# StorePreflight

[![CI](https://github.com/The-Crocop/storepreflight/actions/workflows/ci.yml/badge.svg)](https://github.com/The-Crocop/storepreflight/actions/workflows/ci.yml)
[![Pages](https://github.com/The-Crocop/storepreflight/actions/workflows/pages.yml/badge.svg)](https://github.com/The-Crocop/storepreflight/actions/workflows/pages.yml)

**Know whether this exact iOS or Android release is store-ready before you submit it.**

**Documentation:** https://the-crocop.github.io/storepreflight/

StorePreflight is a local-first release preflight engine. The open-source CLI and GitHub Action inspect the final IPA/AAB, surface deterministic findings, and compare a release with the last store-approved baseline. StorePreflight Cloud adds GitHub sign-in, projects, scoped CI tokens, release history and team workflows.

> Status: early MVP. Rules are intentionally conservative and evidence-based; StorePreflight does not pretend to predict Apple or Google review outcomes.

## Quick start

```bash
go run ./cmd/storepreflight scan MyApp.ipa
go run ./cmd/storepreflight scan --format json app-release.aab
```

Local baseline files still work:

```bash
storepreflight baseline MyApp-2.3.0.ipa
storepreflight scan --baseline storepreflight-baseline.json MyApp-2.4.0.ipa
```

## StorePreflight Cloud: approved-release diff

Cloud uses GitHub OAuth for people and project-scoped `spf_` tokens for CI.

The workflow is:

1. Sign in and create a StorePreflight project.
2. Generate a CI token and store it in your secret manager.
3. After Apple/Google approves a release, mark that artifact as the approved baseline.
4. Every later `push` compares the new release against that approved baseline.
5. New privacy-sensitive permissions and SDK signals are surfaced as release-diff risks.

```bash
export STOREPREFLIGHT_CLOUD_URL=https://your-cloud.example
export STOREPREFLIGHT_API_KEY=spf_...

# After this exact release has been approved by the store:
storepreflight approve --project my-mobile-app MyApp-2.3.0.ipa

# During the next release:
storepreflight push --project my-mobile-app MyApp-2.4.0.ipa
```

Example result:

```text
Compared against approved Cloud baseline a83d7c2e91bf
⚠️ SPF-DIFF-001-background-location — New privacy-sensitive permission since approved baseline
⚠️ SPF-DIFF-002-Firebase Analytics — New SDK signal since approved baseline
```

The artifact itself stays local. StorePreflight Cloud receives the artifact SHA-256 plus normalized readiness counts, permissions and SDK signals.

Use `--no-cloud-baseline` when a particular scan intentionally should not compare against the approved baseline. Passing `--baseline <file>` takes precedence over the Cloud baseline.

## GitHub Action

Local-only scan:

```yaml
- uses: The-Crocop/storepreflight@main
  with:
    artifact: build/MyApp.ipa
    baseline: .storepreflight/approved.json
```

Cloud-backed release history and automatic approved-release diff:

```yaml
- uses: The-Crocop/storepreflight@main
  with:
    artifact: build/MyApp.ipa
    project: my-mobile-app
    cloud-url: https://your-cloud.example
    api-key: ${{ secrets.STOREPREFLIGHT_API_KEY }}
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

The CLI remains useful without an account and scans artifacts locally. StorePreflight Cloud stores normalized scan/report metadata by default, not customer IPA/AAB binaries.

Cloud already has:
- GitHub OAuth user sign-in
- workspaces and projects
- hashed, revocable project-scoped CI tokens
- release history
- approved-release baselines

Planned next:
- GitHub App and native PR checks
- App Store Connect / Google Play correlation
- policy/deadline rule feed
- organization invites and roles
- billing and plan enforcement

## Development

```bash
go test ./...
go vet ./...
go build ./cmd/storepreflight
```

## License

Apache-2.0. See `LICENSE`.
