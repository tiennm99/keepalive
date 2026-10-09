# How it works

Each configured service runs in its own goroutine and goes through the same loop:

1. **Connect.** Open a client and create the minimum resource the adapter owns (a key, table row, or document). One attempt, including this setup, has 1 minute (Couchbase: `ready_timeout` plus 1 minute).
2. **Write now.** Increment the counter once right away, so every start or restart writes even when `interval` is long.
3. **Write on schedule.** Increment once per `interval`. Each write has 3 seconds and logs `[name] counter: N`.
4. **Recover.** When a connect or a write fails, log one line, close the connection, wait 1 minute, and go back to step 1. Reconnecting re-runs setup, so a dropped table, row, key, or collection is recreated.

Services are independent: one failing service never stops or slows the others.

## What each adapter writes

`key` below is the service's `counter_key` (default `counter`).

| Adapter      | On connect | On each tick |
| ------------ | ---------- | ------------ |
| `redis`      | `PING`, then `SETNX key 0` (key is `namespace:key` when `namespace` is set) | `INCR key` |
| `postgresql` | `CREATE TABLE IF NOT EXISTS keepalive (key TEXT PRIMARY KEY, value BIGINT)`, then insert `key` with `0` if missing | `UPDATE keepalive SET value = value + 1 WHERE key = $1 RETURNING value` |
| `mysql`      | `CREATE TABLE IF NOT EXISTS keepalive`, then `INSERT IGNORE` `key` with `0` | `UPDATE` then `SELECT` the value, in one transaction |
| `mongodb`    | Upsert `{_id: key, count: 0}` | `FindOneAndUpdate({_id: key}, {$inc: {count: 1}})` with upsert |
| `couchbase`  | Create the bucket when `bucket_ram_quota_mb` is set, create the scope and collection when missing, insert `key = 0` if missing | Atomic binary increment of `key` |

## Timeouts

| Step                    | Limit |
| ----------------------- | ----- |
| Connect and setup       | 1 minute; Couchbase gets `ready_timeout` plus 1 minute |
| PostgreSQL handshake    | `connect_timeout=30` seconds, added to the connection string unless you set one |
| One write               | 3 seconds |
| Retry after a failure   | 1 minute |
| Shutdown                | 7 seconds, then keepalive exits |

## Logging

keepalive logs one line per write and one line per failure, prefixed with the service name. Connection strings never reach the logs: errors from a malformed URL or DSN are replaced with a generic hint, and the generated service name uses only the host. Driver-internal logging from go-redis is silenced, since keepalive already reports each failure once.

## History

This repository replaces six single-datastore repositories (`redis-keepalive`, `valkey-keepalive`, `postgresql-keepalive`, `mysql-keepalive`, `mongodb-keepalive`, `couchbase-keepalive`). Their histories were absorbed here; earlier commits keep each implementation under its own subfolder.
