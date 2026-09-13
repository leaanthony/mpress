package operations

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/leaanthony/mpress/internal/check"
	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/deploy"
	"github.com/leaanthony/mpress/internal/site"
)

type CheckResult struct {
	Build       site.BuildResult   `json:"build"`
	Broken      []check.BrokenLink `json:"broken"`
	Performance CheckPerformance   `json:"performance"`
}

type CheckPerformance struct {
	StartedAt  time.Time   `json:"startedAt"`
	DurationMS float64     `json:"durationMs"`
	Steps      []CheckStep `json:"steps"`
}

type CheckStep struct {
	Name       string  `json:"name"`
	Label      string  `json:"label"`
	OffsetMS   float64 `json:"offsetMs"`
	DurationMS float64 `json:"durationMs"`
	Status     string  `json:"status"`
	Detail     string  `json:"detail,omitempty"`
}

func Check(project string, drafts bool) (response CheckResult, checkErr error) {
	started := time.Now()
	response.Performance.StartedAt = started
	defer func() {
		response.Performance.DurationMS = milliseconds(time.Since(started))
	}()
	collector := check.NewCollector()
	result, buildErr := site.Build(project, site.BuildOptions{Strict: true, IncludeDrafts: drafts, MinifyAssets: true, PurgeUnusedCSS: true, LinkCollector: collector})
	response.Build = result
	for _, timing := range result.Timings {
		detail := ""
		if timing.Name == "render" {
			detail = fmt.Sprintf("%d pages generated and indexed", result.Pages)
		}
		response.Performance.Steps = append(response.Performance.Steps, CheckStep{
			Name: timing.Name, Label: timing.Label, OffsetMS: timing.OffsetMS,
			DurationMS: timing.DurationMS, Status: timing.Status, Detail: detail,
		})
	}
	if buildErr != nil {
		return response, buildErr
	}
	linksStarted := time.Now()
	response.Broken = collector.Finalize()
	linksStatus := "passed"
	if len(response.Broken) > 0 {
		linksStatus = "failed"
	}
	linksDetail := fmt.Sprintf("%d generated pages checked", result.Pages)
	if len(response.Broken) > 0 {
		linksDetail = fmt.Sprintf("%d issue(s) found", len(response.Broken))
	}
	response.Performance.Steps = append(response.Performance.Steps, CheckStep{
		Name: "links", Label: "Resolve links and assets", OffsetMS: milliseconds(linksStarted.Sub(started)),
		DurationMS: milliseconds(time.Since(linksStarted)), Status: linksStatus, Detail: linksDetail,
	})
	if len(response.Broken) > 0 {
		return response, fmt.Errorf("%d broken link(s) or asset(s)", len(response.Broken))
	}
	return response, nil
}

func milliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}

func Deploy(ctx context.Context, project, targetName, environment string) (deploy.Result, error) {
	if _, err := Check(project, false); err != nil {
		return deploy.Result{}, fmt.Errorf("deployment checks failed: %w", err)
	}
	cfg, err := config.Load(project)
	if err != nil {
		return deploy.Result{}, err
	}
	if targetName == "" {
		targetName = cfg.Deploy.Default
	}
	if targetName == "" {
		return deploy.Result{}, errors.New("no deployment target is configured")
	}
	target, ok := cfg.Deploy.Targets[targetName]
	if !ok {
		return deploy.Result{}, fmt.Errorf("deployment target %q is not configured", targetName)
	}
	if environment == "" {
		environment = "preview"
	}
	if environment != "preview" && environment != "production" {
		return deploy.Result{}, fmt.Errorf("deployment environment must be preview or production")
	}
	switch target.Provider {
	case "cloudflare-pages":
		return deploy.CloudflarePages(ctx, cfg.OutputPath(project), targetName, environment, target, deploy.Options{Token: os.Getenv("CLOUDFLARE_API_TOKEN")})
	case "netlify":
		return deploy.NetlifyPages(ctx, cfg.OutputPath(project), targetName, environment, target, deploy.NetlifyOptions{Token: os.Getenv("NETLIFY_AUTH_TOKEN")})
	default:
		return deploy.Result{}, fmt.Errorf("unsupported deploy provider %q", target.Provider)
	}
}
