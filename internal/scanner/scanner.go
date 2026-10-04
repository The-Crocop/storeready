package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/The-Crocop/storeready/internal/model"
)

func Scan(path string) (model.ScanResult, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return model.ScanResult{}, err
	}
	if stat.IsDir() {
		return model.ScanResult{}, fmt.Errorf("%s is a directory; expected an IPA or AAB", path)
	}

	sum, err := checksum(path)
	if err != nil {
		return model.ScanResult{}, err
	}
	base := model.ScanResult{
		SchemaVersion: 1,
		Artifact: filepath.Base(path),
		SHA256: sum,
		SizeBytes: stat.Size(),
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".ipa":
		base.Platform = "ios"
		return scanIPA(path, base)
	case ".aab":
		base.Platform = "android"
		return scanAAB(path, base)
	default:
		base.Platform = "unknown"
		base.Findings = append(base.Findings, model.Finding{
			ID: "SR-ARTIFACT-001", Severity: model.SeverityBlocker,
			Title: "Unsupported artifact type",
			Evidence: filepath.Ext(path),
			Fix: "Provide an iOS .ipa or Android .aab release artifact.",
		})
		base.Recount()
		return base, nil
	}
}

func checksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func addPassed(r *model.ScanResult) { r.Summary.Passed++ }

func appendUnique(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}
