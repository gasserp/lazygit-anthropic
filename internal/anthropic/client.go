// Package anthropic wraps the official Anthropic Go SDK with a small helper
// for the deterministic, single-shot generation this tool performs.
package anthropic

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/gasserp/lazygit-anthropic/internal/config"
)

// Client is a thin wrapper around the Anthropic SDK client plus the resolved
// model.
type Client struct {
	api   anthropic.Client
	model string
}

// New constructs a Client. The API key is taken from cfg.APIKey when set;
// otherwise the SDK resolves ANTHROPIC_API_KEY from the environment.
func New(cfg *config.Config) *Client {
	var opts []option.RequestOption
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	return &Client{
		api:   anthropic.NewClient(opts...),
		model: cfg.Model,
	}
}

// Generate sends a single user message with the given system prompt and
// returns the concatenated text of the response. Thinking is intentionally
// omitted: these are short, deterministic tasks where it adds latency and
// cost for no benefit.
func (c *Client) Generate(ctx context.Context, system, user string, maxTokens int64) (string, error) {
	resp, err := c.api.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: maxTokens,
		System: []anthropic.TextBlockParam{
			{Text: system},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(user)),
		},
	})
	if err != nil {
		return "", wrapAPIError(err)
	}

	var sb strings.Builder
	for _, block := range resp.Content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			sb.WriteString(tb.Text)
		}
	}
	return strings.TrimSpace(sb.String()), nil
}

// wrapAPIError turns an SDK error into a message that includes the HTTP status
// code when available, so callers can print something actionable to stderr.
func wrapAPIError(err error) error {
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		return fmt.Errorf("anthropic API error (status %d): %s", apiErr.StatusCode, apiErr.Error())
	}
	return err
}
