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

	"github.com/The-Crocop/storeready/internal/model"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type Submission struct {
	ProjectKey  string   `json:"projectKey"`
	ArtifactName string   `json:"artifactName"`
	Platform    string   `json:"platform"`
	SHA256      string   `json:"sha256"`
	Passed      int      `json:"passed"`
	Warnings    int      `json:"warnings"`
	Blockers    int      `json:"blockers"`
	Permissions []string `json:"permissions"`
	SDKs        []string `json:"sdks"`
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
	body, err := json.Marshal(payload)
	if err != nil {
		return ScanRecord{}, fmt.Errorf("encode cloud payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/scans", bytes.NewReader(body))
	if err != nil {
		return ScanRecord{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "StoreReady CLI")
	req.Header.Set("X-StoreReady-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ScanRecord{}, fmt.Errorf("upload scan: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		detail := strings.TrimSpace(string(message))
		if detail == "" {
			detail = http.StatusText(resp.StatusCode)
		}
		return ScanRecord{}, fmt.Errorf("cloud returned HTTP %d: %s", resp.StatusCode, detail)
	}

	var record ScanRecord
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&record); err != nil {
		return ScanRecord{}, fmt.Errorf("decode cloud response: %w", err)
	}
	if record.ID == "" {
		return ScanRecord{}, fmt.Errorf("cloud response did not contain a scan id")
	}
	return record, nil
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
