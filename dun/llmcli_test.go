package dun

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveLLMCLIUsesDocumentedOrder(t *testing.T) {
	available := map[string]string{
		"claude": "/tools/claude",
		"codex":  "/tools/codex",
		"llm":    "/tools/llm",
	}
	lookup := func(name string) (string, error) {
		if path, ok := available[name]; ok {
			return path, nil
		}
		return "", os.ErrNotExist
	}

	inv, err := resolveLLMCLIWithLookup(llmCLIAuto, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if inv.provider != llmCLIClaude || inv.executable != "/tools/claude" {
		t.Fatalf("resolved %+v, want Claude", inv)
	}
}

func TestResolveLLMCLIPrefersDirectCopilot(t *testing.T) {
	lookup := func(name string) (string, error) {
		if name == "copilot" {
			return "/tools/copilot", nil
		}
		return "", os.ErrNotExist
	}

	inv, err := resolveLLMCLIWithLookup(llmCLIAuto, lookup)
	if err != nil || inv.executable != "/tools/copilot" {
		t.Fatalf("resolved %+v, err %v; want direct Copilot", inv, err)
	}

}

func TestResolveLLMCLIExplicitMissingProviderDoesNotFallback(t *testing.T) {
	_, err := resolveLLMCLIWithLookup(llmCLIClaude, func(string) (string, error) {
		return "", os.ErrNotExist
	})
	if err == nil || !strings.Contains(err.Error(), "Claude Code CLI is not installed") {
		t.Fatalf("error = %v, want explicit missing Claude error", err)
	}
}

func TestNormalizeLLMCLI(t *testing.T) {
	if got := normalizeLLMCLI(""); got != llmCLIAuto {
		t.Fatalf("empty setting = %q, want auto", got)
	}
	if got := normalizeLLMCLI("  CODEX "); got != llmCLICodex {
		t.Fatalf("normalized setting = %q, want codex", got)
	}
	if got := normalizeLLMCLI("not-a-cli"); got != llmCLIAuto {
		t.Fatalf("invalid setting = %q, want auto", got)
	}
}

func TestNormalizeLLMModel(t *testing.T) {
	if got := normalizeLLMModel("  gpt-6-luna  "); got != "gpt-6-luna" {
		t.Fatalf("normalized model = %q, want gpt-6-luna", got)
	}
}

func TestLoadConfigNormalizesLLMCLI(t *testing.T) {
	withTempDunnitDir(t)
	path := filepath.Join(DunnitDir(), "config.toml")
	if err := os.WriteFile(path, []byte("llm_cli = \"unknown\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLMCLI != llmCLIAuto {
		t.Fatalf("LLMCLI = %q, want auto", cfg.LLMCLI)
	}
}

func TestBuildLLMCLICommands(t *testing.T) {
	const instructions = "instructions"
	const ledger = "ledger with $pecial chars"
	fullPrompt := instructions + "\n\n" + ledger

	cases := []struct {
		name       string
		inv        llmCLIInvocation
		wantStdin  string
		wantArgs   []string
		wantAbsent string
	}{
		{
			name:      "copilot",
			inv:       llmCLIInvocation{provider: llmCLICopilot, executable: "/bin/copilot"},
			wantStdin: fullPrompt,
			wantArgs:  []string{"--silent", "--allow-all-tools", "--no-ask-user", "--available-tools="},
		},
		{
			name:      "claude",
			inv:       llmCLIInvocation{provider: llmCLIClaude, executable: "/bin/claude"},
			wantStdin: fullPrompt,
			wantArgs:  []string{"-p", "--no-session-persistence", "--tools", ""},
		},
		{
			name:      "codex",
			inv:       llmCLIInvocation{provider: llmCLICodex, executable: "/bin/codex"},
			wantStdin: fullPrompt,
			wantArgs:  []string{"exec", "--ephemeral", "--sandbox", "read-only", "-"},
		},
		{
			name:      "gemini",
			inv:       llmCLIInvocation{provider: llmCLIGemini, executable: "/bin/gemini"},
			wantStdin: ledger,
			wantArgs:  []string{"-p", instructions, "--output-format", "text", "--approval-mode", "plan"},
		},
		{
			name:      "llm",
			inv:       llmCLIInvocation{provider: llmCLILLM, executable: "/bin/llm"},
			wantStdin: fullPrompt,
			wantArgs:  []string{"prompt", "--no-stream", "--no-log"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args, stdin := buildLLMCLICommand(tc.inv, instructions, ledger)
			if stdin != tc.wantStdin {
				t.Fatalf("stdin = %q, want %q", stdin, tc.wantStdin)
			}
			for _, want := range tc.wantArgs {
				if !containsLLMCLIString(args, want) {
					t.Errorf("args %q missing %q", args, want)
				}
			}
		})
	}
}

func TestBuildLLMCLICommandPassesConfiguredModel(t *testing.T) {
	args, _ := buildLLMCLICommand(
		llmCLIInvocation{provider: llmCLICodex, executable: "/bin/codex", model: "gpt-6-luna"},
		"instructions", "ledger")
	if !containsLLMCLIString(args, "--model") || !containsLLMCLIString(args, "gpt-6-luna") {
		t.Fatalf("model was not passed to Codex CLI: %q", args)
	}

	args, _ = buildLLMCLICommand(
		llmCLIInvocation{provider: llmCLILLM, executable: "/bin/llm", model: "gpt-6-luna"},
		"instructions", "ledger")
	if !containsLLMCLIString(args, "-m") || !containsLLMCLIString(args, "gpt-6-luna") {
		t.Fatalf("model was not passed to llm CLI: %q", args)
	}
}

func TestRunLLMCLIUsesStdinAndTrimsOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-llm")
	if err := os.WriteFile(path, []byte("#!/bin/sh\ncat\nprintf '\\n  result  \\n'\n"), 0755); err != nil {
		t.Fatal(err)
	}

	result, err := runLLMCLI(
		llmCLIInvocation{provider: llmCLILLM, executable: path},
		"instructions", "ledger", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if result != "instructions\n\nledger\n  result" {
		t.Fatalf("result = %q", result)
	}
}

func TestRunLLMCLIReportsStderrAndDoesNotFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-llm")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho provider-auth-failed >&2\nexit 7\n"), 0755); err != nil {
		t.Fatal(err)
	}

	_, err := runLLMCLI(
		llmCLIInvocation{provider: llmCLIClaude, executable: path},
		"instructions", "ledger", time.Second)
	if err == nil || !strings.Contains(err.Error(), "provider-auth-failed") || !strings.Contains(err.Error(), "Claude Code CLI failed") {
		t.Fatalf("error = %v, want provider and stderr", err)
	}
}

func TestRunLLMCLITimesOut(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-llm")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nsleep 1\n"), 0755); err != nil {
		t.Fatal(err)
	}

	_, err := runLLMCLI(
		llmCLIInvocation{provider: llmCLICodex, executable: path},
		"instructions", "ledger", 10*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "Codex CLI timed out") {
		t.Fatalf("error = %v, want timeout", err)
	}
}

func TestRunLLMCLICanBeCanceled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-llm")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nsleep 1\n"), 0755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	_, err := runLLMCLIWithContext(ctx,
		llmCLIInvocation{provider: llmCLICopilot, executable: path},
		"instructions", "ledger", time.Second)
	if err == nil || !strings.Contains(err.Error(), "Copilot CLI canceled") {
		t.Fatalf("error = %v, want cancellation", err)
	}
}

func containsLLMCLIString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
