# Security Policy

## Supported Version

Security fixes are maintained on the current `master` branch of
`c3-oss/prosa-webp-widgets`.

## Application Scope

`prosa-webp-widgets` is a Go command-line renderer that produces static WebP
widgets from Prosa analytics data. It reads Prosa API credentials and optional
S3 upload credentials from the runtime environment, renders widgets through
Chromium, and writes WebP files to local storage or the configured S3 bucket.

The Docker image builds a static Go binary and runs it on
`debian:bookworm-slim` with Chromium, CA certificates, and DejaVu fonts
installed. The container entrypoint is `/usr/local/bin/prosa-webp-widgets`.

## Repository Security Controls

Pull requests to `master` run the repository CI workflow:

- Markdown and link linting plus `gitleaks` secret scanning through
  `just quality`.
- Go module tidy verification, `go vet`, race-enabled tests, `golangci-lint`,
  and builds on Ubuntu and macOS.
- `gosec` static security analysis through `just lint-sec`.
- `govulncheck` vulnerability scanning through `just lint-vuln`.

Dependabot checks Go modules, GitHub Actions, npm tooling, and Docker
dependencies weekly. Tagged releases use GoReleaser with Syft to publish
per-archive SPDX SBOMs and SHA-256 checksums.

## Reporting a Vulnerability

Do not report security vulnerabilities through public GitHub issues.

Send a private report to [security@c3.do](mailto:security@c3.do) with:

- A short description of the vulnerability.
- Steps to reproduce or validate the issue.
- The affected command, configuration, environment variable, image, or release.
- The expected impact, including any credential exposure, unauthorized access,
  data integrity issue, or supply-chain concern.
- Any safe proof of concept or relevant logs with secrets removed.

The maintainers acknowledge private reports, triage the impact, prepare fixes
on private branches when needed, and coordinate public disclosure after a fix is
available.
