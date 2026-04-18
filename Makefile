.PHONY: test build run-health run-customer-list run-mcp-http fmt

test:
	go test ./...

build:
	go build ./...

run-health:
	go run ./cmd/cli health

run-customer-list:
	go run ./cmd/cli customer list

run-mcp-http:
	go run ./cmd/mcp-http

fmt:
	gofmt -w ./cmd ./internal
