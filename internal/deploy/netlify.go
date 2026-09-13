package deploy

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/leaanthony/mpress/internal/config"
)

const (
	netlifyAPI       = "https://api.netlify.com/api/v1"
	maxNetlifyFiles  = 25000
	maxNetlifyZip    = 100 << 20
	netlifyPollEvery = time.Second
)

// NetlifyOptions contains the credential and transport used by NetlifyPages.
// The token is never read from project configuration.
type NetlifyOptions struct {
	Token  string
	APIURL string
	Client *http.Client
}

type netlifySite struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	DeployURL string `json:"deploy_url"`
}

type netlifyDeploy struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	URL       string `json:"url"`
	DeployURL string `json:"deploy_url"`
	Error     string `json:"error_message"`
}

// NetlifyPages publishes the already-built static output as an atomic Netlify
// deploy. Preview deploys are draft deploys and do not replace production.
func NetlifyPages(ctx context.Context, outputDir, targetName, environment string, target config.DeployTarget, options NetlifyOptions) (Result, error) {
	if strings.TrimSpace(options.Token) == "" {
		return Result{}, errors.New("NETLIFY_AUTH_TOKEN is not set; create a Netlify personal access token and restart M-Press")
	}
	if target.Provider != "netlify" {
		return Result{}, fmt.Errorf("unsupported deploy provider %q", target.Provider)
	}
	if strings.TrimSpace(target.Project) == "" {
		return Result{}, errors.New("Netlify site ID, domain, or name is required")
	}
	if environment == "" {
		environment = "preview"
	}
	if environment != "preview" && environment != "production" {
		return Result{}, fmt.Errorf("deployment environment must be preview or production")
	}
	if options.APIURL == "" {
		options.APIURL = netlifyAPI
	}
	if options.Client == nil {
		options.Client = &http.Client{Timeout: 90 * time.Second}
	}
	archive, files, err := zipOutput(outputDir)
	if err != nil {
		return Result{}, err
	}
	if len(archive) > maxNetlifyZip {
		return Result{}, fmt.Errorf("Netlify deploy archive is larger than %d MiB", maxNetlifyZip>>20)
	}
	site, err := ensureNetlifySite(ctx, target, options)
	if err != nil {
		return Result{}, err
	}
	path := "/sites/" + url.PathEscape(site.ID) + "/deploys"
	if environment != "production" {
		path += "?draft=true"
	}
	var deployment netlifyDeploy
	if err := netlifyRequest(ctx, http.MethodPost, options.APIURL+path, options.Token, bytes.NewReader(archive), "application/zip", &deployment, options); err != nil {
		return Result{}, fmt.Errorf("create Netlify deployment: %w", err)
	}
	deployment, err = waitNetlifyDeploy(ctx, deployment, site.ID, options)
	if err != nil {
		return Result{}, err
	}
	deployURL := deployment.DeployURL
	if deployURL == "" {
		deployURL = deployment.URL
	}
	if deployURL == "" {
		deployURL = site.DeployURL
	}
	if deployURL == "" {
		deployURL = site.URL
	}
	if deployURL == "" {
		return Result{}, errors.New("Netlify created the deployment without returning a URL")
	}
	return Result{Provider: "netlify", Target: targetName, Environment: environment, Deployment: deployment.ID, URL: deployURL, Uploaded: files, Reused: 0}, nil
}

func ensureNetlifySite(ctx context.Context, target config.DeployTarget, options NetlifyOptions) (netlifySite, error) {
	identifier := url.PathEscape(strings.TrimSpace(target.Project))
	var site netlifySite
	err := netlifyRequest(ctx, http.MethodGet, options.APIURL+"/sites/"+identifier, options.Token, nil, "", &site, options)
	if err == nil {
		if site.ID == "" {
			site.ID = target.Project
		}
		return site, nil
	}
	if !strings.Contains(err.Error(), "404") && !strings.Contains(strings.ToLower(err.Error()), "not found") {
		return netlifySite{}, fmt.Errorf("find Netlify site %q: %w", target.Project, err)
	}
	createPath := "/sites"
	if strings.TrimSpace(target.AccountID) != "" {
		createPath = "/" + url.PathEscape(strings.TrimSpace(target.AccountID)) + "/sites/"
	}
	body, _ := json.Marshal(map[string]string{"name": strings.TrimSpace(target.Project)})
	if err := netlifyRequest(ctx, http.MethodPost, options.APIURL+createPath, options.Token, bytes.NewReader(body), "application/json", &site, options); err != nil {
		return netlifySite{}, fmt.Errorf("create Netlify site %q: %w", target.Project, err)
	}
	if site.ID == "" {
		return netlifySite{}, errors.New("Netlify created a site without returning an ID")
	}
	return site, nil
}

func waitNetlifyDeploy(ctx context.Context, deployment netlifyDeploy, siteID string, options NetlifyOptions) (netlifyDeploy, error) {
	if deployment.ID == "" {
		return netlifyDeploy{}, errors.New("Netlify created a deployment without returning an ID")
	}
	if deployment.State == "ready" || deployment.State == "uploaded" {
		return deployment, nil
	}
	deadline := time.NewTimer(90 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(netlifyPollEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return netlifyDeploy{}, ctx.Err()
		case <-deadline.C:
			return netlifyDeploy{}, errors.New("timed out waiting for Netlify deployment")
		case <-ticker.C:
			var current netlifyDeploy
			path := options.APIURL + "/sites/" + url.PathEscape(siteID) + "/deploys/" + url.PathEscape(deployment.ID)
			if err := netlifyRequest(ctx, http.MethodGet, path, options.Token, nil, "", &current, options); err != nil {
				return netlifyDeploy{}, fmt.Errorf("poll Netlify deployment: %w", err)
			}
			switch current.State {
			case "ready", "uploaded":
				return current, nil
			case "error", "failed":
				if current.Error == "" {
					current.Error = "Netlify reported a failed deployment"
				}
				return netlifyDeploy{}, errors.New(current.Error)
			}
		}
	}
}

func zipOutput(root string) ([]byte, int, error) {
	var body bytes.Buffer
	archive := zip.NewWriter(&body)
	files := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		files++
		if files > maxNetlifyFiles {
			return fmt.Errorf("Netlify deploy contains more than %d files", maxNetlifyFiles)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		header.Method = zip.Deflate
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		_ = archive.Close()
		return nil, 0, err
	}
	if err := archive.Close(); err != nil {
		return nil, 0, err
	}
	return body.Bytes(), files, nil
}

func netlifyRequest(ctx context.Context, method, endpoint, token string, body io.Reader, contentType string, result any, options NetlifyOptions) error {
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("User-Agent", "M-Press static documentation deploy")
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response, err := options.Client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		message := strings.TrimSpace(string(data))
		if message == "" {
			message = response.Status
		}
		return fmt.Errorf("Netlify request: %s: %s", response.Status, message)
	}
	if result == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(result)
}
