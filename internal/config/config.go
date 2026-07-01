// Package config resolves Anthropic credentials and the model from the
// environment and an optional YAML config file.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefaultModel is used when no model is configured anywhere.
const DefaultModel = "claude-opus-4-8"

// Provider selects how requests reach Anthropic.
const (
	// ProviderAPI talks to the Messages API directly using a resolved API key,
	// OAuth token, or `ant auth login` profile. This is the default.
	ProviderAPI = "api"
	// ProviderCLI shells out to the `claude` CLI, reusing whatever login it
	// already has (e.g. a Pro/Max subscription). No credential is configured
	// here; the CLI resolves its own auth.
	ProviderCLI = "cli"
)

// DefaultProvider is used when no provider is configured anywhere.
const DefaultProvider = ProviderAPI

// fileConfig mirrors the optional YAML config file. All keys are optional.
type fileConfig struct {
	APIKey       string `yaml:"api_key"`
	AuthToken    string `yaml:"auth_token"`
	Model        string `yaml:"model"`
	Provider     string `yaml:"provider"`
	Instructions string `yaml:"instructions"`
}

// Config holds the resolved credentials, model, and provider.
type Config struct {
	// APIKey is the resolved Anthropic API key (x-api-key auth), or empty.
	APIKey string
	// AuthToken is the resolved OAuth bearer token (Authorization: Bearer),
	// or empty. Only set when no APIKey is configured.
	AuthToken string
	// Model is the resolved model ID (never empty).
	Model string
	// Provider is the resolved backend ("api" or "cli"; never empty).
	Provider string
	// Instructions holds free-form, user-supplied text appended to every
	// system prompt (commit and PR generation alike). It's the intended place
	// to teach the model project conventions: naming rules, commit style,
	// scopes to prefer, things to never mention, etc. Empty by default.
	Instructions string
	// configPath is the path we looked for the config file at, used for
	// error messages.
	configPath string
}

// configDir returns the base config directory, honoring XDG_CONFIG_HOME and
// falling back to ~/.config.
func configDir() string {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return base
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Best effort: a relative path still produces a sensible error.
		return ".config"
	}
	return filepath.Join(home, ".config")
}

// configFilePath returns the expected lazygit-ai config file path.
func configFilePath() string {
	return filepath.Join(configDir(), "lazygit-ai", "config.yml")
}

// loadFile reads and parses the YAML config file. A missing file is not an
// error and yields a zero-valued fileConfig.
func loadFile(path string) (fileConfig, error) {
	var fc fileConfig
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fc, nil
		}
		return fc, fmt.Errorf("reading config file %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return fc, fmt.Errorf("parsing config file %s: %w", path, err)
	}
	return fc, nil
}

// Resolve loads configuration, applying the documented precedence.
//
// API key:    ANTHROPIC_API_KEY env > config file `api_key`.
// Auth token: ANTHROPIC_AUTH_TOKEN env > config file `auth_token`
//
//	(only consulted when no API key is set).
//
// Model:        modelFlag > LAZYGIT_AI_MODEL env > config file `model` > DefaultModel.
// Provider:     providerFlag > LAZYGIT_AI_PROVIDER env > config file `provider` > DefaultProvider.
// Instructions: config file `instructions` only (empty by default).
//
// When neither an API key nor an auth token is configured, the Anthropic SDK
// still resolves credentials from an `ant auth login` profile at call time.
//
// The modelFlag and providerFlag arguments are the values of the --model and
// --provider CLI flags ("" if unset).
func Resolve(modelFlag, providerFlag string) (*Config, error) {
	path := configFilePath()
	fc, err := loadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{configPath: path}

	// API key: env wins, then file. Prefer the API key over a bearer token —
	// sending both makes the API reject the request.
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		cfg.APIKey = v
	} else if fc.APIKey != "" {
		cfg.APIKey = fc.APIKey
	} else if v := os.Getenv("ANTHROPIC_AUTH_TOKEN"); v != "" {
		cfg.AuthToken = v
	} else if fc.AuthToken != "" {
		cfg.AuthToken = fc.AuthToken
	}

	// Model precedence.
	switch {
	case modelFlag != "":
		cfg.Model = modelFlag
	case os.Getenv("LAZYGIT_AI_MODEL") != "":
		cfg.Model = os.Getenv("LAZYGIT_AI_MODEL")
	case fc.Model != "":
		cfg.Model = fc.Model
	default:
		cfg.Model = DefaultModel
	}

	// Provider precedence.
	switch {
	case providerFlag != "":
		cfg.Provider = providerFlag
	case os.Getenv("LAZYGIT_AI_PROVIDER") != "":
		cfg.Provider = os.Getenv("LAZYGIT_AI_PROVIDER")
	case fc.Provider != "":
		cfg.Provider = fc.Provider
	default:
		cfg.Provider = DefaultProvider
	}
	switch cfg.Provider {
	case ProviderAPI, ProviderCLI:
	default:
		return nil, fmt.Errorf("invalid provider %q: must be %q or %q", cfg.Provider, ProviderAPI, ProviderCLI)
	}

	cfg.Instructions = fc.Instructions

	return cfg, nil
}

// antProfileExists reports whether an `ant auth login` credential store is
// present, in which case the SDK can resolve a profile at call time. It honors
// ANTHROPIC_CONFIG_DIR and falls back to <config-dir>/anthropic.
func antProfileExists() bool {
	dir := os.Getenv("ANTHROPIC_CONFIG_DIR")
	if dir == "" {
		dir = filepath.Join(configDir(), "anthropic")
	}
	if info, err := os.Stat(filepath.Join(dir, "credentials")); err == nil && info.IsDir() {
		return true
	}
	// Older/alternate layouts keep tokens directly under the config dir.
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return true
	}
	return false
}

// BuildSystemPrompt appends the configured Instructions, if any, to a base
// system prompt. This is how project-specific conventions (naming, commit
// style, scopes, things to avoid) reach the model without editing Go code.
func (c *Config) BuildSystemPrompt(base string) string {
	if strings.TrimSpace(c.Instructions) == "" {
		return base
	}
	return base + "\n\nAdditional instructions from the user, which take precedence over the above where they conflict:\n" + strings.TrimSpace(c.Instructions)
}

// HasCredentials reports whether some Anthropic credential is resolvable: an
// API key, an OAuth bearer token, or an `ant auth login` profile.
func (c *Config) HasCredentials() bool {
	return c.APIKey != "" || c.AuthToken != "" || antProfileExists()
}

// RequireCredentials returns a clear error if no credential is available.
//
// Under the CLI provider there is nothing to check here: the `claude` binary
// resolves its own auth, and its presence on PATH is validated when the
// generator is constructed.
func (c *Config) RequireCredentials() error {
	if c.Provider == ProviderCLI || c.HasCredentials() {
		return nil
	}
	return fmt.Errorf("no Anthropic credentials: set ANTHROPIC_API_KEY, add api_key or auth_token to %s, run `ant auth login`, or set provider: cli to use the claude CLI login", c.configPath)
}
