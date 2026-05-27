BINARY    := git-explain
BUILD_DIR := bin
MAIN      := ./cmd/git-explain

GOFLAGS   := -ldflags="-s -w"

.PHONY: build install clean test lint

build:
	go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY) $(MAIN)

install: build
	cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)

release:
	GOOS=darwin  GOARCH=amd64 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 $(MAIN)
	GOOS=darwin  GOARCH=arm64 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 $(MAIN)
	GOOS=linux   GOARCH=amd64 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64  $(MAIN)
	GOOS=linux   GOARCH=arm64 go build $(GOFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64  $(MAIN)

test:
	go test ./... -v -count=1

lint:
	go vet ./...

clean:
	rm -rf $(BUILD_DIR)
