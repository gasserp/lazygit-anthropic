# Dependency Updates

This file documents the dependency updates made to address dashboard warnings:

## Go Dependencies

- `github.com/anthropics/anthropic-sdk-go`: Updated from v1.53.0 to v1.56.0

## GitHub Actions Workflows

All actions are pinned to full commit SHAs with a version comment:

- `actions/checkout`: v5.0.0 -> v7.0.0 (`9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0`)
- `actions/setup-go`: v5.4.0 -> v6.5.0 (`924ae3a1cded613372ab5595356fb5720e22ba16`)
- `github/codeql-action/init` and `analyze`: codeql-bundle-v2.25.6 (`c35d1b164463ee62a100735382aaaa525c5d3496`)
- `aquasecurity/trivy-action`: v0.36.0 unchanged (`ed142fd0673e97e23eac54620cfb913e5ce36c25`)

All updates have been verified with build, test, vet, and format checks.
