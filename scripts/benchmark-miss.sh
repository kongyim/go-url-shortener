#!/usr/bin/env bash

set -euo pipefail
export LC_ALL=C

SHORT_CODE="${1:?Usage: $0 <short-code>}"
REQUESTS=100
BASE_URL="http://localhost:8080"
REDIS_CONTAINER="go-url-shortener-redis"
REDIS_KEY="url:$SHORT_CODE"
OUTPUT="cache-miss.txt"

# Confirm the short URL exists before benchmarking.
status=$(curl -sS -o /dev/null \
  -w "%{http_code}" \
  "$BASE_URL/$SHORT_CODE")

if [[ "$status" != "302" ]]; then
  echo "Preflight failed: HTTP $status" >&2
  exit 1
fi

TEMP_OUTPUT=$(mktemp "${OUTPUT}.tmp.XXXXXX")
trap 'rm -f "$TEMP_OUTPUT"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

for ((i = 1; i <= REQUESTS; i++)); do
  # Remove the cached entry before every request.
  docker exec "$REDIS_CONTAINER" \
    redis-cli DEL "$REDIS_KEY" > /dev/null

  # Verify the key does not exist.
  exists=$(docker exec "$REDIS_CONTAINER" \
    redis-cli EXISTS "$REDIS_KEY")

  if [[ "$exists" != "0" ]]; then
    echo "Request $i: Redis key still exists" >&2
    exit 1
  fi

  # Measure the HTTP redirect response.
  result=$(curl -sS -o /dev/null \
    -w "%{http_code} %{time_total}" \
    "$BASE_URL/$SHORT_CODE")

  read -r status latency <<< "$result"

  if [[ "$status" != "302" ]]; then
    echo "Request $i failed: HTTP $status" >&2
    exit 1
  fi

  if [[ ! "$latency" =~ ^[0-9]+([.][0-9]+)?$ ]]; then
    echo "Request $i returned an invalid latency: $latency" >&2
    exit 1
  fi

  echo "$latency" >> "$TEMP_OUTPUT"
done

mv "$TEMP_OUTPUT" "$OUTPUT"

echo "Collected $REQUESTS Cache Miss samples in $OUTPUT"
