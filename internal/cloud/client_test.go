package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/The-Crocop/storeready/internal/model"
)

func TestPush(t *testing.T) {
	var got Submission
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/scans" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-StoreReady-Key") != "secret" {
			t.Fatalf("missing API key")
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"scan-1","projectKey":"mobile","artifactName":"App.ipa","platform":"ios","sha256":"abc","passed":7,"warnings":1,"blockers":0,"permissions":["camera"],"sdks":[],"createdAt":"2026-10-04T19:00:00Z"}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "secret")
	if err != nil {
		t.Fatal(err)
	}

	record, err := client.Push(context.Background(), "mobile", model.ScanResult{
		Artifact: "App.ipa",
		Platform: "ios",
		SHA256: "abc",
		Summary: model.Summary{Passed: 7, Warnings: 1},
		Metadata: model.Metadata{Permissions: []string{"camera"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != "scan-1" {
		t.Fatalf("id = %q", record.ID)
	}
	if got.ProjectKey != "mobile" || got.ArtifactName != "App.ipa" || got.Warnings != 1 {
		t.Fatalf("unexpected payload: %#v", got)
	}
}

func TestApproveAndFetchBaseline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/baselines":
			var got BaselineSubmission
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if got.ProjectKey != "mobile" || got.Platform != "ios" || len(got.Permissions) != 1 {
				t.Fatalf("unexpected baseline payload: %#v", got)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"baseline-1","projectKey":"mobile","platform":"ios","artifactSha256":"abc","permissions":["camera"],"sdks":[],"approvedAt":"2026-10-04T19:00:00Z"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/baselines":
			if r.URL.Query().Get("projectKey") != "mobile" || r.URL.Query().Get("platform") != "ios" {
				t.Fatalf("unexpected baseline query: %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"id":"baseline-1","projectKey":"mobile","platform":"ios","artifactSha256":"abc","permissions":["camera"],"sdks":[],"approvedAt":"2026-10-04T19:00:00Z"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "secret")
	if err != nil {
		t.Fatal(err)
	}
	result := model.ScanResult{
		Platform: "ios",
		SHA256: "abc",
		Metadata: model.Metadata{Permissions: []string{"camera"}},
	}
	record, err := client.ApproveBaseline(context.Background(), "mobile", result)
	if err != nil {
		t.Fatal(err)
	}
	if record.ID != "baseline-1" {
		t.Fatalf("id = %q", record.ID)
	}

	value, found, err := client.LatestBaseline(context.Background(), "mobile", "ios")
	if err != nil {
		t.Fatal(err)
	}
	if !found || value.ArtifactSHA256 != "abc" || len(value.Permissions) != 1 {
		t.Fatalf("unexpected baseline: found=%v value=%#v", found, value)
	}
}

func TestLatestBaselineNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	client, err := New(server.URL, "secret")
	if err != nil {
		t.Fatal(err)
	}
	_, found, err := client.LatestBaseline(context.Background(), "missing", "android")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("expected no baseline")
	}
}

func TestNewRejectsInvalidURLAndKey(t *testing.T) {
	if _, err := New("storeready.example", "key"); err == nil {
		t.Fatal("expected invalid URL error")
	}
	if _, err := New("https://example.com", ""); err == nil {
		t.Fatal("expected missing API key error")
	}
}
