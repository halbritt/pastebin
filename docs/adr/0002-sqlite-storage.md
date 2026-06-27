# SQLite Storage

Pastebin stores paste content and metadata in SQLite on local disk. SQLite fits the intended tailnet deployment because pastes are small, writes are modest, expiration metadata is first-class, and the service should remain easy to run without a separate database; flat files were rejected because metadata and cleanup would become ad hoc, and Postgres was rejected as unnecessary operational weight for v1.
