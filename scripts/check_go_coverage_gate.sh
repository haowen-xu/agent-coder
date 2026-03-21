#!/usr/bin/env bash
set -euo pipefail

THRESHOLD="${GO_COVERAGE_THRESHOLD:-80}"
PROFILE_PATH="${GO_COVERAGE_PROFILE:-/tmp/agent-coder.coverage.out}"

cleanup() {
  rm -f "${PROFILE_PATH}"
}
trap cleanup EXIT

printf '[coverage-gate] running go test with coverage profile\n'
go test ./... -count=1 -covermode=atomic -coverprofile="${PROFILE_PATH}"

total_cov="$(go tool cover -func="${PROFILE_PATH}" | awk '/^total:/{gsub("%", "", $3); print $3}')"
if [[ -z "${total_cov}" ]]; then
  echo "[coverage-gate] ERROR: failed to parse total coverage"
  exit 1
fi

printf '[coverage-gate] total coverage: %s%% (threshold: %s%%)\n' "${total_cov}" "${THRESHOLD}"
if ! awk -v cov="${total_cov}" -v th="${THRESHOLD}" 'BEGIN { exit ((cov + 0) >= (th + 0)) ? 0 : 1 }'; then
  echo "[coverage-gate] FAIL: total coverage below threshold"
  exit 1
fi

zero_cov_files="$(awk '
  NR == 1 { next }
  {
    split($1, p, ":")
    f = p[1]
    stmts = $2 + 0
    hits = $3 + 0
    total[f] += stmts
    if (hits > 0) covered[f] += stmts
  }
  END {
    for (f in total) {
      if (total[f] > 0 && covered[f] == 0) {
        print f
      }
    }
  }
' "${PROFILE_PATH}" | sort)"

if [[ -n "${zero_cov_files}" ]]; then
  echo "[coverage-gate] FAIL: files with logic statements but 0% coverage"
  echo "${zero_cov_files}" | sed 's/^/  - /'
  exit 1
fi

echo "[coverage-gate] PASS"
