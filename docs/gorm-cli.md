# GORM CLI — Field Helpers (Type-Safe Queries Without Strings)

This project uses the [GORM CLI](https://github.com/go-gorm/cli) code generator
to produce **model-based field helpers**: typed, compile-checked values like
`generated.User.Username.Eq("alice")` that you drop into `gorm.G[T]` builders
instead of hand-writing `WHERE username = ?`.

> TL;DR: add/edit a struct in `internal/models/`, run `make gorm-gen`, then use
> `generated.<Model>.<Field>.<Op>(...)` from your repository.

---

## 1. One-time setup

None. The CLI is a **`tool` directive in `go.mod`** (`gorm.io/cli/gorm`), so it
is version-pinned with the rest of the module and fetched by the ordinary
module download — no `go install`, no GOPATH/PATH setup. To bump its version:

```bash
go get gorm.io/cli/gorm@latest
go mod tidy
```

## 2. Regenerate

```bash
# from repo root
make gorm-gen
# or, equivalently
go generate ./internal/models/...
```

The `//go:generate` directive lives at
[internal/models/generate.go](../internal/models/generate.go) and runs
`go tool gorm gen -i . -o ../generated`. This reads every `.go` file under
`internal/models/` and writes field helpers into `internal/generated/`
(created on first run, committed to the repo). **Never hand-edit anything
under `internal/generated/` — the next generate pass will overwrite it.**

Re-run whenever you:

- add, rename, or remove a field on a `models.*` struct;
- add a new model file under `internal/models/`.

---

## 3. Directory layout

```text
internal/
├── models/         hand-written GORM models — source of truth + go:generate anchor
│   ├── user.go, admin_role.go, config.go, refresh_token.go, log.go, common.go
│   └── generate.go    //go:generate go tool gorm gen -i . -o ../generated
└── generated/      OUTPUT — do not edit, do not grep for bugs here
    └── user.go, admin_role.go, config.go, refresh_token.go, log.go, common.go
```

Each file in `internal/generated/` declares a package-level `var <Model>` whose
fields are typed descriptors — `field.String`, `field.Number[uint]`,
`field.Time`, `field.Bool`, `field.Struct[T]`, `field.Slice[T]`, etc. Those
descriptors expose the predicates and setters you actually call.

---

## 4. Using the helpers

### 4a. Simple lookups

```go
import (
    "github.com/PhantomX7/athleton/internal/generated"
    "github.com/PhantomX7/athleton/internal/models"
    "gorm.io/gorm"
)

user, err := gorm.G[models.User](db).
    Where(generated.User.Username.Eq(username)).
    First(ctx)
```

### 4b. Multi-condition queries

Each call to `.Where(...)` AND's its arguments together; `.Or(...)` adds an OR
branch.

```go
q := gorm.G[models.RefreshToken](db).
    Where(generated.RefreshToken.UserID.Eq(uid)).
    Where(generated.RefreshToken.RevokedAt.IsNull()).
    Where(generated.RefreshToken.ExpiresAt.Gt(time.Now()))

count, err := q.Count(ctx, "*")
```

### 4c. Updates

```go
_, err := gorm.G[models.RefreshToken](db).
    Where(generated.RefreshToken.UserID.Eq(uid)).
    Set(generated.RefreshToken.RevokedAt.Set(time.Now())).
    Update(ctx)
```

`.Set(...)` accepts any number of setters. Use `.Incr(n)`, `.Decr(n)`,
`.Mul(n)`, `.Concat(s)`, `.Upper()`, `.SetExpr(clause.Expr{...})` for
arithmetic, string, and raw-SQL-fragment updates respectively.

### 4d. Deletes

```go
_, err := gorm.G[models.RefreshToken](db).
    Where(generated.RefreshToken.ExpiresAt.Lt(now)).
    Or(generated.RefreshToken.RevokedAt.IsNotNull()).
    Delete(ctx)
```

### 4e. Soft delete is automatic

Any model embedding `gorm.DeletedAt` (directly or via
[internal/models/common.go](../internal/models/common.go)'s `Timestamp` mixin)
gets automatic `deleted_at IS NULL` filtering on `First` / `Find` / `Count` /
`Delete`. Don't add it manually — GORM injects it. To include deleted rows
use `.Unscoped()`.

### 4f. Cheat sheet

| Op | For types | Example |
| --- | --- | --- |
| `.Eq`, `.Neq` | all | `generated.User.ID.Eq(1)` |
| `.Gt`, `.Gte`, `.Lt`, `.Lte`, `.Between` | number / time | `generated.User.AdminRoleID.Gt(0)` |
| `.In`, `.NotIn` | all | `generated.User.Email.In("a@b", "c@d")` |
| `.Like`, `.NotLike`, `.ILike`, `.Regexp` | string | `generated.User.Name.Like("%alice%")` |
| `.IsNull`, `.IsNotNull` | nullable / time | `generated.RefreshToken.RevokedAt.IsNull()` |
| `.Asc`, `.Desc` | all | `.Order(generated.User.CreatedAt.Desc())` |
| `.Set` | all | `.Set(generated.User.Name.Set("alice"))` |
| `.Incr`, `.Decr`, `.Mul` | number | `.Set(generated.Order.Count.Incr(1))` |
| `.Concat`, `.Upper`, `.Lower`, `.Trim`, `.Substring` | string | `.Set(generated.User.Name.Upper())` |
| `.SetExpr` | all | raw SQL update fragment |

---

## 4g. The generic base repository (`pkg/repository`)

`Repository[T]` is the interface every module repository satisfies.
`BaseRepository[T]` is the concrete struct module repositories embed, always
constructed via `NewBaseRepository[T](db, opts...)` so the cached entity name
and slow-query thresholds are populated — a zero-value `BaseRepository` has
a zero threshold, which would flag every query as slow.

The CRUD methods run on GORM's generics API (`gorm.G[T]`) wherever it is
cleaner, and stay on the classic API where generics genuinely can't express
the same semantics:

| Method | API | Why |
| --- | --- | --- |
| `Create` | `gorm.G[T].Create(ctx, entity)` | type-safe, no entity pointer in `.Error` plumbing |
| `FindByID(id, preloads...)` | `gorm.G[T].Where("id = ?", id).Preload(p, nil)...First(ctx)` | generics + chained preloads; the variadic takes `repository.Association`, so callers pass typed helpers like `generated.User.AdminRole` (compile-error on typos) instead of raw strings |
| `FindAll(pg)` | Classic `db.Scopes(pg.Apply).Find(&entities)` | Pagination's scopes are `func(*gorm.DB) *gorm.DB`; generics' `Scopes` expects `func(*gorm.Statement)`. A "pre-apply to `*gorm.DB` then wrap with `gorm.G[T]`" bridge does **not** work either — `gorm.G[T]` calls `db.Session(&Session{NewDB: true})` on every finisher, which discards the accumulated clauses (verified by the test suite). Rewriting every pagination scope as a `func(*Statement)` is the only way to generify this, and offers no material gain |
| `Count(pg)` | Classic `db.Scopes(pg.ApplyWithoutMeta).Model(new(T)).Count(&count)` | Same reasoning as `FindAll` |
| `Update` | `db.Save(entity)` (classic) | `gorm.G[T]` has no `Save`. Matching it in generics would require reflecting PK columns from the schema manually — GORM already does this in `Save` |
| `Delete(entity)` | `db.Delete(entity)` (classic) | `gorm.G[T].Delete(ctx)` requires an explicit `Where`; `db.Delete(entity)` extracts the PK from the entity via schema reflection |

`BaseRepository[T].GetDB(ctx)` still returns the transaction-scoped `*gorm.DB`
when the context carries one, so the transaction manager keeps working
unchanged.

`FindByID` returns `(nil, err)` on **any** error, including not-found, so
services never accidentally dereference a zero-value entity after an `err != nil`
check.

### Slow-query thresholds

`NewBaseRepository` accepts functional options for the slow-query thresholds:

```go
// Defaults: 1s (reads), 500ms (writes).
r := repository.NewBaseRepository[models.User](db)

// Per-repository overrides — e.g. a hot endpoint that needs tighter alerting.
r := repository.NewBaseRepository[models.User](db,
    repository.WithSlowReadThreshold(200*time.Millisecond),
    repository.WithSlowWriteThreshold(100*time.Millisecond),
)
```

Inside custom methods on an embedding repo, call `r.LogSlowRead(ctx, "FindByEmail", duration)`
or `r.LogSlowWrite(ctx, "RevokeByToken", duration)` so the configured threshold
is honored; the lower-level `LogSlowQuery(ctx, op, duration, threshold)` is
kept for the rare one-off threshold.

### Typed preloads

`FindByID`'s variadic is `...repository.Association`, an interface satisfied
by every `field.Struct[T]` / `field.Slice[T]` the generator emits. In
practice:

```go
// Typed — renaming the User.AdminRole field in internal/models breaks this
// call at compile time after the next `make gorm-gen`.
user, err := s.userRepo.FindByID(ctx, id, generated.User.AdminRole)

// Multiple preloads are just more arguments:
user, err := s.userRepo.FindByID(ctx, id, generated.User.AdminRole, generated.User.Logs)

// Runtime-dynamic name (rare): repository.Preload wraps a raw string.
assoc := repository.Preload(cfgDecidedAtStartup)
user, err := s.userRepo.FindByID(ctx, id, assoc)
```

Callers never build a string literal in normal code, so a future model rename
can't silently leave a dead `"AdminRole"` preload behind.

## 5. How this repo uses it

| Repository | Bespoke method | Implementation |
| --- | --- | --- |
| `modules/user/repository` | `FindByUsername` | `gorm.G[User].Where(generated.User.Username.Eq(...)).First(ctx)` |
| `modules/user/repository` | `FindByEmail` | same with normalized (lowercase, trimmed) email via `generated.User.Email.Eq` |
| `modules/admin_role/repository` | `FindByName` | `gorm.G[AdminRole].Where(generated.AdminRole.Name.Eq(...)).First(ctx)` |
| `modules/admin_role/repository` | `CountUsersWithRole` | `gorm.G[User].Where(generated.User.AdminRoleID.Eq(id)).Count(ctx, "*")` — soft-delete filter auto-applied |
| `modules/config/repository` | `FindByKey` | `gorm.G[Config].Where(generated.Config.Key.Eq(...)).First(ctx)` |
| `modules/refresh_token/repository` | all six methods | `gorm.G[RefreshToken]` + `generated.RefreshToken.*` predicates and setters; the "active token" predicate (`RevokedAt IS NULL AND ExpiresAt > now`) lives in a local `activeTokenPredicates` helper so the six methods share one definition |

The generic CRUD in [pkg/repository/repository.go](../pkg/repository/repository.go)
is untouched — it still owns slow-query logging, pagination, and error wrapping,
which the generator intentionally does not replace.

### Behavioural delta to note

The old `RevokeByToken` returned `ErrNotFound` when `RowsAffected == 0`
(i.e. the token was already revoked or missing). The field-helper version
only surfaces database errors. Both existing callers in
[internal/modules/auth/jwt/jwt.go](../internal/modules/auth/jwt/jwt.go) either
call `FindByToken` first or ignore the error, so the change is safe — but if
you add a caller that needs the "nothing to revoke" signal, call `FindByToken`
before revoking.

---

## 6. Adding a new bespoke query

1. Write the query with field helpers in the appropriate repository file.
2. Share multi-field predicates by pulling them into a local helper that
   returns `[]clause.Expression` (see `activeTokenPredicates` in
   refresh_token/repository/repository.go for the pattern).
3. No generation step required — field helpers already cover every column on
   every model.

## 7. When to reach for SQL templates instead

Field helpers do **not** cover:

- Dynamic `UPDATE ... SET` where the list of columns depends on runtime
  conditions (GORM CLI's `{{set}}` / `{{if}}` templates are designed for this).
- Iteration (`{{for _, x := range xs}}`) to build `VALUES (...), (...)` or
  chained `OR` branches.
- Truly dynamic column names (`@@column`).

If you hit one of these, create an `internal/queries/` package, define a Go
interface with SQL-template comments in its method docstrings, add a
`//go:generate go tool gorm gen -i . -o ../generated` directive there, and register
its name in a local `genconfig.Config{IncludeInterfaces: ...}`. The generator
writes a concrete, type-safe implementation alongside the field helpers. A
working starting template is in the GORM CLI README:
<https://github.com/go-gorm/cli>.

## 8. Adding a new model

1. Create the struct in `internal/models/`.
2. Run `make gorm-gen` — helpers appear as `generated.<ModelName>.<FieldName>`.

No config edits required: the directive scans the entire models package.

## 9. Transactions

The repositories call `r.GetDB(ctx)` which returns a transaction-scoped
`*gorm.DB` when the context carries one (placed there by
`libs/transaction_manager`). `gorm.G[T]` is a thin wrapper over `*gorm.DB`, so
it plays cleanly with the existing transaction manager:

```go
_ = s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
    if err := s.userRepo.Create(txCtx, user); err != nil {
        return err
    }
    // The repository uses r.GetDB(txCtx), which picks up the tx.
    return s.authJWT.RevokeAllUserTokensExcept(txCtx, user.ID, except)
})
```

## 10. Troubleshooting

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| `no required module provides package ".../internal/generated"` | You have not run `make gorm-gen` yet. | Run `make gorm-gen`. |
| `go: no such tool "gorm"` | The `tool` directive is missing from `go.mod` (or go.sum is stale). | `go get -tool gorm.io/cli/gorm && go mod tidy`. |
| `missing go.sum entry for module providing package ...` when building | `gorm.io/cli/gorm`'s runtime deps (e.g. `golang.org/x/exp`) are not tracked. | `go mod tidy`. |
| Generated file references a field you just renamed | Stale output. | `make gorm-gen`. |
| `deleted_at` filter missing / applied twice | Manually adding `deleted_at IS NULL` on a model that embeds `gorm.DeletedAt` — GORM already does this. | Remove the manual predicate. Use `.Unscoped()` to opt out. |
| CI compiles locally but not in CI | `internal/generated/` is not committed. | Commit the generated code (recommended) or add `make gorm-gen` to the CI script before `go build` — the CLI comes with the module via the `tool` directive, nothing to install. |

## 11. Further reading

- Repo: <https://github.com/go-gorm/cli>
- Field-helper reference: <https://github.com/go-gorm/cli#field-helpers>
- GORM generics API (`gorm.G[T]`): <https://gorm.io/docs/generics.html>
