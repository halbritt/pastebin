# Authenticate Public-Hostname Publication

Trusted Publishers may submit Public Documents through the same public hostname used for reading. `POST /` requires a dedicated bearer credential that the server checks before reading or storing the request body; anonymous reads remain available, and anonymous creation fails. The credential is stored in a restricted file and sent by the CLI only when its `--public` flag selects the dedicated `~/.config/pastebin/public` profile. Publication still writes exclusively to the separate Public Pastebin database established by ADR 0007, so this change does not expose or promote private Pastes.

The public web form also permits a Publisher to enter that credential and paste Markdown directly. The browser sends the credential in the `Authorization` header to the same `POST /` route. The page never embeds the server credential, and the form does not put the entered credential in the submitted body or URL.

Leaving publication tailnet-only did not meet the requested interface. Anonymous creation was rejected because it would let any internet client add durable content. Cloudflare Access service credentials were also rejected because they would couple the CLI to proxy-specific headers and policy. Application-level bearer authentication keeps the enforcement and failure behavior in the service and can be tested before storage.
