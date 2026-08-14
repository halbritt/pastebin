# Separate Public Document Collection

The Public Pastebin runs as a separate process with its own SQLite database and Tailscale Serve port. Trusted Publishers explicitly submit Public Documents to that instance, while `pastebin.harm.org` exposes only that collection. Sharing the private database was rejected because knowing any private Paste Code made that Paste publicly retrievable; no private record is copied or promoted into the public collection. ADR 0008 adds authenticated publication through the public hostname without changing this storage boundary.
