# Public Read-Only Ingress

This deployment keeps Paste creation on the tailnet while making Paste bearer
links readable at a custom public hostname.

The request path is:

```text
public reader → Cloudflare → cloudflared → Tailscale Serve → loopback Pastebin
```

Configure Pastebin with the public receipt URL and host boundary:

```sh
PASTEBIN_BASE_URL=https://pastebin.example.com
PASTEBIN_PUBLIC_HOST=pastebin.example.com
```

Keep the CLI's `server` setting on the Tailscale Serve URL. Private creation
then returns public Paste and Raw Paste URLs.

Add the hostname to the locally managed Cloudflare Tunnel. Use the Tailscale
Serve HTTPS URL as the service so public traffic still crosses the Tailscale
proxy:

```yaml
ingress:
  - hostname: pastebin.example.com
    service: https://node.example.ts.net:18080
  - service: http_status:404
```

The application, not the Tunnel rule, is the write authority. Requests whose
Host matches `PASTEBIN_PUBLIC_HOST` use a route set without `POST /`. The
Tunnel forwards the whole hostname so a public write probe exercises that
application boundary instead of relying on a path-only proxy rule.

Validate the Tunnel config before restarting it:

```sh
cloudflared tunnel ingress validate
cloudflared tunnel ingress rule https://pastebin.example.com/p/example
```

Create or update the public DNS route with the tunnel's existing name or UUID:

```sh
cloudflared tunnel route dns TUNNEL pastebin.example.com
```

After deployment, create a fresh Paste through the tailnet URL and verify:

```sh
curl -fsS https://pastebin.example.com/p/PASTE_CODE >/dev/null
curl -fsS https://pastebin.example.com/raw/PASTE_CODE >/dev/null
curl -sS -o /dev/null -w '%{http_code}\n' \
  -X POST --data 'must not be stored' https://pastebin.example.com/
```

The two reads must return 200 and the public write must return 405. Also check
that public responses include `Cache-Control: private, no-store`,
`Referrer-Policy: no-referrer`, and
`X-Robots-Tag: noindex, nofollow, noarchive`.
