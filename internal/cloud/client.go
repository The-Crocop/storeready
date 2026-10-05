package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/The-Crocop/storepreflight/internal/baseline"
	"github.com/The-Crocop/storepreflight/internal/model"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type Submission struct {
	ProjectKey   string   `json:"projectKey"`
	ArtifactName string   `json:"artifactName"`
	Platform     string   `json:"platform"`
	SHA256       string   `json:"sha256"`
	Passed       int      `json:"passed"`
	Warnings     int      `json:"warnings"`
	Blockers     int      `json:"blockers"`
	Permissions  []string `json:"permissions"`
	SDKs         []string `json:"sdks"`
}

type ScanRecord struct {
	ID           string   `json:"id"`
	ProjectKey   string   `json:"projectKey"`
	ArtifactName string   `json:"artifactName"`
	Platform     string   `json:"platform"`
	SHA256       string   `json:"sha256"`
	Passed       int      `json:"passed"`
	Warnings     int      `json:"warnings"`
	Blockers     int      `json:"blockers"`
	Permissions  []string `json:"permissions"`
	SDKs         []string `json:"sdks"`
	CreatedAt    string   `json:"createdAt"`
}

type BaselineSubmission struct {
	ProjectKey     string   `json:"projectKey"`
	Platform       string   `json:"platform"`
	ArtifactSHA256 string   `json:"artifactSha256"`
	Permissions    []string `json:"permissions"`
	SDKs           []string `json:"sdks"`
}

type BaselineRecord struct {
	ID             string   `json:"id"`
	ProjectKey     string   `json:"projectKey"`
	Platform       string   `json:"platform"`
	ArtifactSHA256 string   `json:"artifactSha256"`
	Permissions    []string `json:"permissions"`
	SDKs           []string `json:"sdks"`
	ApprovedAt     string   `json:"approvedAt"`
}

func New(baseURL, apiKey string) (*Client, error) {
	baseURL = strings.TrimSpace(strings.TrimRight(baseURL, "/"))
	if baseURL == "" {
		return nil, fmt.Errorf("cloud URL is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("invalid cloud URL %q", baseURL)
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("cloud API key is required")
	}
	return &Client{
		baseURL: baseURL,
		apiKey: strings.TrimSpace(apiKey),
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}, nil
}

func (c *Client) Push(ctx context.Context, project string, result model.ScanResult) (ScanRecord, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		return ScanRecord{}, fmt.Errorf("project key is required")
	}
	payload := Submission{
		ProjectKey: project,
		ArtifactName: result.Artifact,
		Platform: result.Platform,
		SHA256: result.SHA256,
		Passed: result.Summary.Passed,
		Warnings: result.Summary.Warnings,
		Blockers: result.Summary.Blockers,
		Permissions: nonNil(result.Metadata.Permissions),
		SDKs: nonNil(result.Metadata.SDKs),
	}
	var record ScanRecord
	if err := c.postJSON(ctx, "/api/v1/scans", payload, &record); err != nil {
		return ScanRecord{}, err
	}
	if record.ID == "" {
		return ScanRecord{}, fmt.Errorf("cloud response did not contain a scan id")
	}
	return record, nil
}

func (c *Client) ApproveBaseline(ctx context.Context, project string, result model.ScanResult) (BaselineRecord, error) {
	project = strings.TrimSpace(project)
	if project == "" {
		return BaselineRecord{}, fmt.Errorf("project key is required")
	}
	payload := BaselineSubmission{
		ProjectKey: project,
		Platform: result.Platform,
		ArtifactSHA256: result.SHA256,
		Permissions: nonNil(result.Metadata.Permissions),
		SDKs: nonNil(result.Metadata.SDKs),
	}
	var record BaselineRecord
	if err := c.postJSON(ctx, "/api/v1/baselines", payload, &record); err != nil {
		return BaselineRecord{}, err
	}
	if record.ID == "" {
		return BaselineRecord{}, fmt.Errorf("cloud response did not contain a baseline id")
	}
	return record, nil
}

func (c *Client) LatestBaseline(ctx context.Context, project, platform string) (baseline.Baseline, bool, error) {
	values := url.Values{}
	values.Set("projectKey", strings.TrimSpace(project))
	values.Set("platform", strings.TrimSpace(platform))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v1/baselines?"+values.Encode(),
		nil,
	)
	if err != nil {
		return baseline.Baseline{}, false, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return baseline.Baseline{}, false, fmt.Errorf("fetch cloud baseline: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return baseline.Baseline{}, false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return baseline.Baseline{}, false, responseError(resp)
	}

	var record BaselineRecord
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&record); err != nil {
		return baseline.Baseline{}, false, fmt.Errorf("decode cloud baseline: %w", err)
	}
	return baseline.Baseline{
		SchemaVersion: 1,
		ArtifactSHA256: record.ArtifactSHA256,
		Platform: record.Platform,
		Permissions: record.Permissions,
		SDKs: record.SDKs,
	}, true, nil
}

func (c *Client) postJSON(ctx context.Context, path string, payload any, target any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode cloud payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cloud request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseError(resp)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(target); err != nil {
		return fmt.Errorf("decode cloud response: %w", err)
	}
	return nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "StorePreflight CLI")
	req.Header.Set("X-StorePreflight-Key", c.apiKey)
}

func responseError(resp *http.Response) error {
	message, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	detail := strings.TrimSpace(string(message))
	if detail == "" {
		detail = http.StatusText(resp.StatusCode)
	}
	return fmt.Errorf("cloud returned HTTP %d: %s", resp.StatusCode, detail)
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
