package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/The-Crocop/storepreflight/internal/baseline"
	"github.com/The-Crocop/storepreflight/internal/cloud"
	"github.com/The-Crocop/storepreflight/internal/model"
	"github.com/The-Crocop/storepreflight/internal/report"
	"github.com/The-Crocop/storepreflight/internal/scanner"
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
	case "approve":
		os.Exit(runApprove(os.Args[2:]))
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
	fmt.Fprintf(os.Stderr, `StorePreflight - release preflight for iOS and Android

Usage:
  storepreflight scan [flags] <artifact.ipa|artifact.aab>
  storepreflight push [flags] <artifact.ipa|artifact.aab>
  storepreflight approve [flags] <artifact.ipa|artifact.aab>
  storepreflight baseline [flags] <artifact.ipa|artifact.aab>
  storepreflight version

Cloud flags:
  --project <key>       StorePreflight Cloud project key
  --url <url>           Cloud base URL; defaults to STOREPREFLIGHT_CLOUD_URL
  --api-key <key>       Cloud API key; defaults to STOREPREFLIGHT_API_KEY

Push automatically compares with the latest approved Cloud baseline unless
--baseline points to a local baseline or --no-cloud-baseline is set.

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
	project := fs.String("project", "", "StorePreflight Cloud project key")
	cloudURL := fs.String("url", envOr("STOREPREFLIGHT_CLOUD_URL", ""), "StorePreflight Cloud base URL")
	apiKey := fs.String("api-key", envOr("STOREPREFLIGHT_API_KEY", ""), "StorePreflight Cloud API key")
	format := fs.String("format", "text", "text, json, markdown or sarif")
	output := fs.String("output", "", "write report to file")
	baselinePath := fs.String("baseline", "", "compare against a local approved-release baseline")
	noCloudBaseline := fs.Bool("no-cloud-baseline", false, "do not compare with the latest approved Cloud baseline")
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

	client, err := cloud.New(*cloudURL, *apiKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cloud:", err)
		return 2
	}

	result, err := scanner.Scan(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan failed:", err)
		return 2
	}

	if *baselinePath != "" {
		value, err := baseline.Load(*baselinePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "baseline:", err)
			return 2
		}
		baseline.ApplyDiff(&result, value)
		result.Recount()
	} else if !*noCloudBaseline {
		value, found, err := client.LatestBaseline(context.Background(), *project, result.Platform)
		if err != nil {
			fmt.Fprintln(os.Stderr, "cloud baseline:", err)
			return 2
		}
		if found {
			baseline.ApplyDiff(&result, value)
			result.Recount()
			fmt.Fprintf(os.Stderr, "Compared against approved Cloud baseline %s\n", shortSHA(value.ArtifactSHA256))
		} else {
			fmt.Fprintln(os.Stderr, "No approved Cloud baseline yet; recording this scan without release-diff checks")
		}
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
	fmt.Fprintf(os.Stderr, "Uploaded StorePreflight scan %s for project %s\n", record.ID, record.ProjectKey)
	return resultExitCode(result, *failOnWarning)
}

func runApprove(args []string) int {
	fs := flag.NewFlagSet("approve", flag.ContinueOnError)
	project := fs.String("project", "", "StorePreflight Cloud project key")
	cloudURL := fs.String("url", envOr("STOREPREFLIGHT_CLOUD_URL", ""), "StorePreflight Cloud base URL")
	apiKey := fs.String("api-key", envOr("STOREPREFLIGHT_API_KEY", ""), "StorePreflight Cloud API key")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "approve requires exactly one IPA or AAB artifact")
		return 2
	}
	if strings.TrimSpace(*project) == "" {
		fmt.Fprintln(os.Stderr, "approve requires --project")
		return 2
	}

	result, err := scanner.Scan(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan failed:", err)
		return 2
	}
	if result.Summary.Blockers > 0 {
		fmt.Fprintln(os.Stderr, "refusing to approve an artifact with blockers")
		return 2
	}

	client, err := cloud.New(*cloudURL, *apiKey)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cloud:", err)
		return 2
	}
	record, err := client.ApproveBaseline(context.Background(), *project, result)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cloud:", err)
		return 2
	}

	fmt.Printf(
		"Approved %s baseline for project %s (%s) at %s\n",
		record.Platform,
		record.ProjectKey,
		shortSHA(record.ArtifactSHA256),
		record.ApprovedAt,
	)
	return 0
}

func runBaseline(args []string) int {
	fs := flag.NewFlagSet("baseline", flag.ContinueOnError)
	output := fs.String("output", "storepreflight-baseline.json", "baseline file")
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

func shortSHA(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}
