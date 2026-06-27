# Pastebin Deployment

This deployment shape runs Pastebin on localhost and exposes it inside a
trusted tailnet with Tailscale Serve. Do not enable Tailscale Funnel for this
service.

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

Edit `/etc/pastebin/pastebin.env` and keep `PASTEBIN_BASE_URL` set to the
Tailscale HTTPS name for this node, for example `https://paste.example.ts.net`.

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

Keep the service private to the trusted tailnet. Do not run `tailscale funnel`
for Pastebin.

## CLI Smoke Test

```sh
export PASTEBIN_URL=https://paste.example.ts.net
printf 'tailnet paste\n' | pastebin
pastebin get abc234def567ghjk
```

Replace `abc234def567ghjk` with the Paste Code returned by the create command.
