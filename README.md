# Athleton

Go API service built on [gin](https://github.com/gin-gonic/gin), [GORM](https://gorm.io/), [uber-go/fx](https://github.com/uber-go/fx), and [Atlas](https://atlasgo.io/) for schema migrations. Search is backed by [Bleve](https://blevesearch.com/) and asset storage by S3.

## Prerequisites

- **Go** 1.24+ (toolchain pinned to 1.24.7 in `go.mod`)
- **PostgreSQL** 14+ (configurable via `DATABASE_DRIVER`)
- **Atlas CLI** — `curl -sSf https://atlasgo.sh | sh` (used by `make migrate-*`)
- **swag** (optional, for regenerating API docs) — `go install github.com/swaggo/swag/cmd/swag@latest`

Dev tooling installed via Make targets:

```bash
make lint-install    # golangci-lint v2.11.0
make hooks-install   # lefthook + git hooks (pre-commit, pre-push)
```

## Quick start

```bash
cp .env.example .env          # fill in DB + S3 + JWT values
make dep                      # go mod tidy
make migrate-up               # apply schema
make seed                     # optional: seed admin user + roles
make run                      # start on SERVER_PORT (default 8080)
```

Swagger UI is served at `http://localhost:${SERVER_PORT}/swagger/index.html` once the server is up.

## Project layout

```
cmd/              Entry points
  main.go           API server bootstrap (fx wiring)
  generate/         Module scaffolder — `make module name=foo`
database/
  main.go           GORM connection + Atlas loader entrypoint
  migrations/       Atlas-generated SQL migrations (golang-migrate format)
  schema/           GORM models exposed to Atlas for diffing
  seeder/           Seed runner + per-domain seeds
internal/         Application code (not importable by other modules)
  bootstrap/        fx providers wiring config, DB, logger, S3, bleve, etc.
  middlewares/      auth, has_role, request logging, recovery
  models/           GORM entities (source of truth for schema)
  modules/          Vertical slices — one folder per domain
                    (auth, user, admin_role, config, cron, log, post, refresh_token)
                    Each module has controller/ service/ repository/ + routes.go.
  routes/           Route registration per module
  dto/              Shared request/response DTOs
  audit/            Audit log helpers
  generated/        gorm-cli field helpers (regenerated, do not edit)
libs/             Third-party adapters
  bleve/            Search index + pagination helpers
  casbin/           RBAC enforcer
  s3/               S3 / DigitalOcean Spaces client
  transaction_manager/  DB transaction orchestration
pkg/              Reusable, framework-agnostic primitives
  config/ constants/ errors/ generator/ logger/ pagination/
  repository/ response/ utils/ validator/
docs/             Swagger output (swagger.json / swagger.yaml / docs.go)
```

### Adding a new module

```bash
make module module_name=widget                    # module + model + DTO (default)
make module module_name=widget model=0 dto=0      # module only (model/DTO already exist)
make module module_name=widget force=1            # overwrite an existing module
```

The scaffolder wires everything automatically: registers the module in
[internal/modules/modules.go](internal/modules/modules.go), mounts admin CRUD
routes via the module's `routes.go`, and refreshes the typed field helpers in
[internal/generated/](internal/generated/) (`make gorm-gen`). Existing files are
never overwritten unless `force=1` is passed. After scaffolding: fill in the
business logic, create a migration (`make migrate-create`), and replace the
placeholder tests. See [cmd/generate/README_GENERATOR.md](cmd/generate/README_GENERATOR.md)
for details.

## Make targets

### Build & run
| Target | Description |
| --- | --- |
| `make run` | Run the API via `go run cmd/main.go` |
| `make dev` | Build to `bin/athleton` and execute |
| `make build` | Cross-compile for `linux/amd64` (deploy artifact) |
| `make dep` | `go mod tidy` |
| `make vendor` | `go mod vendor` |

### Database & migrations
| Target | Description |
| --- | --- |
| `make migrate-create name=add_foo` | Diff GORM schema → new migration file |
| `make migrate-up` | Apply pending migrations |
| `make migrate-down` | Revert the last applied migration |
| `make migrate-status` | Show pending vs. applied migrations |
| `make migrate-hash` | Re-hash migration files after manual edits |
| `make seed` | Run the seeder (`database/seeder/main.go`) |

### Quality
| Target | Description |
| --- | --- |
| `make test` | `go test ./... -coverprofile cp.out` |
| `make test-html` | Run tests and open HTML coverage report |
| `make lint` | `golangci-lint run ./...` |
| `make lint-fix` | `golangci-lint run --fix ./...` |
| `make fmt` | `golangci-lint fmt` (gofmt + goimports) |
| `make hooks-install` | Install lefthook git hooks |
| `make hooks-run` | Run pre-commit checks across all files |

### Codegen
| Target | Description |
| --- | --- |
| `make swag` | Regenerate Swagger docs |
| `make module module_name=foo [model=0] [dto=0] [force=1]` | Scaffold a new module (+ model + DTO by default) |
| `make gorm-gen` | Regenerate GORM CLI field helpers |

## Testing

```bash
make test         # all packages with coverage
make test-html    # HTML report from cp.out
```

Tests run through `lefthook` on `pre-push` with `-race`. To skip hooks temporarily: `git push --no-verify` (prefer fixing the failure).

## Migrations

Schema is declared in GORM models under [internal/models/](internal/models/). Atlas diffs those models against the current migration state to produce SQL.

```bash
# Edit a model, then:
make migrate-create name=add_widget_table
make migrate-up
```

Migration files live in [database/migrations/](database/migrations/) in `golang-migrate` format so they can be applied by any compatible tool in production.

## API docs

Swagger annotations live alongside controllers. Regenerate with `make swag` whenever a handler signature, tag, or response type changes. Committed outputs are `docs/swagger.json`, `docs/swagger.yaml`, and `docs/docs.go`.

## Configuration

All config is loaded from `.env` via [pkg/config](pkg/config/). See [.env.example](.env.example) for the full list. Key sections:

- `SERVER_*` — bind host/port + timeouts
- `DATABASE_*` — connection string components
- `JWT_*` — secret, access + refresh token TTLs
- `S3_*` — storage credentials and CDN URL
- `BLEVE_*` — search index path/type
- `LOG_*` — log level, rotation, console output

## Git hooks

`make hooks-install` wires up [lefthook.yml](lefthook.yml):

- **pre-commit** — `golangci-lint fmt` on staged files (auto-staged), `go vet`, `golangci-lint run --fast-only`
- **pre-push** — `go test -race ./...`
