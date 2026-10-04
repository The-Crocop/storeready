# StoreReady

**Know whether this exact iOS or Android release is store-ready before you submit it.**

StoreReady is a local-first release preflight engine. The open-source CLI and GitHub Action inspect the final IPA/AAB, surface deterministic findings, and compare a release with the last approved baseline. StoreReady Cloud adds policy updates, store integrations, release history and team workflows.

> Status: early MVP. Rules are intentionally conservative and evidence-based; StoreReady does not pretend to predict Apple or Google review outcomes.

## Quick start

```bash
go run ./cmd/storeready scan MyApp.ipa
go run ./cmd/storeready scan app-release.aab --format json
```

Create an approved-release baseline:

```bash
storeready baseline MyApp-2.3.0.ipa
storeready scan MyApp-2.4.0.ipa --baseline storeready-baseline.json
```

## GitHub Action

```yaml
- uses: The-Crocop/storeready@main
  with:
    artifact: build/MyApp.ipa
    baseline: .storeready/approved.json
```

For production use, pin a released major version once v1 is published.

## Output formats

Terminal, Markdown, JSON and SARIF are supported. A blocker returns exit code 2. Warnings do not fail CI unless `--fail-on-warning` is used.

## Current checks

iOS: IPA structure, Info.plist, privacy manifest presence, privacy-sensitive usage keys, common SDK signals and release-diff risks.

Android: AAB structure, base manifest, common sensitive permissions and SDK signals, Data safety review prompts and release-diff risks.

## Product direction

The CLI remains useful without an account and scans artifacts locally. StoreReady Cloud will store normalized scan/report metadata by default, not customer IPA/AAB binaries.

Planned Cloud features include App Store Connect and Google Play correlation, policy/deadline rules, last-approved-release history, team dashboards, GitHub App checks and agency workflows.

## Development

```bash
go test ./...
go vet ./...
go build ./cmd/storeready
```

## License

Apache-2.0. See `LICENSE`.
