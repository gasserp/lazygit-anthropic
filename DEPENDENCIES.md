# Dependency Updates

This file documents the dependency updates made to address dashboard warnings:

## Go Dependencies

- `github.com/anthropics/anthropic-sdk-go`: Updated from v1.53.0 to v1.56.0

## GitHub Actions Workflows

- `actions/checkout`: Updated from v5.0.0 to v7.0.0 (commit 9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0)
- `actions/setup-go`: Updated from v5.4.0 to v6.5.0
- `github/codeql-action/init` and `analyze`: Using codeql-bundle-v2.25.6 tag directly
- `aquasecurity/trivy-action`: Updated from v0.36.0 to v0.36.0 (using tag directly)

All updates have been verified with build, test, vet, and format checks.