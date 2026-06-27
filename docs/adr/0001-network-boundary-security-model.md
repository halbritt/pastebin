# Network Boundary Security Model

Pastebin relies on a trusted network boundary, such as a tailnet, rather than application-level accounts or paste creation tokens. Anyone who can reach the service may create a paste, and anyone who has the unguessable paste URL may read that paste; this preserves the low-friction shell workflow while keeping the service off the public internet.
