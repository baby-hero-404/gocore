# Proposal: Adopt go-library ideas into gocore (quality gates, mocks, pagination, gRPC-ready layout)

## Problem
`go-library` (internal predecessor, `gitlab.nexsoft.duckdns.org/icom/server/go-library`) holds several ideas gocore lacks: generated mocks, cursor/offset pagination, and a gRPC server. It also shows how a shared library decays without mechanical enforcement: 35 library files call `log.Fatal`/`os.Exit`, 217 raw `fmt.Errorf`, 36 test files for 648 sources, no `doc.go`, no CI, secrets committed in `docker-compose.yml`.

gocore's architecture is sound, but most of its rules are `[review]` only, there is no CI or `golangci-lint` config, `make lint` cannot fail on formatting, and coverage has no floor. As gocore grows (often by AI agents), it can drift toward the same state.

## Goal
Take the useful ideas from go-library, not its code, and make gocore's rules mechanically enforced so it can be extended safely.

## Scope (decided with the user, 2026-10-28)
| Item | Decision |
|---|---|
| Mockery (generated mocks) | **In.** Installed as a Go tool dependency (`go get -tool`), mocks generated into `<pkg>/<pkg>mock`. |
| Pagination (cursor + offset) | **In.** Redesigned, not copied: fixes the multi-column cursor bug, HMAC-signed single-token cursor. |
| In-memory lock | Out. |
| gRPC / ConnectRPC server | **Out for now**, but the architecture is prepared (rank slot, vendor owner, transport-neutral helpers, ADR). |
| Utilities (crypto, guid, snowflake, timer, otp) | Out for now. |
| Domain packages (blockchain, payment, email, sign-in, socket, http3) | Out permanently (non-goal). |
| Agent/quality rules | **In.** New rule IDs, archtest checks, golangci-lint, CI (GitHub Actions), coverage baseline. |
| Dependencies | Always latest stable versions (existing DEP-1). |

## Success
- `make verify` fails on: unformatted code, lint findings, archtest violations, stale mocks, coverage dropping below baseline.
- GitHub Actions runs `make verify` on every push and PR.
- `database/pagination` returns correct, gap-free and duplicate-free pages for 1..N sort columns with mixed ASC/DESC, proven by tests against SQLite.
- A future `grpcserver` package can be added without changing any existing package's public API.

## Non-goals
- Implementing a gRPC server.
- Backward pagination (`prev` cursor). Forward-only for now; the cursor format is versioned so it can be added later.
- Raising the 7 under-covered packages to 80% in this change (baseline ratchet only).

## Trade-offs
- **Mockery via `go tool`**: version-locked in `go.mod`, but mockery's dependency tree enters gocore's module graph (consumers do not compile it; module graph pruning keeps it out of their builds). Accepted by the user.
- **Generated mocks replace hand-written ones**: mock API changes for consumers (`.EXPECT()` style). Accepted because gocore has no tagged release yet.
- **HMAC key mandatory for cursors**: one more config value per service, in exchange for tamper-proof cursors.
