package model

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityBlocker Severity = "blocker"
)

type Finding struct {
	ID       string   `json:"id"`
	Severity Severity `json:"severity"`
	Title    string   `json:"title"`
	Evidence string   `json:"evidence,omitempty"`
	Fix      string   `json:"fix,omitempty"`
	Source   string   `json:"source,omitempty"`
}

type Summary struct {
	Passed   int `json:"passed"`
	Info     int `json:"info"`
	Warnings int `json:"warnings"`
	Blockers int `json:"blockers"`
}

type Metadata struct {
	Permissions []string `json:"permissions,omitempty"`
	SDKs        []string `json:"sdks,omitempty"`
}

type ScanResult struct {
	SchemaVersion int       `json:"schemaVersion"`
	Artifact      string    `json:"artifact"`
	Platform      string    `json:"platform"`
	SHA256        string    `json:"sha256"`
	SizeBytes     int64     `json:"sizeBytes"`
	Summary       Summary   `json:"summary"`
	Metadata      Metadata  `json:"metadata"`
	Findings      []Finding `json:"findings"`
}

func (r *ScanResult) Recount() {
	r.Summary.Info = 0
	r.Summary.Warnings = 0
	r.Summary.Blockers = 0
	for _, f := range r.Findings {
		switch f.Severity {
		case SeverityInfo:
			r.Summary.Info++
		case SeverityWarning:
			r.Summary.Warnings++
		case SeverityBlocker:
			r.Summary.Blockers++
		}
	}
}
