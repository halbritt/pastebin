ROUGHLY RIGHT-SIZED | ON TRACK | confidence: high | biggest risk: Markdown browser rendering quietly changes the product from exact plain-text transfer into lightweight document rendering. Assumption I would least like to be wrong about: this repo is the product itself, not a test article for prompt/agent workflows; README.md:3-9, CONTEXT.md:7-89, and the five-commit history on 2026-06-27 point to an actual private tailnet utility. Evidence that would flip the verdict: if the browser page is intended to become a rendered note-sharing product, the Markdown dependency is not drift and the north-star needs to move.

## Files Reviewed

- .gitignore
- AGENTS.md
- CONTEXT.md
- Makefile
- README.md
- cmd/pastebin/main.go
- cmd/pastebin/main_test.go
- cmd/pastebind/main.go
- cmd/pastebind/main_test.go
- docs/adr/0001-network-boundary-security-model.md
- docs/adr/0002-sqlite-storage.md
- docs/adr/0003-go-service-and-cli.md
- docs/deployment/README.md
- docs/deployment/pastebin.env.example
- docs/deployment/pastebin.service
- docs/deployment/tailscale-serve.md
- go.mod
- internal/client/client.go
- internal/client/client_test.go
- internal/paste/paste.go
- internal/paste/paste_test.go
- internal/server/assets/app.js
- internal/server/assets/style.css
- internal/server/markdown.go
- internal/server/server.go
- internal/server/server_test.go
- internal/server/templates.go
- internal/server/templates/home.html
- internal/server/templates/paste.html
- internal/storage/sqlite/store.go
- internal/storage/sqlite/store_test.go
- tests/acceptance_test.go
- tests/core_contract_test.go

## Files Skipped

- go.sum: skipped as a lock/checksum file. Dependency shape was reviewed through go.mod:5-25 and read-only module metadata.
- bin/pastebin: skipped as ignored generated build output; `file` identifies it as an ELF executable, and .gitignore:1 excludes bin/.
- bin/pastebind: skipped as ignored generated build output; `file` identifies it as an ELF executable, and .gitignore:1 excludes bin/.
- Remote GitHub issues: not queried because the review prompt gates network calls without explicit authority. No local issue tracker files are present in the reviewed tree.

## Executive Summary

- stated: Pastebin is a private tailnet service for turning plain text into retrievable Paste URLs, with the trusted tailnet as the security boundary and no application accounts in v1 (README.md:3-9, docs/adr/0001-network-boundary-security-model.md:1-3).
- actual: the implementation matches that core: one Go HTTP daemon, one Go CLI, SQLite local storage, embedded web assets, bearer URLs, bounded UTF-8 content, expiration, and no user identity model (cmd/pastebind/main.go:31-83, cmd/pastebin/main.go:64-193, internal/storage/sqlite/store.go:246-266).
- mine: the project is not overbuilt in its core path. The package boundaries are simple enough to carry, and the tests are stronger than the maturity stage normally earns.
- actual: the one architectural drift is Markdown rendering in the browser view. It adds goldmark, bluemonday, and their transitive dependency tree for a feature not named in CONTEXT.md, whose language says "plain text," "Raw Paste," and exact text preservation (CONTEXT.md:3, CONTEXT.md:67-72, internal/server/markdown.go:8-22, go.mod:5-25).
- mine: that drift is not a blocker, but it is the most important product decision in the repo. Either make rendered Markdown a first-class Paste View policy or delete it and render escaped preformatted text.
- actual: expiration is enforced on reads and cleanup exists in the store, but no runtime path calls CleanupExpired, so expired content stops being retrievable but remains in SQLite until an external caller or future job removes it (internal/storage/sqlite/store.go:129-165, internal/paste/paste.go:54-58).
- mine: for a single-user tailnet service, lazy retention is acceptable for now only because there is a max TTL and small max size. If the service handles private operational notes, add a tiny startup/ticker cleanup, not a scheduler framework.
- actual: history is short and coherent: one initial build commit plus focused follow-ups for Markdown view, CLI config fallback, agent workflow docs, and CLI help, all on 2026-06-27. Churn is concentrated in cmd/pastebin and server presentation, which matches the user-facing shell/browser workflow.
- mine: the next month should be subtraction and hardening: settle Markdown, add runtime cleanup, document backup/restore, and keep the no-account tailnet boundary intact.

## What Pastebin Is Trying To Be

stated: Pastebin's domain is deliberately narrow. CONTEXT.md defines Pastebin as a private work/team service that stores a Paste and gives back a Paste URL reachable only inside a trusted network boundary (CONTEXT.md:7-9). A Paste is the shared text itself, not an upload, document, blob, or message (CONTEXT.md:11-13). Creator is explicitly not account identity (CONTEXT.md:15-17). Immutable Paste forbids edits and revisions (CONTEXT.md:19-21). Bearer Link says URL possession grants read access without login, token, or account check (CONTEXT.md:79-81). Trusted Network Boundary says reachability, not application auth, limits creation (CONTEXT.md:87-89).

actual: those principles are encoded directly. There is no account table in the SQLite schema, no creator field in paste.Paste, no owner checks in server routes, and no auth middleware in mountRoutes (internal/paste/paste.go:40-58, internal/storage/sqlite/store.go:254-264, internal/server/server.go:64-71). The CLI creates from stdin or one file and retrieves by URL/code, which matches "moving plain text between machines, shells, and browsers" rather than document management (cmd/pastebin/main.go:21-40, cmd/pastebin/main.go:224-229).

mine: the product boundary is good. The trusted tailnet model is not a placeholder for real auth; it is the point. Adding login, per-paste ACLs, edit history, or public hosting would fight the stated utility. The architecture should optimize for the shortest path from text bytes to an unguessable URL and back to the same text bytes.

stated: SQLite is chosen because pastes are small, writes are modest, expiration metadata matters, and a separate database is too much for v1 (docs/adr/0002-sqlite-storage.md:1-3). Go is chosen for static-ish binaries, low operational overhead, standard HTTP, and straightforward packaging (docs/adr/0003-go-service-and-cli.md:1-3).

actual: the binaries and deployment docs follow that: Makefile builds bin/pastebind and bin/pastebin (Makefile:3-5), the systemd unit runs /usr/local/bin/pastebind as a pastebin user with StateDirectory and ReadWritePaths limited to /var/lib/pastebin (docs/deployment/pastebin.service:6-23), and Tailscale Serve is the documented exposure layer (docs/deployment/README.md:55-71).

mine: the only incompatible goal is implicit, not documented: Markdown rendering turns the browser view into a formatted document surface. CONTEXT.md says a Paste View may add chrome but does not redefine Raw Paste (CONTEXT.md:71-73); internal/server/templates.go:56-75 renders stored content as sanitized Markdown. That is a product fork. It can be correct, but it must stop being accidental.

## Current Architecture

stated: the project has an HTTP service, companion CLI, shared code where useful, embedded web assets, and local SQLite storage (README.md:25-29, docs/adr/0003-go-service-and-cli.md:1-3).

actual: the component graph is small:

- cmd/pastebind parses env/flags, opens SQLite, constructs the HTTP server, starts net/http, and handles SIGINT/SIGTERM shutdown (cmd/pastebind/main.go:31-83).
- internal/server owns routes, request decoding, response negotiation, URL receipts, template rendering, and asset embedding (internal/server/server.go:44-71, internal/server/templates.go:14-18).
- internal/storage/sqlite owns schema creation, create/get/cleanup, retry on code collision, and normalized UTC timestamps (internal/storage/sqlite/store.go:48-68, internal/storage/sqlite/store.go:102-165, internal/storage/sqlite/store.go:246-287).
- internal/paste owns the domain primitives: Paste, Store interface, content validation, TTL parsing, code normalization, and random code generation (internal/paste/paste.go:15-58, internal/paste/paste.go:60-147).
- internal/client owns HTTP request/response mechanics for create/get, URL/code resolution, and response errors (internal/client/client.go:27-63, internal/client/client.go:83-143, internal/client/client.go:164-241).
- cmd/pastebin owns CLI command parsing, server config lookup, stdin/file reads, and stdout/stderr behavior (cmd/pastebin/main.go:64-193, cmd/pastebin/main.go:231-305).

actual: runtime state is one SQLite database. The schema has code, content, created_at, expires_at, and size with checks for non-empty lowercase code, length consistency, and expiry after creation (internal/storage/sqlite/store.go:251-266). Open sets MaxOpenConns(1), busy_timeout, and WAL, which is a conservative SQLite posture for a single-process service (internal/storage/sqlite/store.go:57-63, internal/storage/sqlite/store.go:251-253).

actual: public surfaces are a browser form at GET /, creation at POST /, browser paste view at GET /p/{code}, raw paste at GET /raw/{code}, JSON negotiation on create/view, and GET /healthz (README.md:46-54, internal/server/server.go:64-70, internal/server/server.go:188-220). The CLI exposes create, get, help, and version (cmd/pastebin/main.go:21-58, cmd/pastebin/main.go:64-83).

actual: test posture is strong for this size. There are 8 test files and 1,707 lines of tests in a 4,442-line tracked tree. Tests cover content validation and code generation (internal/paste/paste_test.go:12-162), SQLite round trip, expiration, cleanup, and collision retry (internal/storage/sqlite/store_test.go:16-199), server routing and error mapping including Markdown sanitization (internal/server/server_test.go:37-345), client URL resolution and request shape (internal/client/client_test.go:14-162), CLI behavior (cmd/pastebin/main_test.go:15-244), server config parsing (cmd/pastebind/main_test.go:11-65), and compiled binary acceptance tests that start a real server and drive the CLI (tests/acceptance_test.go:42-135).

actual: release posture is minimal and appropriate. Makefile has build/test/clean only (Makefile:1-11). README says use go test ./... and make build (README.md:18-29). AGENTS.md raises the local gate to go test -count=1 ./... and go vet ./... before behavior commits (AGENTS.md:7-14). There is no CI metadata, release packaging, migration tool, or version injection beyond cmd/pastebin/main.go:19.

mine: this is the right amount of architecture for a small tailnet service. It has boundaries where failures and tests need them, not ceremony. The one exception is presentation format: Markdown adds dependency and security surface to a product whose explicit core is exact text.

## Value-Vs-Complexity Ledger

| Component | What it does | Value it delivers | Complexity it carries | Verdict | If CUT/SIMPLIFY: what breaks, what replaces it |
| --- | --- | --- | --- | --- | --- |
| internal/paste | Domain types, validation, TTL policy, code generation, Store interface | Central contract for exact text, bounded size, TTL, unguessable codes | 149 LOC, no external deps, one interface with one implementation but useful for server tests | KEEP | Cutting it would smear validation across CLI/server/storage; keep it. |
| internal/storage/sqlite | SQLite schema, create/get/cleanup, collision retry | Durable local storage with TTL metadata and no external DB | 288 LOC, modernc.org/sqlite direct dep plus large transitive tree, SQL schema | KEEP | Replacing with flat files would make metadata and cleanup ad hoc, exactly what ADR 0002 rejected. |
| internal/server routing/API | HTTP routes, body limits, JSON/text negotiation, status mapping, URL receipts | Gives browser, CLI, and curl users one small protocol | 344 LOC, standard library, several content-type branches | KEEP | Simplifying negotiation too far would make CLI/browser worse; current branch count is acceptable. |
| internal/server templates/assets | Browser create form, paste view, copy buttons, embedded CSS/JS | Useful manual workflow from a browser inside the tailnet | 130 Go LOC, 503 asset/template LOC, copy/fetch JS behavior | SIMPLIFY | Keep browser form and copy buttons; remove any formatting behavior not part of the product decision. |
| internal/server Markdown rendering | Converts stored text to sanitized HTML for /p/{code} | Better visual reading for Markdown notes, if that is a real use case | 23 LOC but 2 direct deps and many transitive deps; security policy surface; semantics differ from exact text | SIMPLIFY or CUT | If cut, /p/{code} shows escaped preformatted text and raw/copy still work. If kept, document "Paste View renders Markdown" as a deliberate policy. |
| internal/client | HTTP client and URL/code resolution for CLI | Keeps CLI command code small and tests transport behavior separately | 289 LOC, standard library, URL path resolution cases | KEEP | Could merge into cmd/pastebin, but tests would become noisier; boundary earns its keep. |
| cmd/pastebin | CLI create/get/help/version and config fallback | Primary shell workflow; supports bare `pastebin file` through config | 306 LOC plus 244 LOC tests | KEEP | Help text could move to docs later, but the command surface is not overbuilt. |
| cmd/pastebind | Server config, SQLite open, net/http lifecycle | One daemon with env/flag config and graceful shutdown | 175 LOC plus 65 LOC tests | KEEP | No framework needed; current code is simpler than introducing one. |
| docs/deployment | systemd, env, Tailscale Serve deployment instructions | Makes the homelab runtime reproducible | 147 LOC across docs and unit/env files | KEEP | Docs should be reconciled with AGENTS.md user-service reality, not deleted. |
| tests/acceptance | Builds binaries, starts server, drives CLI/HTTP | Catches integration failures unit tests miss | 397 LOC, subprocess/local HTTP execution during tests | KEEP | For this repo, the acceptance harness is high value because the product is the CLI-service loop. |
| AGENTS.md | Repo-specific agent workflow and runtime facts | Keeps future code changes disciplined in a small repo | 37 LOC | KEEP | It is operational metadata; do not let it become product docs. |

Roll-up: 11 meaningful components/subsystems; I would cut or simplify 2, both in the browser presentation path. The tracked tree is 4,442 lines including go.sum; the reviewed non-lockfile tree is 4,383 lines: 1,704 production Go lines, 1,707 test lines, 503 web asset/template lines, and 469 docs/config lines. The total direct dependency count is 3: modernc.org/sqlite for storage, goldmark for Markdown parsing, and bluemonday for sanitization (go.mod:5-9). The worst over-engineering is sanitized Markdown rendering: not because 23 LOC is large, but because it imports a document-rendering concept and dependency tree into a product whose domain model says plain text. It likely exists from a reasonable UX impulse, not resume-driven architecture, but it is still a semantic expansion.

## Inverse Check: What Is Actually Missing

1. actual: CleanupExpired is implemented and tested, but unused by the daemon (internal/paste/paste.go:54-58, internal/storage/sqlite/store.go:150-165, cmd/pastebind/main.go:31-83). mine: this is the only under-built runtime behavior. Expiration is enforced on reads, so it is not a user-visible correctness blocker, but stale private text remains on disk longer than the retention policy implies.

2. actual: deployment docs describe a system service at /etc/systemd/system with a pastebin runtime user and /var/lib/pastebin (docs/deployment/README.md:16-45), while AGENTS.md says the local service is pastebin.service, exposed with Tailscale Serve, installed CLI is ~/.local/bin/pastebin, and config lives at ~/.config/pastebin/config (AGENTS.md:16-22). mine: this is not architectural breakage, but the repo has two operating models: generic host install and this maintainer's user-service install. Put the user-service facts in a deployment note or mark AGENTS.md as local operator guidance only.

3. actual: there is no backup/restore note for the SQLite file. README names the production data path (README.md:131-133), and deployment docs create /var/lib/pastebin (docs/deployment/README.md:16-20), but no backup command or restore warning exists. mine: for local-first SQLite, a three-command backup/restore section is worth more than any new feature.

Everything else people might ask for is deliberately absent: auth, audit logs, admin UI, edit/delete, multi-node storage, hosted deployment, and user identity. Those omissions are correct under the stated boundary.

## Strengths Worth Preserving

stated: no application accounts in v1, tailnet reachability is the security boundary, and bearer URLs are sufficient to read (README.md:6-9, CONTEXT.md:79-89). actual: the code has no identity concept anywhere in the Paste schema or route path. mine: preserve this. The moment user identity lands, this becomes a different service with different abuse, recovery, and UI problems.

actual: the domain package is small and concrete. ValidateContent rejects empty, oversized, invalid UTF-8, and control characters except tab/newline/carriage return without normalizing content (internal/paste/paste.go:60-80). Tests explicitly assert non-mutation and exact whitespace preservation (internal/paste/paste_test.go:42-52, tests/core_contract_test.go:12-21). mine: this is the right center of gravity. Exact text preservation should remain the product invariant.

actual: SQLite is opened with one connection, busy timeout, WAL, primary-key code collision protection, and indexed expiration (internal/storage/sqlite/store.go:57-63, internal/storage/sqlite/store.go:220-266). mine: this is pragmatic storage, not overbuild. It gives durability and metadata without dragging in a database service.

actual: the CLI can create from stdin or one file, retrieve raw by URL/code, and infer server from a full URL when config is absent (cmd/pastebin/main.go:116-124, cmd/pastebin/main.go:160-193, cmd/pastebin/main.go:289-301). Tests lock stdout to exact URL/raw content behavior (cmd/pastebin/main_test.go:45-79, cmd/pastebin/main_test.go:163-233). mine: stdout discipline is critical for shell workflows; keep it tight.

actual: the acceptance tests compile both binaries, start pastebind on 127.0.0.1:0, create pastes through the CLI, retrieve by URL/code/raw URL, check JSON receipt, reject empty/invalid UTF-8, and verify expired vs unknown statuses (tests/acceptance_test.go:42-135, tests/acceptance_test.go:137-232). mine: this test harness is not too much. It protects the actual product loop.

## Concerns, Ranked

serious: Markdown rendering is an undocumented semantic expansion of Paste View.

- stated: Pastebin is for plain text, Raw Paste preserves exact submitted UTF-8 text bytes, and Paste View may add chrome but not redefine Raw Paste (CONTEXT.md:3, CONTEXT.md:67-73).
- actual: GET /p/{code} renders found.Content through goldmark and bluemonday before inserting template.HTML into the page (internal/server/templates.go:56-75, internal/server/markdown.go:17-22). The test expects Markdown HTML, sanitized script removal, and unsafe link removal (internal/server/server_test.go:214-248).
- mine: this is the only concern that might change the product identity. The safe version of a browser Paste View for "plain text" is escaped text in a `<pre>`, not interpreted Markdown. If rendered Markdown is truly desired, update CONTEXT.md and README.md to say Paste View renders Markdown while Raw Paste remains exact.

serious: expired paste cleanup is implemented but not scheduled.

- stated: expiration means a Paste is no longer retrievable regardless of whether its stored record has already been cleaned up (CONTEXT.md:59-61).
- actual: Get returns ErrExpired when now is not before ExpiresAt (internal/storage/sqlite/store.go:129-145), and CleanupExpired deletes expired rows (internal/storage/sqlite/store.go:150-165), but cmd/pastebind never invokes cleanup during startup or runtime (cmd/pastebind/main.go:31-83).
- mine: the docs permit delayed cleanup, so this is not a correctness bug. It is a retention gap. A local pastebin often carries secrets/config snippets; "expired but still in the DB" should be intentionally bounded by a small in-process cleanup loop.

smell: server-side TTL options are duplicated rather than derived from the domain policy.

- actual: paste.AllowedTTLs defines 1h, 1d, 7d, 30d (internal/paste/paste.go:25-30), while templates.go repeats the same four values for the select element (internal/server/templates.go:104-110), and README repeats them again (README.md:67).
- mine: this is harmless today and visible in tests, but it is a classic tiny-drift point. The browser options should be generated from one ordered list in internal/paste, not from a map plus a hand-written template slice.

smell: deployment docs and local runtime guidance disagree in shape.

- actual: docs/deployment targets system-wide /usr/local/bin, /etc/pastebin, /etc/systemd/system, and /var/lib/pastebin (docs/deployment/README.md:16-45). AGENTS.md says the local service is pastebin.service, current tailnet URL is https://proximal.tail0ecc2e.ts.net:18080/, installed CLI is ~/.local/bin/pastebin, and config is ~/.config/pastebin/config (AGENTS.md:16-22).
- mine: both can be valid, but they are not presented as two profiles. The repo should label one "generic system install" and one "this host/user service install" before future agents copy the wrong commands.

smell: release/version posture is still placeholder.

- actual: cmd/pastebin has `var version = "dev"` and prints it (cmd/pastebin/main.go:19, cmd/pastebin/main.go:71-80), but Makefile does not inject a version (Makefile:3-5), and no tag/release history exists in the reviewed git state.
- mine: this is fine for day-one utility, but a version command that always prints "dev" is not much of a version command. Either inject commit/date in make build or delete the command until releases exist.

## On Track?

stated: the project wants a private tailnet paste URL workflow, not a public paste site or file transfer service (CONTEXT.md:7-13, README.md:3-9). actual: history supports that. The initial commit, `1706a2a Build private tailnet pastebin`, created the domain docs, Go service, CLI, SQLite store, web form, deployment docs, and tests. Follow-up commits were focused: `ce95007 Render markdown paste views safely`, `7c25bd9 Load CLI default server from config`, `5905211 Document agent git workflow`, and `2519e8b Add pastebin CLI help`, all on 2026-06-27.

actual: most churn after the initial commit is in cmd/pastebin/main.go and cmd/pastebin/main_test.go, each touched three times; server template/style/test files and go.mod/go.sum were touched twice. That means the repo has been refining the shell workflow and browser presentation rather than inventing new subsystems.

mine: on track, with one caution. CLI config fallback and help are directly aligned with the product; Markdown rendering is adjacent but not yet justified by the stated model. There is no evidence of abandoned branches, merge commits, tags, or half-merged directories in the local git view. The repo is young, so trajectory confidence comes from coherence more than time.

## North-Star Architecture

mine: greenfield, I would build almost exactly this:

- One Go module named pastebin.
- One paste domain package with validation, TTL policy, and code generation.
- One SQLite store with schema-on-open, WAL, single writer posture, and collision retry.
- One net/http server with five routes: home, create, paste view, raw, health.
- One CLI binary for create/get, using stdout only for machine-consumable URL/content.
- Embedded minimal HTML/CSS/JS for browser convenience.
- systemd user or system service plus Tailscale Serve as the network boundary.

The delta from current code is small. I would change the browser Paste View to an exact escaped text view unless Markdown is now explicitly part of the product. I would add an in-process cleanup loop in pastebind, probably once on startup and then every hour, using the existing Store method. I would add backup/restore docs. I would not add a framework, a migration tool, auth middleware, a job runner, an admin console, or a second storage backend.

mine: the current code is close enough to the north-star that architectural rewrites would be self-harm. The right motion is decision cleanup, not restructuring.

## Future Direction

Next month: stop expanding surface area. Decide browser rendering semantics, wire cleanup, and document data backup. Those are hours-to-day tasks with high operational payoff.

Next quarter: add only usage-proven shell ergonomics. Examples: `pastebin delete <code>` only if the maintainer actually needs early revocation despite bearer-link semantics; `pastebin open <code>` only if browser workflows dominate; a `pastebin health` command only if service diagnosis is recurring. Each of those forecloses some simplicity, so require observed friction first.

One year: if this becomes team infrastructure rather than a personal tailnet helper, the likely bet is not accounts. It is operational polish: install/update script, documented restore drill, optional max-retention enforcement, and maybe request logging that excludes paste content. If the service needs public sharing, fork the product boundary instead of slowly turning this one public.

## Recommended Changes

| priority | change | rationale | benefit | risk | rough effort |
| --- | --- | --- | --- | --- | --- |
| P0 | Decide Markdown: either remove internal/server/markdown.go and render escaped preformatted text in paste.html, or update CONTEXT.md/README.md to state that Paste View renders Markdown while Raw Paste remains exact. | Current behavior conflicts with the plain-text domain language. | Preserves product identity and prevents dependency/security surface from being accidental. | Removing Markdown may disappoint browser readers; keeping it requires owning sanitizer behavior. | hours |
| P1 | Call CleanupExpired from pastebind on startup and on a simple ticker, shutting it down with the existing signal context. | Expiration blocks retrieval but does not bound on-disk retention. | Aligns data retention with user expectations without adding infrastructure. | Bad interval/error logging could be noisy; keep it simple. | hours |
| P1 | Generate browser expiration options from an ordered domain list in internal/paste and reuse it in tests/docs where practical. | TTL values are repeated in paste.AllowedTTLs, templates.go, README, and tests. | Removes a small policy drift point. | Over-abstracting this would be worse than the duplication; keep an ordered slice. | hours |
| P1 | Add a deployment subsection for backup/restore of the SQLite database. | Local SQLite is the durable product state. | Gives the maintainer a recovery path before data matters. | None beyond keeping commands accurate. | hours |
| P2 | Split deployment docs into "generic system service" and "local user-service on this host" profiles, or explicitly label AGENTS.md as local operator facts. | Current docs and AGENTS.md describe different install shapes. | Prevents future agents/operators from mixing paths and service models. | Minor doc churn. | hours |
| P2 | Inject CLI version at build time or remove the version command until release tags exist. | `pastebin version` currently prints "dev" by default. | Makes the command useful for installed-binary diagnosis. | Build command becomes slightly longer if using -ldflags. | hours |

## Functionality I Would Add

| priority | change | rationale | benefit | risk | rough effort |
| --- | --- | --- | --- | --- | --- |
| P1 | Add `pastebin doctor` or `pastebin health` only if local service diagnosis recurs; it should hit /healthz and print the configured server. | The service already has /healthz and config fallback. | Helps the sole operator debug config without curl. | Adds CLI surface that may not be needed. | hours |
| P2 | Add optional early deletion by code only if bearer URLs are used for sensitive snippets that need revocation before TTL. | Expiration max is 30 days; operational secrets may need shorter manual revocation. | Gives a recovery action after accidental sharing. | Delete authorization is awkward without auth; on a tailnet, possession/config may be the only gate. | days |
| P2 | Add a `--expires` default in the CLI config only if users repeatedly want non-7-day defaults. | Server default already exists; CLI-side default could make shell use easier. | Reduces repeated flags. | More config semantics in a tool that is currently simple. | hours |

## Open Questions

- Is Markdown rendering an intentional product decision or just a nicer browser view that slipped in after the plain-text boundary was written?
- Is the active deployment meant to be the generic system service in docs/deployment, the user-service shape captured in AGENTS.md, or both as supported profiles?
- Are expired pastes expected to be physically removed promptly because content may include secrets, or is "not retrievable through the URL" enough for the current threat model?
- Should `pastebin version` identify installed builds for operator support, or is it only a placeholder?
- Are there remote GitHub issues or private notes that change the roadmap? I did not query network-backed issue state under the prompt's execution gate.
