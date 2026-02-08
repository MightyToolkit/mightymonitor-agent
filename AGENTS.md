# AGENTS.md

This file defines practical guidance for coding agents working in this repository.

## Project Snapshot
- Product: MightyMonitor Agent
- Stack: Go (single binary)
- Purpose: Collect host metrics and send them to the MightyMonitor API server

## Repo Map
- `cmd/mightymonitor-agent/`: CLI entrypoint (enroll/send)
- `internal/`: metric collection, buffering, API client, config

## Local Development
- Run:
  - `go run ./cmd/mightymonitor-agent`
- Tests:
  - `go test ./...`

## Release Builds (GitHub Actions)
- Linux builds (expected by install script):
  - `mightymonitor-agent-linux-amd64`
  - `mightymonitor-agent-linux-arm64`

## Change Guidelines
- Keep changes focused; avoid broad refactors unless requested.
- Preserve low-overhead collection behavior and retry safety.
- Do not commit secrets or real credentials.

## Required Checks Before Finishing
- `go test ./...`

