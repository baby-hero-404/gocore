.PHONY: verify test lint tidy mod mocks mocks-check secrets cover-check cover-baseline vuln

# Pinned tool versions (latest stable at time of writing; bump deliberately via `make mod` review).
GITLEAKS := github.com/zricethezav/gitleaks/v8@v8.30.1
GOLANGCI := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0
GOVULNCHECK := golang.org/x/vuln/cmd/govulncheck@v1.9.0

verify: lint mocks-check secrets test cover-check

test:
	go test ./... -race -covermode=atomic -coverprofile=coverage.out

lint:
	@test -z "$$(gofmt -l $$(git ls-files -co --exclude-standard '*.go' | grep -v '^vendor/') 2>/dev/null)" || { echo 'gofmt: unformatted files (run gofmt -w):'; gofmt -l $$(git ls-files -co --exclude-standard '*.go' | grep -v '^vendor/'); exit 1; }
	go vet ./...
	go run $(GOLANGCI) run ./...

tidy:
	go mod tidy

mod:
	go get -u ./...
	go mod tidy
	go mod vendor

# Regenerate all mocks from .mockery.yml (mockery runs as a `go tool` pinned in go.mod).
mocks:
	go tool mockery

# Fail when committed mocks no longer match the interfaces (run `make mocks` to fix).
mocks-check:
	@before=$$(find . -path ./vendor -prune -o -name 'mock_*.go' -print | sort | xargs cat | sha256sum); \
	go tool mockery; \
	after=$$(find . -path ./vendor -prune -o -name 'mock_*.go' -print | sort | xargs cat | sha256sum); \
	if [ "$$before" != "$$after" ]; then echo "mocks are stale: 'make mocks' regenerated them, review and commit the result"; exit 1; fi

# SEC-1: scan the working tree for committed secrets (config in .gitleaks.toml).
secrets:
	go run $(GITLEAKS) dir . --config .gitleaks.toml --no-banner --redact

# TEST-3: per-package coverage may not drop below coverage-baseline.txt (needs `make test` first).
cover-check:
	go run ./internal/tools/covercheck -profile coverage.out -baseline coverage-baseline.txt

# Rewrite the baseline after intentionally raising coverage (needs `make test` first).
cover-baseline:
	go run ./internal/tools/covercheck -profile coverage.out -baseline coverage-baseline.txt -update

# Known-vulnerability scan of dependencies. Not part of `verify`: it needs network access to the
# vulnerability database and a newly published advisory would fail unrelated work. CI runs it on
# every PR and weekly.
vuln:
	go run $(GOVULNCHECK) ./...
