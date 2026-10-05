#!/usr/bin/env bash
set -u

report="$RUNNER_TEMP/storepreflight-report.md"
common=(--format markdown --output "$report")

if [[ -n "${STOREPREFLIGHT_BASELINE:-}" ]]; then
  common+=(--baseline "$STOREPREFLIGHT_BASELINE")
fi
if [[ "${STOREPREFLIGHT_FAIL_ON_WARNING:-false}" == "true" ]]; then
  common+=(--fail-on-warning)
fi

if [[ -n "${STOREPREFLIGHT_PROJECT:-}" ]]; then
  if [[ -z "${STOREPREFLIGHT_CLOUD_URL:-}" || -z "${STOREPREFLIGHT_API_KEY:-}" ]]; then
    echo "::error::StorePreflight Cloud upload requires cloud-url and api-key when project is configured."
    exit 2
  fi
  "$RUNNER_TEMP/storepreflight" push     --project "$STOREPREFLIGHT_PROJECT"     --url "$STOREPREFLIGHT_CLOUD_URL"     --api-key "$STOREPREFLIGHT_API_KEY"     "${common[@]}"     "$STOREPREFLIGHT_ARTIFACT"
  status=$?
else
  "$RUNNER_TEMP/storepreflight" scan "${common[@]}" "$STOREPREFLIGHT_ARTIFACT"
  status=$?
fi

if [[ -f "$report" ]]; then
  cat "$report" >> "$GITHUB_STEP_SUMMARY"
fi

exit "$status"
