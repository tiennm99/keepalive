# Deployment

keepalive is a background worker: it opens no port, serves no HTTP, and needs only outbound network access to your datastores. Every option lives in the [config file](configuration.md).

The image is built from the repository's [`Dockerfile`](../Dockerfile): a static `CGO_ENABLED=0` binary on `gcr.io/distroless/static-debian12:nonroot`, running as user `65532`.

## Docker Compose

```bash
cp config.example.yml config.yml   # then edit it
docker compose up -d --build
docker compose logs -f
```

[`compose.yml`](../compose.yml) mounts `./config.yml` read-only at `/config.yml` and restarts the container unless you stop it.

Before the first start:

- **Create `config.yml` first.** If it is missing, Docker mounts an empty directory in its place and keepalive exits with `/config.yml is a directory, not a config file`.
- **Make it readable by user `65532`.** A file with mode `0600` owned by your user cannot be read inside the container; `0644` works.

After editing `config.yml`, restart the service (`docker compose restart`); keepalive reads its config only at startup.

## Coolify

1. Create an application from the repository with the **Docker Compose** build pack and compose file `/compose.yml`.
2. Coolify turns the `./config.yml` bind mount into an editable file storage. Paste your config there.
3. Leave the service without a domain; keepalive has no port and no healthcheck.
4. Deploy, then restart after any config change.

## Docker

```bash
docker build -t keepalive:local .
docker run -d --name keepalive --restart unless-stopped \
  -v "$PWD/config.yml:/config.yml:ro" \
  keepalive:local
```

To mount the config into a working directory instead of the container root:

```bash
docker run -d --name keepalive --restart unless-stopped \
  --workdir /workspace \
  -v "$PWD/config.yml:/workspace/config.yml:ro" \
  keepalive:local
```

## From source

Requires Go 1.26 or later.

```bash
git clone https://github.com/tiennm99/keepalive
cd keepalive
cp config.example.yml config.yml   # then edit it
go run .
```

## Stopping

On `SIGTERM` or `SIGINT`, keepalive stops every service and exits within 7 seconds, inside Docker's default 10-second stop grace period.
