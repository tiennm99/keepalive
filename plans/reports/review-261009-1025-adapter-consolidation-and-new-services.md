---
type: review-report
topic: adapter-consolidation-and-new-services
created_at: 2026-10-09T10:25:00+07:00
project: keepalive
---

# Review Report: Adapter Consolidation and New Services

## Summary

This review covers the whole `tiennm99/keepalive` module: `main.go`, `config.go`, `runner.go`, `service_name.go` and every adapter under `adapter/`. It confirmed ten findings. Two are high or high-leaning: the Couchbase increment resets the counter on any read error, and malformed connection URLs leak passwords into logs. Four are medium reliability gaps in the runner and config loading, and four are low.

On consolidation, one merge is clearly worth doing: serve both `redis` and `valkey` with `github.com/redis/go-redis/v9` and drop `github.com/valkey-io/valkey-go`. A scratch build showed this cuts the stripped binary from 36,503,712 B to 27,394,208 B (about 25 percent). It also removes a real connect failure against servers without client-side caching. The merge needs a URL normalizer that is more careful than the first design, and the corrections are listed below. Merging `postgresql` and `mysql` behind one `database/sql` dialect struct is possible, but the review recommends skipping it on KISS grounds. Swapping `lib/pq` for `pgx` is rejected.

On new services, many hosted free tiers already work with the existing adapters and only need README entries: Redis Cloud, Upstash, YugabyteDB Aeon, CockroachDB Basic, Azure DocumentDB, FerretDB Cloud and Aiven. Among services that need new code, the most valuable additions are a small set of plain `net/http` adapters (Qdrant, Zilliz, Neo4j Aura, Astra DB, Turso, Weaviate, InfluxDB), an AMQP adapter for CloudAMQP, a Kafka adapter for Aiven Kafka, an MQTT adapter for EMQX Serverless, and a pure-Go Oracle adapter for the Oracle Autonomous Database. Every candidate is listed for the user to decide on, and nothing has been added.

## Confirmed review findings

Every finding below was confirmed by reading the code, and most were also reproduced with a scratch probe outside the repo. Severity reflects the verifier's correction where one was made.

| # | Severity | File:line | Finding | Fix |
| --- | --- | --- | --- | --- |
| 1 | High | `adapter/couchbase.go:110` | `Increment` discards its context (`_ context.Context`) and treats any `Get` error as "missing document", so a timeout, temporary failure, RBAC error or missing collection silently resets the counter to 1. `Get` then `Upsert` is not atomic. The WAN development profile sets a 20s KV timeout, so one tick can block for about 40s and ignore the runner's 3s tick timeout, and on SIGTERM `wg.Wait()` can outlast Docker's default 10s stop grace period. | Replace both calls with one atomic `a.coll.Binary().Increment(a.docID, &gocb.IncrementOptions{Delta: 1, Initial: 1, Context: ctx})` and return `int64(res.Content())`. The document that `ensureDocument` seeds holds the digit text `0`, so the first tick still returns 1. |
| 2 | Medium (reported high) | `runner.go:36` | When a URL fails `net/url` parsing, `lib/pq`, `redis.ParseURL` and `valkey.ParseURL` return a `*url.Error` that contains the full raw URL, password included. The runner logs it every 10s. A probe reproduced it for all three adapters with a password containing `%9z`. An unencoded `#` leaks only the part of the password before the `#`. Severity is lowered because only a malformed URL triggers it and the logs belong to the credential owner, but the line repeats and reaches Coolify and log-shipper storage. | Validate the `url` key once at config load. Alternatively, have the URL-based adapters share one helper that uses `errors.As` to detect a `*url.Error` and returns only `urlErr.Err` or the `Redacted()` URL. |
| 3 | Medium | `runner.go:24`, `config.go` (`normalizeConfig`) | An unknown adapter name or a missing required key is caught only inside the service goroutine, which logs once and returns. `main` keeps waiting for a signal, so `restart: unless-stopped` never fires. Plain `yaml.Unmarshal` also drops unknown keys, so a `url:` written at service level instead of under `config:` passes loading. With several services, the bad one goes quiet while the others keep running. | Call `adapter.New(adapterType, cfg)` for each service in `normalizeConfig` (factories do no I/O) and return the error, so `main` exits through `log.Fatalf`. Optionally decode with `yaml.NewDecoder(...)` and `KnownFields(true)`. |
| 4 | Medium | `runner.go:47`, `runner.go:63-66` | After a successful `Connect`, `runService` never re-enters its reconnect loop, and `runConnectedService` logs `Increment` errors forever. Initialization runs only in `Connect`. If a provider restore or a manual drop removes the row or table, PostgreSQL and MySQL fail every tick with `sql: no rows in result set` or "relation does not exist", and no write lands. A dropped Couchbase collection fails the same way. Redis, Valkey and MongoDB recreate the key on their own. | Return from `runConnectedService` after N consecutive failures (for example 3). At line 47, return only when `ctx.Err() != nil`; otherwise wait `reconnectDelay` and loop, which re-runs `New` and `Connect`. Optionally make the increments self-seeding. PostgreSQL: `INSERT ... ON CONFLICT (key) DO UPDATE SET value = keepalive.value + 1 RETURNING value`. MySQL: one `Exec` of `INSERT ... ON DUPLICATE KEY UPDATE value = LAST_INSERT_ID(value+1)`, then the count is 1 when `RowsAffected()==1` and `LastInsertId()` otherwise. Do not issue a separate `SELECT LAST_INSERT_ID()`, which can run on another pooled connection. |
| 5 | Medium | `runner.go:31` | `Connect` gets the process-lifetime context, which has no deadline. A probe against a listener that accepts TCP and never answers showed `lib/pq` hanging past 6s, with or without TLS, even under a 2s context. MySQL returned at 2s. A stalled pooler handshake therefore blocks that service forever and logs nothing. | Two parts. (a) In the postgres adapter, add `connect_timeout` to the DSN when the user has not set one, because `lib/pq` v1.12.3 applies the context only to the TCP dial. (b) Wrap each `Connect` in `context.WithTimeout`, for example 60s, which is longer than Couchbase's default `ready_timeout` of 30s. That bounds MySQL and the initialization statements. |
| 6 | Low | `adapter/redis.go:34-41`, `adapter/couchbase.go:101` | On a failed `Connect`, Redis closes `a.client` but leaves it set, and Couchbase sets `a.cluster` before `ensureDocument` and closes it in a deferred cleanup. The runner's `closeService` then closes it again, logging `close: redis: client is closed` or `close: cluster closed` every 10s after the real error. This is log noise only. | Keep the client in a local variable and assign the field only on success, or set it to nil after closing. |
| 7 | Low | `service_name.go:18-31` | Generated suffixed names (`base-N`) are never recorded or checked, so they can collide with an explicit name in either order. Two services then share a log prefix. | Raise the suffix until the candidate is unused, and record every emitted name in `usedNames`. |
| 8 | Low | `config.go:102-106`, `adapter/valkey.go` | `normalizeConfig` always overwrites `config.counter_key` with the service-level or global value. The valkey factory ignores `namespace`, so a redis block copied to valkey writes the bare key `counter`. Neither use is documented, so the core defect is that unknown or reserved keys are accepted silently. | Reject `config.counter_key` in `normalizeConfig`. The redis/valkey merge below fixes `namespace` if the user approves honoring it. |
| 9 | Low | `runner.go:52-70` | The first `Increment` runs one full interval after `Connect`, and no adapter's `Connect` performs a real write once the row or key exists. With a long interval such as 24h and restarts more often than that, no write ever lands. The default 1m interval hides this. | Run one tick right after a successful `Connect`, before the ticker loop. |
| 10 | Low | `compose.yml:15`, `config.go` (`firstExistingConfigFile`), `Dockerfile` | A missing `./config.yml` makes Docker mount an empty root-owned directory, which the loader accepts and then fails to read, so the container crash-loops. The image runs as uid 65532, so a 0600 host file is unreadable. With no `.dockerignore`, `COPY . .` puts a local `config.yml` and `.git` into the build stage (not the final image). | Reject directories in `firstExistingConfigFile` with a clear message. Add a `.dockerignore` listing `config.yml`, `config.yaml`, `.env`, `.git` and `plans`. Document the uid 65532 read requirement. Optionally use long bind syntax with `create_host_path: false`. |

## Adapter consolidation

### Merge 1: redis and valkey on go-redis/v9 (recommended, with changes)

Both adapters run `SETNX key 0` and `INCR key` over RESP, so one client is enough. `go-redis/v9` is the better choice for four reasons.

- **Size.** In a scratch build (`CGO_ENABLED=0`, `-trimpath -s -w`), removing `valkey-go` saved 9.1 MB, while removing `go-redis` would save only 4.5 MB. `go.mod` loses one direct requirement.
- **Robustness.** `valkey-go` enables client-side caching by default and sends `CLIENT TRACKING ON OPTIN` on connect. Against a RESP2-only or tracking-less server it fails with `ErrNoCache` unless the URL carries `?client_cache=0`, and `adapter/valkey.go` never sets `DisableCache`. Upstash, Garnet and older KeyDB or Dragonfly builds are the likely cases, but none was tested live. `go-redis` falls back from `HELLO 3` to `AUTH` over RESP2 on its own.
- **URL features.** `go-redis` `ParseURL` handles `redis://` and `rediss://` (TLS 1.2+ with ServerName), ACL user and password, the `/db` path, timeouts, pool options and `skip_verify`.
- **Existing behaviour.** The `namespace` key and its tests already live on the go-redis adapter.

**Config compatibility.** Register both names on one factory: `Registry["redis"]` and `Registry["valkey"]` both map to `newRedisAdapter`. Generated service names come from the configured adapter string, so `valkey-<host>` does not change. The counter key does not change, and `SETNX` never overwrites, so existing counters continue.

**Corrections the implementation must apply.** The verifier found the first normalizer sketch too thin.

1. `go-redis` rejects the `valkey://` and `valkeys://` schemes. Rewrite them to `redis://` and `rediss://` with a prefix replace on the raw string, not a full `url.Parse` and `u.String()` round trip, so userinfo is never re-serialized.
2. Several query keys behave differently between the clients.
   - `max_retries=0` disables retries in `valkey-go` but means "default 3" in `go-redis`, so rewrite it to `-1`.
   - A bare `?skip_verify` (empty value) is true in `valkey-go` but false in `go-redis`. Rewrite it to `skip_verify=true`, or a Heroku-style URL silently turns certificate checks back on.
   - `skip_verify` on a non-TLS URL is ignored by `valkey-go` but left over in `go-redis`, which then fails with `unexpected option`. Drop it on non-TLS URLs.
   - `write_timeout` is applied as a dial timeout in `valkey-go` v1.0.75 (a copy-paste bug) and as a socket write timeout in `go-redis`. Keep it, because the `go-redis` meaning is the intended one, and note the change.
   - `protocol` accepts only integers in `go-redis`. This is minor.
3. `valkey-go` silently ignores unknown query keys, while `go-redis` rejects them. For `valkey` configs only, keep the shared keys (`db`, `dial_timeout`, `write_timeout`, `protocol`, `client_name`, `max_retries`, `skip_verify`), drop `client_cache` and any other unknown key, and return a clear error for `addr` and `master_set` (cluster seeds and sentinel), which have no single-client equivalent. Keep `redis` configs exactly as strict as they are today.
4. Keep the existing `Ping` in `Connect`. The sketch removed it, which is an unrequested change to the redis adapter.
5. Honoring `namespace` for `valkey` changes the key only for a config that already carries an ignored `namespace`. That is a user decision. Leave it out of the default change unless the user approves it.

**Risks.** A single `go-redis` client does not follow `MOVED`, so a cluster-mode endpoint that worked through `valkey-go` auto-detection would break. No free-tier target needs cluster mode, so an opt-in through `redis.ParseClusterURL` is not worth adding now. `go-redis` v9.23.0 is the latest release; the repo pins v9.19.0, and the bump is optional.

**Hosted services this adapter covers.** Redis Cloud (free 30 MB), Upstash Redis (`rediss://`), Aiven for Valkey, Render Key Value, Dragonfly Cloud and self-hosted Dragonfly, KeyDB, Microsoft Garnet, and self-hosted Valkey or Redis. Heroku Key-Value Store and Railway Redis are wire-compatible but have no idling free tier, so they are not keepalive targets. Fly.io's Upstash Redis is reachable only from inside Fly's private network. Koyeb does not appear to sell managed Redis (unconfirmed).

### Merge 2: postgresql and mysql behind one database/sql dialect (recommended: skip)

The two adapters share a lifecycle (`sql.Open`, ping, `CREATE TABLE IF NOT EXISTS`, seed, increment in a READ COMMITTED transaction, close) and differ in driver, config key (`url` against `dsn`), DDL quoting, seed statement and read-back. A dialect struct would save about 55 to 60 net lines, not 70. It would add an indirection layer, a `selectBack == ""` branch and a deferred rollback with a named return, and bring no dependency or size win. The next SQL-like targets (Oracle, libSQL over HTTP) would not fit the struct cleanly. The review therefore recommends skipping the merge. If pool sizing is the actual goal, change only the `SetMaxOpenConns(10)` and `SetMaxIdleConns(10)` line in `adapter/mysql.go`.

If the user wants the merge anyway, keep the registry names `postgresql`, `postgres` and `mysql` and the config keys `url` and `dsn`. Be aware that the sketch's pool settings (`SetMaxOpenConns(2)`, `SetConnMaxLifetime(3m)`) also change PostgreSQL behaviour, not only MySQL. CockroachDB silently upgrades READ COMMITTED to SERIALIZABLE unless READ COMMITTED is enabled; this is unchanged by the merge but worth a smoke test.

**Hosted services these adapters cover.** PostgreSQL (`lib/pq`): Supabase (session pooler on port 5432), Aiven for PostgreSQL, YugabyteDB Aeon (YSQL, port 5433), CockroachDB Cloud Basic, Nhost, Neon, Timescale/Tiger Cloud, Prisma Postgres (direct TCP), Render and Railway Postgres. MySQL (`go-sql-driver/mysql`): Aiven for MySQL, TiDB Cloud Starter, SingleStore Helios, MariaDB/SkySQL, Railway MySQL. PlanetScale is wire-compatible but dropped its free Hobby tier in April 2024.

### Rejected merges

- **`lib/pq` to `pgx/v5` stdlib.** The "maintenance mode" premise is stale: `lib/pq` shipped v1.12.0 to v1.12.3 between March and April 2026, and its README no longer carries the notice. A scratch build with `pgx` was 3.3 MB larger and added three indirect dependencies, for no feature this daemon uses. Note that a future `lib/pq` release may change the default `sslmode` from `require` to `prefer`, so keep `?sslmode=require` in examples.
- **MongoDB-compatible services.** There is nothing to merge. `adapter/mongodb.go` is already one adapter with a `mongo` alias, and every compatible service connects through URI options. For the README: `mongo-driver` v2 requires server 4.2 or later, Cosmos DB RU and AWS DocumentDB need `retrywrites=false`, and DocumentDB needs `tlsCAFile`. The driver can move from v2.6.0 to v2.9.2 independently.
- **Couchbase with anything.** Its KV protocol has no sibling among the other adapters. It is the heaviest dependency (grpc, otel, protobuf, zap), but removing it means dropping Capella support, which is a user decision.
- **Redis cluster or sentinel auto-detection.** No free-tier target needs it, so the merged adapter returns an explicit error for `addr` and `master_set` instead.

**Hosted services the unchanged adapters cover.** MongoDB (`mongo-driver/v2`): MongoDB Atlas Free, Azure DocumentDB free tier, FerretDB Cloud, Azure Cosmos DB for MongoDB (API 4.2 or later). Couchbase (`gocb/v2`): Couchbase Capella free tier.

## Candidate new services for the user to decide

The user decides which of these to add; nothing has been added. "Unconfirmed" means the policy could not be read on an official page, or the page leaves the key point undefined. Idle policies were checked on 2026-10-09.

### Works today with existing adapters (README or example change only)

| Service | Idle policy | Activity that counts | Adapter | ToS risk | Value | Source |
| --- | --- | --- | --- | --- | --- | --- |
| Supabase Free | Paused after about 7 days of low activity, with an email a week before. Restorable for up to 1 year (the page anchor still says 90 days). | User database queries, API calls, Dashboard visits. Whether a direct Postgres query counts is inferred, not stated. | `postgresql` (session pooler port 5432, `sslmode=require`; direct host is IPv6-only) | Low | High | supabase.com/docs/guides/platform/free-project-pausing |
| MongoDB Atlas Free | Paused after 30 days of inactivity, with an email 7 days before. | Inactivity is "zero connections"; a driver write counts. | `mongodb` | Low | High | mongodb.com/docs/atlas/pause-terminate-cluster/ |
| Couchbase Capella free tier | Turned off after 72 hours idle and deleted after 30 days idle. | Not defined; an SDK write is the safe choice (inferred). | `couchbase` (interval 12 to 24h) | Low | High | docs.couchbase.com/cloud/get-started/get-started.html |
| Redis Cloud Free 30 MB | Deleted permanently after 14 days with no Redis commands; no backups. Heroku listing says 30 days. **Unconfirmed**: the support article returns 403, so this comes from search excerpts and Redis Insight 2.64 notes. | Redis read or write commands only. Console logins do not count. | `redis` (daily) | Low | High | support.redislabs.com article 34131767970450 |
| Upstash Redis Free | Archived after at least 30 days of inactivity. Data is backed up but restored into a new database, so the URL changes. One source mentions 14 days; the live FAQ says 30. | Not defined; any command should count. | `redis` (`rediss://`) | Low | High | upstash.com/docs/redis/help/faq |
| YugabyteDB Aeon Sandbox | Paused after 10 days of inactivity and deleted after 15 days. | Any SELECT, UPDATE, INSERT or DELETE from a connected application. | `postgresql` (port 5433; egress IP on the allow list) | Low | High | docs.yugabyte.com (create-clusters-free) |
| Aiven free PostgreSQL, MySQL, Valkey | May be powered off with "no continuative activity" or no initial use in the first few hours. No threshold is published. Powered-off services are deleted after 180 days. | Not defined; real writes are the safe reading. Whether DB traffic alone is enough is **unconfirmed**. | `postgresql` / `mysql` / `valkey` | Low to medium | High | aiven.io/docs/platform/howto/custom-plans |
| CockroachDB Cloud Basic | Deleted after 6 months with no activity; cannot be restored. | Not defined; SQL traffic is assumed. | `postgresql` (`sslmode=verify-full`, port 26257) | Low | Medium | docs.cockroachlabs.com (basic-cluster-management) |
| Azure DocumentDB free tier | Paused after 60 days of inactivity. No backups on the free tier. | Not defined (**unconfirmed**). | `mongodb` (live-check the upsert and `$inc`) | None found | Medium | learn.microsoft.com/azure/documentdb/free-tier |
| FerretDB Cloud free tier | "May be stopped or deleted if inactive"; no duration given. | Not defined. | `mongodb` | Low | Medium | docs.ferretdb.io/installation/ferretdb-cloud/ |
| Nhost Starter | Project paused after 1 week of inactivity; only one active free project. | Not documented (**unconfirmed** whether DB traffic counts). | `postgresql` (public connection string) | Low to unknown | Medium | nhost.io/pricing |
| SingleStore Helios trial | Trial group warned after 2 days with no query, suspended 24h later, terminated 30 days after suspension. The free Shared edition has no idle rule. | Any query. | `mysql` (live-check the DDL dialect) | Medium (defeats a trial) | Low | docs.singlestore.com (sign-up-to-try) |

Several services match an existing adapter but gain nothing from a keepalive, so they should not be listed as targets.

- **Neon Free and Koyeb Postgres** scale to zero after 5 minutes and keep the data. Pinging only burns compute quota (100 CU-hours a month on Neon, 5 hours a month on Koyeb).
- **Render Free Postgres** expires 30 days after creation whatever its activity, and **Render Key Value** loses data on restart.
- **Railway Free** sleeps services with no outbound traffic, but staying awake burns the $1 monthly credit (Free-plan forced sleep is **unconfirmed**).
- **TiDB Cloud Starter, Nile, Prisma Postgres and Xata** have no documented idle policy, or no free cloud tier.

### Needs a new adapter

| Service | Idle policy | Activity that counts | New adapter shape | ToS risk | Value | Source |
| --- | --- | --- | --- | --- | --- | --- |
| Oracle Autonomous AI Database (Always Free) | Stopped after 7 days inactive; may be deleted after 90 cumulative days stopped. A keepalive cannot start a stopped DB. | A SQL*Net or HTTPS connection that runs SQL resets the measurement. | `oracle` via pure-Go `github.com/sijms/go-ora/v2` (database/sql), or HTTPS POST to the ORDS REST-enabled SQL endpoint | Low | High | docs.oracle.com (autonomous-always-free) |
| Neo4j AuraDB Free | Paused after 72 hours without writes; deleted after 30 days paused. | Writes (from a staff forum quote of a support article). | Plain HTTP POST to Query API v2 with a single-line `MERGE ... SET k.count = coalesce(k.count,0)+1` | None found | High | neo4j.com/docs/aura/managing-instances/instance-actions/ |
| Qdrant Cloud free tier | Suspended after 1 week unused; deleted after 4 weeks. | Not defined; an upsert is safest. | Plain HTTP: create a size-1 collection once, then read point 1 and upsert `count+1` | None found | High | qdrant.tech/documentation/cloud/create-cluster/ |
| Zilliz Cloud Free | Suspended after 7 days of inactivity; no deletion timeline in the docs (support KB returned 403). | Not defined; use an upsert. | Plain HTTP REST v2: create collection once, then query and upsert | None found | High | docs.zilliz.com/docs/free-and-serverless-clusters |
| DataStax Astra DB Serverless free | Hibernated after 48 hours idle; deleted after 30 days hibernated. | Any Data API or CQL request; a request also triggers resume (503 at first). | Plain HTTP `findOneAndUpdate` with `$inc` on `count` and `upsert: true` (interval 12h or less) | None found | High | docs.datastax.com (database-statuses) |
| Turso Free | Archived after 10 days of inactivity; manual unarchive. | Database requests. | Plain HTTP POST to `/v2/pipeline` with a bearer token running an upsert `RETURNING value` (avoid cgo `go-libsql`) | Low | High | docs.turso.tech/cli/group/unarchive |
| CloudAMQP Little Lemur / Loyal Lemming | Deleted after two months with no publish or consume; Little Lemur also drops queues idle for 28 days. | Publishing or consuming at least one message. | `amqp` via pure-Go `github.com/rabbitmq/amqp091-go`: declare a queue, publish one message, `basic.get` it back | Low | High | cloudamqp.com/pricing |
| Aiven free Kafka | Powered off after 24 hours with no produce or consume (stated on the create page; the overview page omits it). Data is lost on power-off, and services are deleted after 180 days. | Producing or consuming messages. | `kafka` via pure-Go `github.com/twmb/franz-go`: produce one record (hourly) | Low to medium | High | aiven.io/docs/products/kafka/free-tier/create-free-tier-kafka-service |
| EMQX Cloud Serverless | Stopped after 30 days with no client connections; may be deleted 30 days later. | An active MQTT client connection. | `mqtt` via pure-Go `github.com/eclipse/paho.mqtt.golang` with TLS SNI: connect, publish one QoS1 message, disconnect. Needs a synthetic count or a "touch" variant of `Increment`. | Low | High | docs.emqx.com/en/cloud/latest/create/serverless.html |
| Weaviate Cloud free | Suspended after 7 days of inactivity; deleted after 30 days. | Not defined (**unconfirmed**). | Plain HTTP: create a vectorless collection once, then read and update one object with a fixed UUID | Low to medium | Medium | docs.weaviate.io/cloud/manage-clusters/create |
| InfluxDB Cloud Free | Account "may be closed" after 30 days with zero writes and zero queries. | Writes, or task and query executions. | Plain HTTP POST of one line-protocol point to `/api/v2/write`; read back with a Flux `last()` | Low | Medium | influxdata.com/influxdb-cloud-pricing-faq/ |
| Aiven OpenSearch free | A new service with no use in 24h is powered off; the ongoing rule has no stated duration. **Unconfirmed**. | Indexing or querying (stated only for the first 24h). | Plain HTTP `POST /<index>/_update/<id>` with a painless script and upsert; shared with Bonsai | Low to medium | Medium | aiven.io/docs/products/opensearch/concepts/opensearch-free-tier |
| Algolia Build | "May remove Subscriber Data" after 30 days inactive; support timelines (2 months, 8 to 10 weeks) **unconfirmed** (403). | Searches, indexing or configuration changes (support snippets). | Plain HTTP partial update with `Increment` and `createIfNotExists=true`; the write is asynchronous | Low to medium ("evaluation purposes") | Medium | algolia.com/policies/build-plan-specific-terms |
| Auth0 Free tenant | All tenants scheduled for deletion after 150 days with no activity. | Dashboard logins, user logins or API calls (support article; API alone **unconfirmed**). | Plain HTTP: client-credentials token, then `PATCH /api/v2/clients/{id}` metadata | Low to medium | Medium | auth0.com/docs/secure/data-privacy-and-compliance/data-processing |
| GitHub Actions schedules (public repos) | Scheduled workflows disabled after 60 days with no repository activity. | Not defined; commits are commonly understood to count. | Plain HTTP `PUT .../actions/workflows/{id}/enable` with a fine-grained PAT | Medium | Medium | docs.github.com (disable-and-enable-workflows) |
| Render free web services | Spin down after 15 minutes without inbound traffic; 750 instance hours a month. | Inbound HTTP requests and WebSocket messages. | Generic HTTP GET with a 60 to 90s timeout | Medium (AUP "bypass usage restrictions"; quota drain) | Medium | render.com/docs/free |
| Bonsai Sandbox | Deleted after "several weeks" unused (2023 docs). Free plan now appears only on Heroku. **Unconfirmed**. | Any incoming traffic. | Shared OpenSearch/Elasticsearch HTTP adapter | Low | Low | bonsai.io/docs (periodic-cluster-cleanup) |
| Upstash Vector | No Vector-specific policy; the 30-day archive rule is documented only for Redis. **Unconfirmed**. | Not defined. | Plain HTTP fetch and upsert | None found | Low | upstash.com/docs/redis/help/faq |
| Hugging Face Spaces (cpu-basic) | Sleeps after 48 hours inactive; new free CPU Gradio and Docker Spaces now need a paid plan. | Visitors (whether a bare GET counts is undocumented). | Generic HTTP GET (24h or less) | Low to medium | Low | huggingface.co/docs/hub/spaces-gpus |
| Koyeb Free Instance | Scales to zero after 1 hour; new signups need a paid plan since February 2026. | Internet traffic over HTTP/1.1 (HTTP/2 cannot wake it). | Generic HTTP GET | Low to medium | Low | koyeb.com/docs/run-and-scale/scale-to-zero |

The following were checked and are not worth an adapter.

- **No idle policy:** Cloudflare D1, KV and R2; Azure Cosmos DB for NoSQL; Pinecone Starter (third-party pause claims are **unconfirmed**); Firebase Spark; Convex; Synadia; Docker Hub; MotherDuck; Chroma; Tiger Cloud.
- **Trial or time-based expiry:** Elastic Cloud, ClickHouse Cloud, Confluent Cloud, Meilisearch Cloud, Typesense Cloud, ScyllaDB Cloud.
- **Discontinued or ending:** Glitch, Upstash Kafka, HiveMQ Serverless (ends 2026-12-31).
- **Policy keyed to something a remote write cannot reach:** Appwrite (Console activity only), PythonAnywhere (owner confirmation), Streamlit Community Cloud (WebSocket visits and a manual wake button), Google Colab (UI only, automation prohibited), Oracle Always Free compute (on-host CPU, memory and network use).
- **Cost features, not idle policies:** Fly.io autostop, Railway app sleeping, Replit Autoscale, PocketHost hibernation (a few seconds), Google Cloud Pub/Sub subscription expiry (set it to never expire instead).

## Proposed new adapters ranked

The ranking weighs how many confirmed, high-value services an adapter unlocks against the dependency it adds. All stay pure Go.

1. **`http` (generic, plain `net/http`, no new dependency).** One configurable request (method, URL, headers, body, expected status) covers Render web services, Hugging Face Spaces, Koyeb, GitHub workflow enable, Auth0 (with a token step), Oracle ORDS and Nhost GraphQL. It has the widest reach and the lowest cost. The current interface expects `Increment` to return a counter, so this adapter needs a local tick count or a "touch" variant of the interface. A JSON path for reading a counter back would let it serve some database APIs too, but that adds config surface.
2. **Thin HTTP-database adapters sharing one helper.** Qdrant, Zilliz, Neo4j Query API, Astra Data API, Turso pipeline, Weaviate and InfluxDB each need a setup call, an increment-like write and a JSON response parse. They share auth headers, a JSON client and error handling. Either write one small adapter per service on a shared `httpjson` helper, or treat them as presets of the generic `http` adapter. Qdrant, Neo4j, Astra and Turso have the best-confirmed policies.
3. **`oracle` via `github.com/sijms/go-ora/v2`.** Oracle's Always Free database has an explicitly documented reset condition and a hard 90-day deletion risk. It follows the existing SQL adapter pattern. Use a PL/SQL block that tolerates ORA-00955 for table creation, then `MERGE` and `UPDATE ... RETURNING`. Wallet-less TLS needs mTLS set to "not required" with an ACL.
4. **`amqp` via `github.com/rabbitmq/amqp091-go`.** It covers CloudAMQP (RabbitMQ and LavinMQ) and any AMQP 0-9-1 broker, with a confirmed policy and a small dependency.
5. **`kafka` via `github.com/twmb/franz-go`.** It covers Aiven free Kafka, whose 24h power-off loses data, and any Kafka-compatible broker. The dependency is moderate in size.
6. **`mqtt` via `github.com/eclipse/paho.mqtt.golang`.** It covers EMQX Serverless and self-hosted brokers. Its value is lower because the idle window is 30 days and HiveMQ's free plan is ending.

## Unresolved questions

1. Should the merged `valkey` adapter honor `namespace`? Doing so changes the key for any valkey config that already carries an ignored `namespace`.
2. Is any deployment pointing `adapter: valkey` at a cluster-mode or sentinel endpoint? The merge would break it, and the answer decides whether cluster support is needed now.
3. Should the `Adapter` interface gain a "touch" operation without a counter, which the generic `http`, `mqtt` and `kafka` adapters need, or should those adapters return a local count?
4. Should `normalizeConfig` reject unknown YAML keys (`KnownFields(true)`)? That would be stricter for existing configs that carry stray keys.
5. Does plain database traffic prevent Aiven's free power-off, or is a Console login also needed? Only a live test can answer this.
6. Should README examples for Couchbase, Astra and Neo4j recommend a shorter interval, given their 48 to 72 hour windows, and should Supabase docs warn against the 6543 transaction pooler with `lib/pq`?
7. Is the user willing to accept the Medium ToS risk of keeping Render web services or GitHub scheduled workflows alive, or should the generic `http` adapter be documented for warm-start use only?
