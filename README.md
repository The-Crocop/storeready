# StoreReady

[![CI](https://github.com/The-Crocop/storeready/actions/workflows/ci.yml/badge.svg)](https://github.com/The-Crocop/storeready/actions/workflows/ci.yml)
[![Pages](https://github.com/The-Crocop/storeready/actions/workflows/pages.yml/badge.svg)](https://github.com/The-Crocop/storeready/actions/workflows/pages.yml)

**Know whether this exact iOS or Android release is store-ready before you submit it.**

**Documentation:** https://the-crocop.github.io/storeready/

StoreReady is a local-first release preflight engine. The open-source CLI and GitHub Action inspect the final IPA/AAB, surface deterministic findings, and compare a release with the last store-approved baseline. StoreReady Cloud adds release history, policy correlation, store integrations and team workflows.

> Status: early MVP. Rules are intentionally conservative and evidence-based; StoreReady does not pretend to predict Apple or Google review outcomes.

## Quick start

```bash
go run ./cmd/storeready scan MyApp.ipa
go run ./cmd/storeready scan --format json app-release.aab
```

Local baseline files still work:

```bash
storeready baseline MyApp-2.3.0.ipa
storeready scan --baseline storeready-baseline.json MyApp-2.4.0.ipa
```

## StoreReady Cloud: approved-release diff

The key Cloud workflow is:

1. Apple/Google approves a release.
2. Mark that exact artifact as the approved baseline.
3. Every later `push` automatically compares the new artifact against that approved release.
4. New privacy-sensitive permissions and SDK signals are surfaced as release-diff risks.

```bash
export STOREREADY_CLOUD_URL=http://localhost:8080
export STOREREADY_API_KEY=local-dev-key

# After this exact release has been approved by the store:
storeready approve --project my-mobile-app MyApp-2.3.0.ipa

# During the next release:
storeready push --project my-mobile-app MyApp-2.4.0.ipa
```

Example result:

```text
Compared against approved Cloud baseline a83d7c2e91bf
⚠️ SR-DIFF-001-background-location — New privacy-sensitive permission since approved baseline
⚠️ SR-DIFF-002-Firebase Analytics — New SDK signal since approved baseline
```

The artifact itself stays local. StoreReady Cloud receives the artifact SHA-256 plus normalized readiness counts, permissions and SDK signals.

Use `--no-cloud-baseline` when a particular scan intentionally should not compare against the approved baseline. Passing `--baseline <file>` takes precedence over the Cloud baseline.

## GitHub Action

Local-only scan:

```yaml
- uses: The-Crocop/storeready@main
  with:
    artifact: build/MyApp.ipa
    baseline: .storeready/approved.json
```

Cloud-backed release history and automatic approved-release diff:

```yaml
- uses: The-Crocop/storeready@main
  with:
    artifact: build/MyApp.ipa
    project: my-mobile-app
    cloud-url: https://app.storeready.dev
    api-key: ${{ secrets.STOREREADY_API_KEY }}
```

For production use, pin a released major version once v1 is published.

## Output formats

- human-readable terminal output
- Markdown for CI summaries
- JSON for automation
- SARIF for security/code-scanning pipelines

A blocker returns exit code `2`. Warnings do not fail CI unless `--fail-on-warning` is used.

## Current checks

### iOS
- validates IPA structure and presence of the app Info.plist
- checks for PrivacyInfo.xcprivacy
- detects common privacy-sensitive usage-description keys
- detects signals for common analytics/tracking SDKs
- compares privacy-sensitive signals against the last approved release

### Android
- validates AAB structure and base manifest
- detects common sensitive permissions from compiled manifest data when visible
- detects common SDK signals
- prompts Data safety review when the release has privacy-sensitive changes
- compares privacy-sensitive signals against the last approved release

## Product direction

The CLI remains useful without an account and scans artifacts locally. StoreReady Cloud stores normalized scan/report metadata by default, not customer IPA/AAB binaries.

Planned Cloud features:
- App Store Connect and Google Play Console correlation
- policy/deadline rule feed
- project-scoped CI tokens
- GitHub App and PR checks
- team dashboards
- notification and agency workflows

## Development

```bash
go test ./...
go vet ./...
go build ./cmd/storeready
```

## License

Apache-2.0. See `LICENSE`.
