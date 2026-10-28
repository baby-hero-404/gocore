# CLAUDE.md

Context guide for Claude Code when working in `github.com/baby-hero-404/gocore`.

## Build, Test & Lint Commands
- **Run all tests with race detector**: `make test` (or `go test ./... -race -cover`)
- **Lint and formatting check**: `make lint` (or `gofmt -l . && go vet ./...`)
- **Tidy dependencies**: `make tidy`

## Architecture & Code Rules
This repository is a Go foundation toolkit (microservice chassis).
Strict adherence to workspace rules is required:
- Read full architectural context in [AGENTS.md](AGENTS.md)
- Follow Go coding standards in [.agents/rules/coding-standards.md](.agents/rules/coding-standards.md) (No global state, no panic, all errors wrapped in `stderr`)
- Manage dependencies per [.agents/rules/dependency-management.md](.agents/rules/dependency-management.md) (Always latest versions, standard library first, zero bloat)
- Follow package hierarchy in [.agents/rules/architecture.md](.agents/rules/architecture.md) (Flat packages, bottom-up deps)
- Write table-driven unit tests per [.agents/rules/testing.md](.agents/rules/testing.md)
- Follow commit format in [.agents/rules/git-workflow.md](.agents/rules/git-workflow.md)
- Active OpenSpec: [docs/openspecs/consumer-fit-v0.3.0/](docs/openspecs/consumer-fit-v0.3.0/) (previous: `golib-adoption`, `gocore-foundation`)
