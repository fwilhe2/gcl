package gcl

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Platform string

const (
	Unknown          Platform = "Unknown"
	GitLab           Platform = "GitLab"
	Forgejo          Platform = "Forgejo"
	Gogs             Platform = "Gogs"
	GitHubEnterprise Platform = "GitHub Enterprise"
	CGit             Platform = "cgit"
)

// DetectPlatform tries to identify the Git hosting software behind a URL.
func DetectPlatform(rawURL string) (Platform, error) {
	base, err := normalizeBaseURL(rawURL)
	if err != nil {
		return Unknown, err
	}

	client := &http.Client{
		Timeout: 8 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// --- 1) Check HTTP headers for GitLab fingerprint ---
	req, _ := http.NewRequestWithContext(ctx, "HEAD", base, nil)
	resp, err := client.Do(req)
	if err == nil {
		for h := range resp.Header {
			if strings.HasPrefix(strings.ToLower(h), "x-gitlab") {
				return GitLab, nil
			}
		}
		resp.Body.Close()
	}

	// --- 2) Probe known API endpoints (software fingerprints) ---
	checks := []struct {
		path     string
		platform Platform
		contains string // optional content hint
	}{
		{"/api/v4/version", GitLab, ""},
		{"/api/v1/version", Forgejo, "version"},
		{"/api/v3/meta", GitHubEnterprise, ""},
		{"/api/v1/version", Gogs, "go version"}, // weak hint
	}

	for _, c := range checks {
		ok, body := probe(ctx, client, base+c.path)
		if ok {
			if c.contains == "" || strings.Contains(strings.ToLower(body), strings.ToLower(c.contains)) {
				return c.platform, nil
			}
		}
	}

	// --- 3) HTML fingerprint fallback ---
	ok, body := probe(ctx, client, base)
	if ok {
		l := strings.ToLower(body)
		switch {
		case strings.Contains(l, "gitlab"):
			return GitLab, nil
		case strings.Contains(l, "gitea"):
			return Forgejo, nil
		case strings.Contains(l, "gogs"):
			return Gogs, nil
		case strings.Contains(l, "cgit"):
			return CGit, nil
		}
	}

	return Unknown, nil
}

func probe(ctx context.Context, client *http.Client, url string) (bool, string) {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return true, string(data)
	}
	return false, ""
}

func normalizeBaseURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	u.Path = ""
	u.RawQuery = ""
	u.Fragment = ""
	if !strings.HasSuffix(u.String(), "/") {
		return u.String() + "/", nil
	}
	return u.String(), nil
}
