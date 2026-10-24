#!/usr/bin/env bash
# Native equivalents of test:standard:*; run every selected suite after failures.
set -uo pipefail
root=$(cd "$(dirname "$0")/.." && pwd)
selection=${1:-standard}
case "$selection" in ordinary|checker|unit|integration|conformance|standard|all) ;; *) echo "Unknown suite: $selection" >&2; exit 2;; esac
cleanup=false
if [ -z "${CTRF_STANDARD_ARTIFACTS:-}" ]; then
  CTRF_STANDARD_ARTIFACTS=$(mktemp -d "${TMPDIR:-/tmp}/ctrf-go-results.XXXXXX")
  cleanup=true
fi
case "$CTRF_STANDARD_ARTIFACTS" in /*) ;; *) echo 'CTRF_STANDARD_ARTIFACTS must be absolute' >&2; exit 2;; esac
tool_dir=$(mktemp -d "${TMPDIR:-/tmp}/ctrf-go-tools.XXXXXX")
finish() {
  rm -rf "$tool_dir"
  if $cleanup; then rm -rf "$CTRF_STANDARD_ARTIFACTS"; fi
}
trap finish EXIT
mkdir -p "$CTRF_STANDARD_ARTIFACTS/diagnostics"
for output in reporter-reports/root reporter-reports/native checker-reports/checker reports/unit reports/integration; do
  case "$selection:$output" in
    all:*|ordinary:reporter-reports/*|checker:checker-reports/*|unit:reports/unit|integration:reports/integration|conformance:reports/*|standard:checker-reports/*|standard:reports/*)
      rm -f "$CTRF_STANDARD_ARTIFACTS/$output.json" "$CTRF_STANDARD_ARTIFACTS/diagnostics/$output.jsonl" "$CTRF_STANDARD_ARTIFACTS/diagnostics/$output.stderr" ;;
  esac
done
cd "$root/tests"
go build -o "$tool_dir/assessment-reporter" ./native/report-results || exit 1
status=0
run() {
  local scope=$1 output=$2 location=$3
  shift 3
  local runner_exit=0
  local inventory_exit=0
  if [[ "$scope" == reporter-tests ]]; then
    (cd "$location" && go test -list . -json "$@") > "$CTRF_STANDARD_ARTIFACTS/diagnostics/$output-inventory.jsonl" 2> "$CTRF_STANDARD_ARTIFACTS/diagnostics/$output-inventory.stderr" || inventory_exit=$?
  fi
  (cd "$location" && go test -json -count=1 "$@") > "$CTRF_STANDARD_ARTIFACTS/diagnostics/$output.jsonl" 2> "$CTRF_STANDARD_ARTIFACTS/diagnostics/$output.stderr" || runner_exit=$?
  if [[ "$scope" == reporter-tests && "$inventory_exit" -eq 0 ]]; then
    "$tool_dir/assessment-reporter" -scope "$scope" -exit "$runner_exit" -input "$CTRF_STANDARD_ARTIFACTS/diagnostics/$output.jsonl" -output "$CTRF_STANDARD_ARTIFACTS/$output.json" -inventory "$CTRF_STANDARD_ARTIFACTS/diagnostics/$output-inventory.jsonl" || status=1
  else
    "$tool_dir/assessment-reporter" -scope "$scope" -exit "$runner_exit" -input "$CTRF_STANDARD_ARTIFACTS/diagnostics/$output.jsonl" -output "$CTRF_STANDARD_ARTIFACTS/$output.json" || status=1
  fi
  if [ "$inventory_exit" -ne 0 ]; then status=1; fi
  if [ "$runner_exit" -ne 0 ]; then status=1; fi
}
if [[ "$selection" == ordinary || "$selection" == all ]]; then
  mkdir -p "$CTRF_STANDARD_ARTIFACTS/reporter-reports" "$CTRF_STANDARD_ARTIFACTS/diagnostics/reporter-reports"
  if [[ "${CTRF_COVERAGE:-false}" == true ]]; then
    run reporter-tests reporter-reports/root "$root" -race -covermode=atomic "-coverprofile=$root/coverage.out" ./...
  else
    run reporter-tests reporter-reports/root "$root" -race ./...
  fi
  run reporter-tests reporter-reports/native "$root/tests" -race ./native/...
fi
if [[ "$selection" == checker || "$selection" == standard || "$selection" == all ]]; then
  mkdir -p "$CTRF_STANDARD_ARTIFACTS/checker-reports" "$CTRF_STANDARD_ARTIFACTS/diagnostics/checker-reports"
  run checker checker-reports/checker "$root/tests" ./standard/checker
fi
for level in unit integration; do
  if [[ "$selection" == "$level" || "$selection" == conformance || "$selection" == standard || "$selection" == all ]]; then
    mkdir -p "$CTRF_STANDARD_ARTIFACTS/reports" "$CTRF_STANDARD_ARTIFACTS/diagnostics/reports"
    run "$level" "reports/$level" "$root/tests" "./standard/conformance/$level"
  fi
done
exit "$status"
