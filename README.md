# Go URL Shortener

## Development commands

Copy `.env.example` to `.env` for local configuration and update the database
credentials to match your PostgreSQL container. The API uses `godotenv.Load()` at
startup to load `.env` from the current working directory, with both `make run`
and `go run ./cmd/api`. Existing environment variables take precedence over values
in `.env`. A missing `.env` is allowed; other loading errors stop startup.
Use `KEY='value'` assignments. `.env` is ignored by Git.

The API uses Redis at `REDIS_ADDR` (default `localhost:6379`) to cache short-code
lookups for one hour. Successful URL creation also populates the cache. Redirects
fall back to PostgreSQL on a cache miss or Redis error; cache write failures do
not fail requests. Each cache operation has a 200 ms timeout with retries disabled.
PostgreSQL remains the source of truth. Tests use a mock cache and do not require
a running Redis server.

- `make run` starts the API.
- `make test` runs all Go tests.
- `make terraform-init` initializes Terraform in `infrastructure/local`.
- `make terraform-plan` previews infrastructure changes.
- `make terraform-apply` creates the local PostgreSQL data directory and applies infrastructure changes.
- `make migrate-up` creates the `urls` table using migration 001.
- `make migrate-down` drops the `urls` table and its data using migration 001.

Migration commands require the `go-url-shortener-postgres` Docker container to be
running. They execute the specified SQL files directly and do not track migration
history; running `make migrate-up` again skips creation if the table already exists.

PostgreSQL stores its data in `.local/postgres/`, which is
ignored by Git and mounted at `/var/lib/postgresql/data` in the container.
When running Terraform directly, create this directory before applying.
Switching from the previous Docker volume starts a fresh database; existing data
is not copied automatically. The previous `go-url-shortener-postgres-data` volume
remains managed by Terraform to preserve that data during the switch.

## Performance Benchmark

Measured on **16 November 2026 at 15:31 UTC** using `cmd/lookupbench` on
macOS ARM64 with Go 1.27.3. PostgreSQL 16.16 and Redis 7.4.12 ran in local Docker
containers accessed through their published localhost ports. The `urls` table
contained **2 rows**.

### Method

The benchmark selects one existing URL and compares
`URLRepository.GetByShortCode` with a Redis cache hit returning the same original
URL. Redis uses a unique temporary key, removed when the benchmark finishes and
given a 10-minute fallback TTL. PostgreSQL data is not modified.

Each backend receives **100 unmeasured warm-up lookups**, then **5,000 measured
lookups**, with one operation in flight at a time. Each iteration measures both
backends, alternating which runs first. Connection setup and Redis seeding are
excluded; each sample includes the client call, response decoding, and a returned
URL check. Redis samples also include creation and cancellation of its 200 ms
timeout context. A failed lookup aborts the run rather than being omitted.

Average is the arithmetic mean. p50 and p95 use the nearest-rank method over each
backend's 5,000 samples. These are client-side lookup durations, not HTTP response
times or server-only query execution times.

### Results

| Lookup | Average (ms) | p50 (ms) | p95 (ms) |
| --- | ---: | ---: | ---: |
| PostgreSQL SELECT | 0.118301 | 0.117625 | 0.131709 |
| Redis GET (cache hit) | 0.120851 | 0.119917 | 0.132833 |
| Speedup: PostgreSQL / Redis | 0.979× | 0.981× | 0.992× |

Speedup ratios are dimensionless and computed from unrounded measurements.
A ratio above 1 means Redis was faster; below 1 means Redis was slower.

### Interpretation

Redis did **not** improve lookup latency in this run: its average was about 2.2%
higher than PostgreSQL's, with slightly higher p50 and p95 as well. Both backends
averaged roughly 0.12 ms. This result does not establish a meaningful performance
difference across repeated runs.

A cache hit still avoids a PostgreSQL lookup in the application. That can reduce
database read traffic, but this benchmark does not measure throughput or prove
an improvement under load. The PostgreSQL measurement is a direct repository
lookup; it does not include the Redis GET miss or subsequent SET that the API's
cache-miss path performs.

### Limitations

- This is one local run with concurrency 1, a two-row table, and repeated reads of
  one URL. It does not represent a large dataset, mixed traffic, or contention.
- Connections are warmed and reused. PostgreSQL's repeated lookup can benefit
  from its own buffers and OS caching; this is not a cold-disk comparison.
- Local Docker networking, client overhead, and host scheduling contribute to
  the measurements. No confidence intervals or repeated-run analysis were used.
- The Redis key is benchmark-specific and longer than a normal short-code key.
  PostgreSQL returns a full URL row, while Redis returns only the original URL.
- HTTP routing, middleware, redirects, cache misses, cache writes, failures, and
  URL creation are excluded. These results are not end-to-end API latency or a
  load test.
- `/metrics` exposes cumulative counts and average lookup latency, not p50 or
  p95. The percentiles above come from individual benchmark samples.

### Reproduce

Start PostgreSQL and Redis, apply the schema with `make migrate-up` if needed,
and ensure the database contains at least one URL. Set `DATABASE_URL` and
`REDIS_ADDR` in `.env` (Redis defaults to `localhost:6379`), then run:

```sh
go run ./cmd/lookupbench -samples 5000 -warmup 100
```

The command prints JSON measurements and speedup ratios. It connects directly to
the backends, so the HTTP API does not need to be running. Results will vary by
environment and run.

### Running Both HTTP Benchmarks Automatically

Start the API at `http://localhost:8080`, PostgreSQL (with the `urls` table), and
the `go-url-shortener-redis` container. The command requires Bash, curl, jq, Docker,
and awk; Redis CLI must be available inside the Redis container.

```sh
./scripts/benchmark.sh
```

This creates a short URL for `https://example.com` using `POST /shorten`, requires
HTTP 201 and a valid `shortCode`, then runs the cache-hit and cache-miss benchmarks
with that same code. It prints the report only after both benchmarks succeed.
Results are staged in a temporary directory; the project-root `cache-hit.txt` and
`cache-miss.txt` files are replaced only after both benchmarks and report validation
succeed. An earlier failure stops the workflow and preserves previous results.

The wrapper validates `/metrics` before and after each phase. Cache Hit must add
101 hits and no misses or PostgreSQL lookups (including warm-up). Cache Miss must
add one hit for preflight, 100 misses, and 100 PostgreSQL lookups. Both phases must
add 101 Redis GETs and zero cache errors. Missing or invalid metrics, or unexpected
counter changes, stop the workflow before publishing results. Run against one API
process with no other traffic or restarts: these are global counters, not per-URL
traces. The standalone collection scripts do not automatically verify counters.
Both collectors reject zero, negative, and malformed latency samples.

Each run creates one PostgreSQL URL record, which is retained even if a later step
fails. Successful creation also populates Redis. Redirect destinations are not
visited. The individual scripts below remain available for manual runs.

### Reproducing the Cache Hit Benchmark

Start the API, PostgreSQL, and Redis services, then create a short URL using
`POST /shorten`.

Run the benchmark from the project root using the returned short code:

```sh
./scripts/benchmark-hit.sh YOUR_SHORT_CODE
```

The script warms up the cache, performs 100 sequential HTTP GET requests, verifies
HTTP 302 responses, and saves the observed request durations (in seconds) to
`cache-hit.txt`. The output file is replaced only after all 100 requests succeed. A failed run
removes its temporary samples and preserves any previous output. Redirects are
not followed.

Check `GET /metrics` before and after running the script to confirm the requests
were served from Redis without additional PostgreSQL lookups. With an already
cached URL and no other traffic, `hits` and `redisGetCount` should each increase
by 101 (one warm-up plus 100 measured requests), while `postgresLookupCount`
stays unchanged. A 302 response alone does not prove a cache hit because the API
can fall back to PostgreSQL.

This script measures HTTP request latency using a separate curl invocation for
each request. It is a separate benchmark from the direct backend lookups reported
above; the results table was not generated by this script.

### Reproducing the Cache Miss Benchmark

**Prerequisites:** Docker, the Redis container (`go-url-shortener-redis`) with
Redis CLI available inside it, PostgreSQL, and the Go API must be running.

Run the benchmark from the project root using an existing short code:

```sh
./scripts/benchmark-miss.sh YOUR_SHORT_CODE
```

The script first verifies the short URL with an unmeasured preflight request,
then performs 100 sequential HTTP GET requests. Before each measured request,
it removes the corresponding Redis key and verifies that the key is absent.

Only successful HTTP 302 responses are recorded; the script stops on a failed
request. Request durations are saved in seconds to `cache-miss.txt`. The output
file is replaced only after all 100 requests succeed; a failed run removes its
temporary samples and preserves any previous output. Redirects are not followed. Redis key deletion and absence
checks are outside the measured HTTP request duration.

Verify the cache miss and PostgreSQL lookup counter increments using
`GET /metrics` before and after running the script. With healthy services and no
other traffic, the measured requests should add 100 to both `misses` and
`postgresLookupCount`. The preflight request adds one more cache lookup and may
also add one miss and PostgreSQL lookup if the URL was not already cached.

This method is intended for local, sequential benchmarking. It does not
guarantee cache misses if other requests concurrently repopulate the same Redis
key. It measures the cache-miss HTTP path, including Redis GET, PostgreSQL lookup,
and Redis SET, rather than a PostgreSQL-only baseline.

### Generating the Benchmark Report

After running both benchmark scripts, generate the summary report from the
project root:

```sh
./scripts/benchmark-report.sh
```

The script requires exactly 100 positive numeric samples in each of
`cache-hit.txt` and `cache-miss.txt`, with one value per line and a positive cache
hit average. Invalid or incomplete inputs fail without printing a report. Since
failed benchmark runs preserve previous results, confirm both runs completed
successfully before generating a new comparison.

The script calculates:

- Average HTTP response latency
- Median latency (p50)
- p95 latency using the nearest-rank method
- Average latency ratio between uncached and cached redirects (Cache Miss / Cache Hit)

All latency values are reported in milliseconds. The latency ratio is
dimensionless.

The comparison measures the application-level redirect paths under local,
sequential HTTP requests. It does not represent a standalone Redis versus
PostgreSQL benchmark.

## Formatting

Format Go files with `gofmt` and Terraform files with
`terraform -chdir=infrastructure/local fmt`. In VS Code, install the recommended
extensions listed in `.vscode/extensions.json` for editor formatting support.
