# keepalive

Pluggable Go daemon that periodically touches external services to prevent idle shutdowns, pauses, or cold starts.

The current adapters perform cheap datastore writes for Redis Cloud, Valkey, Aiven, Neon, Supabase, MongoDB Atlas, Couchbase Capella, and similar hosted services.

Successor to the `*-keepalive` family: one binary, one image, six datastore adapters.

## Configuration

By default, keepalive reads `keepalive.yaml` from the current working directory. One deployment can keep any number of services alive.

```yaml
interval: 1m
counter_key: counter

services:
  - adapter: redis
    config:
      url: redis://default@redis-a.example.com:6379

  - adapter: mongodb
    config:
      uri: mongodb+srv://user:pass@mongo-a.example.com
      database: keepalive
      collection: counter

  - adapter: couchbase
    config:
      connection_string: couchbases://couchbase-a.example.com
      username: user
      password: pass
      bucket_name: keepalive
      scope_name: _default
      collection_name: _default
```

`name` is optional. When omitted, keepalive generates a name from `adapter` and the connection host, such as `redis-redis-a-example-com`. Duplicate generated names get suffixes like `redis-redis-a-example-com-2`.

`interval` and `counter_key` can be set globally or per service. Per-service values override global values.

## Supported adapters

| `adapter`    | Driver                              | `config` keys |
| ------------ | ----------------------------------- | ------------- |
| `redis`      | `github.com/redis/go-redis/v9`      | `url` |
| `valkey`     | `github.com/valkey-io/valkey-go`    | `url` |
| `postgresql` | `github.com/lib/pq`                 | `url` |
| `mysql`      | `github.com/go-sql-driver/mysql`    | `dsn` |
| `mongodb`    | `go.mongodb.org/mongo-driver/v2`    | `uri`, `database`, `collection` |
| `couchbase`  | `github.com/couchbase/gocb/v2`      | `connection_string`, `username`, `password`, `bucket_name`, `scope_name`, `collection_name` |

## Quick start (Docker)

```bash
docker run -d --name keepalive --restart unless-stopped \
  -v "$PWD/keepalive.yaml:/keepalive.yaml:ro" \
  ghcr.io/tiennm99/keepalive:latest
```

## Quick start (local)

```bash
git clone https://github.com/tiennm99/keepalive
cd keepalive
cp keepalive.example.yaml keepalive.yaml
go run .
```

## How it works

On every tick the chosen adapter performs the cheapest write that proves the cluster is alive. `counter_key` selects the key/doc ID and defaults to `counter`.

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

Seed the value with your configured `counter_key` when it is not `counter`. MySQL uses backticked identifiers — see `adapter/mysql.go`.

## Adding a new adapter

1. Create `adapter/<name>.go`.
2. Implement the `Adapter` interface in `adapter/adapter.go` (`Connect`, `Increment`, `Close`).
3. Register the factory in `init()`:
   ```go
   func init() { Registry["<name>"] = func(cfg Config) (Adapter, error) { return &myAdapter{}, nil } }
   ```
4. Add an `import _ "your driver"` if needed, and the adapter config keys to `keepalive.example.yaml` and the table above.

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
