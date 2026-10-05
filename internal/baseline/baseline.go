package baseline

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/The-Crocop/storepreflight/internal/model"
)

type Baseline struct {
	SchemaVersion  int      `json:"schemaVersion"`
	ArtifactSHA256 string   `json:"artifactSha256"`
	Platform       string   `json:"platform"`
	Permissions    []string `json:"permissions,omitempty"`
	SDKs           []string `json:"sdks,omitempty"`
	FindingIDs     []string `json:"findingIds,omitempty"`
}

func FromScan(result model.ScanResult) Baseline {
	ids := make([]string, 0, len(result.Findings))
	for _, f := range result.Findings {
		ids = append(ids, f.ID)
	}
	return Baseline{
		SchemaVersion: 1,
		ArtifactSHA256: result.SHA256,
		Platform: result.Platform,
		Permissions: result.Metadata.Permissions,
		SDKs: result.Metadata.SDKs,
		FindingIDs: ids,
	}
}

func Save(path string, value Baseline) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write baseline: %w", err)
	}
	return nil
}

func Load(path string) (Baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Baseline{}, err
	}
	var value Baseline
	if err := json.Unmarshal(data, &value); err != nil {
		return Baseline{}, fmt.Errorf("parse baseline: %w", err)
	}
	return value, nil
}

func ApplyDiff(result *model.ScanResult, old Baseline) {
	if old.Platform != "" && old.Platform != result.Platform {
		result.Findings = append(result.Findings, model.Finding{
			ID: "SPF-DIFF-000", Severity: model.SeverityBlocker,
			Title: "Baseline platform does not match artifact",
			Evidence: old.Platform + " -> " + result.Platform,
		})
		return
	}
	for _, p := range difference(result.Metadata.Permissions, old.Permissions) {
		result.Findings = append(result.Findings, model.Finding{
			ID: "SPF-DIFF-001-" + p, Severity: model.SeverityWarning,
			Title: "New privacy-sensitive permission since approved baseline",
			Evidence: p,
			Fix: "Confirm store privacy/data-safety metadata and review notes cover this new capability.",
		})
	}
	for _, sdk := range difference(result.Metadata.SDKs, old.SDKs) {
		result.Findings = append(result.Findings, model.Finding{
			ID: "SPF-DIFF-002-" + sdk, Severity: model.SeverityWarning,
			Title: "New SDK signal since approved baseline",
			Evidence: sdk,
			Fix: "Review privacy disclosures, SDK manifests and store declarations for this dependency.",
		})
	}
}

func difference(current, previous []string) []string {
	seen := make(map[string]bool, len(previous))
	for _, item := range previous {
		seen[item] = true
	}
	var out []string
	for _, item := range current {
		if !seen[item] {
			out = append(out, item)
		}
	}
	return out
}
