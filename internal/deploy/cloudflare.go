package deploy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/leaanthony/mpress/internal/config"
	"github.com/leaanthony/mpress/internal/projectfs"
	"github.com/zeebo/blake3"
)

const (
	cloudflareAPI    = "https://api.cloudflare.com/client/v4"
	cloudflareAssets = "https://api.cloudflare.com/client/v4"
	maxUploadFiles   = 100
	maxUploadBytes   = 48 << 20
)

type Options struct {
	Token     string
	APIURL    string
	AssetsURL string
	Client    *http.Client
	// Files borrows an already pinned output boundary. When nil, the provider
	// opens and closes outputDir itself.
	Files *projectfs.FS
}

type Result struct {
	Provider    string `json:"provider"`
	Target      string `json:"target"`
	Environment string `json:"environment"`
	Deployment  string `json:"deploymentId"`
	URL         string `json:"url"`
	Uploaded    int    `json:"uploaded"`
	Reused      int    `json:"reused"`
}

type asset struct {
	Name        string
	Hash        string
	ContentType string
	Size        int64
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type envelope[T any] struct {
	Success bool       `json:"success"`
	Result  T          `json:"result"`
	Errors  []apiError `json:"errors"`
}

func CloudflarePages(ctx context.Context, outputDir, targetName, environment string, target config.DeployTarget, options Options) (Result, error) {
	if strings.TrimSpace(options.Token) == "" {
		return Result{}, errors.New("CLOUDFLARE_API_TOKEN is not set; create a token with Pages Write permission and restart M-Press")
	}
	if target.Provider != "cloudflare-pages" {
		return Result{}, fmt.Errorf("unsupported deploy provider %q", target.Provider)
	}
	if target.AccountID == "" || target.Project == "" {
		return Result{}, errors.New("Cloudflare accountID and project are required")
	}
	if options.APIURL == "" {
		options.APIURL = cloudflareAPI
	}
	if options.AssetsURL == "" {
		options.AssetsURL = cloudflareAssets
	}
	if options.Client == nil {
		options.Client = &http.Client{Timeout: 90 * time.Second}
	}
	root, closeRoot, err := outputRoot(outputDir, options.Files)
	if err != nil {
		return Result{}, err
	}
	defer closeRoot()
	files, err := collectAssets(root)
	if err != nil {
		return Result{}, err
	}
	if len(files) == 0 {
		return Result{}, errors.New("the build output is empty")
	}
	if len(files) > 20000 {
		return Result{}, fmt.Errorf("Cloudflare Pages accepts at most 20,000 files; the build contains %d", len(files))
	}
	if err := ensurePagesProject(ctx, target, options); err != nil {
		return Result{}, err
	}
	jwt, err := uploadToken(ctx, target, options)
	if err != nil {
		return Result{}, err
	}
	missing, err := missingAssets(ctx, files, jwt, options)
	if err != nil {
		return Result{}, err
	}
	uploaded, err := uploadAssets(ctx, root, files, missing, jwt, options)
	if err != nil {
		return Result{}, err
	}
	_ = upsertHashes(ctx, files, jwt, options)
	deployment, err := createDeployment(ctx, outputDir, environment, target, files, options)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Provider: "cloudflare-pages", Target: targetName, Environment: environment,
		Deployment: deployment.ID, URL: deployment.URL, Uploaded: uploaded, Reused: len(files) - uploaded,
	}, nil
}

func collectAssets(files *projectfs.FS) ([]asset, error) {
	var collected []asset
	err := files.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("cannot deploy non-regular file %s", path)
		}
		if info.Size() > 25<<20 {
			return fmt.Errorf("%s is larger than Cloudflare Pages' 25 MiB file limit", path)
		}
		data, err := files.ReadFile(path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(path)
		contentType := mime.TypeByExtension(filepath.Ext(name))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		collected = append(collected, asset{Name: name, Hash: assetHash(name, data), ContentType: contentType, Size: info.Size()})
		return nil
	})
	sort.Slice(collected, func(i, j int) bool { return collected[i].Name < collected[j].Name })
	return collected, err
}

func assetHash(name string, data []byte) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
	sum := blake3.Sum256([]byte(base64.StdEncoding.EncodeToString(data) + ext))
	return hex.EncodeToString(sum[:16])
}

func ensurePagesProject(ctx context.Context, target config.DeployTarget, options Options) error {
	path := fmt.Sprintf("/accounts/%s/pages/projects/%s", target.AccountID, target.Project)
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, options.APIURL+path, nil)
	request.Header.Set("Authorization", "Bearer "+options.Token)
	response, err := options.Client.Do(request)
	if err != nil {
		return fmt.Errorf("check Cloudflare Pages project: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return nil
	}
	if response.StatusCode != http.StatusNotFound {
		return cloudflareResponseError("check Cloudflare Pages project", response)
	}
	branch := target.ProductionBranch
	if branch == "" {
		branch = "main"
	}
	body, _ := json.Marshal(map[string]string{"name": target.Project, "production_branch": branch})
	return call(ctx, http.MethodPost, options.APIURL+fmt.Sprintf("/accounts/%s/pages/projects", target.AccountID), options.Token, bytes.NewReader(body), "application/json", nil, options.Client)
}

func uploadToken(ctx context.Context, target config.DeployTarget, options Options) (string, error) {
	var result struct {
		JWT string `json:"jwt"`
	}
	path := fmt.Sprintf("/accounts/%s/pages/projects/%s/upload-token", target.AccountID, target.Project)
	if err := call(ctx, http.MethodGet, options.APIURL+path, options.Token, nil, "", &result, options.Client); err != nil {
		return "", fmt.Errorf("get Cloudflare upload token: %w", err)
	}
	if result.JWT == "" {
		return "", errors.New("Cloudflare returned an empty asset upload token")
	}
	return result.JWT, nil
}

func missingAssets(ctx context.Context, files []asset, jwt string, options Options) (map[string]bool, error) {
	hashes := make([]string, 0, len(files))
	for _, file := range files {
		hashes = append(hashes, file.Hash)
	}
	body, _ := json.Marshal(map[string]any{"hashes": hashes})
	var missing []string
	if err := call(ctx, http.MethodPost, options.AssetsURL+"/pages/assets/check-missing", jwt, bytes.NewReader(body), "application/json", &missing, options.Client); err != nil {
		return nil, fmt.Errorf("check Cloudflare assets: %w", err)
	}
	set := make(map[string]bool, len(missing))
	for _, hash := range missing {
		set[hash] = true
	}
	return set, nil
}

func uploadAssets(ctx context.Context, files *projectfs.FS, assets []asset, missing map[string]bool, jwt string, options Options) (int, error) {
	type metadata struct {
		ContentType string `json:"contentType"`
	}
	type item struct {
		Key      string   `json:"key"`
		Value    string   `json:"value"`
		Metadata metadata `json:"metadata"`
		Base64   bool     `json:"base64"`
	}
	var batch []item
	batchBytes := int64(0)
	uploaded := 0
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		body, _ := json.Marshal(batch)
		if err := call(ctx, http.MethodPost, options.AssetsURL+"/pages/assets/upload", jwt, bytes.NewReader(body), "application/json", nil, options.Client); err != nil {
			return fmt.Errorf("upload Cloudflare assets: %w", err)
		}
		uploaded += len(batch)
		batch = nil
		batchBytes = 0
		return nil
	}
	for _, file := range assets {
		if !missing[file.Hash] {
			continue
		}
		if len(batch) >= maxUploadFiles || batchBytes+file.Size > maxUploadBytes {
			if err := flush(); err != nil {
				return uploaded, err
			}
		}
		data, err := files.ReadFile(filepath.FromSlash(file.Name))
		if err != nil {
			return uploaded, err
		}
		if assetHash(file.Name, data) != file.Hash {
			return uploaded, fmt.Errorf("%s changed after its content hash was collected; rebuild before deploying", file.Name)
		}
		batch = append(batch, item{Key: file.Hash, Value: base64.StdEncoding.EncodeToString(data), Metadata: metadata{ContentType: file.ContentType}, Base64: true})
		batchBytes += file.Size
	}
	return uploaded, flush()
}

func upsertHashes(ctx context.Context, files []asset, jwt string, options Options) error {
	hashes := make([]string, 0, len(files))
	for _, file := range files {
		hashes = append(hashes, file.Hash)
	}
	body, _ := json.Marshal(map[string]any{"hashes": hashes})
	return call(ctx, http.MethodPost, options.AssetsURL+"/pages/assets/upsert-hashes", jwt, bytes.NewReader(body), "application/json", nil, options.Client)
}

type deploymentResponse struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Environment string `json:"environment"`
}

func createDeployment(ctx context.Context, outputDir, environment string, target config.DeployTarget, files []asset, options Options) (deploymentResponse, error) {
	manifest := make(map[string]string, len(files))
	for _, file := range files {
		manifest["/"+file.Name] = file.Hash
	}
	manifestJSON, _ := json.Marshal(manifest)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("manifest", string(manifestJSON))
	_ = writer.WriteField("pages_build_output_dir", filepath.Base(outputDir))
	_ = writer.WriteField("commit_dirty", "true")
	_ = writer.WriteField("commit_message", "Deploy from M-Press")
	branch := target.ProductionBranch
	if branch == "" {
		branch = "main"
	}
	if environment != "production" {
		branch = "mpress-preview"
	}
	_ = writer.WriteField("branch", branch)
	_ = writer.Close()
	path := fmt.Sprintf("/accounts/%s/pages/projects/%s/deployments", target.AccountID, target.Project)
	var result deploymentResponse
	if err := call(ctx, http.MethodPost, options.APIURL+path, options.Token, &body, writer.FormDataContentType(), &result, options.Client); err != nil {
		return result, fmt.Errorf("create Cloudflare deployment: %w", err)
	}
	if result.URL == "" {
		return result, errors.New("Cloudflare created the deployment without returning a URL")
	}
	return result, nil
}

func call(ctx context.Context, method, url, token string, body io.Reader, contentType string, result any, client *http.Client) error {
	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return cloudflareResponseError("Cloudflare request", response)
	}
	if result == nil {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil
	}
	var payload envelope[json.RawMessage]
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return err
	}
	if !payload.Success {
		if len(payload.Errors) > 0 {
			return errors.New(payload.Errors[0].Message)
		}
		return errors.New("Cloudflare request failed")
	}
	return json.Unmarshal(payload.Result, result)
}

func cloudflareResponseError(action string, response *http.Response) error {
	data, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	var payload envelope[json.RawMessage]
	if json.Unmarshal(data, &payload) == nil && len(payload.Errors) > 0 {
		return fmt.Errorf("%s: %s", action, payload.Errors[0].Message)
	}
	message := strings.TrimSpace(string(data))
	if message == "" {
		message = response.Status
	}
	return fmt.Errorf("%s: %s", action, message)
}
