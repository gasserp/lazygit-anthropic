// Command lazygit-ai generates git commit messages and PR descriptions via the
// Anthropic Messages API. It is designed to be invoked from lazygit custom
// commands.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/gasserp/lazygit-anthropic/internal/commit"
	"github.com/gasserp/lazygit-anthropic/internal/config"
	"github.com/gasserp/lazygit-anthropic/internal/generator"
	"github.com/gasserp/lazygit-anthropic/internal/git"
	"github.com/gasserp/lazygit-anthropic/internal/pr"
)

const usage = `lazygit-ai - generate commit messages and PR descriptions via the Anthropic API

Usage:
  lazygit-ai commit [--commit] [--edit] [--model <id>] [--provider <name>]
  lazygit-ai pr [--base <branch>] [--create] [--model <id>] [--provider <name>]

Commands:
  commit    Generate a commit message from the staged diff and print it to stdout.
            With --commit, create the commit directly (no shell pipe needed);
            add --edit to review the message in $EDITOR first.
  pr        Generate a PR title and description for the current branch.

Global flags:
  --model <id>       Override the Anthropic model.
  --provider <name>  Backend: "api" (default) or "cli" (shell out to claude).
  -h, --help         Show this help.

Providers:
  api   Talk to the Anthropic Messages API directly (default). Needs a
        credential (see Authentication below).
  cli   Shell out to the 'claude' CLI in print mode, reusing its existing
        login (e.g. a Pro/Max subscription). No credential is configured
        here; 'claude' must be installed and logged in.

Authentication (api provider; first match wins):
  ANTHROPIC_API_KEY     API key, or api_key in the config file.
  ANTHROPIC_AUTH_TOKEN  OAuth bearer token, or auth_token in the config file.
                        Use a subscription token from 'claude setup-token'.
  ant auth login        An Anthropic CLI profile, resolved by the SDK.

Environment:
  ANTHROPIC_API_KEY     API key.
  ANTHROPIC_AUTH_TOKEN  OAuth bearer token (sk-ant-oat...).
  LAZYGIT_AI_MODEL      Default model override.
  LAZYGIT_AI_PROVIDER   Default provider override ("api" or "cli").

Config file: $XDG_CONFIG_HOME/lazygit-ai/config.yml (or ~/.config/lazygit-ai/config.yml)
  YAML keys: api_key, auth_token, model, provider, instructions

  instructions: free-form text appended to every system prompt (commit and
  pr alike), for teaching the model project conventions such as naming
  rules, commit message style, or scopes to prefer. See README.md for an
  example.
`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}

	// Handle top-level help before treating the first arg as a subcommand.
	switch args[0] {
	case "-h", "--help":
		fmt.Print(usage)
		return 0
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "commit":
		return runCommit(rest)
	case "pr":
		return runPR(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
}

func runCommit(args []string) int {
	fs := flag.NewFlagSet("commit", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	model := fs.String("model", "", "override the Anthropic model")
	provider := fs.String("provider", "", `backend: "api" (default) or "cli"`)
	doCommit := fs.Bool("commit", false, "create the commit directly instead of printing the message to stdout")
	edit := fs.Bool("edit", false, "with --commit, open the generated message in $EDITOR before committing")
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.Resolve(*model, *provider)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := cfg.RequireCredentials(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	client, err := generator.New(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	msg, err := commit.Generate(context.Background(), client, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *doCommit {
		if err := git.Commit(msg, *edit); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	}

	fmt.Println(msg)
	return 0
}

func runPR(args []string) int {
	fs := flag.NewFlagSet("pr", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	base := fs.String("base", "", "base branch (defaults to origin/HEAD, then main/master)")
	create := fs.Bool("create", false, "create the PR via gh pr create")
	model := fs.String("model", "", "override the Anthropic model")
	provider := fs.String("provider", "", `backend: "api" (default) or "cli"`)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := config.Resolve(*model, *provider)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := cfg.RequireCredentials(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	baseBranch, err := pr.ResolveBase(*base)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	client, err := generator.New(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	result, err := pr.Generate(context.Background(), client, baseBranch, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if !*create {
		fmt.Printf("%s\n\n%s\n", result.Title, result.Body)
		return 0
	}

	head, err := git.CurrentBranch()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if err := pr.Create(baseBranch, head, result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
