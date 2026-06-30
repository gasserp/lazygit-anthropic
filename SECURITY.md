# Security Policy

## Reporting a Vulnerability

Please report security vulnerabilities **privately** — do not open a public
issue for a suspected vulnerability.

- Preferred: use GitHub **Private Vulnerability Reporting** for this repository
  (the **Report a vulnerability** button under the *Security* tab). This opens a
  private advisory visible only to maintainers.
- If that is unavailable, contact the maintainer directly.

Please include enough detail to reproduce: affected version/commit, steps,
impact, and any proof-of-concept.

We aim to acknowledge reports within a few days and to ship a fix or mitigation
as quickly as the severity warrants. Please give us a reasonable window to
remediate before any public disclosure.

## Scope

`lazygit-ai` is a local CLI that reads your git working tree and calls the
Anthropic API.

- Your **Anthropic API key** is read from the `ANTHROPIC_API_KEY` environment
  variable or a local config file. It is never written to the repository, never
  logged, and only sent to the official Anthropic API endpoint via the official
  SDK.
- Keep your config file (`~/.config/lazygit-ai/config.yml`) out of version
  control. The repository `.gitignore` excludes common secret-bearing files.
- The tool shells out to `git` and (for `pr --create`) `gh`; it does not execute
  arbitrary code from model output.

## Supported Versions

This project is pre-1.0; security fixes are applied to the latest `main`.
