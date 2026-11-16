#!/usr/bin/env bash

set -euo pipefail
export LC_ALL=C

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

expect_failure() {
    local test_name="$1"
    local expected_error="$2"

    local status=0
    bash "$SCRIPT_DIR/benchmark-report.sh" > output.txt 2> error.txt || status=$?

    if [[ "$status" != "1" ]]; then
        echo "FAIL: $test_name expected exit 1, got $status" >&2
        exit 1
    fi

    if ! grep -Fq -- "$expected_error" error.txt; then
        echo "FAIL: $test_name returned the wrong error" >&2
        echo "Expected: $expected_error" >&2
        echo "Actual output:" >&2
        cat error.txt >&2
        exit 1
    fi

    if [[ -s output.txt ]]; then
        echo "FAIL: $test_name produced a report" >&2
        exit 1
    fi

    echo "PASS: $test_name"
}

# Test: Invalid latency value
for ((i = 1; i <= 100; i++)); do
    if ((i == 50)); then
        echo "NaN"
    else
        echo "0.001"
    fi
done > cache-hit.txt

expect_failure "invalid latency" "Invalid latency value"

# Test: Incorrect sample count
: > cache-hit.txt

for ((i = 1; i <= 99; i++)); do
    echo "0.001" >> cache-hit.txt
done

expect_failure "incorrect sample count" "Both files must contain exactly 100 latency samples"
