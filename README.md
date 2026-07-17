# Athleton

Go API service built on [gin](https://github.com/gin-gonic/gin), [GORM](https://gorm.io/), [uber-go/fx](https://github.com/uber-go/fx), and [Atlas](https://atlasgo.io/) for schema migrations. Search is backed by [Bleve](https://blevesearch.com/) and asset storage by S3.

## Prerequisites

- **Go** 1.26+ (toolchain pinned to 1.26.4 in `go.mod`)
- **PostgreSQL** 14+ (configurable via `DATABASE_DRIVER`)
- **Atlas CLI** — `curl -sSf https://atlasgo.sh | sh` (used by `make migrate-*`)
- **swag** (optional, for regenerating API docs) — `go install github.com/swaggo/swag/cmd/swag@latest`

Dev tooling installed via Make targets:

```bash
make lint-install    # golangci-lint v2.12.2
make hooks-install   # lefthook + git hooks (pre-commit, pre-push)
```

## Quick start

```bash
cp .env.example .env          # fill in DB + S3 + JWT + ADMIN_* values
make dep                      # go mod tidy
make migrate-up               # apply schema
make seed                     # seed the root + admin users (uses ADMIN_* env vars)
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
  middlewares/      auth/role/permission guards, rate limit, body limit,
                    request ID, logging, timeout, recovery, error handler
  models/           GORM entities (source of truth for schema)
  modules/          Vertical slices — one folder per domain
                    (auth, user, admin_role, config, cron, log, refresh_token)
                    Each module has controller/ service/ repository/ + routes.go.
  routes/           Route registration + the shared /admin middleware stack
  dto/              Shared request/response DTOs
  audit/            Audit log helpers
  integration/      End-to-end tests: a shared harness/ package boots the real
                    app, one sub-package per domain (auth, authz, user, …)
  generated/        gorm-cli field helpers (regenerated, do not edit)
libs/             Third-party adapters
  bleve/            Search index + pagination helpers
  casbin/           RBAC enforcer
  s3/               S3 / DigitalOcean Spaces client
  transaction_manager/  DB transaction orchestration
pkg/              Reusable, framework-agnostic primitives
  config/ constants/ errors/ generator/ ginx/ logger/
  pagination/ repository/ response/ utils/ validator/
docs/             Swagger output (swagger.json / swagger.yaml / docs.go)
```

Besides the API, the server also mounts static routes from `APP_ASSETS`:
`/assets`, `/sitemaps`, and `/sitemap.xml`
([internal/bootstrap/bootstrap.go](internal/bootstrap/bootstrap.go)).

### Adding a new module

```bash
make module module_name=widget                        # module + model + DTO + permissions (default)
make module module_name=widget model=0 dto=0          # module only (model/DTO already exist)
make module module_name=widget permissions=0          # skip permission registration
make module module_name=widget force=1                # overwrite an existing module
```

The scaffolder wires everything automatically:

- registers the module in [internal/modules/modules.go](internal/modules/modules.go)
- mounts **permission-guarded** admin CRUD routes via the module's `routes.go`
- registers `widget:create/read/update/delete` in
  [pkg/constants/permissions/permissions.go](pkg/constants/permissions/permissions.go)
  so the generated routes compile and the permissions are assignable to roles
- refreshes the typed field helpers in [internal/generated/](internal/generated/) (`make gorm-gen`)

Existing files are never overwritten unless `force=1` is passed. After
scaffolding: fill in the business logic, create a migration
(`make migrate-create`), assign the new permissions to admin roles, and grow
the generated tests. See
[cmd/generate/README_GENERATOR.md](cmd/generate/README_GENERATOR.md) for details.

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
| `make debug name=add_foo` | Echo the migration name a `migrate-create` would use |

### Quality
| Target | Description |
| --- | --- |
| `make test` | `go test ./... -coverprofile cp.out` |
| `make test-html` | Run tests and open HTML coverage report |
| `make lint` | `golangci-lint run ./...` |
| `make lint-fix` | `golangci-lint run --fix ./...` |
| `make fmt` | `golangci-lint fmt` (gofmt + goimports) |
| `make vuln` | `govulncheck ./...` (same scan CI runs weekly) |
| `make hooks-install` | Install lefthook git hooks |
| `make hooks-uninstall` | Remove lefthook git hooks |
| `make hooks-run` | Run pre-commit checks across all files |

### Codegen
| Target | Description |
| --- | --- |
| `make swag` | Regenerate Swagger docs |
| `make swag-format` | Format Swagger annotations only (`swag fmt`) |
| `make module module_name=foo [model=0] [dto=0] [permissions=0] [force=1]` | Scaffold a new module (+ model + DTO + permissions by default) |
| `make gorm-gen` | Regenerate GORM CLI field helpers |
| `make mocks` | Regenerate moq interface mocks (`go generate -run moq ./...`) |

## Testing

```bash
make test         # all packages with coverage
make test-html    # HTML report from cp.out
```

Tests run through `lefthook` on `pre-push`; CI additionally runs them with `-race` (the race detector needs a C toolchain, which not every dev machine has). To skip hooks temporarily: `git push --no-verify` (prefer fixing the failure).

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

## Authorization & roles

Every user row carries one of three roles:

| Role | How it's created | Access |
| --- | --- | --- |
| `user` | Self-registration (`POST /auth/register`) | Public endpoints only |
| `admin` | Created by an admin (`POST /admin/user`) or promoted from a user (`POST /admin/user/:id/admin-role`) | `/admin` routes, gated per-permission by the assigned admin role |
| `root` | Seeder only — never via the API | Bypasses all permission checks |

**Root is protected by construction.** No request field can produce a `root`
account (registration hardcodes `user`, admin-create hardcodes `admin`), and
every mutating user endpoint refuses a root target — update, role assignment,
password change, and delete all return 403. Root rotates its own password only
through the self-service `/auth/change-password`.

**The `/admin` surface is defended in depth.** The group middleware runs
`AdminRateLimiter → RequireAuth → RequireRole(admin, root) →
RequirePasswordChanged` before any handler, so a plain user is rejected even
if a route forgets its per-route guard. Individual routes then authorize admins with fine-grained
`resource:action` permissions enforced via [Casbin](libs/casbin/); the registry
lives in [pkg/constants/permissions](pkg/constants/permissions/). Managing
*admin* accounts requires the stronger `admin_user:*` grants — `user:*` governs
regular accounts only.

**Seeded admin/root accounts must rotate their password.** An account whose
password it did not choose itself (seeded, or created by another admin) has a
null `PasswordChangedAt` and is blocked from `/admin` by the
must-change-default-password gate until it rotates via `/auth/change-password`.

**Public config is opt-in.** The unauthenticated `/public/config` surface only
serves rows explicitly marked `is_public`; everything else is admin-only, so the
config table can safely hold secrets. Toggle visibility with the `is_public`
field on the admin config update.

## Configuration

All config is loaded from `.env` via [pkg/config](pkg/config/). See [.env.example](.env.example) for the full list. Key sections:

- `SERVER_*` — bind host/port, timeouts, request-body cap
  (`SERVER_MAX_BODY_BYTES`), trusted proxies (`SERVER_TRUSTED_PROXIES`), and
  CORS origins (`SERVER_CORS_ALLOWED_ORIGINS`)
- `DATABASE_*` — connection string components
- `JWT_*` — secret, access + refresh token TTLs, per-user session cap
  (`JWT_MAX_ACTIVE_SESSIONS`)
- `APP_*` — app name/version, environment, assets directory
- `S3_*` — storage credentials, CDN URL, upload ACL (`S3_UPLOAD_ACL`)
- `BLEVE_*` — search index path/type
- `LOG_*` — log level, rotation, console output
- `ADMIN_*` — **required**: `ADMIN_DEFAULT_PASSWORD` seeds the root/admin
  accounts and has no default (weak or known values are rejected in
  production); `ADMIN_EMAIL` is seeded into the root account. `make seed`
  fails without them.

## Git hooks

`make hooks-install` wires up [lefthook.yml](lefthook.yml):

- **pre-commit** — `golangci-lint fmt` on staged files (auto-staged), `go vet`, `golangci-lint run --fast-only`
- **pre-push** — `go test ./...` (CI runs the same suite with `-race`)
