
#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)

TEST_DIR=$(mktemp -d)
trap 'rm -rf "$TEST_DIR"' EXIT

cd "$TEST_DIR"

# Create 100 fixed samples per dataset.
for ((i = 1; i <= 100; i++)); do
    echo "0.001" >> cache-hit.txt
    echo "0.005" >> cache-miss.txt
done

# Execute the report in an isolated directory.
report=$(bash "$SCRIPT_DIR/benchmark-report.sh")

# Verify the calculated speedup.
if [[ "$report" != *"5.00x"* ]]; then
    echo "FAIL: Expected 5.00x average latency ratio" >&2
    exit 1
fi

echo "PASS: Benchmark report calculates 5.00x ratio"
