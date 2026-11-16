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
"$SCRIPT_DIR/benchmark-hit.sh" "$short_code"
"$SCRIPT_DIR/benchmark-miss.sh" "$short_code"
"$SCRIPT_DIR/benchmark-report.sh" > report.txt

mv cache-hit.txt "$PROJECT_ROOT/cache-hit.txt"
mv cache-miss.txt "$PROJECT_ROOT/cache-miss.txt"
cat report.txt
