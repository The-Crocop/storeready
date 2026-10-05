package report

import (
	"encoding/json"
	"github.com/The-Crocop/storepreflight/internal/model"
)

func SARIF(result model.ScanResult) ([]byte, error) {
	type message struct { Text string `json:"text"` }
	type sarifResult struct {
		RuleID string `json:"ruleId"`
		Level string `json:"level"`
		Message message `json:"message"`
	}
	results := make([]sarifResult, 0, len(result.Findings))
	for _, f := range result.Findings {
		level := "note"
		if f.Severity == model.SeverityWarning { level = "warning" }
		if f.Severity == model.SeverityBlocker { level = "error" }
		results = append(results, sarifResult{RuleID:f.ID, Level:level, Message:message{Text:f.Title + evidenceSuffix(f.Evidence)}})
	}
	doc := map[string]any{
		"$schema":"https://json.schemastore.org/sarif-2.1.0.json",
		"version":"2.1.0",
		"runs":[]any{map[string]any{
			"tool":map[string]any{"driver":map[string]any{"name":"StorePreflight"}},
			"results":results,
		}},
	}
	return json.MarshalIndent(doc, "", "  ")
}

func evidenceSuffix(value string) string {
	if value == "" { return "" }
	return ": " + value
}
