# Project Instructions

Pastebin is a private tailnet service for turning plain text into retrievable
Paste URLs. The accepted product boundary is documented in `CONTEXT.md` and
`docs/adr/`.

## Development

- Use Go 1.23 or newer.
- Build both binaries with `make build`.
- Run the full verification gate with `go test -count=1 ./...` and
  `go vet ./...` before committing behavior changes.
- Keep `bin/` as ignored build output. Do not commit generated binaries, local
  databases, logs, or scratch artifacts.

## Runtime

- The local user service is `pastebin.service`.
- The service is exposed to the tailnet with Tailscale Serve.
- The current tailnet URL is `https://proximal.tail0ecc2e.ts.net:18080/`.
- The installed CLI lives at `~/.local/bin/pastebin`.
- The CLI default server config lives at `~/.config/pastebin/config`.

## Git Discipline

- Work on `main` for this small repository unless the user explicitly asks for a
  branch or pull request.
- Sync before editing: fetch `origin/main` and make sure local `main` is not
  behind before changing source.
- Keep the checkout clean. Do not leave uncommitted tracked changes, untracked
  deliverables, or stale local-only work at the end of a turn.
- Commit verified source/doc changes directly to `main`.
- Push finished work to `origin/main` without opening a pull request unless the
  user explicitly asks for one.
- Use stored GitHub credentials and unset `GH_TOKEN` for pushes if credential
  conflicts appear.
- After pushing, verify that `HEAD` and `origin/main` point at the same commit.

<!-- BEGIN PROXIMAL PLANE TRACKING -->
## Plane Tracking

This repository is represented in the local/private Plane workspace `Proximal`.

- Plane project: `Pastebin` (`PASTEBIN`)
- Plane URL: `https://proximal.tail0ecc2e.ts.net:10000/`
- GitHub repo: `https://github.com/halbritt/pastebin`
- Use Plane work items for multi-agent planning, claims, submitted artifacts, reviews, and acceptance decisions.
- When updating Plane, include the repo, branch/worktree, `run_id`, `base_sha`, artifact links, verification evidence, and authority scope in the work item description or comments.
- Do not commit Plane API tokens. Local tokens and MCP env files live outside git under `~/.config/plane/`.
<!-- END PROXIMAL PLANE TRACKING -->
