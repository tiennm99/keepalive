# keepalive

**Stop free-tier databases from pausing.** keepalive is a tiny Go daemon that writes to each of your hosted datastores on a schedule, so idle-shutdown policies never trigger.

- **One config, many services.** Keep Redis, PostgreSQL, MySQL, MongoDB, and Couchbase instances alive from a single process.
- **Cheapest possible write.** Each tick increments one counter; nothing else in your database is touched.
- **Self-healing.** Failed services retry every minute and recreate their counter if it disappears, without affecting the others.
- **Small and safe.** A static, distroless, non-root image with no ports, and connection strings are kept out of logs.

Works with Redis Cloud, Upstash, Aiven (PostgreSQL, MySQL, Valkey), Supabase, YugabyteDB, CockroachDB, MongoDB Atlas, Couchbase Capella, and any self-hosted Redis-, Valkey-, PostgreSQL-, MySQL-, MongoDB-, or Couchbase-compatible server.

## Quick start

Write a `config.yml`:

```yaml
interval: 1h

services:
  - name: upstash
    adapter: redis
    config:
      url: rediss://default:PASSWORD@example.upstash.io:6379

  - name: supabase
    adapter: postgresql
    config:
      url: postgresql://USER:PASSWORD@aws-0-region.pooler.supabase.com:5432/postgres?sslmode=require

  - name: atlas
    adapter: mongodb
    config:
      uri: mongodb+srv://USER:PASSWORD@cluster0.example.mongodb.net
      database: keepalive
      collection: counter
```

Run it:

```bash
git clone https://github.com/tiennm99/keepalive && cd keepalive
# put your config.yml here (see config.example.yml for every adapter)
docker compose up -d --build
docker compose logs -f
```

Each service logs a line like `[upstash] counter: 42` on every tick.

## Adapters

| `adapter`                 | Use it for                                                        |
| ------------------------- | ----------------------------------------------------------------- |
| `redis`                   | Redis, Valkey, Dragonfly, KeyDB, Garnet, Upstash, Redis Cloud      |
| `postgresql` / `postgres` | PostgreSQL, Supabase, Aiven, YugabyteDB, CockroachDB               |
| `mysql`                   | MySQL, MariaDB, Aiven for MySQL, TiDB                              |
| `mongodb` / `mongo`       | MongoDB Atlas, Azure DocumentDB, FerretDB, Cosmos DB for MongoDB   |
| `couchbase`               | Couchbase Capella and self-hosted Couchbase                        |

## Documentation

- [Configuration](docs/configuration.md): every option, per-adapter keys, and validation rules.
- [Deployment](docs/deployment.md): Docker Compose, Coolify, plain Docker, and running from source.
- [How it works](docs/how-it-works.md): what each adapter writes, retries, timeouts, and shutdown.
- [Known issues](docs/known-issues.md): permission errors and other setup problems, by log line.
- [Adding an adapter](docs/adding-an-adapter.md): plug in a new datastore.

## License

Apache-2.0. See [LICENSE](LICENSE).
