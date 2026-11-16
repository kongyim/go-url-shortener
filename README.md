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

PostgreSQL stores its data in `infrastructure/local/data/postgres/`, which is
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

## Formatting

Run `npm install` to install the project's Prettier version.

In VS Code, install the workspace's recommended extensions (open the Command
Palette and select **Extensions: Show Recommended Extensions**). Workspace settings
enable formatting whenever you save a file.

Prettier formats supported files such as JSON, Markdown, JavaScript, and YAML.
Go and Terraform files use their respective extensions instead; install Go and
Terraform locally to use those formatters.

- `npm run format` formats all Prettier-supported files.
- `npm run format:check` checks formatting without changing files.
