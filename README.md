# migpgx - Lightweight PostgreSQL Migrations with pgx

## Overview

migpgx is a minimal and efficient migration runner for PostgreSQL, built using [pgx v5](https://github.com/jackc/pgx/). It applies .sql migrations sequentially while ensuring idempotency by tracking applied migrations in a tracking table.

## Installation

```sh
go get github.com/ldummer/migpgx
```

## Usage

### 1. Initialize the Migration Runner

```go
ctx := context.Background()
conn, err := pgx.Connect(ctx, connStr)
if err != nil {
    log.Fatalf("Error connecting to PostgreSQL: %v", err)
}
defer conn.Close(ctx)

migrator := migpgx.NewMigrationRunner(conn, "./migrations")
```

### 2. Apply Migrations

```go
err = migrator.ApplyMigrations(ctx, "schema_migrations")
if err != nil {
    log.Fatalf("Migration failed: %v", err)
}
```

## Migration File Structure

Place SQL migration files in a directory. Files are executed in **lexicographical order**.

Example:

```
/migrations
  ├── 001_init.sql
  ├── 002_create_table.sql
  ├── 003_add_indexes.sql
```

## How It Works

1. **Ensures a migration tracking table exists** (`CREATE TABLE IF NOT EXISTS <migTable>`).
2. **Reads migration files** from the specified directory.
3. **Checks already applied migrations** to prevent re-execution.
4. **Applies new migrations** in a transaction.
5. **Records applied migrations** in the tracking table.

## Customization

- Define a custom migration table name.
- Uses pg\_advisory\_xact\_lock for concurrency safety.
- Errors out on failed migrations.


