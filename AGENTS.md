# AGENTS.md — Workspace Context for AI Coding Assistants

> Repository: `github.com/baby-hero-404/gocore`  
> Project Type: Go Backend Foundation Toolkit (Microservice Chassis)

---

## 1. Project Overview

`gocore` is a Go foundation toolkit (microservice chassis) serving as a shared infrastructure baseline across services.
Goal: Any new microservice can import `gocore` and become production-ready within minutes without reinventing boilerplate infrastructure.

### 3 Core Pillars:
1. **Standardization**: Unified error handling (`stderr`), API response envelope (`stdres`), context-aware logging (`stdlog`), and IAM/Casbin authorization.
2. **Infrastructure Wrappers**: Pre-wired connectors for Postgres, Redis, Kafka, Distributed Lock, and Gin HTTP server with built-in connection pooling, retries, timeouts, and graceful shutdown.
3. **Plug & Play**: Flat top-level layout, zero heavyweight DI container, and independent per-package configurations.

---

## 2. Rules & Guidelines (Rules Index)

All AI agents writing code or refactoring in this repository **must** strictly adhere to the rules located in `.agents/rules/`:

* 📜 [coding-standards.md](.agents/rules/coding-standards.md): Idiomatic Go, no global state, no panics, unified error handling via `stderr`.
* 📦 [dependency-management.md](.agents/rules/dependency-management.md): Always use latest stable versions, zero-bloat policy, standard library first.
* 🏛️ [architecture.md](.agents/rules/architecture.md): Flat package layout, bottom-up acyclic dependency tree, independent configs.
* 🧪 [testing.md](.agents/rules/testing.md): Table-driven tests, race detection (`-race`), interface mocking, coverage target.
* 🌿 [git-workflow.md](.agents/rules/git-workflow.md): Conventional Commits format and branch/PR workflow.

---

## 3. Common Development Commands

```bash
# Run all tests with race detector and coverage
make test

# Code formatting and vet checks
make lint

# Tidy and prune Go modules
make tidy
```

---

## 4. Technical Specifications (OpenSpecs)

Architecture decisions, proposals, and task tracking are located at:
* [docs/openspecs/gocore-foundation/proposal.md](docs/openspecs/gocore-foundation/proposal.md)
* [docs/openspecs/gocore-foundation/design.md](docs/openspecs/gocore-foundation/design.md)
* [docs/openspecs/gocore-foundation/specs.md](docs/openspecs/gocore-foundation/specs.md)
* [docs/openspecs/gocore-foundation/tasks.md](docs/openspecs/gocore-foundation/tasks.md)
