package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolate points config resolution at an empty temp dir and clears every
// environment variable Resolve consults, so tests don't pick up the developer's
// real config file or shell environment.
func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("LAZYGIT_AI_MODEL", "")
	t.Setenv("LAZYGIT_AI_PROVIDER", "")
	t.Setenv("ANTHROPIC_CONFIG_DIR", filepath.Join(dir, "no-anthropic-profile"))
	return dir
}

// writeConfig drops a config.yml into the isolated config dir.
func writeConfig(t *testing.T, dir, body string) {
	t.Helper()
	path := filepath.Join(dir, "lazygit-ai")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "config.yml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveProviderDefault(t *testing.T) {
	isolate(t)
	cfg, err := Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != ProviderAPI {
		t.Fatalf("provider = %q, want %q", cfg.Provider, ProviderAPI)
	}
}

func TestResolveProviderPrecedence(t *testing.T) {
	dir := isolate(t)
	writeConfig(t, dir, "provider: cli\n")

	// File sets cli; env should override back to api; flag should win over env.
	t.Setenv("LAZYGIT_AI_PROVIDER", ProviderAPI)

	cfg, err := Resolve("", ProviderCLI)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != ProviderCLI {
		t.Fatalf("flag should win: provider = %q, want %q", cfg.Provider, ProviderCLI)
	}

	// Without the flag, env wins over the file.
	cfg, err = Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != ProviderAPI {
		t.Fatalf("env should win over file: provider = %q, want %q", cfg.Provider, ProviderAPI)
	}
}

func TestResolveProviderFromFile(t *testing.T) {
	dir := isolate(t)
	writeConfig(t, dir, "provider: cli\n")
	cfg, err := Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != ProviderCLI {
		t.Fatalf("provider = %q, want %q", cfg.Provider, ProviderCLI)
	}
}

func TestResolveInvalidProvider(t *testing.T) {
	isolate(t)
	if _, err := Resolve("", "bogus"); err == nil {
		t.Fatal("expected error for invalid provider, got nil")
	}
}

func TestRequireCredentialsCLIDelegates(t *testing.T) {
	isolate(t)
	// No API key, token, or profile — but the cli provider delegates auth to
	// the claude binary, so this must not error.
	cfg, err := Resolve("", ProviderCLI)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.RequireCredentials(); err != nil {
		t.Fatalf("cli provider should not require configured credentials: %v", err)
	}
}

func TestRequireCredentialsAPINeedsCredential(t *testing.T) {
	isolate(t)
	cfg, err := Resolve("", ProviderAPI)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.RequireCredentials(); err == nil {
		t.Fatal("api provider with no credentials should error")
	}
}

func TestResolveInstructionsFromFile(t *testing.T) {
	dir := isolate(t)
	writeConfig(t, dir, "instructions: |\n  Use nouns, not verbs, in scopes.\n  Never mention file names in the subject line.\n")
	cfg, err := Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "Use nouns, not verbs, in scopes.\nNever mention file names in the subject line.\n"
	if cfg.Instructions != want {
		t.Fatalf("Instructions = %q, want %q", cfg.Instructions, want)
	}
}

func TestBuildSystemPromptNoInstructions(t *testing.T) {
	cfg := &Config{}
	got := cfg.BuildSystemPrompt("base prompt")
	if got != "base prompt" {
		t.Fatalf("BuildSystemPrompt = %q, want unchanged base prompt", got)
	}
}

func TestBuildSystemPromptAppendsInstructions(t *testing.T) {
	cfg := &Config{Instructions: "Use nouns in scopes."}
	got := cfg.BuildSystemPrompt("base prompt")
	if !strings.Contains(got, "base prompt") || !strings.Contains(got, "Use nouns in scopes.") {
		t.Fatalf("BuildSystemPrompt = %q, want it to contain both base prompt and instructions", got)
	}
}

func TestRequireCredentialsAPIWithKey(t *testing.T) {
	isolate(t)
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-api03-test")
	cfg, err := Resolve("", ProviderAPI)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "sk-ant-api03-test" {
		t.Fatalf("APIKey = %q, want the env value", cfg.APIKey)
	}
	if err := cfg.RequireCredentials(); err != nil {
		t.Fatalf("api provider with key should not error: %v", err)
	}
}
