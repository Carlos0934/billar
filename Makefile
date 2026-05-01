BINDIR := $(shell go env GOBIN)
ifeq ($(strip $(BINDIR)),)
BINDIR := $(shell go env GOPATH)/bin
endif

.PHONY: test build install uninstall run-health run-customer-list run-invoice-import fmt

test:
	go test ./...

build:
	go build ./...

install:
	mkdir -p $(BINDIR)
	go build -o $(BINDIR)/billar ./cmd/cli

uninstall:
	rm -f $(BINDIR)/billar

run-health:
	go run ./cmd/cli health

run-customer-list:
	go run ./cmd/cli customer list

run-invoice-import:
	go run ./cmd/cli invoice import --file $(FILE)

fmt:
	gofmt -w ./cmd ./internal
