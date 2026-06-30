// Package config resolves the Anthropic API key and model from the
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

// fileConfig mirrors the optional YAML config file. Both keys are optional.
type fileConfig struct {
	APIKey string `yaml:"api_key"`
	Model  string `yaml:"model"`
}

// Config holds the resolved API key and model.
type Config struct {
	// APIKey is the resolved Anthropic API key. It may be empty: when
	// ANTHROPIC_API_KEY is set in the environment the SDK reads it directly,
	// so we leave APIKey empty and let the SDK pick it up.
	APIKey string
	// Model is the resolved model ID (never empty).
	Model string
	// configPath is the path we looked for the config file at, used for
	// error messages.
	configPath string
}

// configFilePath returns the expected config file path, honoring
// XDG_CONFIG_HOME and falling back to ~/.config.
func configFilePath() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			// Best effort: a relative path still produces a sensible error.
			return filepath.Join(".config", "lazygit-ai", "config.yml")
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "lazygit-ai", "config.yml")
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
// API key: ANTHROPIC_API_KEY env > config file `api_key`.
// Model:   modelFlag > LAZYGIT_AI_MODEL env > config file `model` > DefaultModel.
//
// The modelFlag argument is the value of the --model CLI flag ("" if unset).
func Resolve(modelFlag string) (*Config, error) {
	path := configFilePath()
	fc, err := loadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{configPath: path}

	// API key: env wins. When the env var is set we leave APIKey empty so the
	// SDK reads it from the environment; otherwise fall back to the file.
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		cfg.APIKey = ""
	} else {
		cfg.APIKey = fc.APIKey
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

// HasAPIKey reports whether an API key is resolvable, either from the
// environment or the config file.
func (c *Config) HasAPIKey() bool {
	return os.Getenv("ANTHROPIC_API_KEY") != "" || c.APIKey != ""
}

// RequireAPIKey returns a clear error if no API key is available.
func (c *Config) RequireAPIKey() error {
	if c.HasAPIKey() {
		return nil
	}
	return fmt.Errorf("no Anthropic API key: set ANTHROPIC_API_KEY or api_key in %s", c.configPath)
}
