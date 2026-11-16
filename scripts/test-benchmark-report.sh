
#!/usr/bin/env bash

set -euo pipefail
export LC_ALL=C

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)

TEST_DIR=$(mktemp -d)
trap 'rm -rf "$TEST_DIR"' EXIT

cd "$TEST_DIR"

# Prepare clean test data before each test.
setup_fixture() {
    : > cache-hit.txt
    : > cache-miss.txt

    for ((i = 1; i <= 100; i++)); do
        echo "0.001" >> cache-hit.txt
        echo "0.005" >> cache-miss.txt
    done
}

expect_failure() {
    local test_name="$1"
    local expected_error="$2"

    local status=0

    bash "$SCRIPT_DIR/benchmark-report.sh" \
        > output.txt 2> error.txt || status=$?

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

# Test 1: Happy path
setup_fixture

report=$(bash "$SCRIPT_DIR/benchmark-report.sh")

if [[ "$report" != *"5.00x"* ]]; then
    echo "FAIL: Expected 5.00x average latency ratio" >&2
    exit 1
fi

echo "PASS: Benchmark report calculates 5.00x ratio"

# Test 2: Invalid latency
setup_fixture

awk 'NR == 50 { print "NaN"; next } { print }' \
    cache-hit.txt > cache-hit.tmp

mv cache-hit.tmp cache-hit.txt

expect_failure \
    "invalid latency" \
    "Invalid latency value"

# Test 3: Incorrect sample count
setup_fixture

head -n 99 cache-hit.txt > cache-hit.tmp
mv cache-hit.tmp cache-hit.txt

expect_failure \
    "incorrect sample count" \
    "Both files must contain exactly 100 latency samples"

# Test 4: Missing Cache Hit file
setup_fixture

rm cache-hit.txt

expect_failure \
    "missing Cache Hit file" \
    "Missing file: cache-hit.txt"

# Test 5: Missing Cache Miss file
setup_fixture

rm cache-miss.txt

expect_failure \
    "missing Cache Miss file" \
    "Missing file: cache-miss.txt"
