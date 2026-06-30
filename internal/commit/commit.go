// Package commit generates a commit message from the staged git diff.
package commit

import (
	"context"
	"fmt"
	"strings"

	"github.com/gasserp/lazygit-anthropic/internal/anthropic"
	"github.com/gasserp/lazygit-anthropic/internal/git"
)

const systemPrompt = `You write git commit messages in the Conventional Commits style.

Given a staged diff and a list of changed files, produce a single commit message:
- The first line is a concise subject (<= 72 characters) in the form "type(scope): summary" (scope optional), e.g. "fix(auth): handle expired tokens".
- Then a blank line.
- Then a body explaining what changed and why, wrapped at a reasonable width. Use bullet points if it helps.

Output ONLY the commit message itself. Do not include markdown code fences, backticks, quotes, or any preamble such as "Here is the commit message".`

// Generate reads the staged diff and returns an AI-generated commit message.
// It returns an error if there are no staged changes.
func Generate(ctx context.Context, client *anthropic.Client) (string, error) {
	diff, err := git.StagedDiff()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(diff) == "" {
		return "", fmt.Errorf("no staged changes")
	}

	nameStatus, err := git.StagedNameStatus()
	if err != nil {
		return "", err
	}

	user := fmt.Sprintf("Changed files:\n%s\n\nStaged diff:\n%s", strings.TrimSpace(nameStatus), diff)

	return client.Generate(ctx, systemPrompt, user, 1024)
}
