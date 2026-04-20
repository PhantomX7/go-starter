# GORM CLI — Type-Safe Queries & Field Helpers

This project uses the [GORM CLI](https://github.com/go-gorm/cli) code generator to
produce **type-safe query APIs** and **model-based field helpers** from Go
interfaces and struct definitions. The generated code replaces most hand-written
`WHERE ... = ?` strings in the repository layer with compile-time-checked calls.

> TL;DR: edit `internal/queries/*.go`, run `make gorm-gen`, import
> `github.com/PhantomX7/athleton/internal/generated` from your repository.

---

## 1. One-time setup

```bash
# install the CLI (idempotent)
go install gorm.io/cli/gorm@latest

# pull gorm.io/cli into go.mod (field helpers are a runtime dependency)
go get gorm.io/cli/gorm@latest
go mod tidy
```

`$(go env GOPATH)/bin` must be on your `PATH` so the `gorm` binary is visible
to `go generate`.

## 2. Generate

```bash
# from repo root
make gorm-gen
# or, equivalently
go generate ./internal/queries/...
```

This reads every file in [`internal/queries/`](../internal/queries) and writes
Go code into `internal/generated/` (created on first run, committed to the
repo). **Never hand-edit anything under `internal/generated/` — the next
generate pass will overwrite it.**

You need to re-run the generator whenever you:

- add, rename, or remove a field on a `models.*` struct;
- add, rename, or remove a method on a query interface in
  `internal/queries/`;
- add a new query interface.

---

## 3. Directory layout

```text
internal/
├── models/         hand-written GORM models (source of truth)
├── queries/        hand-written query interfaces — INPUT to the generator
│   ├── doc.go             //go:generate directive + package doc
│   ├── genconfig.go       genconfig.Config (output path, includes)
│   ├── queries.go         generic Query[T any] — GetByID / ExistsByID / DeleteByID
│   ├── user.go            UserQuery          (FindByUsername, FindByEmail, ...)
│   ├── admin_role.go      AdminRoleQuery     (FindByName)
│   ├── config.go          ConfigQuery        (FindByKey)
│   ├── refresh_token.go   RefreshTokenQuery  (revoke/find active, GC invalid)
│   └── log.go             LogQuery           (CountByEntity)
└── generated/      OUTPUT — do not edit, do not grep for bugs here
```

`internal/modules/*/repository/repository.go` imports
`github.com/PhantomX7/athleton/internal/generated` and delegates bespoke
queries to the generated functions, keeping error wrapping and slow-query
logging in the repo layer.

---

## 4. Writing a query interface

Two styles coexist in `internal/queries/`.

### 4a. SQL template interfaces (generated implementations)

For complex predicates, multi-field filters, and UPDATE statements, declare a
Go interface whose method comments contain an SQL template. The generator
emits a concrete implementation.

```go
// internal/queries/user.go
package queries

import "github.com/PhantomX7/athleton/internal/models"

type UserQuery interface {
    // SELECT * FROM @@table WHERE username=@username LIMIT 1
    FindByUsername(username string) (models.User, error)

    // SELECT * FROM @@table WHERE LOWER(email)=LOWER(@email) LIMIT 1
    FindByEmail(email string) (models.User, error)

    // SELECT COUNT(*) FROM @@table WHERE admin_role_id=@roleID AND deleted_at IS NULL
    CountWithAdminRole(roleID uint) (int64, error)
}
```

After `make gorm-gen` the repository calls the generated function (same name
as the interface, with `ctx` auto-injected as the first argument). The
generator emits each query as a generic function `func Name[T any](db) ...`,
so the model type must be supplied explicitly at the call site:

```go
user, err := generated.UserQuery[models.User](r.GetDB(ctx)).FindByUsername(ctx, username)
```

#### Template DSL cheat sheet

| Token | Meaning |
| --- | --- |
| `@@table` | Model's table name, inferred from the method's return type (or `T` in generic interfaces) |
| `@@column` | Dynamic column binding — pass the column name as a parameter |
| `@paramName` | Bind a Go parameter into SQL. Supports dotted paths: `@user.Name` |
| `{{if cond}} ... {{else if}} ... {{else}} ... {{end}}` | Conditional SQL fragments |
| `{{where}} ... {{end}}` | Wraps conditions; emits `WHERE` only if any branch fires, drops leading `AND`/`OR` |
| `{{set}} ... {{end}}` | Same idea for `UPDATE ... SET` — drops the trailing comma |
| `{{for _, x := range xs}} ... {{end}}` | Iteration (OR-of-conditions, VALUES lists, etc.) |

Example of a conditional UPDATE (see `RefreshTokenQuery.RevokeActiveByUserIDExcept`
for a real one in this repo):

```go
// UPDATE @@table
//   {{set}}
//     {{if name != ""}} name=@name, {{end}}
//     age=@age
//   {{end}}
// WHERE id=@id
UpdateInfo(id uint, name string, age int) error
```

### 4b. Generic `Query[T any]`

For methods that apply to every model (get-by-id, exists-by-id, delete-by-id),
declare a generic interface. The generator emits one call site that works for
all models:

```go
user,  err := generated.Query[models.User](db).GetByID(ctx, 123)
cfg,   err := generated.Query[models.Config](db).GetByID(ctx, 42)
```

Our `pkg/repository.Repository[T]` base still uses hand-written code because
it also tracks slow-query timing and wraps errors; the generic `Query[T]`
generated here is available if you want a thinner alternative.

### 4c. Field helpers (no interface needed)

For single-column lookups, use the generated field helpers directly instead
of declaring an interface method:

```go
import (
    "github.com/PhantomX7/athleton/internal/generated"
    "gorm.io/gorm"
)

user, err := gorm.G[models.User](db).
    Where(generated.User.Username.Eq(username)).
    First(ctx)
```

The helpers are fully typed — `Eq`, `Neq`, `Gt`, `Gte`, `Lt`, `Lte`, `In`,
`NotIn`, `Between`, `Like`, `ILike`, `IsNull`, `IsNotNull`, `Asc`, `Desc`,
`Set`, `Incr`, `Decr`, `Mul`, `Concat`, `Upper`, `Lower`, `Trim`,
`Substring`, `SetExpr`, etc. — so typos become compile errors.

---

## 5. How this repo uses it

All four bespoke repositories delegate to generated code; the generic CRUD in
`pkg/repository/repository.go` is untouched because it also owns the
pagination + slow-query + error-wrapping layer.

| Repository | Bespoke method | Delegates to |
| --- | --- | --- |
| `modules/user/repository` | `FindByUsername` | `generated.UserQuery[models.User](db).FindByUsername` |
| `modules/user/repository` | `FindByEmail` | `generated.UserQuery[models.User](db).FindByEmail` |
| `modules/admin_role/repository` | `FindByName` | `generated.AdminRoleQuery[models.AdminRole](db).FindByName` |
| `modules/admin_role/repository` | `CountUsersWithRole` | `generated.UserQuery[models.User](db).CountWithAdminRole` |
| `modules/config/repository` | `FindByKey` | `generated.ConfigQuery[models.Config](db).FindByKey` |
| `modules/refresh_token/repository` | `FindByToken` | `generated.RefreshTokenQuery[models.RefreshToken](db).FindActiveByToken` |
| `modules/refresh_token/repository` | `GetValidCountByUserID` | `… CountActiveByUserID` |
| `modules/refresh_token/repository` | `DeleteInvalidToken` | `… DeleteInvalid` |
| `modules/refresh_token/repository` | `RevokeAllByUserID` | `… RevokeActiveByUserID` |
| `modules/refresh_token/repository` | `RevokeAllByUserIDExcept` | `… RevokeActiveByUserIDExcept` |
| `modules/refresh_token/repository` | `RevokeByToken` | `… RevokeByToken` |

### Behavioural delta to note

The old `RevokeByToken` returned `ErrNotFound` when
`RowsAffected == 0` (i.e. the token was already revoked or missing). The
generated version only surfaces database errors. Both existing callers
(`AuthJWT.RevokeRefreshToken`, `ValidateAndRotateRefreshToken`) either call
`FindByToken` first or ignore the error, so the behaviour change is safe — but
if you add a new caller that needs to distinguish "nothing to revoke" you
must `FindByToken` before revoking.

---

## 6. Adding a new bespoke query

1. Decide whether it is model-specific (→ `internal/queries/<model>.go`) or
   generic across models (→ `internal/queries/queries.go`).
2. Add a method with an SQL-template comment. Prefer `{{where}}` /
   `{{set}}` over hand-building strings — they automatically drop the
   leading `AND`/trailing `,` in the zero-branch case.
3. If you introduced a brand-new interface, add its name to
   `genconfig.Config.IncludeInterfaces` in
   [`internal/queries/genconfig.go`](../internal/queries/genconfig.go).
4. Run `make gorm-gen`.
5. Call it from the repository, wrapping `gorm.ErrRecordNotFound` with
   `cerrors.NewNotFoundError` where the caller expects a
   404 / "not-found" error.

## 7. Adding a new model

1. Create the struct in `internal/models/`.
2. Because `genconfig.Config.IncludeStructs` is `"*"`, the generator picks it
   up automatically — field helpers appear as
   `generated.<ModelName>.<FieldName>` after the next `make gorm-gen`.
3. If you need bespoke queries for it, add an interface file in
   `internal/queries/` and register it in `IncludeInterfaces`.

## 8. Transactions

The repositories call `r.GetDB(ctx)` which returns a transaction-scoped
`*gorm.DB` when the context carries one (placed there by
`libs/transaction_manager`). Generated functions accept any `*gorm.DB`, so
they play cleanly with the existing transaction manager — no special
handling needed:

```go
_ = s.txManager.ExecuteInTransaction(ctx, func(txCtx context.Context) error {
    if err := s.userRepo.Create(txCtx, user); err != nil {
        return err
    }
    // The generated function uses r.GetDB(txCtx), which picks up the tx.
    return s.authJWT.RevokeAllUserTokensExcept(txCtx, user.ID, except)
})
```

## 9. Troubleshooting

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| `no required module provides package ".../internal/generated"` | You have not run `make gorm-gen` yet. | Run `go install gorm.io/cli/gorm@latest && make gorm-gen`. |
| `gorm: command not found` | `$(go env GOPATH)/bin` is not on your `PATH`. | Add it to your shell profile. |
| Generated file references a field you just renamed | Stale output. | `make gorm-gen`. |
| `@@table` renders as the wrong table | The method's return type is ambiguous. | Either move the method into a generic `Query[T]` or replace `@@table` with the literal table name (e.g. `users`). |
| CI compiles locally but not in CI | `internal/generated/` is not committed, or your CI image lacks the CLI. | Either commit the generated code (recommended) or add `go install gorm.io/cli/gorm@latest` and `make gorm-gen` to the CI script before `go build`. |

## 10. Further reading

- Repo: <https://github.com/go-gorm/cli>
- Field helper reference: <https://github.com/go-gorm/cli#field-helpers>
- GORM generics API (`gorm.G[T]`): <https://gorm.io/docs/generics.html>
