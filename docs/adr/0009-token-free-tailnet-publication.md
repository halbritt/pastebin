# Token-Free Tailnet Publication

The CLI's `--public` profile points to the distinct Public Pastebin's tailnet
endpoint and does not require a publishing token. The server returns Public
Document URLs using its configured public base URL. Tailnet reachability is
the authority to create content, just as it is for the private service.

This corrects ADR 0008's requirement that the public CLI profile contain a
credential. Bearer authentication remains required for creation through the
internet-facing hostname, where it prevents anonymous clients from filling the
disk. A profile may still specify a token for publication outside the tailnet.
The CLI must not fall back to the private collection when public publication
fails. The services continue to use separate databases.
