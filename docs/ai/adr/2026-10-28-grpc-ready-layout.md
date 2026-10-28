# ADR: gRPC-ready layout without a gRPC server

Date: 2026-10-28
Status: accepted

## Decision
gocore does not ship a gRPC/ConnectRPC server yet (no consumer needs it). The architecture is prepared so one can be added without changing any existing public API:
- reserved top-level package `grpcserver` at rank 4 (same tier as `httpserver`), the only allowed owner of `google.golang.org/grpc` and `connectrpc.com` (`internal/archtest`);
- transport-neutral helpers extracted from the Gin middleware: `stdlog.ContextWithRequestID` / `RequestIDFromContext`, `stderr.FromPanic`;
- the gRPC status mapping from `stderr.Code` will live in `grpcserver`, keeping `stderr` dependency-free;
- infrastructure packages stay independent (ARCH-5), so a transport cannot become entangled with `database` or `cache`.

## Rationale
Building a server nobody uses adds maintenance and dependency weight (go-library's `server/grpc` and `http3` are examples). Reserving the boundary is nearly free and prevents the first gRPC user from putting gRPC inside `httpserver` or leaking it into core packages.

Rejected: adding gRPC inside `httpserver` (couples two transports and their dependency trees); implementing `grpcserver` now (no consumer, YAGNI).

## Follow-ups
- Done 2026-10-28: `httpserver.Server.Start()` now takes `ctx` (CS-5). It blocks until the server fails or `ctx` is cancelled, then shuts down gracefully within `Config.ShutdownTimeout` and returns nil, so HTTP and a future gRPC server can run side by side under one `errgroup`. This was a breaking signature change, accepted by the user because gocore has no tagged release yet.
