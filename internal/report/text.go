package report

import (
	"fmt"
	"strings"

	"github.com/The-Crocop/storepreflight/internal/model"
)

func Text(result model.ScanResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "StorePreflight\n\nArtifact: %s\nPlatform: %s\nSHA256: %s\n\n", result.Artifact, result.Platform, result.SHA256)
	for _, f := range result.Findings {
		fmt.Fprintf(&b, "%s %s — %s\n", symbol(f.Severity), f.ID, f.Title)
		if f.Evidence != "" { fmt.Fprintf(&b, "   Evidence: %s\n", f.Evidence) }
		if f.Fix != "" { fmt.Fprintf(&b, "   Fix: %s\n", f.Fix) }
	}
	fmt.Fprintf(&b, "\n%d passed · %d info · %d warnings · %d blockers\n", result.Summary.Passed, result.Summary.Info, result.Summary.Warnings, result.Summary.Blockers)
	if result.Summary.Blockers > 0 { b.WriteString("\nNOT READY\n") } else if result.Summary.Warnings > 0 { b.WriteString("\nREADY WITH REVIEW RISKS\n") } else { b.WriteString("\nREADY\n") }
	return b.String()
}

func Markdown(result model.ScanResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## StorePreflight release preflight\n\n**%s** · %s · `%s`\n\n", result.Artifact, result.Platform, short(result.SHA256))
	fmt.Fprintf(&b, "**%d passed · %d warnings · %d blockers**\n\n", result.Summary.Passed, result.Summary.Warnings, result.Summary.Blockers)
	for _, f := range result.Findings {
		fmt.Fprintf(&b, "- %s **%s** — %s", symbol(f.Severity), f.ID, f.Title)
		if f.Evidence != "" { fmt.Fprintf(&b, " — %s", f.Evidence) }
		b.WriteString("\n")
	}
	if result.Summary.Blockers > 0 { b.WriteString("\n### ❌ Not ready to submit\n") } else if result.Summary.Warnings > 0 { b.WriteString("\n### ⚠️ Ready with review risks\n") } else { b.WriteString("\n### ✅ Ready to submit\n") }
	return b.String()
}

func symbol(s model.Severity) string {
	switch s {
	case model.SeverityBlocker: return "❌"
	case model.SeverityWarning: return "⚠️"
	default: return "ℹ️"
	}
}

func short(value string) string {
	if len(value) <= 12 { return value }
	return value[:12]
}
