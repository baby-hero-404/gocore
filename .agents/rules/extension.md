# Extension Rules (adding packages, mocks, examples)

Rules for growing `gocore` safely. Tags: `[test]` enforced by `internal/archtest`/`go test`, `[lint]` by lint/scan tooling in `make verify`, `[review]` by human or agent review only.

| ID | Tag | Rule |
|---|---|---|
| EXT-1 | `[test]` (rank, `doc.go`) / `[review]` (rest) | New package checklist, in order: rank entry in `internal/archtest` (`rank`), `doc.go`, own `Config` + `New*`, interface in `spec.go`, entry in `.mockery.yml`, tests, README package index row. |
| EXT-2 | `[review]` | Never remove or change the signature of an exported symbol. Add a new one and mark the old `// Deprecated:`. A breaking change needs an ADR (`docs/ai/adr/`) and a major version bump. |
| EXT-3 | `[test]` | A new third-party SDK must be added to `vendorOwners` in archtest with its single owner package, plus the reason in the PR. Passes the dependency checks in `dependency-management.md` first. |
| EXT-4 | `[test]` | Every `<pkg>mock` package has a compile-time assertion `var _ pkg.Iface = (*Mock)(nil)` in its hand-written `doc.go`. Mocks are generated, never hand-edited (`make mocks`; `make mocks-check` fails on drift). |
| EXT-5 | `[review]` | A candidate package must answer "is this shared infrastructure?". Domain logic (payments, blockchain, email, sign-in, sockets) belongs in consuming services, not in `gocore`. |
| EXT-6 | `[test]` | No `package main` and no demo code in the library. Examples are `Example*` tests; tools go under `internal/tools`. |

## Generated mocks

* Config: `.mockery.yml` (mockery v3, pinned as a `go tool` in `go.mod`). Output goes to `<pkg>/<pkg>mock/mock_<Interface>.go`.
* Add an interface: list it in `.mockery.yml`, run `make mocks`, add the assertion line to the mock package's `doc.go`.
* Never edit `mock_*.go`; change the interface or the config and regenerate.
