#!/usr/bin/env bash
set -u

report="$RUNNER_TEMP/storeready-report.md"
common=(--format markdown --output "$report")

if [[ -n "${STOREREADY_BASELINE:-}" ]]; then
  common+=(--baseline "$STOREREADY_BASELINE")
fi
if [[ "${STOREREADY_FAIL_ON_WARNING:-false}" == "true" ]]; then
  common+=(--fail-on-warning)
fi

if [[ -n "${STOREREADY_PROJECT:-}" ]]; then
  if [[ -z "${STOREREADY_CLOUD_URL:-}" || -z "${STOREREADY_API_KEY:-}" ]]; then
    echo "::error::StoreReady Cloud upload requires cloud-url and api-key when project is configured."
    exit 2
  fi
  "$RUNNER_TEMP/storeready" push     --project "$STOREREADY_PROJECT"     --url "$STOREREADY_CLOUD_URL"     --api-key "$STOREREADY_API_KEY"     "${common[@]}"     "$STOREREADY_ARTIFACT"
  status=$?
else
  "$RUNNER_TEMP/storeready" scan "${common[@]}" "$STOREREADY_ARTIFACT"
  status=$?
fi

if [[ -f "$report" ]]; then
  cat "$report" >> "$GITHUB_STEP_SUMMARY"
fi

exit "$status"
