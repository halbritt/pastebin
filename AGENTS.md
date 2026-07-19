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
- Issue tracker: Plane (`Proximal` workspace), project `Pastebin` (`PASTEBIN`).
- Plane URL: `https://proximal.tail0ecc2e.ts.net:10000/`
- GitHub repo: `https://github.com/halbritt/pastebin`
- GitHub Issues: deprecated; use Plane work items for new issue tracking, claims, reviews, and issue-state changes.
- Use Plane work items for multi-agent planning, claims, submitted artifacts, reviews, and acceptance decisions.
- When updating Plane, include the repo, branch/worktree, `run_id`, `base_sha`, artifact links, verification evidence, and authority scope in the work item description or comments.
- Do not commit Plane API tokens. Local tokens and MCP env files live outside git under `~/.config/plane/`.
<!-- END PROXIMAL PLANE TRACKING -->


## Branch hygiene

Do not leave unmerged code lying around. If a task uses a branch, merge its authorized work into the intended target branch before reporting completion. If merge authority is absent, report that as a blocker instead of treating the branch as finished. Clean up branches and associated worktrees after merge.

## Parallel work: one worktree per branch

When more than one agent works this repo at once, do not share a working
directory — give each unit of work its own git worktree. A branch can be
checked out in only one worktree at a time, so concurrent edits to shared
files (Makefile, configs, generated/golden files) become impossible.

- One worktree per branch, one agent per worktree; name the dir after the branch.
- Siblings, not nested: create worktrees OUTSIDE this checkout
  (`../pastebin-wt/<branch>`), never inside it — recursive globs, file-count/hash
  gates, and IDE indexers must not scan across worktrees.
- Lifecycle: `git worktree add ../pastebin-wt/<branch> -b <branch>` /
  `git worktree list` / `git worktree remove <path>` after merge /
  `git worktree prune`. Agents with worktree isolation get this for free.
- Shared object store and build caches are fine; worktrees do NOT isolate
  ports, databases, or local services — coordinate those separately.
- Regenerate, don't merge, generated artifacts (golden files, compiled
  indexes): merge the source change, then regenerate once on the merged tree.
