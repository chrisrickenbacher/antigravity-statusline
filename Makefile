# Detect operating system and architecture for local installs
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Darwin)
    OS_SUF := darwin
endif
ifeq ($(UNAME_S),Linux)
    OS_SUF := linux
endif

ifeq ($(UNAME_M),x86_64)
    ARCH_SUF := amd64
endif
ifeq ($(UNAME_M),amd64)
    ARCH_SUF := amd64
endif
ifeq ($(UNAME_M),arm64)
    ARCH_SUF := arm64
endif
ifeq ($(UNAME_M),aarch64)
    ARCH_SUF := arm64
endif

PLATFORM_SUF := $(OS_SUF)-$(ARCH_SUF)

BINARY_NAME=statusline
DAEMON_NAME=statusline-daemon
RELEASES_DIR=releases

.PHONY: all build-local build-releases build-current install clean test

all: build-local

build-local:
	@echo "Building local binaries..."
	go build -o $(BINARY_NAME) ./cmd/statusline/main.go
	go build -o $(DAEMON_NAME) ./cmd/daemon/main.go

build-current:
	@echo "Building binaries for current platform ($(PLATFORM_SUF))..."
	mkdir -p $(RELEASES_DIR)
	go build -ldflags="-s -w" -o $(RELEASES_DIR)/$(BINARY_NAME)-$(PLATFORM_SUF) ./cmd/statusline/main.go
	go build -ldflags="-s -w" -o $(RELEASES_DIR)/$(DAEMON_NAME)-$(PLATFORM_SUF) ./cmd/daemon/main.go

install: build-current
	@echo "Installing locally built binaries..."
	./install.sh

build-releases: clean
	@echo "Cross-compiling releases for all platforms..."
	mkdir -p $(RELEASES_DIR)
	
	# macOS (Darwin) amd64
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(RELEASES_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/statusline/main.go
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(RELEASES_DIR)/$(DAEMON_NAME)-darwin-amd64 ./cmd/daemon/main.go
	
	# macOS (Darwin) arm64
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(RELEASES_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/statusline/main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(RELEASES_DIR)/$(DAEMON_NAME)-darwin-arm64 ./cmd/daemon/main.go
	
	# Linux amd64
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(RELEASES_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/statusline/main.go
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(RELEASES_DIR)/$(DAEMON_NAME)-linux-amd64 ./cmd/daemon/main.go
	
	# Linux arm64
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(RELEASES_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/statusline/main.go
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(RELEASES_DIR)/$(DAEMON_NAME)-linux-arm64 ./cmd/daemon/main.go
	
	@echo "Releases successfully compiled in $(RELEASES_DIR)/"

clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME) $(DAEMON_NAME)
	rm -rf $(RELEASES_DIR)

test:
	@echo "Running tests..."
	go test -v ./...
