# keepalive

Pluggable Go daemon that periodically touches external services to prevent idle shutdowns, pauses, or cold starts.

The current adapters perform cheap datastore writes for Redis Cloud, Valkey, Aiven, Neon, Supabase, MongoDB Atlas, Couchbase Capella, and similar hosted services.

Successor to the `*-keepalive` family: one binary, one image, six datastore adapters.

## Configuration

By default, keepalive reads the first config file it finds: `config.yml`, `config.yaml`, `/config.yml`, then `/config.yaml`. One deployment can keep any number of services alive.

```yaml
# Default interval for every service.
interval: 1m
counter_key: counter

services:
  - adapter: redis
    config:
      url: redis://default@redis-a.example.com:6379
      namespace: keepalive

  - adapter: valkey
    # One service can override the global interval and counter key.
    interval: 30s
    counter_key: valkey-counter
    config:
      url: valkey://default@valkey-a.example.com:6379

  - adapter: postgresql
    config:
      url: postgresql://user:pass@postgres-a.example.com:5432/keepalive?sslmode=require

  - adapter: mysql
    config:
      dsn: user:pass@tcp(mysql-a.example.com:3306)/keepalive

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

`interval` at the root sets the default schedule for every service and defaults to `1m`.
`interval` inside a service overrides that default only for that service.
In the example above, every service runs every `1m` except `valkey`, which runs every `30s`.
Interval values use Go duration syntax, for example `30s`, `5m`, `1h`, `1h30m`, or `1.5h`. Plain integers are treated as seconds, so `90` means `90s`.

`counter_key` at the root sets the default counter key for every service and defaults to `counter`.
`counter_key` inside a service overrides that default only for that service.
In the example above, every service writes `counter` except `valkey`, which writes `valkey-counter`.

## Supported adapters

| `adapter`    | Driver                              | `config` keys |
| ------------ | ----------------------------------- | ------------- |
| `redis`      | `github.com/redis/go-redis/v9`      | `url`, optional `namespace` |
| `valkey`     | `github.com/valkey-io/valkey-go`    | `url` |
| `postgresql` | `github.com/lib/pq`                 | `url` |
| `mysql`      | `github.com/go-sql-driver/mysql`    | `dsn` |
| `mongodb`    | `go.mongodb.org/mongo-driver/v2`    | `uri`, `database`, `collection` |
| `couchbase`  | `github.com/couchbase/gocb/v2`      | `connection_string`, `username`, `password`, `bucket_name`, `scope_name`, `collection_name`, optional `ready_timeout`, optional `bucket_ram_quota_mb` |

## Quick start (Compose)

```bash
cp config.example.yml config.yml
docker compose up -d --build
```

## Quick start (Docker)

```bash
docker build -t keepalive:local .
docker run -d --name keepalive --restart unless-stopped \
  -v "$PWD/config.yml:/config.yml:ro" \
  keepalive:local
```

If you prefer to mount the config into a working directory instead of the container root, set the container working directory and mount the file there:

```bash
docker run -d --name keepalive --restart unless-stopped \
  --workdir /workspace \
  -v "$PWD/config.yml:/workspace/config.yml:ro" \
  keepalive:local
```

## Quick start (local)

```bash
git clone https://github.com/tiennm99/keepalive
cd keepalive
cp config.example.yml config.yml
go run .
```

## How it works

On startup each adapter initializes the minimum resource it owns, then every tick performs the cheapest write that proves the cluster is alive. `counter_key` selects the key/doc ID and defaults to `counter`.

- **Redis** — initialize with `SETNX key 0`, then `INCR key`. When `namespace` is empty, the key is `counter`; when `namespace: keepalive`, the key is `keepalive:counter`.
- **Valkey** — initialize with `SETNX key 0`, then `INCR key`
- **PostgreSQL** — `CREATE TABLE IF NOT EXISTS keepalive`, seed `key`, then `UPDATE ... RETURNING`
- **MySQL** — `CREATE TABLE IF NOT EXISTS keepalive`, seed `key`, then `UPDATE` + `SELECT`
- **MongoDB** — upsert `{_id: key, count: 0}` on connect, then `FindOneAndUpdate({_id: key}, {$inc: {count: 1}}, upsert)`
- **Couchbase** — optionally create the bucket when `bucket_ram_quota_mb` is set, create configured scope/collection when missing, insert `key = 0` if missing, then `GET key` -> `++` -> `UPSERT key`

Each configured service starts independently. If one service cannot connect, it logs the error and retries without stopping other services in the same deployment.

For hosted Couchbase/Capella clusters, `ready_timeout` defaults to `30s`. If Couchbase reports `CONNECTION_ERROR`, check the connection string, bucket name, database user permissions, and Capella allowed IP/network access.

## Adding a new adapter

1. Create `adapter/<name>.go`.
2. Implement the `Adapter` interface in `adapter/adapter.go` (`Connect`, `Increment`, `Close`).
3. Register the factory in `init()`:
   ```go
   func init() { Registry["<name>"] = func(cfg Config) (Adapter, error) { return &myAdapter{}, nil } }
   ```
4. Add an `import _ "your driver"` if needed, and the adapter config keys to `config.example.yml` and the table above.

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
