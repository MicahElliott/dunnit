# Session handoff — 2026-09-16 — Go/Fyne build cache diagnosis

## Goal

Investigate why Codex repeatedly spent many minutes building Dunnit’s Go/cgo/Fyne dependencies and fell back to `/tmp`, while the user’s builds from `~/proj/dunnit` are fast.

## Completed

- Confirmed the repository root is `/home/mde/proj/dunnit`; `make build` writes the `./dunnit` binary there.
- Confirmed the source and native Fyne/cgo toolchain are healthy when the existing caches are available.
- Confirmed the sandbox denies writes to the normal Go build cache `/home/mde/.cache/go-build`, which caused prior sessions to override `GOCACHE` with a temporary path.
- Found existing warm fallback caches under `/home/mde/tmp/codex-go-cache/dunnit` and `dunzo`; no cache deletion is needed.
- Confirmed there is no persistent `GOCACHE=/tmp` override in the shell or Go user environment. The `/tmp` fallback was session-local.
- Found `~/.codex/config.toml` trusts `/home/mde/tmp` but has no entry for `/home/mde/proj/dunnit`.
- Verified with the normal home cache after one elevated run:
  - `make build`: passed in 1.7 seconds.
  - `make vet`: passed in 2.2 seconds.
  - `go test ./...`: passed in 5.4 seconds.

## Required user configuration

Add this to `~/.codex/config.toml`, preserving any existing tables, then restart Codex:

```toml
[projects."/home/mde/proj/dunnit"]
trust_level = "trusted"

[sandbox_workspace_write]
writable_roots = ["/home/mde/.cache/go-build"]
```

Leave `GOCACHE` unset so Go uses `/home/mde/.cache/go-build`. The module cache at `/home/mde/go/pkg/mod` only needs read access for the current build; grant write access separately if Codex should download or modify dependencies without approval.

## Current repository state

- Branch: `main`
- HEAD: `5608e67 EOD/SOD improvs`
- Existing uncommitted source changes: `dun/minicalendar.go`, `dun/recurring.go`; preserved as unrelated user work.
- Existing untracked handoff note from the prior session: `notes/sessions/SESSION-SAVE-2026-09-16-main-agent-shell-copilot-defaults.md`.
- `git diff --check` passed.

## Next step

After the user updates Codex permissions and starts a new session, run `go env GOCACHE` and one warm `make build` from `/home/mde/proj/dunnit`. If a cache permission error appears, stop immediately, report the path, and request access rather than retrying for minutes or redirecting to `/tmp`.

Completion timestamp: 2026-09-16 07:31:11 MST.
