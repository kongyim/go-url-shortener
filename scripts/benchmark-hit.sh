#!/usr/bin/env bash

set -euo pipefail
export LC_ALL=C

SHORT_CODE="${1:?Usage: $0 <short-code>}"
REQUESTS=100
BASE_URL="http://localhost:8080"
OUTPUT="cache-hit.txt"

# Warm up the cache.
status=$(curl -sS -o /dev/null \
  -w "%{http_code}" \
  "$BASE_URL/$SHORT_CODE")

if [[ "$status" != "302" ]]; then
  echo "Warm-up failed: HTTP $status" >&2
  exit 1
fi

# Collect HTTP latency samples.
TEMP_OUTPUT=$(mktemp "${OUTPUT}.tmp.XXXXXX")
trap 'rm -f "$TEMP_OUTPUT"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

for ((i = 1; i <= REQUESTS; i++)); do
  result=$(curl -sS -o /dev/null \
    -w "%{http_code} %{time_total}" \
    "$BASE_URL/$SHORT_CODE")

  read -r status latency <<< "$result"

  if [[ "$status" != "302" ]]; then
    echo "Request $i failed: HTTP $status" >&2
    exit 1
  fi

  if [[ ! "$latency" =~ ^[0-9]+([.][0-9]+)?$ || ! "$latency" =~ [1-9] ]]; then
    echo "Request $i returned an invalid latency: $latency" >&2
    exit 1
  fi

  echo "$latency" >> "$TEMP_OUTPUT"
done

mv "$TEMP_OUTPUT" "$OUTPUT"

echo "Collected $REQUESTS Cache Hit samples in $OUTPUT"
