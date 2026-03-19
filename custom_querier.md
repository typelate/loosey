# Custom Version Table

By default, loosey uses the `goose_db_version` table to track migration state.
To use a different table name, generate your own sqlc implementation from the
bundled SQL files and pass it to `loosey.New`.

## Install sqlc

```
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

## Steps

### 1. Copy the SQL files for your database engine

Copy both `schema.sql` and `queries.sql` from the matching dialect into your
project:

```
mkdir -p internal/migrations
```

| Database   | Source path        |
|------------|--------------------|
| PostgreSQL | `internal/postgres/` |
| SQLite     | `internal/sqlite3/`  |
| MySQL      | `internal/mysql/`    |
| libsql     | `internal/libsql/`   |

### 2. Replace the table name

Find-and-replace `goose_db_version` with your table name in both files:

```
sed -i 's/goose_db_version/my_migrations/g' internal/migrations/schema.sql
sed -i 's/goose_db_version/my_migrations/g' internal/migrations/queries.sql
```

### 3. Add a sqlc configuration

Create `sqlc.yaml` in your project root. SQLite and MySQL require a column
override so `version_id` maps to `int64`. PostgreSQL does not need the override.

```yaml
version: "2"
sql:
  - engine: "sqlite"
    schema: "internal/migrations/schema.sql"
    queries: "internal/migrations/queries.sql"
    gen:
      go:
        package: "migrations"
        out: "internal/migrations"
        overrides:
          - column: "my_migrations.version_id"
            go_type: "int64"
```

### 4. Generate

```
sqlc generate
```

This produces a `Queries` type in your package that implements `loosey.Querier`.

### 5. Wire it up with loosey.New

```go
package main

import (
    "context"
    "database/sql"
    "os"

    "github.com/typelate/loosey"

    "example.com/internal/migrations"
)

func main() {
    db, _ := sql.Open("sqlite3", "app.db")
    ctx := context.Background()

    q := migrations.New(db)
    m, err := loosey.New(ctx, db, os.DirFS("migrations"), q, (*migrations.Queries).WithTx)
    if err != nil {
        panic(err)
    }

    results, err := m.Up(ctx)
    // ...
}
```

The generated `Queries` type has the same shape as the built-in dialects:
`New(db)` creates one, `WithTx(tx)` creates a transaction-bound copy, and the
five query methods (`EnsureTable`, `InsertVersion`, `DeleteVersion`,
`ListApplied`, `LatestVersion`) satisfy `loosey.Querier`.
