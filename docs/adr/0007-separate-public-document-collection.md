# Separate Public Document Collection

The Public Pastebin runs as a separate process with its own SQLite database and Tailscale Serve port. Trusted Publishers explicitly submit Public Documents through that instance's tailnet endpoint, while `pastebin.harm.org` provides read-only access to only that collection. Sharing the private database was rejected because knowing any private Paste Code made that Paste publicly retrievable; no private record is copied or promoted into the public collection.
