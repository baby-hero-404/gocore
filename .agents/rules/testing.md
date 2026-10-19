# Testing Standards & Guidelines

This document specifies unit testing and integration testing standards for the `gocore` repository.

---

## 1. General Requirements

As a shared foundation toolkit (chassis library) powering critical services, `gocore` requires high reliability:
* **Table-Driven Tests**: Mandatory pattern for unit testing logic (especially `stderr`, `stdres`, `env`, and config validation).
* **Race Condition Detection**: All tests must execute and pass cleanly under the `-race` flag:
  ```bash
  go test ./... -race -cover
  ```
* **Coverage Target**: Minimum 80% test coverage for core packages (`stderr`, `stdlog`, `stdres`, `lock`).

---

## 2. Standard Table-Driven Test Structure

Example pattern for Go unit tests:

```go
func TestError_Format(t *testing.T) {
    tests := []struct {
        name     string
        code     Code
        msg      string
        expected string
    }{
        {
            name:     "valid bad request error",
            code:     ErrInvalidInput,
            msg:      "invalid username",
            expected: "[1001] invalid username",
        },
        {
            name:     "internal server error",
            code:     ErrInternalServer,
            msg:      "db timeout",
            expected: "[5000] db timeout",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := New(tt.code, tt.msg)
            if got := err.Error(); got != tt.expected {
                t.Errorf("got %q, want %q", got, tt.expected)
            }
        })
    }
}
```

---

## 3. Mocking & External Dependencies

* **Narrow Mock Interfaces**: When testing business logic or handlers, only mock narrow interfaces defined locally at the consumption site.
* **Database / Redis Testing**:
  * Unit tests: Prefer `sqlmock` or `miniredis` for fast, lightweight in-memory tests without requiring Docker.
  * Integration tests: Tag with `//go:build integration` when testing against real instances (e.g. testcontainers or docker-compose).
* **Resource Cleanup**: Always use `t.Cleanup()` for tearing down temporary resources instead of scattered inline `defer` statements.
