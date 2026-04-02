.PHONY: test build run-health fmt

test:
	go test ./...

build:
	go build ./...

run-health:
	go run ./cmd/cli health

fmt:
	gofmt -w ./cmd ./internal
