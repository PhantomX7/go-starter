# Pagination Library — Usage Guide

A type-safe, injection-proof query pagination and filtering library for [GORM](https://gorm.io). It turns HTTP query parameters into safe `WHERE`, `ORDER BY`, `LIMIT`, and `OFFSET` clauses.

---

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
  - [FilterDefinition](#filterdefinition)
  - [FilterConfig](#filterconfig)
  - [SortConfig](#sortconfig)
  - [PaginationOptions](#paginationoptions)
  - [Pagination](#pagination)
- [Filter Types](#filter-types)
  - [ID](#id)
  - [Number](#number)
  - [String](#string)
  - [Bool](#bool)
  - [Date](#date)
  - [DateTime](#datetime)
  - [Enum](#enum)
- [Filter Operators](#filter-operators)
  - [Operators by Type Matrix](#operators-by-type-matrix)
- [Query Parameter Format](#query-parameter-format)
  - [Pagination Parameters](#pagination-parameters)
  - [Sorting Parameters](#sorting-parameters)
  - [Filtering Parameters](#filtering-parameters)
- [Advanced Usage](#advanced-usage)
  - [Multi-Field Search](#multi-field-search)
  - [Table-Prefixed Fields](#table-prefixed-fields)
  - [Custom Operators per Filter](#custom-operators-per-filter)
  - [Custom GORM Scopes](#custom-gorm-scopes)
  - [Count Queries (Without Pagination Meta)](#count-queries-without-pagination-meta)
  - [Page Calculation Helpers](#page-calculation-helpers)
- [Full HTTP Handler Example](#full-http-handler-example)
- [Security Model](#security-model)
- [Defensive Limits](#defensive-limits)
- [Error Handling Philosophy](#error-handling-philosophy)

---

## Installation

```bash
go get <your-module-path>/pagination
```

The library depends on:

| Dependency | Purpose |
|---|---|
| `gorm.io/gorm` | ORM query builder |
| Go 1.21+ | Uses `maps.Clone`, `slices.Contains`, `strings.Cut` |

---

## Quick Start

```go
package main

import (
    "net/http"

    "gorm.io/gorm"
    "your-module/pagination"
)

type User struct {
    ID    uint
    Name  string
    Email string
    Role  string
    Age   int
}

// 1. Define filters and sorts (typically once at init or handler setup)
var userFilters = pagination.NewFilterDefinition().
    AddFilter("name", pagination.FilterConfig{
        Field: "name",
        Type:  pagination.FilterTypeString,
    }).
    AddFilter("email", pagination.FilterConfig{
        Field: "email",
        Type:  pagination.FilterTypeString,
    }).
    AddFilter("role", pagination.FilterConfig{
        Field:      "role",
        Type:       pagination.FilterTypeEnum,
        EnumValues: []string{"admin", "editor", "viewer"},
    }).
    AddFilter("age", pagination.FilterConfig{
        Field: "age",
        Type:  pagination.FilterTypeNumber,
    }).
    AddSort("name", pagination.SortConfig{Allowed: true}).
    AddSort("id", pagination.SortConfig{Allowed: true}).
    AddSort("age", pagination.SortConfig{Allowed: true})

func ListUsers(db *gorm.DB, r *http.Request) ([]User, int64, error) {
    // 2. Collect query parameters into map[string][]string
    conditions := map[string][]string{}
    for key, values := range r.URL.Query() {
        conditions[key] = values
    }

    // 3. Create pagination instance
    opts := pagination.DefaultPaginationOptions()
    p := pagination.NewPagination(conditions, userFilters, opts)

    // 4. Count total (without limit/offset)
    var total int64
    if err := p.ApplyWithoutMeta(db.Model(&User{})).Count(&total).Error; err != nil {
        return nil, 0, err
    }

    // 5. Fetch paginated results
    var users []User
    if err := p.Apply(db.Model(&User{})).Find(&users).Error; err != nil {
        return nil, 0, err
    }

    return users, total, nil
}
```

**Resulting HTTP requests:**

```
GET /users?limit=10&offset=0&sort=name asc&name=like:john&role=in:admin,editor
GET /users?age=gte:18&age=lte:65&sort=age desc
GET /users?email=is_not_null&limit=50
```

---

## Core Concepts

### FilterDefinition

The registry of all allowed filters and sorts. Created once, typically at package or handler level. Both `AddFilter` and `AddSort` are chainable.

```go
fd := pagination.NewFilterDefinition().
    AddFilter("status", pagination.FilterConfig{ /* ... */ }).
    AddFilter("name",   pagination.FilterConfig{ /* ... */ }).
    AddSort("status", pagination.SortConfig{Allowed: true}).
    AddSort("name",   pagination.SortConfig{Allowed: true})
```

> **Key insight:** Only filters and sorts registered here are recognized. Unknown query parameters are silently ignored. This is the allowlist.

---

### FilterConfig

Defines how a single query parameter maps to a database column.

```go
type FilterConfig struct {
    Field        string           // Database column name
    SearchFields []string         // Multiple columns for OR-search (overrides Field)
    Type         FilterType       // Data type (see Filter Types)
    TableName    string           // Optional table prefix (e.g., "users")
    Operators    []FilterOperator // Override allowed operators (optional)
    EnumValues   []string         // Valid values for FilterTypeEnum
}
```

| Property | Required | Description |
|---|---|---|
| `Field` | Yes* | The DB column. Ignored when `SearchFields` is set. |
| `SearchFields` | No | Multiple columns to OR-search across. |
| `Type` | Yes | Determines parsing, validation, and allowed operators. |
| `TableName` | No | Prefixed to fields as `tablename.field` in SQL. |
| `Operators` | No | Restrict to a subset of operators for this type. |
| `EnumValues` | Required for `FilterTypeEnum` | The allowlist of valid values. |

---

### SortConfig

Defines how a sort key maps to a database column.

```go
type SortConfig struct {
    Field     string // Actual DB column (if different from the query key)
    TableName string // Optional table prefix
    Allowed   bool   // Must be true or the sort key is rejected
}
```

**Example: aliased sort key**

```go
// ?sort=created desc  →  ORDER BY created_at desc
fd.AddSort("created", pagination.SortConfig{
    Field:   "created_at",
    Allowed: true,
})
```

**Example: table-prefixed sort**

```go
// ?sort=users.name asc  →  ORDER BY users.name asc
fd.AddSort("name", pagination.SortConfig{
    TableName: "users",
    Allowed:   true,
})
```

---

### PaginationOptions

Controls pagination defaults and limits.

```go
type PaginationOptions struct {
    DefaultLimit int            // Default page size (default: 20)
    MaxLimit     int            // Maximum page size (default: 100)
    DefaultOrder string         // Fallback ORDER BY (default: "id desc")
    Timezone     *time.Location // For date/datetime parsing (default: Asia/Jakarta)
}
```

Use `DefaultPaginationOptions()` for sensible defaults:

```go
opts := pagination.DefaultPaginationOptions()
// Override as needed:
opts.MaxLimit = 200
opts.DefaultOrder = "created_at desc"
opts.Timezone = time.UTC
```

| Option | Default | Notes |
|---|---|---|
| `DefaultLimit` | 20 | Falls back to 20 if ≤ 0 |
| `MaxLimit` | 100 | Falls back to 100 if ≤ 0; user requests above this are clamped |
| `DefaultOrder` | `"id desc"` | Must be a valid order literal or falls back to `"id desc"` |
| `Timezone` | `Asia/Jakarta` | Used by `FilterTypeDate` and `FilterTypeDateTime` |

---

### Pagination

The main runtime object. Created per-request from query parameters.

```go
p := pagination.NewPagination(conditions, filterDef, opts)
```

| Method | Description |
|---|---|
| `Apply(db)` | Applies filters + scopes + `LIMIT` + `OFFSET` + `ORDER BY` |
| `ApplyWithoutMeta(db)` | Applies filters + scopes only (for `COUNT(*)` queries) |
| `AddCustomScope(scopes...)` | Adds arbitrary GORM scopes (chainable) |
| `GetConditions()` | Returns a shallow clone of the raw conditions |
| `GetPage()` | Current page number (1-indexed) |
| `GetPageSize()` | Current page size (`Limit`) |
| `GetTotalPages(total)` | Calculates total page count from a row count |

---

## Filter Types

### ID

Integer-only identifiers. Floats are rejected (no silent rounding at 2^53+).

```go
pagination.FilterConfig{
    Field: "id",
    Type:  pagination.FilterTypeID,
}
```

```
?id=eq:42
?id=in:1,2,3
?id=between:100,200
?id=gt:50
```

**Allowed operators:** `eq`, `neq`, `in`, `not_in`, `between`, `gt`, `gte`, `lt`, `lte`, `is_null`, `is_not_null`

---

### Number

Integers and floats. Parses as `int64` first, falls back to `float64`.

```go
pagination.FilterConfig{
    Field: "price",
    Type:  pagination.FilterTypeNumber,
}
```

```
?price=gte:9.99
?price=between:10,100
?price=in:10,20,30
```

**Allowed operators:** `eq`, `neq`, `in`, `not_in`, `between`, `gt`, `gte`, `lt`, `lte`, `is_null`, `is_not_null`

---

### String

Text values. Supports `LIKE` with automatic wildcard escaping.

```go
pagination.FilterConfig{
    Field: "name",
    Type:  pagination.FilterTypeString,
}
```

```
?name=eq:John
?name=like:john           → LOWER(name) LIKE LOWER('%john%') ESCAPE '\'
?name=in:Alice,Bob
?name=neq:Charlie
```

**LIKE escaping:** User input containing `%`, `_`, or `\` is automatically escaped. `?name=like:50%` matches the literal string "50%", not "50" followed by anything.

**Allowed operators:** `eq`, `neq`, `in`, `not_in`, `like`, `is_null`, `is_not_null`

---

### Bool

Boolean values. Accepts all formats recognized by Go's `strconv.ParseBool`: `1`, `t`, `true`, `TRUE`, `0`, `f`, `false`, `FALSE`, etc.

```go
pagination.FilterConfig{
    Field: "is_active",
    Type:  pagination.FilterTypeBool,
}
```

```
?is_active=eq:true
?is_active=eq:1
?is_active=eq:false
```

**Allowed operators:** `eq`, `is_null`, `is_not_null`

---

### Date

Date-only values in `YYYY-MM-DD` format. Parsed in the configured timezone.

```go
pagination.FilterConfig{
    Field: "birth_date",
    Type:  pagination.FilterTypeDate,
}
```

```
?birth_date=eq:2024-01-15
?birth_date=between:2024-01-01,2024-12-31
?birth_date=gte:2024-06-01
?birth_date=lte:2024-06-30
```

**How date equality works:** `eq:2024-01-15` generates `field >= 2024-01-15 00:00:00 AND field < 2024-01-16 00:00:00` (half-open interval). This is precision-safe across Postgres `timestamptz` and MySQL `DATETIME`.

**How `lte` works:** `lte:2024-06-30` generates `field < 2024-07-01` (inclusive of the entire day).

**Allowed operators:** `eq`, `between`, `gte`, `lte`, `is_null`, `is_not_null`

---

### DateTime

Full timestamp values in `YYYY-MM-DD HH:MM:SS` format. Parsed in the configured timezone.

```go
pagination.FilterConfig{
    Field: "created_at",
    Type:  pagination.FilterTypeDateTime,
}
```

```
?created_at=eq:2024-01-15 10:30:00
?created_at=between:2024-01-01 00:00:00,2024-12-31 23:59:59
?created_at=gte:2024-06-01 08:00:00
?created_at=lte:2024-06-30 17:00:00
```

**Allowed operators:** `eq`, `between`, `gte`, `lte`, `is_null`, `is_not_null`

---

### Enum

String values restricted to a predefined allowlist. Invalid values are dropped individually — a single typo doesn't discard the entire filter.

```go
pagination.FilterConfig{
    Field:      "status",
    Type:       pagination.FilterTypeEnum,
    EnumValues: []string{"draft", "published", "archived"},
}
```

```
?status=eq:published
?status=in:draft,published
?status=in:draft,typo,published    → filters on draft,published (typo dropped)
?status=eq:invalid                 → returns 0 rows (fail-closed)
```

**Allowed operators:** `eq`, `in`, `is_null`, `is_not_null`

---

## Filter Operators

| Operator | Wire Format | Description | Example |
|---|---|---|---|
| Equals | `eq` | Exact match | `?name=eq:John` |
| Not Equals | `neq` | Exclude exact match | `?name=neq:John` |
| In | `in` | Match any in set | `?id=in:1,2,3` |
| Not In | `not_in` | Exclude set | `?id=not_in:4,5` |
| Like | `like` | Case-insensitive substring | `?name=like:john` |
| Between | `between` | Inclusive range (2 values) | `?age=between:18,65` |
| Greater Than | `gt` | Strictly greater | `?price=gt:100` |
| Greater or Equal | `gte` | Greater or equal | `?price=gte:100` |
| Less Than | `lt` | Strictly less | `?price=lt:50` |
| Less or Equal | `lte` | Less or equal | `?price=lte:50` |
| Is Null | `is_null` | NULL check | `?deleted_at=is_null` |
| Is Not Null | `is_not_null` | NOT NULL check | `?deleted_at=is_not_null` |

**Default (no operator):** When no `operator:` prefix is provided, `eq` is assumed.

```
?name=John        → equivalent to ?name=eq:John
```

**Colon handling:** Values containing colons (e.g., URLs) are handled correctly. Only recognized operator prefixes are parsed:

```
?website=http://example.com   → eq match on "http://example.com" (not parsed as operator)
```

### Operators by Type Matrix

| Operator | ID | Number | String | Bool | Date | DateTime | Enum |
|---|---|---|---|---|---|---|---|
| `eq` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `neq` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `in` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ✅ |
| `not_in` | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `like` | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ |
| `between` | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ |
| `gt` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| `gte` | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ |
| `lt` | ✅ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ |
| `lte` | ✅ | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ |
| `is_null` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| `is_not_null` | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

---

## Query Parameter Format

### Pagination Parameters

| Parameter | Type | Default | Description |
|---|---|---|---|
| `limit` | int | 20 | Page size. Clamped to `MaxLimit`. |
| `offset` | int | 0 | Number of rows to skip. |

```
?limit=25&offset=50    → Page 3 of 25-row pages
```

### Sorting Parameters

```
?sort=field [asc|desc][,field2 [asc|desc],...]
```

| Example | Result |
|---|---|
| `?sort=name` | `ORDER BY name asc` |
| `?sort=name desc` | `ORDER BY name desc` |
| `?sort=name asc,id desc` | `ORDER BY name asc, id desc` |
| `?sort=users.name asc` | `ORDER BY users.name asc` (if SortConfig declares that table) |

**Rules:**
- Direction defaults to `asc` if omitted.
- Every field must be registered via `AddSort` with `Allowed: true`.
- Table prefixes in the request must match the `SortConfig.TableName`.
- If any part is invalid, the entire sort falls back to `DefaultOrder`.
- Maximum of 16 comma-separated sort parts.

### Filtering Parameters

```
?field=operator:value
?field=operator:value1,value2
?field=value                    (implies eq:value)
?field=is_null
?field=is_not_null
```

**Repeated keys** are treated as separate `AND` conditions:

```
?age=gte:18&age=lte:65   → age >= 18 AND age <= 65
```

---

## Advanced Usage

### Multi-Field Search

Use `SearchFields` to search across multiple columns with a single parameter. Matching operators (`eq`, `like`, `in`, `is_null`) combine with `OR`. Excluding operators (`neq`, `not_in`, `is_not_null`) combine with `AND`.

```go
fd.AddFilter("search", pagination.FilterConfig{
    SearchFields: []string{"name", "email", "phone"},
    Type:         pagination.FilterTypeString,
})
```

```
?search=like:john
```

Generates:

```sql
WHERE (
    LOWER(name) LIKE LOWER('%john%') ESCAPE '\'
    OR LOWER(email) LIKE LOWER('%john%') ESCAPE '\'
    OR LOWER(phone) LIKE LOWER('%john%') ESCAPE '\'
)
```

```
?search=neq:test
```

Generates:

```sql
WHERE (name != 'test' AND email != 'test' AND phone != 'test')
```

---

### Table-Prefixed Fields

When joining tables, use `TableName` to disambiguate columns:

```go
fd.AddFilter("user_name", pagination.FilterConfig{
    Field:     "name",
    Type:      pagination.FilterTypeString,
    TableName: "users",
})

fd.AddFilter("order_status", pagination.FilterConfig{
    Field:     "status",
    Type:      pagination.FilterTypeEnum,
    TableName: "orders",
    EnumValues: []string{"pending", "completed", "cancelled"},
})
```

Generates `users.name = ?` and `orders.status = ?` respectively.

Fields that already contain a dot (e.g., from a subquery alias) are not re-prefixed:

```go
fd.AddFilter("metric", pagination.FilterConfig{
    SearchFields: []string{"stats.value", "name"},
    Type:         pagination.FilterTypeString,
    TableName:    "metrics",
})
// → SearchFields resolve to: ["stats.value", "metrics.name"]
//   "stats.value" already qualified, left as-is
//   "name" gets prefixed to "metrics.name"
```

---

### Custom Operators per Filter

Override the default operators for a type:

```go
fd.AddFilter("status", pagination.FilterConfig{
    Field:      "status",
    Type:       pagination.FilterTypeString,
    Operators:  []pagination.FilterOperator{
        pagination.OperatorEquals,
        pagination.OperatorIn,
    },
    // Only eq and in are allowed; like, neq, not_in are rejected
})
```

---

### Custom GORM Scopes

Add arbitrary GORM scopes that run alongside the filters:

```go
p := pagination.NewPagination(conditions, filterDef, opts)

// Add a tenant isolation scope
p.AddCustomScope(func(db *gorm.DB) *gorm.DB {
    return db.Where("tenant_id = ?", currentTenantID)
})

// Add a soft-delete scope
p.AddCustomScope(func(db *gorm.DB) *gorm.DB {
    return db.Where("deleted_at IS NULL")
})

// Scopes are applied in order alongside filter scopes
var results []MyModel
db = p.Apply(db.Model(&MyModel{})).Find(&results)
```

---

### Count Queries (Without Pagination Meta)

Use `ApplyWithoutMeta` for count queries that need filters but not `LIMIT`/`OFFSET`/`ORDER BY`:

```go
p := pagination.NewPagination(conditions, filterDef, opts)

// Count with filters only
var total int64
p.ApplyWithoutMeta(db.Model(&User{})).Count(&total)

// Fetch with full pagination
var users []User
p.Apply(db.Model(&User{})).Find(&users)
```

---

### Page Calculation Helpers

```go
p := pagination.NewPagination(conditions, filterDef, opts)

p.GetPage()        // Current page (1-indexed): (offset / limit) + 1
p.GetPageSize()    // Current limit
p.GetTotalPages(total)  // Ceiling division: ceil(total / limit)

// Build a JSON response envelope:
response := map[string]any{
    "data":        users,
    "total":       total,
    "page":        p.GetPage(),
    "page_size":   p.GetPageSize(),
    "total_pages": p.GetTotalPages(total),
}
```

---

## Full HTTP Handler Example

```go
package handler

import (
    "encoding/json"
    "net/http"

    "gorm.io/gorm"
    "your-module/pagination"
)

type Product struct {
    ID        uint    `json:"id"`
    Name      string  `json:"name"`
    Category  string  `json:"category"`
    Price     float64 `json:"price"`
    InStock   bool    `json:"in_stock"`
    CreatedAt time.Time `json:"created_at"`
}

// Define once at package level
var productFilters = pagination.NewFilterDefinition().
    AddFilter("id", pagination.FilterConfig{
        Field: "id",
        Type:  pagination.FilterTypeID,
    }).
    AddFilter("name", pagination.FilterConfig{
        Field: "name",
        Type:  pagination.FilterTypeString,
    }).
    AddFilter("search", pagination.FilterConfig{
        SearchFields: []string{"name", "description", "sku"},
        Type:         pagination.FilterTypeString,
    }).
    AddFilter("category", pagination.FilterConfig{
        Field:      "category",
        Type:       pagination.FilterTypeEnum,
        EnumValues: []string{"electronics", "clothing", "food", "books"},
    }).
    AddFilter("price", pagination.FilterConfig{
        Field: "price",
        Type:  pagination.FilterTypeNumber,
    }).
    AddFilter("in_stock", pagination.FilterConfig{
        Field: "in_stock",
        Type:  pagination.FilterTypeBool,
    }).
    AddFilter("created_at", pagination.FilterConfig{
        Field: "created_at",
        Type:  pagination.FilterTypeDate,
    }).
    AddSort("id", pagination.SortConfig{Allowed: true}).
    AddSort("name", pagination.SortConfig{Allowed: true}).
    AddSort("price", pagination.SortConfig{Allowed: true}).
    AddSort("created", pagination.SortConfig{
        Field:   "created_at", // alias: ?sort=created maps to created_at
        Allowed: true,
    })

var productOpts = pagination.PaginationOptions{
    DefaultLimit: 25,
    MaxLimit:     100,
    DefaultOrder: "created_at desc",
    Timezone:     time.UTC,
}

func ListProducts(db *gorm.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Convert query params to conditions
        conditions := make(map[string][]string)
        for key, values := range r.URL.Query() {
            conditions[key] = values
        }

        // Create pagination
        p := pagination.NewPagination(conditions, productFilters, productOpts)

        // Count
        var total int64
        if err := p.ApplyWithoutMeta(db.Model(&Product{})).Count(&total).Error; err != nil {
            http.Error(w, "internal error", http.StatusInternalServerError)
            return
        }

        // Fetch
        var products []Product
        if err := p.Apply(db.Model(&Product{})).Find(&products).Error; err != nil {
            http.Error(w, "internal error", http.StatusInternalServerError)
            return
        }

        // Respond
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]any{
            "data":        products,
            "total":       total,
            "page":        p.GetPage(),
            "page_size":   p.GetPageSize(),
            "total_pages": p.GetTotalPages(total),
        })
    }
}
```

**Example requests against this handler:**

```bash
# Basic pagination
GET /products?limit=10&offset=0

# Search across name, description, sku
GET /products?search=like:wireless

# Filter by category and price range
GET /products?category=in:electronics,books&price=gte:10&price=lte:100

# In-stock electronics sorted by price
GET /products?category=eq:electronics&in_stock=eq:true&sort=price asc

# Products created in January 2024
GET /products?created_at=between:2024-01-01,2024-01-31

# Multi-sort: cheapest first, then newest
GET /products?sort=price asc,created desc

# NULL checks
GET /products?category=is_not_null
```

---

## Security Model

The library is designed to prevent SQL injection at multiple layers:

| Layer | Protection |
|---|---|
| **Registration (startup)** | `AddFilter` and `AddSort` reject identifiers that fail `isIdent` / `isQualifiedIdent` checks. Only `[a-zA-Z_][a-zA-Z0-9_]*` (optionally dot-separated) are accepted. |
| **Filter values (runtime)** | All user values are passed as parameterized `?` placeholders via GORM — never interpolated into SQL strings. |
| **Sort clause (runtime)** | `parseOrder` reconstructs the `ORDER BY` from validated tokens. Raw user input never reaches `db.Order()`. Table prefixes must match the registered `SortConfig.TableName`. |
| **LIKE patterns** | Wildcard characters (`%`, `_`, `\`) in user input are escaped via `likeEscaper` with an explicit `ESCAPE '\'` clause. |
| **DefaultOrder (config)** | Validated by `isValidOrderLiteral` at construction; malformed values fall back to `"id desc"`. |
| **Enum values** | Compared against the `EnumValues` allowlist; invalid values are dropped individually. |

---

## Defensive Limits

To prevent a single malicious request from consuming unbounded memory or CPU:

| Constant | Value | Purpose |
|---|---|---|
| `maxFilterValues` | 256 | Max comma-separated values in `in:` / `not_in:` lists |
| `maxSortParts` | 16 | Max comma-separated parts in `?sort=` |

Requests exceeding these limits are silently rejected (the filter is dropped or the sort falls back to default).

---

## Error Handling Philosophy

The library follows a **fail-closed, silent-drop** strategy:

| Scenario | Behavior |
|---|---|
| Unknown query parameter | Ignored (not in filter registry) |
| Invalid operator for type | Filter dropped (no rows affected) |
| Unparseable number/date | Filter dropped |
| Invalid enum value | Value dropped; if all values invalid → 0 rows returned |
| Invalid sort field | Entire sort falls back to `DefaultOrder` |
| `in:` with all unparseable values | 0 rows returned (`WHERE 1 = 0`) |
| `not_in:` with all unparseable values | Filter becomes no-op (no exclusion) |
| Unsafe identifier at registration | `AddFilter` / `AddSort` silently skips it |

This design ensures that malformed input never widens the result set beyond what the user explicitly asked for.