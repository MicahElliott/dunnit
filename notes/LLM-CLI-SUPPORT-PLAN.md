# Support multiple local LLM CLIs

## Goal

Allow every Dunnit one shot LLM feature to use a CLI the user has already
installed and authenticated. Dunnit will never ask for or store an API token.
The first supported providers will be Copilot, Claude Code, Codex CLI, Gemini
CLI, and `llm`.

The implementation should preserve the current Copilot behavior for existing
users, while making provider selection explicit enough that an unattended
daily or weekly report does not unexpectedly switch providers after a failure.

## Decisions

### Provider selection

Add a TOML setting named `llm_cli`, with these values:

- `auto` — choose the first available provider from the documented priority
  order.
- `copilot`
- `claude`
- `codex`
- `gemini`
- `llm`

Default to `auto`. The Settings window should expose the same choices with
human readable labels and should save the value through the existing config
path.

For `auto`, resolve availability when a request starts with `exec.LookPath`.
Use this priority order:

1. `copilot`;
2. `claude`;
3. `codex`;
4. `gemini`;
5. `llm`.

Document the order so users can pin a provider when more than one is present.

An explicitly selected provider must not silently fall through to another
provider. If it is absent, return an actionable error naming the provider and
pointing the user to GUIDE. In `auto`, fall through only when the executable
cannot be found. Once a child process has started, report its failure rather
than retrying with another provider; retrying could duplicate charges and
would make report results nondeterministic.

### Process boundary

Replace the Copilot-specific runner in `dun/summarize.go` with a provider
neutral runner. All current report and Ask AI call sites already funnel
through this path, so they should inherit provider support without separate
integration logic.

Pass the ledger content through stdin wherever the CLI supports it. This
avoids command line length limits and keeps private ledger text out of process
argument listings. Keep the short instruction portion in the provider's
supported prompt argument when that is required to enter headless mode; test
the resulting prompt composition for each adapter.

Use `exec.CommandContext` with a fixed report timeout of five minutes. Capture
stdout and stderr separately. On failure, include the provider, exit status or
signal, and a bounded stderr excerpt; do not include the full ledger in an
error message. Trim surrounding whitespace from successful output and treat
empty output as an error.

Do not run provider authentication checks from Dunnit. Availability and
authentication are different states, and probing may open a browser, consume
network requests, or otherwise have side effects. Let the selected CLI report
its own authentication error.

Every asynchronous LLM request must expose a visible **Stop** control while
it is running. The control cancels the process context, closes or clears the
progress surface, and suppresses a cancellation error or partial result. The
same cancellation path applies to background EOD drafting; its finalize-time
draft must have its own visible progress window rather than continuing
unobservably after the EOD form closes. Closing a progress window also cancels
the active request.

### Provider commands

Keep command construction in small provider-specific functions so CLI syntax
does not spread through report code. The exact flags should be checked against
the minimum supported versions during implementation.

- **Copilot:** use the direct `copilot` executable in programmatic mode, with
  clean response output, no clarification prompt, no color, and the narrowest
  tool/MCP configuration supported by the installed CLI. Preserve the
  existing noninteractive behavior represented by `--silent` and
  `--allow-all-tools`, but constrain tools if the current CLI accepts an empty
  allowlist.
- **Claude Code:** use `claude -p` with text output, disable session
  persistence, and disable built-in tools/MCP access for this read-only
  summarization task. Limit turns if the supported version provides that
  option. Do not use a permission-bypass flag when tools can be disabled.
- **Codex CLI:** use `codex exec -` so the complete prompt comes from stdin;
  use ephemeral execution, read-only sandboxing, no color, and skip the Git
  repository check because Dunnit is not asking Codex to modify a repository.
  Prefer a machine-readable output mode if its final-response event format is
  stable enough to parse; otherwise normalize the plain final output behind
  the adapter.
- **Gemini CLI:** use headless `-p` mode, pipe the ledger as stdin context,
  request plain text output, and do not enable YOLO or any automatic tool
  approval. Keep extensions and MCP access out of this invocation if the
  supported CLI provides a stable restriction for them.
- **`llm`:** use a one shot prompt, disable streaming, and pass `--no-log` so
  Dunnit does not cause the ledger to be copied into `llm`'s local prompt log
  in addition to whatever provider-side retention the user has chosen. Let
  `llm` continue to use its own configured default model and provider.

The commands should not pass tokens, model names, or provider credentials from
Dunnit. Each CLI remains responsible for its own login, model configuration,
quotas, and billing.

## Configuration and UI changes

Add an `LLMCLI string` field to the existing `Config` struct with the TOML key
`llm_cli`, and set it to `auto` in `defaultConfig`. Loading should tolerate an
empty value from an older or hand-edited config by treating it as `auto`.
Invalid values should produce a clear config error or be repaired to `auto`
using the existing config conventions; choose one behavior and cover it in
tests.

Add a provider selector to Settings near the existing reporting/automation
controls. Explain briefly that Dunnit uses the selected CLI's existing login
and never requests a token. Show `Auto` plus the five explicit providers. The
selector should retain unrelated settings when saving.

Change progress and error text that currently names Copilot to use either
the resolved provider label or a generic `LLM CLI` label. Showing the resolved
provider in progress text is useful when `auto` is selected. Errors should
identify the command that was attempted and distinguish “not installed” from
“installed but authentication/request failed.”

## GUIDE changes

Add an “AI/LLM CLI setup” section to `docs/GUIDE.md` covering:

- Dunnit's supported providers and the `Auto` priority order;
- how to select `Auto` or pin a provider in Settings;
- the fact that Dunnit does not request or store tokens;
- where to authenticate/configure each CLI, with links to the official setup
  documentation;
- `llm` provider/plugin setup as a separate user-side concern;
- the fact that report prompts contain ledger data and may be subject to the
  selected CLI/provider's own history, logging, retention, quota, and billing;
- the “no provider available” and authentication error cases;
- why scheduled reporting can be slow or incur provider usage.

Mention that Dunnit invokes `llm` with `--no-log` by default, while making clear
that this does not control logging or retention performed by the remote model
provider or by the other CLIs.

Update any nearby README or design-note wording that implies only Copilot is
supported, but keep the main setup explanation in GUIDE.

Useful upstream references for the GUIDE and implementation review:

- [Copilot programmatic use](https://docs.github.com/en/copilot/how-tos/copilot-cli/automate-copilot-cli/run-cli-programmatically)
- [Copilot CLI reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference)
- [Claude Code CLI reference](https://code.claude.com/docs/en/cli-usage)
- [Codex CLI source and exec interface](https://github.com/openai/codex/blob/main/codex-rs/exec/src/cli.rs)
- [Gemini headless mode](https://geminicli.com/docs/cli/headless/)
- [Gemini automation and stdin](https://geminicli.com/docs/cli/tutorials/automation/)
- [`llm` usage](https://llm.datasette.io/en/stable/usage.html)
- [`llm` setup and logging](https://github.com/simonw/llm/blob/main/docs/setup.md)

## Tests and verification

Add focused tests around the provider layer rather than tests that duplicate
the report formatting logic:

- config default, TOML round trip, empty value, invalid value, and Settings
  persistence behavior;
- automatic selection order, explicit
  selection, and missing-provider errors;
- no fallback after a selected process starts and exits unsuccessfully;
- exact stdin/prompt composition, including large text and special
  characters;
- provider argument construction, including the noninteractive, no-tools,
  no-session, no-stream, no-log, and read-only flags;
- stdout trimming, empty output, stderr reporting, nonzero exit status, and
  timeout/cancellation.

Use fake executables in a temporary PATH or inject lookup/command execution so
tests never call a real model or require credentials. Preserve the unrelated
existing worktree change while editing.

After implementation, run `make build`, `make vet`, and the relevant Go tests.
Manually verify one successful run with each locally available provider and
one missing-provider case. Confirm that scheduled and interactive report
flows all use the same configured provider and that no provider waits for
interactive input.

## Scope boundary and deferred candidates

**Gemini CLI should be included now.** It has a documented headless mode,
stdin support, plain/JSON output, and existing authentication that can be
prepared outside Dunnit.

**Aider should be deferred.** Its scripting mode is centered on applying edits
to files, git integration, and confirmation/commit behavior. It can answer in
ask mode, but making it a safe general report backend would require a separate
adapter contract and more tool/file-system policy than this feature needs.

**OpenCode should be deferred.** It has a useful `opencode run` mode, but its
normal architecture includes a background server, sessions, agents, and tool
execution. It is a reasonable future adapter after Dunnit has an explicit
agent/tool policy, but it is not as clean a fit for the initial text-only
provider set.

Do not add Ollama, the OpenAI REST CLI, Amazon Q, or other provider-specific
tools in this pass. Ollama is worth revisiting if local/offline models become
a product requirement; the REST CLI requires its own API key and does not
provide the “use an already authenticated agent CLI” behavior that motivates
this change.

## Risks to resolve during implementation

- CLI flags and output formats can drift. Keep adapters isolated and document
  the minimum versions or graceful degradation policy.
- Providers may choose different models and produce different report styles.
  Do not add a Dunnit model setting in this pass; use each CLI's existing
  default and revisit model selection after the provider boundary is stable.
- Ledger entries are user-controlled text and can contain instructions that
  resemble prompt injection. Disable tools, MCP, extensions, and ambient
  project instructions wherever each CLI permits it.
- A report may exceed a provider's context window, especially annual reviews.
  Preserve the existing hierarchical summarization strategy and surface a
  useful error when the selected CLI rejects the input.
- Automatic provider selection is convenient but can change cost, privacy,
  latency, and output behavior when the installed CLI set changes. The GUIDE
  and Settings selector must make the current choice visible.
