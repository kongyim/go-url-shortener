#!/usr/bin/env bash

set -euo pipefail
export LC_ALL=C

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
PROJECT_ROOT=$(cd -- "$SCRIPT_DIR/.." && pwd)
BASE_URL="http://localhost:8080"

for dependency in curl jq docker awk; do
  if ! command -v "$dependency" > /dev/null 2>&1; then
    echo "Missing required command: $dependency" >&2
    exit 1
  fi
done

# Stage both datasets together so a failed run preserves previous results.
WORK_DIR=$(mktemp -d "$PROJECT_ROOT/.benchmark.XXXXXX")
trap 'rm -rf "$WORK_DIR"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# Require complete counter snapshots; absent fields must not default to zero.
read_metrics() {
  local destination="$1" status
  status=$(curl -sS --connect-timeout 5 --max-time 15 \
    -o "$destination" -w "%{http_code}" "$BASE_URL/metrics")
  if [[ "$status" != "200" ]]; then
    echo "Metrics request failed: HTTP $status" >&2
    return 1
  fi
  if ! jq -e 'type == "object" and
    ([.hits, .misses, .errors, .redisGetCount, .postgresLookupCount] |
      all(.[]; type == "number" and . >= 0 and . == floor))' "$destination" > /dev/null; then
    echo "Invalid metrics response" >&2
    return 1
  fi
}

verify_metrics() {
  local before="$1" after="$2" expected_hits="$3" expected_misses="$4" phase="$5"
  if ! jq -en --slurpfile before "$before" --slurpfile after "$after" \
    --argjson hits "$expected_hits" --argjson misses "$expected_misses" '
      $before[0] as $b | $after[0] as $a |
      ($a.hits - $b.hits == $hits) and
      ($a.misses - $b.misses == $misses) and
      ($a.errors - $b.errors == 0) and
      ($a.redisGetCount - $b.redisGetCount == 101) and
      ($a.postgresLookupCount - $b.postgresLookupCount == $misses)
    ' > /dev/null; then
    echo "$phase cache-path verification failed; check Redis, API restarts, and concurrent traffic" >&2
    return 1
  fi
  echo "$phase cache-path counters verified"
}

status=$(curl -sS --connect-timeout 5 --max-time 15 \
  -o "$WORK_DIR/create-response.json" -w "%{http_code}" \
  -X POST "$BASE_URL/shorten" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com"}')

if [[ "$status" != "201" ]]; then
  echo "Failed to create benchmark URL: HTTP $status" >&2
  exit 1
fi

if ! short_code=$(jq -er '.shortCode | select(type == "string") | select(test("^[A-Za-z0-9_-]+$"))' "$WORK_DIR/create-response.json"); then
  echo "Create response must contain a valid shortCode" >&2
  exit 1
fi

echo "Benchmark short code: $short_code"
cd -- "$WORK_DIR"
read_metrics before-hit.json
"$SCRIPT_DIR/benchmark-hit.sh" "$short_code"
read_metrics after-hit.json
verify_metrics before-hit.json after-hit.json 101 0 "Cache Hit"

read_metrics before-miss.json
"$SCRIPT_DIR/benchmark-miss.sh" "$short_code"
read_metrics after-miss.json
# The preflight hits the key populated by the successful hit benchmark.
verify_metrics before-miss.json after-miss.json 1 100 "Cache Miss"
"$SCRIPT_DIR/benchmark-report.sh" > report.txt

mv cache-hit.txt "$PROJECT_ROOT/cache-hit.txt"
mv cache-miss.txt "$PROJECT_ROOT/cache-miss.txt"
cat report.txt
