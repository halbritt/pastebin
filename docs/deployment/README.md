# Pastebin Deployment

This deployment runs two isolated Pastebin instances: a private Paste service
and a Public Pastebin containing only explicitly published documents. They use
different processes, ports, and SQLite databases.

## Build On The Host

```sh
go test ./...
make build
```

## Install Files

Create the runtime user and separate data directories:

```sh
id -u pastebin >/dev/null 2>&1 || sudo useradd --system --user-group --home-dir /var/lib/pastebin --shell /usr/sbin/nologin pastebin
sudo install -d -o pastebin -g pastebin -m 0750 /var/lib/pastebin
sudo install -d -o pastebin -g pastebin -m 0750 /var/lib/pastebin-public
```

Install binaries:

```sh
sudo install -D -o root -g root -m 0755 bin/pastebind /usr/local/bin/pastebind
sudo install -D -o root -g root -m 0755 bin/pastebin /usr/local/bin/pastebin
```

Install configuration and the systemd unit:

```sh
sudo install -D -o root -g pastebin -m 0640 docs/deployment/pastebin.env.example /etc/pastebin/pastebin.env
sudo install -D -o root -g pastebin -m 0640 docs/deployment/pastebin-public.env.example /etc/pastebin/pastebin-public.env
sudo install -D -o root -g root -m 0644 docs/deployment/pastebin.service /etc/systemd/system/pastebin.service
sudo install -D -o root -g root -m 0644 docs/deployment/pastebin-public.service /etc/systemd/system/pastebin-public.service
```

Set the private instance's base URL to its Tailscale HTTPS endpoint. Set the
public instance's base URL and public host to the public hostname. Do not reuse
the private database path in the public environment file.

## Start Pastebin

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now pastebin pastebin-public
sudo systemctl status pastebin pastebin-public
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS http://127.0.0.1:8081/healthz
```

Logs are written through stdout and stderr to the systemd journal:

```sh
sudo journalctl -u pastebin -u pastebin-public -f
```

## Expose With Tailscale Serve

On the same node, publish the localhost service to the tailnet:

```sh
tailscale serve --bg --https=18080 http://127.0.0.1:8080
tailscale serve --bg --https=18081 http://127.0.0.1:8081
tailscale serve status
```

Then check the tailnet URL from a tailnet-connected machine:

```sh
curl -fsS https://paste.example.ts.net:18080/healthz
curl -fsS https://paste.example.ts.net:18081/healthz
```

Keep both listeners and both creation surfaces private to the trusted tailnet.
Only the Public Pastebin's read host crosses Cloudflare. See [Public Read-Only
Ingress](cloudflare-public-read.md).

## CLI Smoke Test

```sh
mkdir -p ~/.config/pastebin
printf 'server=https://paste.example.ts.net\n' > ~/.config/pastebin/config
printf 'tailnet paste\n' | pastebin
pastebin get abc234def567ghjk

printf 'explicit public document\n' | \
  pastebin --server https://paste.example.ts.net:18081
```

Replace `abc234def567ghjk` with the Paste Code returned by the create command.
