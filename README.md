# db-keepalive

Pluggable Go daemon that performs a periodic counter increment to prevent free-tier database clusters (Redis Cloud, Valkey, Aiven, Neon, Supabase, MongoDB Atlas, Couchbase Capella, etc.) from being auto-paused for inactivity.

Successor to the `*-keepalive` family: one binary, one image, six adapters.

## Supported drivers

| `DB_TYPE`    | Driver                              | Env vars |
| ------------ | ----------------------------------- | -------- |
| `redis`      | `github.com/redis/go-redis/v9`      | `REDIS_URL` |
| `valkey`     | `github.com/valkey-io/valkey-go`    | `VALKEY_URL` |
| `postgresql` | `github.com/lib/pq`                 | `SERVICE_URI` |
| `mysql`      | `github.com/go-sql-driver/mysql`    | `DATA_SOURCE_NAME` |
| `mongodb`    | `go.mongodb.org/mongo-driver/v2`    | `MONGODB_URI`, `MONGODB_DATABASE`, `MONGODB_COLLECTION` |
| `couchbase`  | `github.com/couchbase/gocb/v2`      | `COUCHBASE_CONNECTION_STRING`, `COUCHBASE_USERNAME`, `COUCHBASE_PASSWORD`, `COUCHBASE_BUCKET_NAME`, `COUCHBASE_SCOPE_NAME`, `COUCHBASE_COLLECTION_NAME` |

Optional: `INTERVAL` (e.g. `30s`, `5m`; default `1m`), `COUNTER_KEY` (default `counter`).

## Quick start (Docker)

```bash
docker run -d --name db-keepalive --restart unless-stopped \
  -e DB_TYPE=redis \
  -e REDIS_URL='redis://default@host:6379' \
  ghcr.io/tiennm99/db-keepalive:latest
```

## Quick start (local)

```bash
git clone https://github.com/tiennm99/db-keepalive
cd db-keepalive
cp .env.example .env       # then edit DB_TYPE + the driver's env vars
go run .
```

## How it works

On every tick the chosen adapter performs the cheapest write that proves the cluster is alive:

- **Redis/Valkey** — `INCR counter`
- **PostgreSQL** — `UPDATE keepalive SET value = value + 1 WHERE key = 'counter' RETURNING value`
- **MySQL** — `UPDATE` + `SELECT` inside a transaction
- **MongoDB** — `FindOneAndUpdate({_id: "counter"}, {$inc: {count: 1}}, upsert)`
- **Couchbase** — `GET counter` → `++` → `UPSERT counter`

The PostgreSQL and MySQL adapters expect a table:

```sql
CREATE TABLE keepalive (key TEXT PRIMARY KEY, value BIGINT NOT NULL DEFAULT 0);
INSERT INTO keepalive (key, value) VALUES ('counter', 0);
```

(MySQL uses backticked identifiers — see `adapter/mysql.go`.)

## Adding a new database

1. Create `adapter/<name>.go`.
2. Implement the `Adapter` interface in `adapter/adapter.go` (`Connect`, `Increment`, `Close`).
3. Register the factory in `init()`:
   ```go
   func init() { Registry["<name>"] = func() (Adapter, error) { return &myAdapter{}, nil } }
   ```
4. Add an `import _ "your driver"` if needed, and the env vars to `.env.example` and the table above.

## Migrated from

This repo replaces six single-database repos. All are archived with a pointer here:

- [redis-keepalive](https://github.com/tiennm99/redis-keepalive)
- [valkey-keepalive](https://github.com/tiennm99/valkey-keepalive)
- [postgresql-keepalive](https://github.com/tiennm99/postgresql-keepalive)
- [mysql-keepalive](https://github.com/tiennm99/mysql-keepalive)
- [mongodb-keepalive](https://github.com/tiennm99/mongodb-keepalive)
- [couchbase-keepalive](https://github.com/tiennm99/couchbase-keepalive)

## License

Apache-2.0 — see [LICENSE](LICENSE).
