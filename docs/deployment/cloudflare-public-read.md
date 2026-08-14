# Public Read-Only Ingress

This deployment makes explicitly published Public Documents readable at a
custom hostname without exposing the private Paste collection.

The request path is:

```text
public reader → Cloudflare → cloudflared → Tailscale Serve → loopback Public Pastebin
```

Configure a distinct Public Pastebin process with its own database:

```sh
PASTEBIN_BASE_URL=https://pastebin.example.com
PASTEBIN_PUBLIC_HOST=pastebin.example.com
PASTEBIN_LISTEN=127.0.0.1:8081
PASTEBIN_DB=/var/lib/pastebin-public/pastebin.db
```

Do not use the private Pastebin database path. Keep the CLI's default server on
the private instance and pass the Public Pastebin's tailnet URL only for an
explicit publication.

Add the hostname to the locally managed Cloudflare Tunnel. Use the Tailscale
Serve HTTPS URL as the service so public traffic still crosses the Tailscale
proxy:

```yaml
ingress:
  - hostname: pastebin.example.com
    service: https://node.example.ts.net:18081
  - service: http_status:404
```

The Public Pastebin application, not the Tunnel rule, is the write authority.
Requests whose Host matches `PASTEBIN_PUBLIC_HOST` use a route set without
`POST /`. The Tunnel forwards the whole hostname so a public write probe
exercises that application boundary instead of relying on a path-only proxy
rule.

Validate the Tunnel config before restarting it:

```sh
cloudflared tunnel ingress validate
cloudflared tunnel ingress rule https://pastebin.example.com/p/example
```

Create or update the public DNS route with the tunnel's existing name or UUID:

```sh
cloudflared tunnel route dns TUNNEL pastebin.example.com
```

After deployment, publish a fresh document through the Public Pastebin's
tailnet URL and verify that a known private Paste Code remains unavailable:

```sh
curl -fsS https://pastebin.example.com/p/PASTE_CODE >/dev/null
curl -fsS https://pastebin.example.com/raw/PASTE_CODE >/dev/null
curl -sS -o /dev/null -w '%{http_code}\n' \
  https://pastebin.example.com/p/KNOWN_PRIVATE_CODE
curl -sS -o /dev/null -w '%{http_code}\n' \
  -X POST --data 'must not be stored' https://pastebin.example.com/
```

The two Public Document reads must return 200, the known private code must
return 404, and the public write must return 405. Also check
that public responses include `Cache-Control: private, no-store`,
`Referrer-Policy: no-referrer`, and
`X-Robots-Tag: noindex, nofollow, noarchive`.
