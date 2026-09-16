# Session handoff — 2026-09-15 — main — LLM CLI cancellation

## Goal

Support multiple already-authenticated local LLM CLIs for every Dunnit
one-shot report flow, document the setup in GUIDE, and add a way to stop an
active request.

## Completed

- Added Copilot, Claude Code, Codex, Gemini, and `llm` support behind the
  `llm_cli` config setting and Settings selector.
- Added Auto selection in the documented priority order:
  `copilot`, `claude`, `codex`, `gemini`, then `llm`.
- Removed the legacy `gh copilot` fallback; Copilot now uses the direct
  `copilot` executable.
- Routed Summarize, Standup, Status Report, Annual Review, Month/period
  Reviews, Navigator Ask AI, and both EOD draft paths through the shared
  provider runner.
- Added stdin prompt handling, bounded stderr errors, five-minute timeouts,
  provider-specific safety flags, read-only/ephemeral Codex execution, and
  `llm --no-log`.
- Added Stop buttons and cancellable contexts to every visible or background
  LLM request. Closing an active progress window also cancels its request.
- Documented provider setup, authentication ownership, privacy/logging,
  costs, context limits, prompt injection concerns, scheduling, and
  troubleshooting in `docs/GUIDE.md`.
- Added provider selection, command construction, stdin, timeout, error, and
  cancellation tests.

## Key files

- `dun/llmcli.go` — provider resolution, command construction, execution,
  timeout, and error handling.
- `dun/llmprogress.go` — cancellable request state and Stop controls.
- `dun/config.go`, `dun/settings.go` — persisted provider selection.
- `dun/{summarize,annualreview,statusreport,standup,monthreview,periodreview,eod,navigator,dailysummary,period}.go` — shared runner and UI integration.
- `dun/llmcli_test.go` — provider and cancellation coverage.
- `docs/GUIDE.md` — user-facing setup and operational guidance.
- `notes/LLM-CLI-SUPPORT-PLAN.md` — design plan and decisions.

## Decisions

- Use `auto` by default, with an explicit provider pin available in Settings.
- Auto selection falls through only for missing executables. A started CLI
  failure is reported without retrying another provider.
- Dunnit owns no credentials; each CLI owns login, model selection, quota,
  billing, and provider-side retention.
- Pass report input through stdin and run from a neutral temporary directory
  to avoid command-line exposure and accidental project context.
- Stop cancels the child process context and suppresses cancellation errors or
  partial results.

## Verification

- `go test ./...` passed.
- `make build` passed.
- `make vet` passed.
- `git diff --check` passed.
- Local help checks passed for Copilot, `gh` wrapper removal context, Codex,
  and Gemini command syntax. No real model requests were made.
- Implementation commit: `5c708c1 Add cancellable multi CLI LLM reporting`.

## Known problems / next step

- Human manual UI verification remains: run the app, start each available
  report flow, confirm the Stop button appears, confirm it terminates the
  request, and verify normal completion still displays the report.
- Authenticate and smoke-test Copilot, Claude Code, Codex, Gemini, and `llm`
  individually when desired; these checks may incur provider usage.
- Before the next commit, add this handoff note if session notes are being
  collected in the usual notes commit.

