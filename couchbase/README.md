# couchbase-keepalive

> [!IMPORTANT]
> **Archived — moved to [tiennm99/db-keepalive](https://github.com/tiennm99/db-keepalive).**
>
> All six `*-keepalive` repos were consolidated into a single binary with pluggable adapters. Use `DB_TYPE=couchbase` with the new image:
>
> ```bash
> docker run -d --restart unless-stopped \
>   -e DB_TYPE=couchbase \
>     -e COUCHBASE_CONNECTION_STRING='...' -e COUCHBASE_USERNAME=... # ...etc \
>   ghcr.io/tiennm99/db-keepalive:latest
> ```
>
> The source here is retained for git history. No further changes will land on this repo.

## Original description

Lightweight Go utility performing periodic pings to keep Couchbase Capella free clusters active.

## License

Apache-2.0 — see [LICENSE](LICENSE).
