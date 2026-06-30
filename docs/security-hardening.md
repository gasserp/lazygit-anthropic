# Security Hardening

This repository is hardened against the failure mode where a project is taken
over because of weak GitHub settings — unprotected default branch, force-pushable
history, mutable third-party Action tags, an over-privileged `GITHUB_TOKEN`, or
leaked secrets. Most of the hardening is committed as code; a few account-level
toggles are applied with a script or in the UI.

## What is enforced as code (in this repo)

| Control | File | Why |
|---|---|---|
| Pinned GitHub Actions (full commit SHA, not tags) | `.github/workflows/*.yml` | A tag like `@v4` is mutable — an attacker who compromises the action repo can repoint it. A SHA is immutable. This is the single most important supply-chain control. |
| Least-privilege `GITHUB_TOKEN` | every workflow (`permissions: contents: read`) | A read-only default token limits blast radius if a workflow or dependency is compromised. Jobs opt into `security-events: write` only where needed. |
| `persist-credentials: false` on checkout | every workflow | Stops the token from being written to `.git/config` where later steps (or malicious code) could read it. |
| CI gate (build, vet, gofmt, tests, govulncheck) | `.github/workflows/ci.yml` | Provides the `build` status check the ruleset requires, and scans Go deps for known vulnerabilities. |
| CodeQL code scanning | `.github/workflows/codeql.yml` | Static analysis for Go on push, PR, and weekly. |
| Trivy scan (vuln + secret + misconfig) → SARIF | `.github/workflows/trivy.yml` | Filesystem scan, results surfaced in the Security tab. |
| Dependabot (gomod + github-actions) | `.github/dependabot.yml` | Weekly updates; also keeps the pinned Action SHAs current. |
| Security policy | `SECURITY.md` | Private reporting instructions. |
| Code owners | `.github/CODEOWNERS` | Enables code-owner review enforcement when turned on. |
| Branch ruleset (as data) | `.github/rulesets/main-branch-protection.json` | Applied via the script below. |

## What the script applies

Run from a machine with `gh` authenticated as a repo admin:

```bash
scripts/apply-security.sh            # defaults to gasserp/lazygit-anthropic
# or: scripts/apply-security.sh owner/repo
```

It enables:

- **Dependabot alerts** and **automated security fixes**.
- **Secret scanning**, **push protection**, non-provider patterns, and validity
  checks (push protection blocks commits that contain credentials *before* they
  land — the direct mitigation for leaked-token takeovers).
- **`GITHUB_TOKEN` default = read-only** at the repo level, and Actions cannot
  approve pull requests.
- **Actions allowlist**: GitHub-owned actions plus an explicit
  `aquasecurity/trivy-action@*` entry — nothing else can run.
- The **branch ruleset** below.

> Secret scanning and code scanning are free on public repos; on private repos
> they require GitHub Advanced Security. The script warns and continues if GHAS
> is not enabled.

## The `main` ruleset

`.github/rulesets/main-branch-protection.json` protects the default branch:

| Rule | Effect |
|---|---|
| `deletion` | The branch cannot be deleted. |
| `non_fast_forward` | **Force-pushes are blocked** — history cannot be rewritten. |
| `required_linear_history` | No merge commits; squash/rebase only. |
| `required_signatures` | Commits on `main` must be signed (merges via GitHub are signed automatically). |
| `pull_request` | Direct pushes are blocked; changes land via PR with 1 approval, stale-review dismissal, last-push approval, and resolved threads. |
| `required_status_checks` | The `build` check must pass and be up to date before merge. |

No bypass actors are configured, so the rules apply to admins too.

### Adjust before/after applying

- **Solo maintainer?** You cannot approve your own PR. Either set
  `required_approving_review_count` to `0` in the JSON, invite a collaborator, or
  add a bypass actor. Leave it at `1` once you have reviewers.
- **Add the scanning checks as required gates** once CodeQL/Trivy run green:
  add `{ "context": "analyze" }` and `{ "context": "scan" }` to
  `required_status_checks`. They are not required by default so a private repo
  without GHAS isn't blocked from merging.
- **Re-applying**: a ruleset name must be unique. Delete the old one first:
  `gh api repos/<owner>/<repo>/rulesets` to find the id, then
  `gh api --method DELETE repos/<owner>/<repo>/rulesets/<id>`.

## Manual UI steps (no stable API)

- **Settings → Code security**: enable **Private vulnerability reporting**.
- **Settings → Actions → General → Fork pull request workflows**: require
  approval for **all external contributors** (prevents drive-by PRs from running
  workflows with secrets).
- **Settings → Code security**: enable code scanning so the CodeQL/Trivy SARIF
  uploads land (auto on public repos with the committed workflows).
- **Organization**: require **two-factor authentication** for all members.

## Updating pinned Action SHAs

Don't hand-edit SHAs. Dependabot (`github-actions` ecosystem) opens PRs that bump
both the SHA and the `# vX.Y.Z` comment. Review and merge those PRs.
