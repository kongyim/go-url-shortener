
#!/usr/bin/env bash

set -euo pipefail

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
: > "$OUTPUT"

for ((i = 1; i <= REQUESTS; i++)); do
  result=$(curl -sS -o /dev/null \
    -w "%{http_code} %{time_total}" \
    "$BASE_URL/$SHORT_CODE")

  read -r status latency <<< "$result"

  if [[ "$status" != "302" ]]; then
    echo "Request $i failed: HTTP $status" >&2
    exit 1
  fi

  echo "$latency" >> "$OUTPUT"
done

echo "Collected $REQUESTS Cache Hit samples in $OUTPUT"
