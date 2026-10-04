package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/The-Crocop/storeready/internal/baseline"
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
  storeready scan <artifact.ipa|artifact.aab> [--format text|json|markdown|sarif] [--output file] [--baseline file] [--fail-on-warning]
  storeready baseline <artifact.ipa|artifact.aab> [--output storeready-baseline.json]
  storeready version
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

	result, err := scanner.Scan(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan failed:", err)
		return 2
	}

	if *baselinePath != "" {
		b, err := baseline.Load(*baselinePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "baseline:", err)
			return 2
		}
		baseline.ApplyDiff(&result, b)
		result.Recount()
	}

	var payload []byte
	switch strings.ToLower(*format) {
	case "text":
		payload = []byte(report.Text(result))
	case "markdown", "md":
		payload = []byte(report.Markdown(result))
	case "json":
		payload, err = json.MarshalIndent(result, "", "  ")
	case "sarif":
		payload, err = report.SARIF(result)
	default:
		err = fmt.Errorf("unsupported format %q", *format)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	if *output == "" {
		fmt.Println(string(payload))
	} else if err := os.WriteFile(*output, append(payload, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write report:", err)
		return 2
	}

	if result.Summary.Blockers > 0 {
		return 2
	}
	if *failOnWarning && result.Summary.Warnings > 0 {
		return 1
	}
	return 0
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
