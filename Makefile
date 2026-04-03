.PHONY: test build run-health run-customer-list run-mcp-http run-mcp fmt

test:
	go test ./...

build:
	go build ./...

run-health:
	go run ./cmd/cli health

run-customer-list:
	BILLAR_SESSION_EMAIL=local@billar.dev go run ./cmd/cli customer list

run-mcp-http:
	go run ./cmd/mcp-http

run-mcp:
	go run ./cmd/mcp

fmt:
	gofmt -w ./cmd ./internal
