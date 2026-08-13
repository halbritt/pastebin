# Pastebin Deployment

This deployment shape runs Pastebin on localhost and exposes creation inside a
trusted tailnet with Tailscale Serve. Public Paste reads can use a separate
Cloudflare hostname that reaches the same service through Tailscale Serve.

## Build On The Host

```sh
go test ./...
make build
```

## Install Files

Create the runtime user and data directory:

```sh
id -u pastebin >/dev/null 2>&1 || sudo useradd --system --user-group --home-dir /var/lib/pastebin --shell /usr/sbin/nologin pastebin
sudo install -d -o pastebin -g pastebin -m 0750 /var/lib/pastebin
```

Install binaries:

```sh
sudo install -D -o root -g root -m 0755 bin/pastebind /usr/local/bin/pastebind
sudo install -D -o root -g root -m 0755 bin/pastebin /usr/local/bin/pastebin
```

Install configuration and the systemd unit:

```sh
sudo install -D -o root -g pastebin -m 0640 docs/deployment/pastebin.env.example /etc/pastebin/pastebin.env
sudo install -D -o root -g root -m 0644 docs/deployment/pastebin.service /etc/systemd/system/pastebin.service
```

For a private-only deployment, set `PASTEBIN_BASE_URL` to the Tailscale HTTPS
name. For public reads, set it to the public hostname and set
`PASTEBIN_PUBLIC_HOST` to that hostname. The CLI still submits through the
Tailscale URL.

## Start Pastebin

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now pastebin
sudo systemctl status pastebin
curl -fsS http://127.0.0.1:8080/healthz
```

Logs are written through stdout and stderr to the systemd journal:

```sh
sudo journalctl -u pastebin -f
```

## Expose With Tailscale Serve

On the same node, publish the localhost service to the tailnet:

```sh
tailscale serve --bg 8080
tailscale serve status
```

Then check the tailnet URL from a tailnet-connected machine:

```sh
curl -fsS https://paste.example.ts.net/healthz
```

Keep the listener and creation surface private to the trusted tailnet. Do not
run `tailscale funnel` for the custom public hostname. See
[Public Read-Only Ingress](cloudflare-public-read.md).

## CLI Smoke Test

```sh
mkdir -p ~/.config/pastebin
printf 'server=https://paste.example.ts.net\n' > ~/.config/pastebin/config
printf 'tailnet paste\n' | pastebin
pastebin get abc234def567ghjk
```

Replace `abc234def567ghjk` with the Paste Code returned by the create command.
