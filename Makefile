.PHONY: test lint tidy

test:
	go test ./... -race -cover

lint:
	gofmt -l .
	go vet ./...

tidy:
	go mod tidy

mod:
	go get -u ./...
	go mod tidy
	go mod vendor