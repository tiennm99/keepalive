# mysql-keepalive

> [!IMPORTANT]
> **Archived — moved to [tiennm99/db-keepalive](https://github.com/tiennm99/db-keepalive).**
>
> All six `*-keepalive` repos were consolidated into a single binary with pluggable adapters. Use `DB_TYPE=mysql` with the new image:
>
> ```bash
> docker run -d --restart unless-stopped \
>   -e DB_TYPE=mysql \
>     -e DATA_SOURCE_NAME='...' \
>   ghcr.io/tiennm99/db-keepalive:latest
> ```
>
> The source here is retained for git history. No further changes will land on this repo.

## Original description

Go utility performs periodic operations to MySQL server.

## License

Apache-2.0 — see [LICENSE](LICENSE).
