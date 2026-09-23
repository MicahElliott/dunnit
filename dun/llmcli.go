package dun

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	llmCLIAuto    = "auto"
	llmCLICopilot = "copilot"
	llmCLIClaude  = "claude"
	llmCLICodex   = "codex"
	llmCLIGemini  = "gemini"
	llmCLILLM     = "llm"
)

const llmCLITimeout = 5 * time.Minute

type llmCLIInvocation struct {
	provider   string
	executable string
	model      string
}

type llmCLILookPath func(string) (string, error)

func validLLMCLI(value string) bool {
	switch value {
	case llmCLIAuto, llmCLICopilot, llmCLIClaude, llmCLICodex, llmCLIGemini, llmCLILLM:
		return true
	default:
		return false
	}
}

func normalizeLLMCLI(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if !validLLMCLI(value) {
		return llmCLIAuto
	}
	return value
}

func normalizeLLMModel(value string) string {
	return strings.TrimSpace(value)
}

func llmCLIProviderLabel(provider string) string {
	switch provider {
	case llmCLICopilot:
		return "Copilot"
	case llmCLIClaude:
		return "Claude Code"
	case llmCLICodex:
		return "Codex"
	case llmCLIGemini:
		return "Gemini"
	case llmCLILLM:
		return "llm"
	default:
		return "LLM CLI"
	}
}

func llmCLISettingOptions() []string {
	return []string{
		"Auto (recommended)",
		"Copilot",
		"Claude Code",
		"Codex",
		"Gemini",
		"llm",
	}
}

func llmCLISettingLabel(setting string) string {
	switch normalizeLLMCLI(setting) {
	case llmCLICopilot:
		return "Copilot"
	case llmCLIClaude:
		return "Claude Code"
	case llmCLICodex:
		return "Codex"
	case llmCLIGemini:
		return "Gemini"
	case llmCLILLM:
		return "llm"
	default:
		return "Auto (recommended)"
	}
}

func llmCLISettingFromLabel(label string) string {
	switch label {
	case "Copilot":
		return llmCLICopilot
	case "Claude Code":
		return llmCLIClaude
	case "Codex":
		return llmCLICodex
	case "Gemini":
		return llmCLIGemini
	case "llm":
		return llmCLILLM
	default:
		return llmCLIAuto
	}
}

func resolveLLMCLI(setting string) (llmCLIInvocation, error) {
	return resolveLLMCLIWithLookup(setting, exec.LookPath)
}

func resolveLLMCLIWithLookup(setting string, lookPath llmCLILookPath) (llmCLIInvocation, error) {
	setting = strings.ToLower(strings.TrimSpace(setting))
	if !validLLMCLI(setting) {
		return llmCLIInvocation{}, fmt.Errorf("invalid llm_cli setting %q", setting)
	}

	candidates := []struct {
		provider   string
		executable string
	}{
		{llmCLICopilot, "copilot"},
		{llmCLIClaude, "claude"},
		{llmCLICodex, "codex"},
		{llmCLIGemini, "gemini"},
		{llmCLILLM, "llm"},
	}

	if setting != llmCLIAuto {
		filtered := candidates[:0]
		for _, candidate := range candidates {
			if candidate.provider == setting {
				filtered = append(filtered, candidate)
			}
		}
		candidates = filtered
	}

	for _, candidate := range candidates {
		path, err := lookPath(candidate.executable)
		if err == nil {
			return llmCLIInvocation{provider: candidate.provider, executable: path}, nil
		}
	}

	if setting == llmCLIAuto {
		return llmCLIInvocation{}, fmt.Errorf("no supported LLM CLI found; install and authenticate Copilot, Claude Code, Codex, Gemini, or llm")
	}
	return llmCLIInvocation{}, fmt.Errorf("%s CLI is not installed or not available on PATH", llmCLIProviderLabel(setting))
}

func buildLLMCLICommand(inv llmCLIInvocation, instructions, ledgerText string) ([]string, string) {
	prompt := instructions + "\n\n" + ledgerText
	modelArgs := func(args []string) []string {
		model := normalizeLLMModel(inv.model)
		if model == "" {
			return args
		}
		return append(args, "--model", model)
	}

	switch inv.provider {
	case llmCLICopilot:
		return modelArgs([]string{
			"--silent", "--allow-all-tools", "--no-ask-user", "--no-color",
			"--no-custom-instructions", "--disable-builtin-mcps", "--available-tools=",
		}), prompt
	case llmCLIClaude:
		return modelArgs([]string{
			"-p", "--output-format", "text", "--no-session-persistence", "--bare",
			"--tools", "", "--disallowedTools", "mcp__*", "--max-turns", "1",
		}), prompt
	case llmCLICodex:
		return modelArgs([]string{
			"exec", "--ephemeral", "--skip-git-repo-check", "--sandbox", "read-only",
			"--color", "never", "--ignore-rules", "-",
		}), prompt
	case llmCLIGemini:
		// Gemini requires -p to enter headless mode. Its documented behavior
		// appends piped stdin to this prompt, so keep the ledger off argv.
		return modelArgs([]string{
			"-p", instructions, "--output-format", "text", "--approval-mode", "plan",
		}), ledgerText
	case llmCLILLM:
		args := []string{"prompt", "--no-stream", "--no-log"}
		if model := normalizeLLMModel(inv.model); model != "" {
			args = append(args, "-m", model)
		}
		return args, prompt
	default:
		return nil, ""
	}
}

func summarizeWithLLMCLIPrompt(instructions, ledgerText string) (string, error) {
	return summarizeWithLLMCLIPromptContext(context.Background(), instructions, ledgerText)
}

func summarizeWithLLMCLI(ledgerText string) (string, error) {
	return summarizeWithLLMCLIContext(context.Background(), ledgerText)
}

func summarizeWithLLMCLIContext(ctx context.Context, ledgerText string) (string, error) {
	return summarizeWithLLMCLIPromptContext(ctx,
		"Summarize this ledger of daily activity entries into a brief "+
			"impact report suitable for a standup or status update. Be concise "+
			"and group related work together.", ledgerText)
}

func summarizeWithLLMCLIPromptContext(ctx context.Context, instructions, ledgerText string) (string, error) {
	cfg := LoadConfig()
	inv, err := resolveLLMCLI(normalizeLLMCLI(cfg.LLMCLI))
	if err != nil {
		return "", err
	}
	inv.model = normalizeLLMModel(cfg.LLMModel)
	return runLLMCLIWithContext(ctx, inv, instructions, ledgerText, llmCLITimeout)
}

func runLLMCLI(inv llmCLIInvocation, instructions, ledgerText string, timeout time.Duration) (string, error) {
	return runLLMCLIWithContext(context.Background(), inv, instructions, ledgerText, timeout)
}

func runLLMCLIWithContext(parent context.Context, inv llmCLIInvocation, instructions, ledgerText string, timeout time.Duration) (string, error) {
	args, stdinText := buildLLMCLICommand(inv, instructions, ledgerText)
	if len(args) == 0 {
		return "", fmt.Errorf("unsupported LLM CLI provider %q", inv.provider)
	}

	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, inv.executable, args...)
	// Reports need only the prompt and the CLI's user-level authentication.
	// Running from a neutral directory avoids accidentally loading project
	// instructions or exposing a repository as implicit context.
	cmd.Dir = os.TempDir()
	cmd.Stdin = strings.NewReader(stdinText)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return "", fmt.Errorf("%s CLI canceled: %w", llmCLIProviderLabel(inv.provider), context.Canceled)
		}
		if ctx.Err() != nil {
			return "", fmt.Errorf("%s CLI timed out after %s", llmCLIProviderLabel(inv.provider), timeout)
		}
		detail := strings.TrimSpace(stderr.String())
		if len(detail) > 4000 {
			detail = detail[:4000] + "…"
		}
		if detail == "" {
			return "", fmt.Errorf("%s CLI failed: %w", llmCLIProviderLabel(inv.provider), err)
		}
		return "", fmt.Errorf("%s CLI failed: %w: %s", llmCLIProviderLabel(inv.provider), err, detail)
	}

	result := strings.TrimSpace(string(out))
	if result == "" {
		return "", fmt.Errorf("%s CLI returned empty output", llmCLIProviderLabel(inv.provider))
	}
	return result, nil
}
