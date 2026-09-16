# Session save — 2026-09-16 — Codex permissions and Git policy

## Goal

Diagnose why Codex could not run the normal Dunnit completion workflow and
configure the project so ledger writes, Git metadata access, and desktop
notifications can work after restart. Record Micah's Git safety preference in
the shared `ai-rules` corpus.

## Completed

- Confirmed `DUNNIT_DIR=/home/mde/proj/mydunnits` and the repository's `.git`
  directories were read-only inside the active sandbox.
- Confirmed the Dunnit CLI writes successfully under `/tmp`, proving the
  application path works when the filesystem is writable.
- Confirmed `notify-send` is installed but DBus access is blocked by the
  current sandbox.
- Updated `/home/mde/.config/ai-rules/rules/git-commit-conventions.md` to
  prohibit routine `git push` and require an explicit request before
  `git commit`; this change is intentionally uncommitted.
- Merged the project-scoped Codex settings in `/home/mde/.codex/config.toml`:
  `sandbox_mode = "danger-full-access"` and
  `approval_policy = "on-request"`.
- Corrected a duplicate TOML project table introduced while adding those
  settings.

## Decisions

- Keep absolute paths such as `/home/mde` in Codex permission configuration;
  tilde expansion is not documented for `writable_roots`.
- Do not commit or push the shared rule change automatically.
- The project settings are expected to take effect only after restarting
  Codex.

## Verification

- `codex exec --help` loaded successfully after the TOML merge.
- `git diff --check` passed in both repositories.
- Dunnit's latest recorded build/vet checks remain passing from the preceding
  implementation session; no source code changed here.
- The completion notification was attempted but DBus remained unavailable in
  the pre-restart session.

## Known problems

- Dunnit's CLI logs ledger write failures but still exits with status `0`.
- The current Dunnit source repository remote uses SSH; the private
  `mydunnits` remote uses HTTPS.
- The active session still has the old sandbox, so post-restart verification
  remains.

## Handoff

Restart Codex, then verify `dunnit DONE '...'`, `notify-send`, and ordinary
read-only Git commands. Keep commits and pushes manual unless Micah explicitly
requests a commit.
