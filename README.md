# Pastebin

Pastebin is a private tailnet service for turning plain text into a Paste URL
that can be opened from another machine, shell, or browser.

The security boundary is the trusted tailnet. The service has no application
accounts in v1: anyone who can reach it may create a Paste, and anyone with a
Paste URL may read that Paste. Deploy it with Tailscale Serve, not Tailscale
Funnel.

## Build

Prerequisites:

- Go 1.23 or newer
- A host joined to the target tailnet for deployment

From the repository root:

```sh
go test ./...
make build
```

`make build` writes:

- `bin/pastebind`: HTTP service
- `bin/pastebin`: CLI

## Run Locally

```sh
mkdir -p /tmp/pastebin
PASTEBIN_BASE_URL=http://127.0.0.1:8080 \
PASTEBIN_LISTEN=127.0.0.1:8080 \
PASTEBIN_DB=/tmp/pastebin/pastebin.db \
bin/pastebind
```

Health check:

```sh
curl -fsS http://127.0.0.1:8080/healthz
```

Routes:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/` | Web Paste form |
| `POST` | `/` | Create Paste |
| `GET` | `/p/{code}` | Browser Paste view |
| `GET` | `/raw/{code}` | Raw Paste text |
| `GET` | `/healthz` | Health check |

## Configuration

| Variable | Default/example | Purpose |
| --- | --- | --- |
| `PASTEBIN_BASE_URL` | `https://paste.example.ts.net` | Public base URL returned in Paste receipts |
| `PASTEBIN_LISTEN` | `127.0.0.1:8080` | HTTP listen address |
| `PASTEBIN_DB` | `/var/lib/pastebin/pastebin.db` | SQLite database path |
| `PASTEBIN_MAX_BYTES` | `1048576` | Maximum Paste size in bytes |
| `PASTEBIN_DEFAULT_TTL` | `168h` | Default expiration, equivalent to 7 days |
| `PASTEBIN_MAX_TTL` | `720h` | Maximum expiration, equivalent to 30 days |

Allowed creation expirations are `1h`, `1d`, `7d`, and `30d`.

## CLI Examples

Set the service URL once with either `PASTEBIN_URL`:

```sh
export PASTEBIN_URL=https://paste.example.ts.net
```

or a config file:

```sh
mkdir -p ~/.config/pastebin
printf 'server=https://paste.example.ts.net\n' > ~/.config/pastebin/config
```

Create a Paste from standard input:

```sh
printf 'hello from host A\n' | bin/pastebin
```

Create a Paste from a file:

```sh
bin/pastebin --expires 1h notes.txt
```

Create a Paste and print a JSON receipt:

```sh
bin/pastebin --json notes.txt
```

Use a one-off server URL:

```sh
bin/pastebin --server https://paste.example.ts.net notes.txt
```

Retrieve by Paste URL, Raw Paste URL, or Paste Code:

```sh
bin/pastebin get https://paste.example.ts.net/p/abc234def567ghjk
bin/pastebin get --raw abc234def567ghjk
bin/pastebin get --json abc234def567ghjk
```

Show the CLI version:

```sh
bin/pastebin version
```

## Deployment

Use the deployment artifacts in `docs/deployment/`:

- `docs/deployment/README.md`
- `docs/deployment/pastebin.env.example`
- `docs/deployment/pastebin.service`
- `docs/deployment/tailscale-serve.md`

The documented production data path is `/var/lib/pastebin/pastebin.db`, the
runtime user is `pastebin`, and service logs go to the systemd journal through
stdout and stderr.
