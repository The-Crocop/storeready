#!/usr/bin/env bash
set -u
report="$RUNNER_TEMP/storeready-report.md"
args=(scan "$STOREREADY_ARTIFACT" --format markdown --output "$report")
if [[ -n "${STOREREADY_BASELINE:-}" ]]; then args+=(--baseline "$STOREREADY_BASELINE"); fi
if [[ "${STOREREADY_FAIL_ON_WARNING:-false}" == "true" ]]; then args+=(--fail-on-warning); fi
"$RUNNER_TEMP/storeready" "${args[@]}"
status=$?
if [[ -f "$report" ]]; then cat "$report" >> "$GITHUB_STEP_SUMMARY"; fi
exit "$status"
