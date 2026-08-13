# Tailscale Serve Example

Paste creation should be reachable only inside the trusted tailnet. Use
Tailscale Serve to publish the local HTTP service. A separate public hostname
may proxy read-only requests through that Serve endpoint; do not expose the
loopback listener or use Tailscale Funnel for a custom hostname.

Pastebin listens on localhost:

```sh
curl -fsS http://127.0.0.1:8080/healthz
```

Publish the local service:

```sh
tailscale serve --bg 8080
tailscale serve status
```

Keep the CLI configured to submit through the tailnet URL. When public reads
are enabled, use the public URL for `PASTEBIN_BASE_URL` and configure its host:

```sh
sudo sed -i 's|^PASTEBIN_BASE_URL=.*|PASTEBIN_BASE_URL=https://pastebin.example.com|' /etc/pastebin/pastebin.env
sudo sed -i 's|^PASTEBIN_PUBLIC_HOST=.*|PASTEBIN_PUBLIC_HOST=pastebin.example.com|' /etc/pastebin/pastebin.env
sudo systemctl restart pastebin
```

Check from a tailnet-connected machine:

```sh
curl -fsS https://paste.example.ts.net/healthz
```

See [Public Read-Only Ingress](cloudflare-public-read.md) for the custom public
hostname and method boundary.

Remove the Serve rule:

```sh
tailscale serve off
```
