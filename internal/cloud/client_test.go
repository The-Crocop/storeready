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

func TestNewRejectsInvalidURLAndKey(t *testing.T) {
	if _, err := New("storeready.example", "key"); err == nil {
		t.Fatal("expected invalid URL error")
	}
	if _, err := New("https://example.com", ""); err == nil {
		t.Fatal("expected missing API key error")
	}
}
