# mongodb-keepalive

> [!IMPORTANT]
> **Archived — moved to [tiennm99/db-keepalive](https://github.com/tiennm99/db-keepalive).**
>
> All six `*-keepalive` repos were consolidated into a single binary with pluggable adapters. Use `DB_TYPE=mongodb` with the new image:
>
> ```bash
> docker run -d --restart unless-stopped \
>   -e DB_TYPE=mongodb \
>     -e MONGODB_URI='...' -e MONGODB_DATABASE=keepalive -e MONGODB_COLLECTION=counter \
>   ghcr.io/tiennm99/db-keepalive:latest
> ```
>
> The source here is retained for git history. No further changes will land on this repo.

## Original description

Go utility performs periodic operations to prevent Mongo DB Atlas free instances from being deleted due to inactivity.

## License

Apache-2.0 — see [LICENSE](LICENSE).
