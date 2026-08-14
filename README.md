# Pastebin

Pastebin turns plain text into a bearer Paste URL. The normal instance is a
private tailnet service. A second instance may host explicitly published Public
Documents, but it must run with a separate process and database.

The service has no application accounts. Anyone who can reach an instance's
tailnet endpoint may create content in that instance. A configured public host
may also accept creation with a dedicated publishing bearer token; requests
without that token cannot create content.

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

When `PASTEBIN_PUBLIC_HOST` is configured, that host serves `GET /`,
`GET /p/{code}`, `GET /raw/{code}`, and `GET /healthz`. If
`PASTEBIN_PUBLISH_TOKEN_FILE` is also configured, `POST /` requires that
publishing bearer token. Configure both only on a distinct Public Pastebin
instance with its own database. Public responses disable shared caching,
referrer transmission, and search indexing.

Without a configured publishing token, the public host returns `405` for
`POST /`. With one configured, a missing or invalid bearer token returns `401`
without reading the request body or calling storage.

## Browser Paste Rendering

`GET /p/{code}` renders the stored text as Markdown with Goldmark's built-in
GitHub Flavored Markdown (GFM) extension bundle and a local text-highlighting
extension. Paste Views support:

- tables
- strikethrough with `~~text~~`
- literal autolinks for `http://`, `https://`, `www.`, and email addresses
- task lists with `- [ ]` and `- [x]`
- highlighted text with `==text==`

FTP URLs remain plain text because the sanitizer does not permit the `ftp`
scheme.

Task-list checkboxes are display-only. They are rendered disabled, and the HTML
sanitizer permits only the checkbox attributes needed for that output.

Markdown rendering happens when the browser requests a Paste View, so the same
dialect applies to pastes created before this support was added. Stored content,
`GET /raw/{code}`, CLI retrieval, and `pastebin get --json` continue to return
the submitted text instead of rendered HTML.

Paste Views do not render raw HTML and do not add footnotes, definition lists,
emoji expansion, math or diagram syntax, code syntax highlighting, or
interactive task state.

## Configuration

| Variable | Default/example | Purpose |
| --- | --- | --- |
| `PASTEBIN_BASE_URL` | `https://paste.example.ts.net` | Base URL returned in Paste receipts |
| `PASTEBIN_PUBLIC_HOST` | empty | Public Document hostname |
| `PASTEBIN_PUBLISH_TOKEN_FILE` | empty | File containing the public publishing bearer token; requires `PASTEBIN_PUBLIC_HOST` |
| `PASTEBIN_LISTEN` | `127.0.0.1:8080` | HTTP listen address |
| `PASTEBIN_DB` | `/var/lib/pastebin/pastebin.db` | SQLite database path |
| `PASTEBIN_MAX_BYTES` | `1048576` | Maximum Paste size in bytes; deployment examples set both services to `10485760` (10 MiB) |
| `PASTEBIN_DEFAULT_TTL` | `168h` | Default expiration, equivalent to 7 days |
| `PASTEBIN_MAX_TTL` | `720h` | Maximum expiration, equivalent to 30 days |

Allowed creation expirations are `1h`, `1d`, `7d`, and `30d`.

## Explicit Public Documents

Run the Public Pastebin as a second process with a dedicated listen port and
SQLite database. Never point the public instance at the private database.

```sh
PASTEBIN_BASE_URL=https://pastebin.example.com \
PASTEBIN_PUBLIC_HOST=pastebin.example.com \
PASTEBIN_PUBLISH_TOKEN_FILE=/etc/pastebin/public-publish-token \
PASTEBIN_LISTEN=127.0.0.1:8081 \
PASTEBIN_DB=/var/lib/pastebin-public/pastebin.db \
bin/pastebind
```

Create a separate CLI config containing the public hostname and a local copy
of the publishing token:

```sh
mkdir -p ~/.config/pastebin
printf '%s\n' \
  'server=https://pastebin.example.com' \
  'publish_token_file=/path/to/public-publish-token' \
  > ~/.config/pastebin/public
bin/pastebin --public public-notes.md
```

The returned URL uses the public hostname. Existing private Pastes are not
published, copied, or made reachable through that hostname. Keep the default
CLI config pointed at the private service so public publication stays explicit.
The `--public` flag always loads `~/.config/pastebin/public`; private
`PASTEBIN_URL`, `PASTEBIN_CONFIG`, and `PASTEBIN_PUBLISH_TOKEN_FILE` values do
not override that profile.

## CLI Examples

Set the private creation endpoint once with either `PASTEBIN_URL`:

```sh
export PASTEBIN_URL=https://proximal.example.ts.net:18080
```

or a config file:

```sh
mkdir -p ~/.config/pastebin
printf 'server=https://proximal.example.ts.net:18080\n' > ~/.config/pastebin/config
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
- `docs/deployment/pastebin-public.env.example`
- `docs/deployment/pastebin-public.service`
- `docs/deployment/tailscale-serve.md`

The documented data paths are `/var/lib/pastebin/pastebin.db` for private
Pastes and `/var/lib/pastebin-public/pastebin.db` for Public Documents. The
runtime user is `pastebin`, and service logs go to the systemd journal through
stdout and stderr.
