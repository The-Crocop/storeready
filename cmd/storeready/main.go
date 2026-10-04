package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/The-Crocop/storeready/internal/baseline"
	"github.com/The-Crocop/storeready/internal/cloud"
	"github.com/The-Crocop/storeready/internal/model"
	"github.com/The-Crocop/storeready/internal/report"
	"github.com/The-Crocop/storeready/internal/scanner"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "scan":
		os.Exit(runScan(os.Args[2:]))
	case "push":
		os.Exit(runPush(os.Args[2:]))
	case "baseline":
		os.Exit(runBaseline(os.Args[2:]))
	case "version", "--version", "-v":
		fmt.Println(version)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `StoreReady - release preflight for iOS and Android

Usage:
  storeready scan [flags] <artifact.ipa|artifact.aab>
  storeready push [flags] <artifact.ipa|artifact.aab>
  storeready baseline [flags] <artifact.ipa|artifact.aab>
  storeready version

Push flags:
  --project <key>       StoreReady Cloud project key (required)
  --url <url>           Cloud base URL; defaults to STOREREADY_CLOUD_URL
  --api-key <key>       Cloud API key; defaults to STOREREADY_API_KEY

Scan and push support --baseline, --format, --output and --fail-on-warning.
Flags must appear before the artifact path.
`)
}

func runScan(args []string) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	format := fs.String("format", "text", "text, json, markdown or sarif")
	output := fs.String("output", "", "write report to file")
	baselinePath := fs.String("baseline", "", "compare against an approved-release baseline")
	failOnWarning := fs.Bool("fail-on-warning", false, "return non-zero when warnings are present")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "scan requires exactly one IPA or AAB artifact")
		return 2
	}

	result, err := scanWithBaseline(fs.Arg(0), *baselinePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan failed:", err)
		return 2
	}
	if err := writeReport(result, *format, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	return resultExitCode(result, *failOnWarning)
}

func runPush(args []string) int {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	project := fs.String("project", "", "StoreReady Cloud project key")
	cloudURL := fs.String("url", envOr("STOREREADY_CLOUD_URL", ""), "StoreReady Cloud base URL")
	apiKey := fs.String("api-key", envOr("STOREREADY_API_KEY", ""), "StoreReady Cloud API key")
	format := fs.String("format", "text", "text, json, markdown or sarif")
	output := fs.String("output", "", "write report to file")
	baselinePath := fs.String("baseline", "", "compare against an approved-release baseline")
	failOnWarning := fs.Bool("fail-on-warning", false, "return non-zero when warnings are present")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "push requires exactly one IPA or AAB artifact")
		return 2
	}
	if strings.TrimSpace(*project) == "" {
		fmt.Fprintln(os.Stderr, "push requires --project")
		return 2
	}

	result, err := scanWithBaseline(fs.Arg(0), *baselinePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan failed:", err)
		return 2
	}

	client, err := cloud.New(*cloudURL, *apiKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cloud:", err)
		return 2
	}
	record, err := client.Push(context.Background(), *project, result)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cloud:", err)
		return 2
	}

	if err := writeReport(result, *format, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	fmt.Fprintf(os.Stderr, "Uploaded StoreReady scan %s for project %s\n", record.ID, record.ProjectKey)
	return resultExitCode(result, *failOnWarning)
}

func runBaseline(args []string) int {
	fs := flag.NewFlagSet("baseline", flag.ContinueOnError)
	output := fs.String("output", "storeready-baseline.json", "baseline file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "baseline requires exactly one IPA or AAB artifact")
		return 2
	}

	result, err := scanner.Scan(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan failed:", err)
		return 2
	}
	if result.Summary.Blockers > 0 {
		fmt.Fprintln(os.Stderr, "refusing to baseline an artifact with blockers")
		return 2
	}

	if err := baseline.Save(*output, baseline.FromScan(result)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	abs, _ := filepath.Abs(*output)
	fmt.Printf("Baseline written to %s\n", abs)
	return 0
}

func scanWithBaseline(artifact, baselinePath string) (model.ScanResult, error) {
	result, err := scanner.Scan(artifact)
	if err != nil {
		return model.ScanResult{}, err
	}
	if baselinePath == "" {
		return result, nil
	}
	value, err := baseline.Load(baselinePath)
	if err != nil {
		return model.ScanResult{}, fmt.Errorf("baseline: %w", err)
	}
	baseline.ApplyDiff(&result, value)
	result.Recount()
	return result, nil
}

func writeReport(result model.ScanResult, format, output string) error {
	var payload []byte
	var err error
	switch strings.ToLower(format) {
	case "text":
		payload = []byte(report.Text(result))
	case "markdown", "md":
		payload = []byte(report.Markdown(result))
	case "json":
		payload, err = json.MarshalIndent(result, "", "  ")
	case "sarif":
		payload, err = report.SARIF(result)
	default:
		return fmt.Errorf("unsupported format %q", format)
	}
	if err != nil {
		return err
	}

	if output == "" {
		fmt.Println(string(payload))
		return nil
	}
	if err := os.WriteFile(output, append(payload, '\n'), 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

func resultExitCode(result model.ScanResult, failOnWarning bool) int {
	if result.Summary.Blockers > 0 {
		return 2
	}
	if failOnWarning && result.Summary.Warnings > 0 {
		return 1
	}
	return 0
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
