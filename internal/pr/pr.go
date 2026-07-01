// Package pr generates pull request titles and descriptions, and optionally
// creates the PR via the gh CLI.
package pr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gasserp/lazygit-anthropic/internal/config"
	"github.com/gasserp/lazygit-anthropic/internal/generator"
	"github.com/gasserp/lazygit-anthropic/internal/git"
)

const systemPrompt = `You write pull request descriptions.

Given the commit history and the diff for a branch, produce:
- A concise, descriptive PR title on the first line.
- Then a blank line.
- Then a markdown body containing a "## Summary" section with a short paragraph, followed by a "## Key changes" section with a bullet list of the most important changes.

Output ONLY the title and body. The first line is the title (no "Title:" prefix, no markdown heading). Do not wrap the output in code fences.`

// Result holds a generated PR title and body.
type Result struct {
	Title string
	Body  string
}

// ResolveBase determines the base branch. If baseFlag is non-empty it is used
// verbatim. Otherwise it resolves origin/HEAD, then falls back to "main" then
// "master", picking whichever exists as a ref.
func ResolveBase(baseFlag string) (string, error) {
	if baseFlag != "" {
		return baseFlag, nil
	}

	if head, err := git.OriginHead(); err == nil && head != "" {
		return head, nil
	}

	for _, candidate := range []string{"main", "master"} {
		if git.RefExists(candidate) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("could not determine base branch: pass --base, or ensure origin/HEAD, main, or master exists")
}

// Generate builds a PR title and body for the given base branch. cfg's
// Instructions, if set, are appended to the system prompt (see
// config.Config.BuildSystemPrompt).
func Generate(ctx context.Context, client generator.Generator, base string, cfg *config.Config) (*Result, error) {
	hasCommits, err := git.HasCommits(base)
	if err != nil {
		return nil, err
	}
	if !hasCommits {
		return nil, fmt.Errorf("no commits between %s and HEAD", base)
	}

	log, err := git.Log(base)
	if err != nil {
		return nil, err
	}

	diff, err := git.RangeDiff(base)
	if err != nil {
		return nil, err
	}

	user := fmt.Sprintf("Commits:\n%s\n\nDiff (%s...HEAD):\n%s", strings.TrimSpace(log), base, diff)

	out, err := client.Generate(ctx, cfg.BuildSystemPrompt(systemPrompt), user, 2048)
	if err != nil {
		return nil, err
	}

	return parse(out), nil
}

// parse splits generated output into a title (first line) and body (the rest,
// trimmed).
func parse(out string) *Result {
	out = strings.TrimSpace(out)
	parts := strings.SplitN(out, "\n", 2)
	title := strings.TrimSpace(parts[0])
	body := ""
	if len(parts) > 1 {
		body = strings.TrimSpace(parts[1])
	}
	return &Result{Title: title, Body: body}
}

// Create runs `gh pr create` with the generated title and body, streaming gh's
// stdout/stderr through. It returns a clear error if gh is not on PATH.
func Create(base, head string, result *Result) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found on PATH: install GitHub CLI (https://cli.github.com) to use --create")
	}

	cmd := exec.Command("gh", "pr", "create",
		"--base", base,
		"--head", head,
		"--title", result.Title,
		"--body", result.Body,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh pr create failed: %w", err)
	}
	return nil
}
