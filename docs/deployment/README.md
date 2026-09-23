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
sudo install -d -o root -g pastebin -m 0750 /etc/pastebin
sudo sh -c "umask 027; openssl rand -base64 16 | tr '+/' '-_' | tr -d '=\\n' > /etc/pastebin/public-publish-token"
sudo chown root:pastebin /etc/pastebin/public-publish-token
sudo install -D -o root -g pastebin -m 0640 docs/deployment/pastebin.env.example /etc/pastebin/pastebin.env
sudo install -D -o root -g pastebin -m 0640 docs/deployment/pastebin-public.env.example /etc/pastebin/pastebin-public.env
sudo install -D -o root -g root -m 0644 docs/deployment/pastebin.service /etc/systemd/system/pastebin.service
sudo install -D -o root -g root -m 0644 docs/deployment/pastebin-public.service /etc/systemd/system/pastebin-public.service
```

Set the private instance's base URL to its Tailscale HTTPS endpoint. Set the
public instance's base URL and public host to the public hostname. The public
environment points at the generated publishing token. Do not reuse the private
database path in the public environment file.

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

Keep both listeners private to the trusted tailnet. Cloudflare exposes Public
Document reads and bearer-authenticated publication from the public instance.
See [Public Document Ingress](cloudflare-public-read.md).

## CLI Smoke Test

```sh
mkdir -p ~/.config/pastebin
printf 'server=https://paste.example.ts.net\n' > ~/.config/pastebin/config
printf 'tailnet paste\n' | pastebin
pastebin get abc234def567ghjk

sudo install -o "$(id -un)" -g "$(id -gn)" -m 0600 \
  /etc/pastebin/public-publish-token ~/.config/pastebin/public-publish-token
printf '%s\n' \
  'server=https://pastebin.example.com' \
  "publish_token_file=$HOME/.config/pastebin/public-publish-token" \
  > ~/.config/pastebin/public
pastebin --public public-notes.md
```

Replace `abc234def567ghjk` with the Paste Code returned by the create command.
An unauthorized `POST` to the public hostname returns `401` and does not create
a Public Document.

To publish from a browser, open the public hostname, paste Markdown into the
text box, and enter the publishing credential from the public CLI profile.
The form keeps the credential only in the current page and sends it as a
Bearer authorization header. Verify the returned Public Document URL and Raw
Paste URL after deployment.

## Rotate The Publishing Credential

Generate a replacement token with the same restricted ownership and mode,
restart `pastebin-public`, and then replace each Publisher's local token copy.
The command above generates a 22-character credential from 16 random bytes.
The old credential stops working as soon as the service restarts. Public reads,
private Pastes, and existing Public Documents do not depend on the credential.
