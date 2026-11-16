
#!/usr/bin/env bash

set -euo pipefail

HIT_FILE="cache-hit.txt"
MISS_FILE="cache-miss.txt"

# Check required input files.
for file in "$HIT_FILE" "$MISS_FILE"; do
  if [[ ! -f "$file" ]]; then
    echo "Missing file: $file" >&2
    exit 1
  fi
done

# Calculate statistics for both datasets.
awk '
FNR == 1 {
    dataset++
}

{
    if ($1 !~ /^[0-9]+([.][0-9]+)?$/) {
        print "Invalid latency value: " $0 > "/dev/stderr"
        exit 1
    }

    count[dataset]++
    values[dataset, count[dataset]] = $1 * 1000
    total[dataset] += $1 * 1000
}

END {
    if (count[1] == 0 || count[2] == 0) {
        print "Both files must contain latency samples" > "/dev/stderr"
        exit 1
    }

    for (d = 1; d <= 2; d++) {
        # Sort each dataset numerically.
        for (i = 1; i <= count[d]; i++) {
            for (j = i + 1; j <= count[d]; j++) {
                if (values[d, i] > values[d, j]) {
                    temp = values[d, i]
                    values[d, i] = values[d, j]
                    values[d, j] = temp
                }
            }
        }

        avg[d] = total[d] / count[d]

        if (count[d] % 2 == 0) {
            p50[d] = (
                values[d, count[d] / 2] +
                values[d, count[d] / 2 + 1]
            ) / 2
        } else {
            p50[d] = values[d, (count[d] + 1) / 2]
        }

        p95Rank = int(0.95 * count[d])
        if (p95Rank < 0.95 * count[d]) {
            p95Rank++
        }

        p95[d] = values[d, p95Rank]
    }

    if (avg[1] <= 0) {
        print "Cache Hit average must be positive" > "/dev/stderr"
        exit 1
    }

    printf "Benchmark Results (HTTP latency)\n\n"
    printf "%-12s %12s %12s\n", "Metric", "Cache Hit", "Cache Miss"
    printf "%-12s %12d %12d\n", "Requests", count[1], count[2]
    printf "%-12s %11.3fms %11.3fms\n", "Average", avg[1], avg[2]
    printf "%-12s %11.3fms %11.3fms\n", "p50", p50[1], p50[2]
    printf "%-12s %11.3fms %11.3fms\n", "p95", p95[1], p95[2]

    printf "\nAverage latency ratio (Miss / Hit): %.2fx\n", avg[2] / avg[1]
}
' "$HIT_FILE" "$MISS_FILE"
