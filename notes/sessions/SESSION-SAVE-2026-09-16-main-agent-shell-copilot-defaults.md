# Session handoff — 2026-09-16 — main — agent-shell and Copilot defaults

## Goal

Record how to make agent-shell start with the preferred model, reasoning
effort, and Copilot permissions.

## Completed

- Confirmed the current agent-shell OpenAI adapter exposes model and session
  mode defaults, but no `agent-shell-openai-default-reasoning-effort` variable.
- Codex reasoning effort can be set with `CODEX_CONFIG`:
  `{"model_reasoning_effort":"high"}`, or globally in
  `~/.codex/config.toml` as `model_reasoning_effort = "high"`.
- Confirmed Copilot supports `COPILOT_MODEL=gpt-5.6-luna` and
  `COPILOT_ALLOW_ALL=true`.
- Confirmed Copilot does not currently document a
  `COPILOT_REASONING_EFFORT` or `COPILOT_EFFORT` environment variable.
- Copilot effort can be persisted in `~/.copilot/settings.json` with
  `"effortLevel": "high"`, or passed to agent-shell with `--effort high`.

## Recommended settings

For agent-shell's Copilot ACP server:

```elisp
(setq agent-shell-github-acp-command
      '("copilot" "--acp"
        "--model" "gpt-5.6-luna"
        "--effort" "high"
        "--allow-all"))
```

To use environment variables from agent-shell:

```elisp
(setq agent-shell-github-environment
      (agent-shell-make-environment-variables
       :inherit-env t
       "COPILOT_MODEL=gpt-5.6-luna"
       "COPILOT_ALLOW_ALL=true"))
```

## Decisions

- Use `COPILOT_MODEL` and `COPILOT_ALLOW_ALL` for shell-wide Copilot
  defaults; use the ACP command flags or settings JSON for reasoning effort.
- Keep the existing OpenAI model setting and add high effort through Codex
  configuration rather than inventing an unsupported agent-shell variable.

## Verification

- Checked the installed `agent-shell` Lisp adapter and local Copilot CLI help.
- Checked current GitHub Copilot CLI documentation for environment variables,
  model precedence, `effortLevel`, and `--allow-all`.
- No project source files were changed.

## Known problems

- `gpt-5.6-luna` must be available to the user's Copilot account/provider;
  model availability is provider/account dependent.
- The existing worktree already had unrelated uncommitted changes in
  `dun/eod.go`, `dun/eod_test.go`, `dun/report.go`, `dun/sched.go`,
  `dun/sod.go`, and `dun/sod_test.go`; they were left untouched.

## Next step

Apply the relevant Emacs Lisp or shell settings on each machine, start a new
agent-shell session, and confirm the model, effort, and permission status.

