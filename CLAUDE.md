# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Tool

This project uses [Task](https://taskfile.dev) (`Taskfile.yml`) as the primary build automation tool — not Make.

```sh
task build       # Format, generate code, vet, and build binary
task test        # Run tests with race detection and coverage
task lint        # Run golangci-lint
task run         # Build and run with -debug flag and config.json
task dev         # Hot reload development (using air)
task generate    # Run sqlc + templ + tailwind code generation
task deps        # Tidy go modules
task setup       # Install npm deps (Tailwind CSS, DaisyUI)
task configcheck # Validate config.json
```

To run a single test:
```sh
go test -race -run TestFunctionName ./internal/server/handlers/
```

Tests require `CGO_ENABLED=1` (race detector needs cgo). The build itself uses `CGO_ENABLED=0`.

## Code Generation

Three generation steps must run before building:

1. **sqlc** (`go tool sqlc generate`) — generates type-safe DB query code in `internal/database/sqlc/`
2. **templ** (`go tool templ fmt . && go tool templ generate`) — generates Go code from `.templ` files
3. **tailwind** (`npx @tailwindcss/cli ...`) — compiles `input.css` → `style.min.css`

Run `task generate` to do all three. Never manually edit generated files.

## Architecture

### Request Flow

```
main.go → server.go → router.go → middleware chain → handlers
```

- `main.go` initializes all services (logger, config, DB, notifications, metrics, HTTP client, cacher, mailer) and passes them into the server via `options.go`
- `internal/server/router/router.go` builds the middleware chain: access log → real IP → real host → secret key → panic recovery
- Handlers return `error` values; the server's error handler processes them, maps to HTTP status codes, and sends notifications for 5xx errors

### Database

- SQLite with WAL mode via `modernc.org/sqlite` (pure Go, no cgo required for build)
- Dual connection pools: writer (1 conn) and reader (up to 100 conns) in `internal/database/database.go`
- Migrations via Goose in `internal/database/migrations/`
- Type-safe queries via SQLC — write SQL in `internal/database/queries/queries.sql`, run `task sqlc`, use generated code from `internal/database/sqlc/`
- Custom queries (non-SQLC) go in `internal/database/database_custom.go`
- `internal/database/mockdb.go` provides a mock for handler tests

### Configuration

Config is parsed by koanf from a JSON file (path via `-config` flag). The struct is defined and validated in `internal/config/config.go`. The config file path defaults to `config.json`; see `config.sample.json` for all options.

### Templates & Frontend

- HTML templates use [Templ](https://templ.guide/) — edit `.templ` files, never the generated `_templ.go` files
- Tailwind CSS v4 + DaisyUI v5 for styling; HTMX for interactivity
- Static assets are embedded in the binary via `//go:embed` in `internal/server/assets/`

### Notifications

`notify.go` configures multi-service notifications (Telegram, Discord, Email, Mailgun, MS Teams) using `github.com/nikoksr/notify`. The notifier is injected into the server and triggered automatically on 5xx errors.

## Linting

`.golangci.yml` enables 40+ linters with strict settings. Key enforced rules:
- Use `log/slog` for logging (not the `log` package, except in `main.go`)
- Use `google.golang.org/protobuf` (not the old `github.com/golang/protobuf`)
- All errors must be checked
- No `math/rand` — use `crypto/rand`

Run `task lint` before submitting changes. The CI also runs golangci-lint on a schedule.

## Three HTTP Servers

The application can run up to three HTTP servers simultaneously (all optional except main):
1. **Main** — configured by `server.listen`
2. **Metrics** — Prometheus `/metrics` endpoint, configured by `server.listen_metrics`
3. **Pprof** — Go profiling, configured by `server.listen_pprof`
