# Network Boundary Security Model

Pastebin relies on a trusted network boundary, such as a tailnet, rather than application-level accounts or paste creation tokens. Anyone who can reach the private service may create a paste, and anyone who has the unguessable Paste URL may read that Paste; this preserves the low-friction shell workflow while keeping creation off the public internet.

ADR 0006 supersedes this decision for Paste retrieval: creation remains inside the trusted network, while a separate public host may serve read-only bearer links.
