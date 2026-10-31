package check

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"
)

// ExternalOptions opts into HTTP requests against references in existing HTML.
// Hosts restricts initial destinations, not redirect targets. Client is optional
// so callers can use a controlled transport; its redirect/timeout policy is replaced.
type ExternalOptions struct {
	BaseURL      string
	Hosts        []string
	URLPrefixes  []string
	Timeout      time.Duration
	Retries      int
	RetryDelay   time.Duration
	MaxRedirects int
	Client       *http.Client
}

type ExternalLink struct {
	URL        string   `json:"url"`
	Pages      []string `json:"pages"`
	Status     string   `json:"status"`
	HTTPStatus int      `json:"http_status,omitempty"`
	FinalURL   string   `json:"final_url,omitempty"`
	Attempts   int      `json:"attempts"`
	Detail     string   `json:"detail,omitempty"`
}

type ExternalReport struct {
	HTMLPages int            `json:"html_pages"`
	Links     []ExternalLink `json:"links"`
}

// ExitCode distinguishes missing destinations from incomplete network evidence.
// Broken takes precedence when both kinds occur. Operational/usage errors use 1.
func (r ExternalReport) ExitCode() int {
	if slices.ContainsFunc(r.Links, func(link ExternalLink) bool { return link.Status == "broken" }) {
		return 1
	}
	if slices.ContainsFunc(r.Links, func(link ExternalLink) bool { return link.Status == "inconclusive" }) {
		return 2
	}
	return 0
}

// External checks HTTP availability only. It does not interpret destination
// content, GitHub tree/blob semantics, fragments, JavaScript or CSS references.
// It never builds, edits source, caches results or refreshes audit exceptions.
func External(ctx context.Context, output string, options ExternalOptions) (ExternalReport, error) {
	report := ExternalReport{Links: []ExternalLink{}}
	if options.Timeout <= 0 || options.Retries < 0 || options.Retries > 5 || options.RetryDelay < 0 || options.MaxRedirects < 0 || options.MaxRedirects > 10 {
		return report, fmt.Errorf("external check requires positive timeout, 0–5 retries, nonnegative delay and 0–10 redirects")
	}
	for _, prefix := range options.URLPrefixes {
		parsed, err := url.Parse(prefix)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return report, fmt.Errorf("URL prefix must be an absolute HTTP(S) URL without credentials, query or fragment")
		}
	}
	base, err := url.Parse(options.BaseURL)
	if err != nil {
		return report, err
	}
	collector := NewCollector()
	if err := collector.IndexOutput(output); err != nil {
		return report, err
	}
	report.HTMLPages = len(collector.index.pages)
	refs := collectExternalRefs(collector.index.pages, base.Host, options.Hosts, options.URLPrefixes)
	client := http.Client{}
	if options.Client != nil {
		client = *options.Client
	}
	client.Timeout = options.Timeout
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) > options.MaxRedirects {
			return fmt.Errorf("redirect limit (%d) reached", options.MaxRedirects)
		}
		if req.URL.Scheme != "http" && req.URL.Scheme != "https" || req.URL.User != nil {
			return fmt.Errorf("unsupported redirect destination")
		}
		return nil
	}
	keys := make([]string, 0, len(refs))
	for key := range refs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		link := probeExternal(ctx, &client, key, options)
		for page := range refs[key] {
			link.Pages = append(link.Pages, page)
		}
		sort.Strings(link.Pages)
		report.Links = append(report.Links, link)
	}
	return report, nil
}

func externalHostAllowed(host string, hosts []string) bool {
	if len(hosts) == 0 {
		return true
	}
	for _, allowed := range hosts {
		if strings.EqualFold(host, allowed) {
			return true
		}
	}
	return false
}

func probeExternal(ctx context.Context, client *http.Client, destination string, options ExternalOptions) ExternalLink {
	link := ExternalLink{URL: destination, Status: "inconclusive"}
	parsed, _ := url.Parse(destination)
	if parsed.User != nil {
		link.Detail = "URLs containing credentials are not requested"
		return link
	}
	for attempt := 0; attempt <= options.Retries; attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(options.RetryDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				link.Detail = ctx.Err().Error()
				return link
			case <-timer.C:
			}
		}
		link.Attempts++
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, destination, nil)
		if err != nil {
			link.Detail = err.Error()
			return link
		}
		req.Header.Set("User-Agent", "M-Press external-link-audit")
		resp, err := client.Do(req)
		if err != nil {
			link.Detail = err.Error()
			continue
		}
		// Availability depends on headers; unread body-close errors do not
		// establish whether the destination exists.
		_ = resp.Body.Close()
		link.HTTPStatus = resp.StatusCode
		link.FinalURL = resp.Request.URL.String()
		link.Detail = resp.Status
		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			link.Status = "ok"
			return link
		case resp.StatusCode == 404 || resp.StatusCode == 410:
			link.Status = "broken"
			return link
		}
		// Never rush a server that explicitly asks us to wait. A later audit can
		// retry; other transient failures get the configured bounded retry count.
		if after := resp.Header.Get("Retry-After"); after != "" {
			link.Detail += "; Retry-After: " + after + " (deferred to next audit)"
			return link
		}
		if resp.StatusCode != 408 && resp.StatusCode != 425 && resp.StatusCode != 429 && resp.StatusCode < 500 {
			return link
		}
	}
	return link
}

func collectExternalRefs(pages []indexedPage, baseHost string, hosts, prefixes []string) map[string]map[string]bool {
	refs := map[string]map[string]bool{}
	for _, page := range pages {
		for _, href := range page.refs {
			target, err := url.Parse(strings.TrimSpace(href))
			if err != nil {
				continue
			}
			if target.Scheme == "" && target.Host != "" {
				target.Scheme = "https"
			}
			if target.Scheme != "http" && target.Scheme != "https" || target.Host == "" {
				continue
			}
			if baseHost != "" && strings.EqualFold(target.Host, baseHost) {
				continue
			}
			if !externalHostAllowed(target.Hostname(), hosts) {
				continue
			}
			target.Fragment = ""
			target.RawFragment = ""
			target.Scheme = strings.ToLower(target.Scheme)
			target.Host = strings.ToLower(target.Host)
			key := target.String()
			if len(prefixes) > 0 && !slices.ContainsFunc(prefixes, func(prefix string) bool { return strings.HasPrefix(key, prefix) }) {
				continue
			}
			if refs[key] == nil {
				refs[key] = map[string]bool{}
			}
			refs[key][page.rel] = true
		}
	}
	return refs
}
