# Configuration

keepalive reads one YAML file: the first that exists of `config.yml`, `config.yaml`, `/config.yml`, then `/config.yaml`. The app reads no environment variables. [`config.example.yml`](../config.example.yml) shows every adapter.

The file holds datastore credentials, so keep it out of git (`.gitignore` and `.dockerignore` already exclude `config.yml` and `config.yaml`).

## Structure

```yaml
interval: 1m            # optional, default 1m
counter_key: counter    # optional, default counter

services:
  - name: cache-a       # optional
    adapter: redis      # required
    interval: 30s       # optional, overrides the root interval
    counter_key: hits   # optional, overrides the root counter_key
    config:             # adapter-specific keys, see below
      url: rediss://default:PASSWORD@cache-a.example.com:6379
      namespace: keepalive
```

| Key           | Where             | Meaning |
| ------------- | ----------------- | ------- |
| `interval`    | root, service     | Time between writes. Go duration syntax (`30s`, `5m`, `1h30m`, `1.5h`); a plain integer means seconds (`90` = `90s`). Must be greater than zero. Default `1m`. |
| `counter_key` | root, service     | Key, row, or document ID the adapter increments. Default `counter`. Set it at the root or on a service, never inside `config`. |
| `name`        | service           | Log prefix for the service. Must be unique. When omitted, keepalive builds one from the adapter and host, such as `redis-cache-a-example-com`, adding `-2`, `-3`, and so on when it is taken. |
| `adapter`     | service           | One of the adapters below. |
| `config`      | service           | Connection settings for that adapter. |

Pick an interval well inside the provider's idle window. The default `1m` suits everything; hourly or daily writes are enough for policies measured in days.

## Adapter keys

### `redis`

Covers every server that speaks the Redis protocol: Redis, Valkey, Dragonfly, KeyDB, Garnet, and hosted variants.

| Key         | Required | Meaning |
| ----------- | -------- | ------- |
| `url`       | yes      | `redis://` or `rediss://` (TLS) URL, with optional user, password, `/db`, and [go-redis query options](https://pkg.go.dev/github.com/redis/go-redis/v9#ParseURL). `valkey://` is not accepted; use `redis://` or `rediss://`. |
| `namespace` | no       | Key prefix. With `namespace: keepalive`, the key is `keepalive:counter`. |

Cluster and Sentinel endpoints are not supported; point `url` at a single node or a proxy endpoint.

### `postgresql` (alias `postgres`)

| Key   | Required | Meaning |
| ----- | -------- | ------- |
| `url` | yes      | `postgres://` or `postgresql://` URL, or a key=value DSN (`host=... user=... dbname=...`). Without `connect_timeout`, keepalive adds `connect_timeout=30`. |

The user must be able to create a table in the database's `public` schema; see [Known issues](known-issues.md#postgresql-permission-denied-for-schema-public).

For Supabase, use the session pooler connection string (port `5432`) with `sslmode=require`; the direct host is IPv6-only.

### `mysql`

| Key   | Required | Meaning |
| ----- | -------- | ------- |
| `dsn` | yes      | [go-sql-driver DSN](https://github.com/go-sql-driver/mysql#dsn-data-source-name), for example `user:pass@tcp(host:3306)/keepalive?tls=true`. |

### `mongodb` (alias `mongo`)

| Key          | Required | Meaning |
| ------------ | -------- | ------- |
| `uri`        | yes      | `mongodb://` or `mongodb+srv://` connection string. |
| `database`   | yes      | Database holding the counter. |
| `collection` | yes      | Collection holding the counter document. |

The driver needs MongoDB 4.2 or later. Cosmos DB and AWS DocumentDB need `retrywrites=false` in the URI.

### `couchbase`

| Key                   | Required | Meaning |
| --------------------- | -------- | ------- |
| `connection_string`   | yes      | `couchbase://` or `couchbases://` (TLS) address. |
| `username`            | yes      | Database user. |
| `password`            | yes      | Database user's password. |
| `bucket_name`         | yes      | Bucket holding the counter. |
| `scope_name`          | yes      | Scope; created when missing. Use `_default` for the default scope. |
| `collection_name`     | yes      | Collection; created when missing. Use `_default` for the default collection. |
| `ready_timeout`       | no       | How long to wait for the bucket and each setup step. Duration or seconds. Default `30s`. |
| `bucket_ram_quota_mb` | no       | When set, create the bucket with this RAM quota if it does not exist. Needs a user allowed to create buckets. |

## Validation

keepalive checks the whole file at startup.

Startup fails, with the offending `services[N]` path in the error, when:

- the file is missing, is a directory, or is not valid YAML;
- `services` is empty;
- a service has no `adapter`, or names an unknown one;
- a required `config` key is missing;
- `counter_key` appears inside `config`;
- an `interval` is not a positive duration;
- two services share a `name`.

Keys the schema does not define only log a warning, and keepalive ignores them:

```
warning: /config.yml: line 1: field intervall not found in type main.appConfig; ignoring it
warning: /config.yml: services[0].config: unknown key "namspace" for adapter redis (known: url, namespace); ignoring it
```

Warnings name the key, never its value.
