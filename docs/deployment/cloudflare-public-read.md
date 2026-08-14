# Public Document Ingress

This deployment makes explicitly published Public Documents readable at a
custom hostname and permits credentialed publication without exposing the
private Paste collection or anonymous creation.

The request path is:

```text
reader or Publisher → Cloudflare → cloudflared → Tailscale Serve → loopback Public Pastebin
```

Configure a distinct Public Pastebin process with its own database:

```sh
PASTEBIN_BASE_URL=https://pastebin.example.com
PASTEBIN_PUBLIC_HOST=pastebin.example.com
PASTEBIN_PUBLISH_TOKEN_FILE=/etc/pastebin/public-publish-token
PASTEBIN_LISTEN=127.0.0.1:8081
PASTEBIN_DB=/var/lib/pastebin-public/pastebin.db
```

Do not use the private Pastebin database path. Keep the CLI's default config on
the private instance and use a separate config containing the public hostname
and publishing token file for Explicit Publication.

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
Requests whose Host matches `PASTEBIN_PUBLIC_HOST` can reach `POST /` only when
`PASTEBIN_PUBLISH_TOKEN_FILE` configures the route. The application verifies
the bearer credential before reading or storing the request body. The Tunnel
forwards the whole hostname so both authorized and unauthorized probes exercise
the application boundary.

Validate the Tunnel config before restarting it:

```sh
cloudflared tunnel ingress validate
cloudflared tunnel ingress rule https://pastebin.example.com/p/example
```

Create or update the public DNS route with the tunnel's existing name or UUID:

```sh
cloudflared tunnel route dns TUNNEL pastebin.example.com
```

After deployment, publish a fresh document through the public hostname and
verify that a known private Paste Code remains unavailable. `PUBLISH_TOKEN`
below is read from the restricted token file and must not be pasted into shell
history:

```sh
PUBLISH_TOKEN="$(sudo cat /etc/pastebin/public-publish-token)"
curl -fsS \
  -H "Authorization: Bearer $PUBLISH_TOKEN" \
  --data-binary 'explicit public document' \
  https://pastebin.example.com/
unset PUBLISH_TOKEN
curl -fsS https://pastebin.example.com/p/PASTE_CODE >/dev/null
curl -fsS https://pastebin.example.com/raw/PASTE_CODE >/dev/null
curl -sS -o /dev/null -w '%{http_code}\n' \
  https://pastebin.example.com/p/KNOWN_PRIVATE_CODE
curl -sS -o /dev/null -w '%{http_code}\n' \
  -X POST --data 'must not be stored' https://pastebin.example.com/
```

The two Public Document reads must return 200, the known private code must
return 404, and the unauthorized public write must return 401 without changing
the Public Document count. Also check
that public responses include `Cache-Control: private, no-store`,
`Referrer-Policy: no-referrer`, and
`X-Robots-Tag: noindex, nofollow, noarchive`.
