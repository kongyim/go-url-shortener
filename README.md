# Go URL Shortener

## Development commands

Copy `.env.example` to `.env` for local configuration and update the database
credentials to match your PostgreSQL container. The API uses `godotenv.Load()` at
startup to load `.env` from the current working directory, with both `make run`
and `go run ./cmd/api`. Existing environment variables take precedence over values
in `.env`. A missing `.env` is allowed; other loading errors stop startup.
Use `KEY='value'` assignments. `.env` is ignored by Git.

- `make run` starts the API.
- `make test` runs all Go tests.
- `make migrate-up` creates the `urls` table using migration 001.
- `make migrate-down` drops the `urls` table and its data using migration 001.

Migration commands require the `go-url-shortener-postgres` Docker container to be
running. They execute the specified SQL files directly and do not track migration
history; running `make migrate-up` again skips creation if the table already exists.

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
