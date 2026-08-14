# Tailscale Serve Example

The private and Public Pastebin instances use separate loopback listeners and
Tailscale Serve ports. Cloudflare reaches only the Public Pastebin's Serve port;
the application requires a publishing credential for public-host creation.

The two instances listen on localhost:

```sh
curl -fsS http://127.0.0.1:8080/healthz
curl -fsS http://127.0.0.1:8081/healthz
```

Publish the local service:

```sh
tailscale serve --bg --https=18080 http://127.0.0.1:8080
tailscale serve --bg --https=18081 http://127.0.0.1:8081
tailscale serve status
```

Keep the normal CLI configuration on the private tailnet URL. Use a separate
CLI config for authenticated publication through the public hostname.

Check from a tailnet-connected machine:

```sh
curl -fsS https://paste.example.ts.net:18080/healthz
curl -fsS https://paste.example.ts.net:18081/healthz
```

See [Public Document Ingress](cloudflare-public-read.md) for the custom public
hostname and authentication boundary.

Remove the Serve rule:

```sh
tailscale serve off
```
