# Known issues

Problems seen when pointing keepalive at hosted datastores, listed by the log line they produce. keepalive creates its table, key, or document on first connect, so a dedicated keepalive user needs write access plus permission to create that object. Until the cause is fixed, the service logs the error and retries every minute; no restart is needed after the fix.

## PostgreSQL: `permission denied for schema public`

```
[aiven-postgres] connect: pq: permission denied for schema public at position 2:28 (42501); retrying in 1m0s
```

**Cause.** On connect, keepalive runs `CREATE TABLE IF NOT EXISTS keepalive` in the `public` schema. Since PostgreSQL 15, only the database owner can create objects in `public` by default, so a non-owner user is refused. PostgreSQL checks this permission even when the table already exists.

**Fix.** Give keepalive its own database and make its user the owner. Run this as an admin user (`avnadmin` on Aiven):

```sql
CREATE DATABASE keepalive OWNER keepalive;      -- new database
ALTER DATABASE keepalive OWNER TO keepalive;    -- existing database
```

Then point `url` at that database, for example `postgres://keepalive:...@host:port/keepalive?sslmode=require`.

On Aiven for PostgreSQL, changing the owner was the fix that worked; `GRANT CREATE ON SCHEMA public TO keepalive` did not. If you try the grant anyway, note that it applies only inside the database it runs in, so a SQL editor connected to `defaultdb` does not change the database in `url`.

## Redis / Valkey: `NOPERM No permissions to access a key`

```
[aiven-valkey] connect: NOPERM No permissions to access a key; retrying in 1m0s
```

**Cause.** The user's ACL key pattern does not cover the counter key. keepalive writes `<namespace>:<counter_key>` when `namespace` is set, otherwise `<counter_key>` (default `counter`).

On Aiven for Valkey, a common trigger is typing the key pattern with a leading `~`. Aiven adds the `~` itself, so `~keepalive:*` becomes a pattern that only matches keys starting with a literal `~`, and every write is refused.

**Fix.** Allow the commands `PING`, `SETNX`, and `INCR`, and set a key pattern that covers the counter key. On Aiven, enter the keys field without the `~`:

| Field      | Value                                                   |
| ---------- | ------------------------------------------------------- |
| Categories | `+@all -@admin -@dangerous` (or `+@connection +@string`) |
| Keys       | `keepalive:*` (matching `namespace: keepalive`)         |

## Couchbase: `CONNECTION_ERROR` or bucket not ready

**Cause.** The cluster is unreachable or the bucket cannot be opened with the configured user.

**Fix.** Check the connection string, `bucket_name`, the database user's permissions, and Capella's allowed IP/network access. If the cluster or bucket was just created, raise `ready_timeout`; keepalive allows `ready_timeout` plus 1 minute for each connect attempt.
