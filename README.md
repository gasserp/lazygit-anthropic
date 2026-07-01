# lazygit-ai

A Go CLI that generates Conventional-Commits-style commit messages and GitHub PR descriptions with Claude, wired into [lazygit](https://github.com/jesseduffield/lazygit) through its Custom Commands feature. It can talk to the Anthropic API directly, or shell out to the `claude` CLI and reuse its login.

## Why custom commands?

lazygit has no plugin system. Custom commands — defined in lazygit's config file — are the upstream-safe, maintainable integration point. This approach requires no fork, no patching, and no changes to lazygit itself. lazygit upgrades won't break it; the only coupling is lazygit's stable `customCommands` config schema.

## Requirements

- **Go** (to build from source)
- **git**
- **lazygit**
- **gh** (GitHub CLI) — optional, only needed for `lazygit-ai pr --create`
- **Credentials** — either an **Anthropic API key** (<https://console.anthropic.com/>) or a subscription token for the `api` provider, or the **`claude` CLI** installed and logged in for the `cli` provider (see [Provider](#provider))

## Install

```sh
# From source
make install

# Or directly with Go
go install github.com/gasserp/lazygit-anthropic/cmd/lazygit-ai@latest
```

Then ensure Go's bin directory is on your PATH:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

Add that line to your shell's rc file (`~/.bashrc`, `~/.zshrc`, etc.) to make it permanent.

## Configure

### Provider

`lazygit-ai` has two backends, selected by the `provider` setting:

| Provider      | How it authenticates                                                                                   |
| ------------- | ------------------------------------------------------------------------------------------------------ |
| `api` (default) | Calls the Anthropic Messages API directly. Needs a credential (see [Authentication](#authentication)). |
| `cli`         | Shells out to the `claude` CLI in print mode and reuses **its** login (e.g. a Pro/Max subscription). No credential is configured here — `claude` must be installed and already logged in. |

Set it via `--provider`, the `LAZYGIT_AI_PROVIDER` env var, or `provider:` in the config file (flag > env > file > default `api`). See [Option C](#option-c--use-the-claude-cli-provider-cli) for the CLI route.

### Authentication

With the default `api` provider, `lazygit-ai` resolves credentials in this order (first match wins):

1. `ANTHROPIC_API_KEY` env var
2. `api_key` in the config file
3. `ANTHROPIC_AUTH_TOKEN` env var
4. `auth_token` in the config file
5. an `ant auth login` profile (resolved by the SDK at call time)

The `cli` provider skips all of this — auth is whatever `claude` already has.

#### Option A — API key (Console, pay-per-token)

```sh
export ANTHROPIC_API_KEY=sk-ant-api03-...
```

or in `~/.config/lazygit-ai/config.yml`:

```yaml
api_key: sk-ant-api03-...
model: claude-opus-4-8
```

Set either auth_token or api_key, never both. Don't also have ANTHROPIC_API_KEY set in your environment — env vars win over the config file and would override your subscription token.

#### Option B — use your Claude Pro/Max subscription

Mint a subscription token with the Claude Code CLI and use it as the
**auth token** (not `api_key`):

```sh
claude setup-token          # requires Claude Code, logged in to your Pro/Max plan
# prints a token like: sk-ant-oat01-...
```

Then either:

```sh
export ANTHROPIC_AUTH_TOKEN=sk-ant-oat01-...
```

or in `~/.config/lazygit-ai/config.yml`:

```yaml
auth_token: sk-ant-oat01-...
model: claude-opus-4-8
```

`lazygit-ai` sends this as an OAuth bearer token with the required
`anthropic-beta: oauth-2025-04-20` header. This draws on your subscription
rather than Console API billing.

> Note: subscription tokens are intended for Claude Code; using one in
> third-party tools is your call and subject to Anthropic's terms.

Environment variables take precedence over the config file. Set **either**
`api_key`/`ANTHROPIC_API_KEY` **or** `auth_token`/`ANTHROPIC_AUTH_TOKEN`, not
both — sending both makes the API reject the request.

#### Option C — use the `claude` CLI (`provider: cli`)

If you already have [Claude Code](https://claude.com/claude-code) installed and
logged in, you can skip credential setup entirely and let `lazygit-ai` shell out
to it:

```yaml
# ~/.config/lazygit-ai/config.yml
provider: cli
model: claude-opus-4-8
```

or per-invocation:

```sh
lazygit-ai commit --provider cli
export LAZYGIT_AI_PROVIDER=cli   # or set it once in your shell
```

`lazygit-ai` runs `claude -p` non-interactively, replacing Claude Code's default
system prompt with its own and disabling the CLI's tools, so the result is the
same single-shot message you'd get from the API. Because it reuses the `claude`
login, this is the least setup — nothing to configure but the provider.

Trade-offs versus the API providers:

- **Slower.** Each call spawns the `claude` process, adding a couple of seconds
  of start-up latency on top of generation.
- **No `max_tokens`/determinism control.** The CLI doesn't expose those knobs;
  the prompts still bound output length in practice.
- **Requires `claude` on PATH**, logged in. Billing follows your `claude`
  login (e.g. subscription) rather than Console API usage.

### Model selection

The default model is `claude-opus-4-8`. You can override it at three levels (highest priority first):

| Method                     | Example                                      |
| -------------------------- | -------------------------------------------- |
| `--model` flag             | `lazygit-ai commit --model claude-haiku-4-5` |
| `LAZYGIT_AI_MODEL` env var | `export LAZYGIT_AI_MODEL=claude-sonnet-4-6`  |
| `model:` in config file    | `model: claude-sonnet-4-6`                   |

For lighter workloads or lower cost, consider:

- `claude-sonnet-4-6` — faster and cheaper, still high quality
- `claude-haiku-4-5` — fastest and cheapest option

### Custom instructions

`instructions` in the config file is free-form text appended to the system
prompt for **both** `commit` and `pr` generation. Use it to teach the model
your project's conventions — naming, scope, tone, formatting rules — without
touching any Go code.

```yaml
# ~/.config/lazygit-ai/config.yml
provider: cli
model: claude-opus-4-8
instructions: |
  Commit message conventions for this project:
  - Use these scopes only: api, db, ui, cli, ci, deps.
  - Subject line: imperative mood ("add", not "adds"/"added"), no trailing period.
  - Never mention file names or line counts in the subject; save specifics for the body.
  - If the diff touches both implementation and tests, say so explicitly in the body
    (e.g. "Adds a regression test alongside the fix").
  - Breaking changes must start the body with "BREAKING CHANGE:" on its own line.
  - Keep the body to what changed and why — skip restating the diff line by line.
```

General best practices worth encoding here:

- **Be specific, not vibes-based.** "Write good commit messages" does nothing;
  "use imperative mood, no period, 72-char subject" is something the model can
  actually follow.
- **Give it a fixed vocabulary.** Enumerate your `type`/`scope` values (e.g.
  `feat`, `fix`, `refactor`, `chore` and your handful of scopes) so messages
  stay consistent across contributors instead of drifting.
- **Call out what to leave out.** Models tend to over-explain; explicitly
  banning things like restating the diff or mentioning filenames in the
  subject keeps messages tight.
- **State precedence for edge cases**, e.g. how to title a commit that spans
  multiple scopes, or when to fall back to no scope at all.

`instructions` is appended to the prompt as-is, after the built-in Conventional
Commits / PR-description instructions, and is explicitly marked as taking
precedence if the two ever conflict — so you can override defaults (e.g. a
different style than Conventional Commits) as well as extend them.

## lazygit integration

Merge the `customCommands` block from `lazygit/config.yml` in this repo into your lazygit config file.

**Find your lazygit config:**

```sh
lazygit --print-config-dir
# Linux default:  ~/.config/lazygit/config.yml
# macOS default:  ~/Library/Application Support/lazygit/config.yml
```

**Add the custom commands:**

```yaml
customCommands:
  - key: '<c-a>'
    context: 'files'
    description: 'AI: generate commit message'
    command: 'lazygit-ai commit --commit --edit'
    subprocess: true
  - key: '<c-p>'
    context: 'localBranches'
    description: 'AI: create PR description'
    command: 'lazygit-ai pr --create'
    subprocess: true
```

**Keybindings:**

| Key      | Panel          | Action                                                                                     |
| -------- | -------------- | ------------------------------------------------------------------------------------------ |
| `Ctrl+A` | Files          | Generate a commit message for staged changes, open it in `$EDITOR` for review, then commit |
| `Ctrl+P` | Local Branches | Generate a PR title + body and open a PR via `gh pr create`                                |

These keybindings can be changed by editing the `key:` fields in your lazygit config.

## Usage

```sh
# Generate a commit message for staged changes (prints to stdout)
lazygit-ai commit

# Generate the message and create the commit directly (no shell pipe),
# opening it in $EDITOR for review first
lazygit-ai commit --commit --edit

# Generate a PR description (prints title and body to stdout)
lazygit-ai pr

# Generate a PR description against a specific base branch
lazygit-ai pr --base main

# Generate and immediately open a GitHub PR
lazygit-ai pr --create

# Use a specific model for one command
lazygit-ai commit --model claude-sonnet-4-6
lazygit-ai pr --create --model claude-haiku-4-5

# Help
lazygit-ai --help
lazygit-ai commit --help
lazygit-ai pr --help
```

`lazygit-ai commit` reads `git diff --cached` and exits with status 1 if nothing is staged.

`lazygit-ai pr` auto-detects the base branch from the origin remote's default branch, falling back to `main` or `master`. Without `--create` it prints `title\n\nbody`; with `--create` it runs `gh pr create`.

## Maintainability

The lazygit integration is config-only. lazygit upgrades will not break anything as long as lazygit continues to support the `customCommands` schema, which is a stable, documented feature. No forking, no patching, no monkey-patching — just two lines in a YAML file.
