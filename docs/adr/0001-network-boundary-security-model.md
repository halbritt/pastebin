# Network Boundary Security Model

Pastebin relies on a trusted network boundary, such as a tailnet, rather than application-level accounts or paste creation tokens. Anyone who can reach the private service may create a paste, and anyone who has the unguessable Paste URL may read that Paste; this preserves the low-friction shell workflow while keeping creation off the public internet.

ADR 0007 extends this decision with a separate Public Pastebin collection. Private Paste creation and retrieval remain inside the trusted network. ADR 0008 adds a dedicated credential only for publishing to the separate public collection; it does not authenticate or expose the private service.
