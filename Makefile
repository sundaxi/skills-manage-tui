.PHONY: build test lint install clean fmt

BINARY   := skill-tui
GO       := go
GOFLAGS  := -trimpath -ldflags="-s -w"

build:
	$(GO) build $(GOFLAGS) -o $(BINARY) .

test:
	$(GO) test ./... -v -count=1

lint: fmt
	$(GO) vet ./...

fmt:
	gofmt -w .

install: build
	cp $(BINARY) /usr/local/bin/

clean:
	rm -f $(BINARY)

run: build
	./$(BINARY)

.PHONY: coverage
coverage:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
