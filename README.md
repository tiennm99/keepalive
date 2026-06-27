# keepalive

Pluggable Go daemon that periodically touches external services to prevent idle shutdowns, pauses, or cold starts.

The current adapters perform cheap datastore writes for Redis Cloud, Valkey, Aiven, Neon, Supabase, MongoDB Atlas, Couchbase Capella, and similar hosted services.

Successor to the `*-keepalive` family: one binary, one image, six datastore adapters.

## Supported adapters

Set `KEEPALIVE_ADAPTER` to one of the values below.

| `KEEPALIVE_ADAPTER` | Driver                              | Env vars |
| ------------------- | ----------------------------------- | ------------------ |
| `redis`             | `github.com/redis/go-redis/v9`      | `KEEPALIVE_REDIS_URL` |
| `valkey`            | `github.com/valkey-io/valkey-go`    | `KEEPALIVE_VALKEY_URL` |
| `postgresql`        | `github.com/lib/pq`                 | `KEEPALIVE_POSTGRESQL_URL` |
| `mysql`             | `github.com/go-sql-driver/mysql`    | `KEEPALIVE_MYSQL_DSN` |
| `mongodb`           | `go.mongodb.org/mongo-driver/v2`    | `KEEPALIVE_MONGODB_URI`, `KEEPALIVE_MONGODB_DATABASE`, `KEEPALIVE_MONGODB_COLLECTION` |
| `couchbase`         | `github.com/couchbase/gocb/v2`      | `KEEPALIVE_COUCHBASE_CONNECTION_STRING`, `KEEPALIVE_COUCHBASE_USERNAME`, `KEEPALIVE_COUCHBASE_PASSWORD`, `KEEPALIVE_COUCHBASE_BUCKET_NAME`, `KEEPALIVE_COUCHBASE_SCOPE_NAME`, `KEEPALIVE_COUCHBASE_COLLECTION_NAME` |

Optional: `KEEPALIVE_INTERVAL` (e.g. `30s`, `5m`; default `1m`), `KEEPALIVE_COUNTER_KEY` (default `counter`).

## Quick start (Docker)

```bash
docker run -d --name keepalive --restart unless-stopped \
  -e KEEPALIVE_ADAPTER=redis \
  -e KEEPALIVE_REDIS_URL='redis://default@host:6379' \
  ghcr.io/tiennm99/keepalive:latest
```

## Quick start (local)

```bash
git clone https://github.com/tiennm99/keepalive
cd keepalive
cp .env.example .env       # then edit KEEPALIVE_ADAPTER + the driver's env vars
go run .
```

## How it works

On every tick the chosen adapter performs the cheapest write that proves the cluster is alive. `KEEPALIVE_COUNTER_KEY` selects the key/doc ID and defaults to `counter`.

- **Redis/Valkey** — `INCR key`
- **PostgreSQL** — `UPDATE keepalive SET value = value + 1 WHERE key = $1 RETURNING value`
- **MySQL** — `UPDATE` + `SELECT` by key inside a transaction
- **MongoDB** — `FindOneAndUpdate({_id: key}, {$inc: {count: 1}}, upsert)`
- **Couchbase** — `GET key` -> `++` -> `UPSERT key`

The PostgreSQL and MySQL adapters expect a table:

```sql
CREATE TABLE keepalive (key TEXT PRIMARY KEY, value BIGINT NOT NULL DEFAULT 0);
INSERT INTO keepalive (key, value) VALUES ('counter', 0);
```

Seed the value with your configured `KEEPALIVE_COUNTER_KEY` when it is not `counter`. MySQL uses backticked identifiers — see `adapter/mysql.go`.

## Adding a new adapter

1. Create `adapter/<name>.go`.
2. Implement the `Adapter` interface in `adapter/adapter.go` (`Connect`, `Increment`, `Close`).
3. Register the factory in `init()`:
   ```go
   func init() { Registry["<name>"] = func() (Adapter, error) { return &myAdapter{}, nil } }
   ```
4. Add an `import _ "your driver"` if needed, and the `KEEPALIVE_*` env vars to `.env.example` and the table above.

## Migrated from

This repo replaces six single-datastore repos. All are archived with a pointer here:

- [redis-keepalive](https://github.com/tiennm99/redis-keepalive)
- [valkey-keepalive](https://github.com/tiennm99/valkey-keepalive)
- [postgresql-keepalive](https://github.com/tiennm99/postgresql-keepalive)
- [mysql-keepalive](https://github.com/tiennm99/mysql-keepalive)
- [mongodb-keepalive](https://github.com/tiennm99/mongodb-keepalive)
- [couchbase-keepalive](https://github.com/tiennm99/couchbase-keepalive)

## License

Apache-2.0 — see [LICENSE](LICENSE).
