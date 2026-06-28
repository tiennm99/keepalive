---
type: research-report
topic: supabase-postgres-fit
created_at: 2026-06-28T23:35:00Z
project: keepalive
---

# Research Report: Supabase Postgres Fit

## Summary

Yes. This project can apply to Supabase Postgres now through the existing `postgresql` adapter. Supabase exposes a normal Postgres connection string, and this repo already uses `database/sql` with `github.com/lib/pq`, which accepts Postgres URL DSNs and `sslmode` parameters.

No source-code change required for basic use. Need only add a Supabase service config with the right connection string. For production polish, add a Supabase example, set Postgres max open connections to `1`, and consider configurable schema/table to avoid creating `public.keepalive`.

## Research Methodology

- Sources consulted: local repo files + 4 official/external docs.
- Date: 2026-06-28.
- Key terms: Supabase Postgres connection string, Supabase pooler, Supavisor, lib/pq DSN, Supabase RLS public schema.
- Scope: compatibility, config, security, performance. No migration or implementation.

## Key Findings

### 1. Current Project Fit

Current adapter: [adapter/postgresql.go](/config/workspace/tiennm99/keepalive/adapter/postgresql.go)

Behavior:

- Accepts `config.url`.
- Supports adapter names `postgresql` and `postgres`.
- Opens `sql.Open("postgres", url)` via `github.com/lib/pq`.
- On connect: `Ping`, `CREATE TABLE IF NOT EXISTS keepalive`, seed key.
- On tick: transaction + `UPDATE keepalive SET value = value + 1 WHERE key = $1 RETURNING value`.

Supabase fit:

| Requirement | Status |
| --- | --- |
| Postgres protocol | Supported |
| URL DSN | Supported by `lib/pq` |
| SSL | Supported via `sslmode` |
| Cheap write | Supported |
| Reconnect after startup failure | Supported |
| Supabase-specific API | Not needed |

### 2. Supabase Connection Choice

Supabase docs say Postgres clients use connection strings. Direct connection is best for long-lived servers, but the direct endpoint is IPv6 unless the project has IPv4 add-on. Shared pooler session mode is for persistent app traffic from IPv4-only networks. Transaction mode is for temporary/serverless clients and does not support prepared statements.

This daemon is long-lived. Recommended order:

1. Direct connection if runtime supports IPv6 or Supabase IPv4 add-on enabled.
2. Shared pooler session mode if runtime is IPv4-only.
3. Transaction pooler only if tested, because Supabase warns it does not support prepared statements.

Example direct:

```yaml
services:
  - name: supabase-postgres
    adapter: postgresql
    interval: 5m
    config:
      url: "postgresql://postgres:[PASSWORD]@db.[PROJECT-REF].supabase.co:5432/postgres?sslmode=require"
```

Example shared pooler session mode:

```yaml
services:
  - name: supabase-postgres
    adapter: postgresql
    interval: 5m
    config:
      url: "postgresql://postgres.[PROJECT-REF]:[PASSWORD]@aws-[REGION].pooler.supabase.com:5432/postgres?sslmode=require"
```

### 3. Security Considerations

Current SQL creates unqualified table `keepalive`. With a normal Supabase connection, that likely means `public.keepalive`.

Supabase docs say RLS must be enabled for tables in exposed schemas, and `public` is exposed by default. Tables created via raw SQL do not automatically get RLS unless configured.

Safer options:

1. Use as-is, then enable RLS on `public.keepalive` and add no API policies.
2. Use a private schema through connection `search_path` if compatible with chosen pooler.
3. Better code hardening: add optional `schema` / `table` config and qualify identifiers safely.

Runtime role:

- Basic config with `postgres` user works.
- Least-privilege role is not ideal today because adapter always runs `CREATE TABLE IF NOT EXISTS`; it needs create/init permissions at startup.
- Hardening option: add `init: false` mode after manually creating table, then grant only `SELECT`, `INSERT`, `UPDATE`.

### 4. Performance Insights

Load is tiny.

At default `interval: 1m`, each Supabase service does about:

- 1 update/minute
- 1,440 updates/day
- 43,200 updates/month

This is negligible for Postgres. It still keeps one Go `sql.DB` open. For Supabase small/free tiers, set `db.SetMaxOpenConns(1)` in the adapter to make connection usage explicit.

### 5. Free Project Pausing

Supabase docs currently state Free Plan projects with low activity over a 7-day period may be paused, while paid projects are not paused for inactivity.

This daemon performs real database writes, so it is technically aligned with a keepalive use case. Do not treat this as SLA. If the project matters, paid Supabase plan is cleaner than relying on keepalive activity.

## Comparative Analysis

| Option | Pros | Cons | Recommendation |
| --- | --- | --- | --- |
| Existing `postgresql` adapter | Works now, simple | Creates `keepalive` in default schema | Use now |
| New `supabase` adapter alias | Friendlier UX | Mostly duplicate of Postgres adapter | Optional only |
| Add Supabase docs/example | Low risk, useful | No behavior change | Do it |
| Add schema/table config | Better security posture | Needs safe identifier handling/tests | Good next hardening |
| Use Supabase Data API | Avoid DB conn details | Wrong tool for daemon counter writes | Do not do |

## Implementation Recommendations

### Quick Start

1. In Supabase dashboard, open project -> Connect.
2. Copy direct or session pooler connection string.
3. Add `?sslmode=require` unless already present.
4. Add service config:

```yaml
services:
  - name: supabase-postgres
    adapter: postgresql
    interval: 5m
    counter_key: supabase-postgres
    config:
      url: "postgresql://postgres.[PROJECT-REF]:[PASSWORD]@aws-[REGION].pooler.supabase.com:5432/postgres?sslmode=require"
```

5. Run:

```bash
go run .
```

6. Confirm in Supabase SQL editor:

```sql
select * from keepalive;
```

### Optional Code Changes

Minimal docs-only improvement:

- Add Supabase example to `config.example.yml`.
- Add README note: direct vs session pooler.

Small hardening:

- In `adapter/postgresql.go`, after `sql.Open`, set:

```go
db.SetMaxOpenConns(1)
db.SetMaxIdleConns(1)
```

Production hardening:

- Add optional `schema` and `table` config.
- Validate identifiers with strict allow-list.
- Quote identifiers safely.
- Add tests for default table and custom schema/table SQL.

## Common Pitfalls

- IPv4-only deploy using direct Supabase host without IPv4 add-on: connection fails. Use shared pooler session mode.
- Transaction pooler with prepared statements: Supabase warns this is unsupported. Prefer direct/session mode for this daemon.
- Raw SQL table in `public`: enable RLS or use private schema.
- Password special chars in URL: URL-encode password.
- Custom least-privilege role: current adapter still needs init permissions.

## Verification

Local tests:

```text
go test ./...
ok github.com/tiennm99/keepalive
ok github.com/tiennm99/keepalive/adapter
```

## Resources & References

- Supabase connect docs: https://supabase.com/docs/guides/database/connecting-to-postgres
- Supabase RLS docs: https://supabase.com/docs/guides/database/postgres/row-level-security
- Supabase free project pausing docs: https://supabase.com/docs/guides/platform/free-project-pausing
- lib/pq docs: https://pkg.go.dev/github.com/lib/pq

## Next Steps

1. Use existing `postgresql` adapter with Supabase session pooler or direct URL.
2. Add README/config example for Supabase.
3. Add `SetMaxOpenConns(1)` hardening.
4. Decide whether `public.keepalive` acceptable; if not, implement schema/table config.

## Unresolved Questions

- Target deployment supports IPv6, or IPv4-only?
- Is `public.keepalive` acceptable, or need private schema by default?
