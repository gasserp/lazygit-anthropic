// Package config resolves Anthropic credentials and the model from the
// environment and an optional YAML config file.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// DefaultModel is used when no model is configured anywhere.
const DefaultModel = "claude-opus-4-8"

// fileConfig mirrors the optional YAML config file. All keys are optional.
type fileConfig struct {
	APIKey    string `yaml:"api_key"`
	AuthToken string `yaml:"auth_token"`
	Model     string `yaml:"model"`
}

// Config holds the resolved credentials and model.
type Config struct {
	// APIKey is the resolved Anthropic API key (x-api-key auth), or empty.
	APIKey string
	// AuthToken is the resolved OAuth bearer token (Authorization: Bearer),
	// or empty. Only set when no APIKey is configured.
	AuthToken string
	// Model is the resolved model ID (never empty).
	Model string
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
// Model:      modelFlag > LAZYGIT_AI_MODEL env > config file `model` > DefaultModel.
//
// When neither an API key nor an auth token is configured, the Anthropic SDK
// still resolves credentials from an `ant auth login` profile at call time.
//
// The modelFlag argument is the value of the --model CLI flag ("" if unset).
func Resolve(modelFlag string) (*Config, error) {
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

// HasCredentials reports whether some Anthropic credential is resolvable: an
// API key, an OAuth bearer token, or an `ant auth login` profile.
func (c *Config) HasCredentials() bool {
	return c.APIKey != "" || c.AuthToken != "" || antProfileExists()
}

// RequireCredentials returns a clear error if no credential is available.
func (c *Config) RequireCredentials() error {
	if c.HasCredentials() {
		return nil
	}
	return fmt.Errorf("no Anthropic credentials: set ANTHROPIC_API_KEY, add api_key or auth_token to %s, or run `ant auth login`", c.configPath)
}
