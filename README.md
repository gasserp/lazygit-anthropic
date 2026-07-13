# lazygit-ai

A Go CLI that generates Conventional-Commits-style commit messages and GitHub PR descriptions via the Anthropic API, wired into [lazygit](https://github.com/jesseduffield/lazygit) through its Custom Commands feature.

## Why custom commands?

lazygit has no plugin system. Custom commands — defined in lazygit's config file — are the upstream-safe, maintainable integration point. This approach requires no fork, no patching, and no changes to lazygit itself. lazygit upgrades won't break it; the only coupling is lazygit's stable `customCommands` config schema.

## Requirements

- **Go** (to build from source)
- **git**
- **lazygit**
- **gh** (GitHub CLI) — optional, only needed for `lazygit-ai pr --create`
- An **Anthropic API key** — get one at <https://console.anthropic.com/>

## Install

```sh
# From source
make install

# Or directly with Go (installs as `lazygit-anthropic`; rename it to match
# the `lazygit-ai` name used in the sample lazygit config)
go install github.com/gasserp/lazygit-anthropic@latest
mv "$(go env GOPATH)/bin/lazygit-anthropic" "$(go env GOPATH)/bin/lazygit-ai"
```

Then ensure Go's bin directory is on your PATH:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

Add that line to your shell's rc file (`~/.bashrc`, `~/.zshrc`, etc.) to make it permanent.

## Configure

### Authentication

`lazygit-ai` resolves credentials in this order (first match wins):

1. `ANTHROPIC_API_KEY` env var
2. `api_key` in the config file
3. `ANTHROPIC_AUTH_TOKEN` env var
4. `auth_token` in the config file
5. an `ant auth login` profile (resolved by the SDK at call time)

#### Option A — API key (Console, pay-per-token)

```sh
export ANTHROPIC_API_KEY=sk-ant-api03-...
```

or in `~/.config/lazygit-ai/config.yml`:

```yaml
api_key: sk-ant-api03-...
model: claude-opus-4-8
```

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

### Model selection

The default model is `claude-opus-4-8`. You can override it at three levels (highest priority first):

| Method | Example |
|---|---|
| `--model` flag | `lazygit-ai commit --model claude-haiku-4-5` |
| `LAZYGIT_AI_MODEL` env var | `export LAZYGIT_AI_MODEL=claude-sonnet-4-6` |
| `model:` in config file | `model: claude-sonnet-4-6` |

For lighter workloads or lower cost, consider:

- `claude-sonnet-4-6` — faster and cheaper, still high quality
- `claude-haiku-4-5` — fastest and cheapest option

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
    command: 'lazygit-ai commit | git commit -F - --edit'
    subprocess: true
  - key: '<c-p>'
    context: 'localBranches'
    description: 'AI: create PR description'
    command: 'lazygit-ai pr --create'
    subprocess: true
```

**Keybindings:**

| Key | Panel | Action |
|---|---|---|
| `Ctrl+A` | Files | Generate a commit message for staged changes, open it in `$EDITOR` for review, then commit |
| `Ctrl+P` | Local Branches | Generate a PR title + body and open a PR via `gh pr create` |

These keybindings can be changed by editing the `key:` fields in your lazygit config.

## Usage

```sh
# Generate a commit message for staged changes (prints to stdout)
lazygit-ai commit

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

## Security

The repository is hardened against supply-chain and takeover risks: GitHub Actions are pinned to immutable commit SHAs, workflows run with a read-only `GITHUB_TOKEN`, CodeQL, Trivy, and govulncheck run on a schedule, and Renovate keeps dependencies (and the pinned Action SHAs) up to date. Branch protection is shipped as a ruleset and applied with `scripts/apply-security.sh`. See [`docs/security-hardening.md`](docs/security-hardening.md) for the full rationale and checklist, and [`SECURITY.md`](SECURITY.md) for how to report a vulnerability.

## Maintainability

The lazygit integration is config-only. lazygit upgrades will not break anything as long as lazygit continues to support the `customCommands` schema, which is a stable, documented feature. No forking, no patching, no monkey-patching — just two lines in a YAML file.
