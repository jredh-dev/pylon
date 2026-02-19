# CONTEXT.md

## Project Overview
- **Purpose**: CLI tool for interacting with deployed infrastructure services. Starting with the cal (calendar) service, designed to grow to other services.
- **Vision**: Single binary (`xor`) that provides a unified interface to all deployed services in the jredh-dev infrastructure.
- **Status**: Initial development - v0.1.0

## Architecture Decisions
- **Tech Stack**: Go, stdlib only (no external CLI framework), consistent with ctl's approach
- **Design Patterns**: Service-namespaced subcommands (`xor cal feed create`, `xor cal event add`), HTTP client per service
- **Trade-offs**: Stdlib flag parsing is more verbose than cobra/urfave but keeps deps at zero and matches ctl conventions
- **JSON casing**: Cal API has mixed casing (snake_case requests, PascalCase list responses, lowercase create responses). Client types match this with explicit json tags.

## Current State
- **What's Working**: Full cal service client (create/list/delete feeds and events, subscribe URLs), CLI routing, config, tests passing
- **What's Not**: No integration tests against live cal service yet (cal not deployed)
- **Next Steps**: Merge PR, deploy cal service, test xor against live API, add more services as they come online

## Task Tracking
### Completed
- [x] Project scaffolding (go.mod, LICENSE, repo setup on GitHub + Gitea)
- [x] Config package (XOR_CAL_URL env var)
- [x] Cal HTTP client (all 8 endpoints)
- [x] CLI entry point with subcommand routing and flag parsing
- [x] Unit tests for cal client and config (20 tests)
- [x] .gitignore, Makefile, release workflow, CODEOWNERS, CHANGELOG

### Available
- [ ] Integration tests against live cal service
- [ ] Add more service namespaces as infrastructure grows
- [ ] Shell completion support

## Development Notes
- **Build**: `make build` (binary at `bin/xor`)
- **Install**: `make install` (copies to GOPATH/bin)
- **Test**: `make test` or `go test ./...`
- **Config**: `export XOR_CAL_URL=http://localhost:8085` (default)
- **Release**: Push a `v*` tag to trigger GitHub release workflow

## Session Log
- **[2026-02-18]**: Created xor repo, wrote cal client + CLI + config + tests. Matching ctl patterns for versioning, Makefile, release workflow, CHANGELOG format.
