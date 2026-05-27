BINARY    := git-explain
BUILD_DIR := bin
MAIN      := ./cmd/git-explain

VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS   := -ldflags="-s -w -X main.version=$(VERSION)"

.PHONY: build install clean test test-verbose test-race lint vet

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) $(MAIN)

install: build
	cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)

release:
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64  $(MAIN)
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64  $(MAIN)
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64   $(MAIN)
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64   $(MAIN)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe $(MAIN)

test:
	go test ./... -count=1

test-verbose:
	go test ./... -v -count=1

test-race:
	go test -race ./... -count=1

vet:
	go vet ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)
