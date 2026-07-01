// Package generator selects and constructs the text-generation backend
// (the direct Anthropic API or the `claude` CLI) from the resolved config.
package generator

import (
	"context"
	"fmt"

	"github.com/gasserp/lazygit-anthropic/internal/anthropic"
	"github.com/gasserp/lazygit-anthropic/internal/claudecli"
	"github.com/gasserp/lazygit-anthropic/internal/config"
)

// Generator produces text for a single system+user prompt. Both the API client
// and the CLI client satisfy it.
type Generator interface {
	Generate(ctx context.Context, system, user string, maxTokens int64) (string, error)
}

// New returns the Generator for the configured provider.
func New(cfg *config.Config) (Generator, error) {
	switch cfg.Provider {
	case config.ProviderCLI:
		if err := claudecli.Available(); err != nil {
			return nil, err
		}
		return claudecli.New(cfg), nil
	case config.ProviderAPI:
		return anthropic.New(cfg), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", cfg.Provider)
	}
}
