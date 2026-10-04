# StoreReady

**Know whether this exact iOS or Android release is store-ready before you submit it.**

StoreReady is a local-first release preflight engine. The open-source CLI and GitHub Action inspect the final IPA/AAB, surface deterministic findings, and compare a release with the last approved baseline. StoreReady Cloud adds release history, policy correlation, store integrations and team workflows.

> Status: early MVP. Rules are intentionally conservative and evidence-based; StoreReady does not pretend to predict Apple or Google review outcomes.

## Quick start

```bash
go run ./cmd/storeready scan MyApp.ipa
go run ./cmd/storeready scan --format json app-release.aab
```

Create an approved-release baseline:

```bash
storeready baseline MyApp-2.3.0.ipa
storeready scan --baseline storeready-baseline.json MyApp-2.4.0.ipa
```

The baseline comparison flags newly introduced privacy-sensitive permissions and SDK signals.

## StoreReady Cloud

The artifact stays local. `push` scans the IPA/AAB locally and uploads only normalized release metadata.

```bash
export STOREREADY_CLOUD_URL=https://app.storeready.dev
export STOREREADY_API_KEY=sr_...

storeready push --project my-mobile-app MyApp.ipa
```

The Cloud payload contains the artifact name/hash, platform, pass/warning/blocker counts and detected permission/SDK signals. It does **not** upload the IPA/AAB.

For local development:

```bash
export STOREREADY_CLOUD_URL=http://localhost:8080
export STOREREADY_API_KEY=local-dev-key
storeready push --project demo MyApp.ipa
```

## GitHub Action

Local-only scan:

```yaml
- uses: The-Crocop/storeready@main
  with:
    artifact: build/MyApp.ipa
    baseline: .storeready/approved.json
```

Scan and keep release history in StoreReady Cloud:

```yaml
- uses: The-Crocop/storeready@main
  with:
    artifact: build/MyApp.ipa
    baseline: .storeready/approved.json
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
- highlights release-diff privacy risks

### Android
- validates AAB structure and base manifest
- detects common sensitive permissions from compiled manifest data when visible
- detects common SDK signals
- prompts Data safety review when the release has privacy-sensitive changes
- release-diff comparison

## Product direction

The CLI remains useful without an account and scans artifacts locally. StoreReady Cloud stores normalized scan/report metadata by default, not customer IPA/AAB binaries.

Planned Cloud features:
- App Store Connect and Google Play Console correlation
- policy/deadline rule feed
- last-approved-release history
- team dashboards and GitHub App checks
- notification and agency workflows

## Development

```bash
go test ./...
go vet ./...
go build ./cmd/storeready
```

## License

Apache-2.0. See `LICENSE`.
