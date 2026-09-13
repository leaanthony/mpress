package lighthouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const defaultTimeout = 3 * time.Minute

type Options struct {
	FormFactor string
	Timeout    time.Duration
}

type Result struct {
	URL         string       `json:"url"`
	FormFactor  string       `json:"formFactor"`
	Version     string       `json:"version"`
	FetchedAt   string       `json:"fetchedAt"`
	DurationMS  int64        `json:"durationMs"`
	Categories  []Category   `json:"categories"`
	Metrics     []Metric     `json:"metrics"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type Category struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Score int    `json:"score"`
}

type Metric struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	DisplayValue string  `json:"displayValue"`
	Score        float64 `json:"score"`
}

type Diagnostic struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	DisplayValue string `json:"displayValue,omitempty"`
	Score        int    `json:"score"`
}

type report struct {
	FinalDisplayedURL string                    `json:"finalDisplayedUrl"`
	LighthouseVersion string                    `json:"lighthouseVersion"`
	FetchTime         string                    `json:"fetchTime"`
	Categories        map[string]reportCategory `json:"categories"`
	Audits            map[string]reportAudit    `json:"audits"`
}

type reportCategory struct {
	Title string   `json:"title"`
	Score *float64 `json:"score"`
}

type reportAudit struct {
	Title        string   `json:"title"`
	Score        *float64 `json:"score"`
	ScoreDisplay string   `json:"scoreDisplayMode"`
	DisplayValue string   `json:"displayValue"`
}

var categoryOrder = []string{"performance", "accessibility", "best-practices", "seo"}
var metricOrder = []string{"first-contentful-paint", "largest-contentful-paint", "total-blocking-time", "cumulative-layout-shift", "speed-index"}

func Run(ctx context.Context, targetURL string, options Options) (Result, error) {
	if strings.TrimSpace(targetURL) == "" {
		return Result{}, errors.New("a page URL is required")
	}
	formFactor := strings.ToLower(strings.TrimSpace(options.FormFactor))
	if formFactor == "" {
		formFactor = "mobile"
	}
	if formFactor != "mobile" && formFactor != "desktop" {
		return Result{}, errors.New("Lighthouse form factor must be mobile or desktop")
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	name, prefix, err := command()
	if err != nil {
		return Result{}, err
	}
	args := append(prefix,
		targetURL,
		"--output=json",
		"--output-path=stdout",
		"--quiet",
		"--only-categories=performance,accessibility,best-practices,seo",
		"--disable-full-page-screenshot",
		`--extra-headers={"X-MPress-Audit":"lighthouse"}`,
		"--chrome-flags=--headless=new --disable-gpu --disable-dev-shm-usage",
	)
	if formFactor == "desktop" {
		args = append(args, "--preset=desktop")
	}
	started := time.Now()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "LIGHTHOUSE_CI=true")
	var stderr strings.Builder
	cmd.Stderr = &stderr
	output, runErr := cmd.Output()
	if ctx.Err() == context.DeadlineExceeded {
		return Result{}, fmt.Errorf("Lighthouse did not finish within %s", timeout.Round(time.Second))
	}
	if runErr != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = runErr.Error()
		}
		return Result{}, fmt.Errorf("Lighthouse failed: %s", compactError(message))
	}
	result, err := Parse(output)
	if err != nil {
		return Result{}, err
	}
	result.FormFactor = formFactor
	result.DurationMS = time.Since(started).Milliseconds()
	return result, nil
}

func command() (string, []string, error) {
	if configured := strings.TrimSpace(os.Getenv("MPRESS_LIGHTHOUSE")); configured != "" {
		path, err := exec.LookPath(configured)
		if err != nil {
			return "", nil, fmt.Errorf("configured Lighthouse executable %q was not found", configured)
		}
		return path, nil, nil
	}
	if path, err := exec.LookPath("lighthouse"); err == nil {
		return path, nil, nil
	}
	if path, err := exec.LookPath("npx"); err == nil {
		if resolved, resolveErr := filepath.EvalSymlinks(path); resolveErr == nil && strings.HasSuffix(resolved, ".js") {
			if node, nodeErr := exec.LookPath("node"); nodeErr == nil {
				return node, []string{resolved, "--yes", "lighthouse@latest"}, nil
			}
		}
		return path, []string{"--yes", "lighthouse@latest"}, nil
	}
	return "", nil, errors.New("Lighthouse is unavailable. Install Node.js 22 or later and run `npm install -g lighthouse`, then try again. M-Press itself does not require Node.js")
}

func Parse(data []byte) (Result, error) {
	var source report
	if err := json.Unmarshal(data, &source); err != nil {
		return Result{}, fmt.Errorf("could not read the Lighthouse report: %w", err)
	}
	result := Result{URL: source.FinalDisplayedURL, Version: source.LighthouseVersion, FetchedAt: source.FetchTime}
	for _, id := range categoryOrder {
		category, ok := source.Categories[id]
		if !ok || category.Score == nil {
			continue
		}
		result.Categories = append(result.Categories, Category{ID: id, Title: category.Title, Score: percentage(*category.Score)})
	}
	for _, id := range metricOrder {
		audit, ok := source.Audits[id]
		if !ok || audit.Score == nil {
			continue
		}
		result.Metrics = append(result.Metrics, Metric{ID: id, Title: audit.Title, DisplayValue: audit.DisplayValue, Score: *audit.Score})
	}
	for id, audit := range source.Audits {
		if audit.Score == nil || *audit.Score >= 0.9 || audit.ScoreDisplay == "notApplicable" || audit.ScoreDisplay == "manual" || contains(metricOrder, id) {
			continue
		}
		result.Diagnostics = append(result.Diagnostics, Diagnostic{ID: id, Title: audit.Title, DisplayValue: audit.DisplayValue, Score: percentage(*audit.Score)})
	}
	sort.Slice(result.Diagnostics, func(i, j int) bool {
		if result.Diagnostics[i].Score != result.Diagnostics[j].Score {
			return result.Diagnostics[i].Score < result.Diagnostics[j].Score
		}
		return result.Diagnostics[i].Title < result.Diagnostics[j].Title
	})
	if len(result.Diagnostics) > 8 {
		result.Diagnostics = result.Diagnostics[:8]
	}
	if len(result.Categories) == 0 {
		return Result{}, errors.New("the Lighthouse report did not contain category scores")
	}
	return result, nil
}

func percentage(value float64) int {
	rounded, _ := strconv.Atoi(fmt.Sprintf("%.0f", value*100))
	return rounded
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func compactError(value string) string {
	lines := strings.Fields(value)
	message := strings.Join(lines, " ")
	if len(message) > 420 {
		return message[:417] + "..."
	}
	return message
}
