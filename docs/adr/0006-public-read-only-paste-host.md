# Public Read-Only Paste Host

Paste creation remains on the trusted tailnet, but Paste Views and Raw Pastes may be read through `pastebin.harm.org`. The application selects a read-only route set for that host, so a proxy rule cannot expose creation by forwarding `POST /`. Cloudflare provides the public hostname and forwards to the existing Tailscale Serve endpoint; the Go service stays loopback-bound.

Public responses use `no-store`, `no-referrer`, and `noindex` policies because a Paste URL is a bearer secret. Private creation receipts use the public hostname, while the CLI continues to submit Paste content through the tailnet endpoint. Tailscale Funnel was rejected because Funnel hostnames are restricted to the tailnet domain and would not supply the requested custom hostname.
