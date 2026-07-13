// Package claudecli implements text generation by shelling out to the `claude`
// CLI (Claude Code) in non-interactive print mode. It reuses whatever login the
// CLI already has — e.g. a Pro/Max subscription — so no credential needs to be
// configured in lazygit-ai itself.
package claudecli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gasserp/lazygit-anthropic/internal/config"
)

// binary is the CLI executable name, resolved from PATH.
const binary = "claude"

// disallowedTools keeps the run a single-shot text task: the CLI is an agent
// and would otherwise be free to read files, run commands, or hit the network.
// We hand it the diff and want only the message back.
var disallowedTools = []string{"Bash", "Edit", "Write", "Read", "WebFetch", "WebSearch"}

// Client generates text via the `claude` CLI.
type Client struct {
	model string
}

// New constructs a Client for the resolved model. It does not verify the binary
// is installed; that happens on the first Generate call so the error is
// actionable at the point of use.
func New(cfg *config.Config) *Client {
	return &Client{model: cfg.Model}
}

// Available reports whether the `claude` CLI is on PATH, returning a clear
// error if not.
func Available() error {
	if _, err := exec.LookPath(binary); err != nil {
		return fmt.Errorf("claude CLI not found on PATH: install Claude Code (https://claude.com/claude-code) and run `claude` once to log in, or use an API key instead")
	}
	return nil
}

// Generate runs `claude -p` with the system prompt appended and the user
// content piped on stdin, returning the trimmed response text.
//
// maxTokens is accepted for parity with the API client but is not enforced: the
// CLI does not expose a max-tokens knob. The prompts already constrain output
// length, so this is not a practical limitation for commit and PR messages.
func (c *Client) Generate(ctx context.Context, system, user string, maxTokens int64) (string, error) {
	if err := Available(); err != nil {
		return "", err
	}

	args := []string{
		"-p",
		"--model", c.model,
		"--output-format", "text",
		// Replace (not append) the default system prompt: Claude Code's default
		// makes the CLI behave like an interactive coding agent, which would
		// answer conversationally instead of emitting just the message.
		"--system-prompt", system,
		"--disallowed-tools", strings.Join(disallowedTools, " "),
	}

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdin = strings.NewReader(user)
	// Run outside the repo so the project's CLAUDE.md and hooks don't steer the
	// output; the diff we pipe in is the only context that should matter.
	cmd.Dir = os.TempDir()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("claude CLI failed: %s", msg)
	}

	return strings.TrimSpace(stdout.String()), nil
}
