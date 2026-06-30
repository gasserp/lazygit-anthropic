#!/usr/bin/env bash
#
# Apply repository security settings that cannot live in the repo as files:
# Dependabot alerts + security updates, secret scanning + push protection,
# least-privilege Actions defaults, a restricted Actions allowlist, and the
# branch ruleset in .github/rulesets/main-branch-protection.json.
#
# Requirements: GitHub CLI (`gh`) authenticated with admin rights on the repo
# (`gh auth login`, scope: repo + admin:org if applicable).
#
# Usage:
#   scripts/apply-security.sh [owner/repo]
# Defaults to gasserp/lazygit-anthropic.
#
# Idempotent: safe to re-run. Steps that require GitHub Advanced Security on a
# private repo will warn (not fail) if GHAS is not enabled.

set -euo pipefail

REPO="${1:-gasserp/lazygit-anthropic}"
RULESET_FILE="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/.github/rulesets/main-branch-protection.json"

echo ">> Target repository: ${REPO}"

api() { gh api -H "Accept: application/vnd.github+json" "$@"; }
soft() { "$@" || echo "   (warning: step failed — see message above; continuing)"; }

echo ">> Enabling Dependabot vulnerability alerts"
soft api --method PUT "repos/${REPO}/vulnerability-alerts" --silent

# Note: Dependabot vulnerability *alerts* stay on (detection feed). We do NOT
# enable Dependabot security *updates* — Renovate (renovate.json) raises the fix
# PRs, so enabling both would produce duplicate update PRs.

echo ">> Enabling secret scanning and push protection"
echo "   (secret scanning on a PRIVATE repo requires GitHub Advanced Security)"
soft api --method PATCH "repos/${REPO}" --input - <<'JSON'
{
  "security_and_analysis": {
    "secret_scanning": { "status": "enabled" },
    "secret_scanning_push_protection": { "status": "enabled" },
    "secret_scanning_non_provider_patterns": { "status": "enabled" },
    "secret_scanning_validity_checks": { "status": "enabled" }
  }
}
JSON

echo ">> Setting default workflow token to read-only; blocking PR approval by Actions"
soft api --method PUT "repos/${REPO}/actions/permissions/workflow" \
  -f default_workflow_permissions=read \
  -F can_approve_pull_request_reviews=false

echo ">> Restricting Actions to GitHub-owned + an explicit third-party allowlist"
soft api --method PUT "repos/${REPO}/actions/permissions" \
  -F enabled=true -f allowed_actions=selected
soft api --method PUT "repos/${REPO}/actions/permissions/selected-actions" --input - <<'JSON'
{
  "github_owned_allowed": true,
  "verified_allowed": false,
  "patterns_allowed": [
    "aquasecurity/trivy-action@*"
  ]
}
JSON

echo ">> Applying branch ruleset from ${RULESET_FILE}"
echo "   (delete an existing ruleset of the same name first if re-applying)"
soft api --method POST "repos/${REPO}/rulesets" --input "${RULESET_FILE}"

cat <<'NOTE'

>> Done with the scriptable settings.

Manual steps that have no stable API (do these in the GitHub UI):
  - Onboard Renovate: install https://github.com/apps/renovate on the repo
    (free for public repos), or self-host. Config lives in renovate.json.
  - Settings > Code security: turn on "Private vulnerability reporting".
  - Settings > Actions > General > Fork pull request workflows:
      "Require approval for all external contributors".
  - Settings > Code security: enable CodeQL/code scanning (free on public
    repos; needs GitHub Advanced Security on private repos) so the CodeQL and
    Trivy SARIF uploads land.
  - Org-level: require two-factor authentication for all members.

See docs/security-hardening.md for the full rationale and checklist.
NOTE
