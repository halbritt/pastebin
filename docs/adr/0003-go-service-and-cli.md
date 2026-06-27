# Go Service and CLI

Pastebin is implemented in Go as a small HTTP service and companion CLI, with shared code where useful and embedded web assets for simple deployment. Go was chosen because the intended tailnet deployment benefits from static-ish binaries, low operational overhead, a strong standard HTTP stack, and straightforward packaging for both the server and command-line workflow.
