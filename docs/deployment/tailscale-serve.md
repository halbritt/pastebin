# Tailscale Serve Example

Pastebin should be reachable only inside the trusted tailnet. Use Tailscale
Serve to publish the local HTTP service; do not use Tailscale Funnel.

Pastebin listens on localhost:

```sh
curl -fsS http://127.0.0.1:8080/healthz
```

Publish the local service:

```sh
tailscale serve --bg 8080
tailscale serve status
```

Use the tailnet HTTPS URL as `PASTEBIN_BASE_URL`:

```sh
sudo sed -i 's|^PASTEBIN_BASE_URL=.*|PASTEBIN_BASE_URL=https://paste.example.ts.net|' /etc/pastebin/pastebin.env
sudo systemctl restart pastebin
```

Check from a tailnet-connected machine:

```sh
curl -fsS https://paste.example.ts.net/healthz
```

Remove the Serve rule:

```sh
tailscale serve off
```
