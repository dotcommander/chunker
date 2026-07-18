set shell := ["bash", "-cu"]

BINARY_NAME := "chunker"
BIN_DIR := "bin"
MAIN_PKG := "./cmd/chunker"
COVERAGE_FILE := "coverage.out"

default:
  @just --list

build:
  mkdir -p {{BIN_DIR}}
  go build -o {{BIN_DIR}}/{{BINARY_NAME}} {{MAIN_PKG}}

run: build
  ./{{BIN_DIR}}/{{BINARY_NAME}}

test:
  go test -v ./...

test-short:
  go test -short -v ./...

test-coverage:
  go test -v -coverprofile={{COVERAGE_FILE}} ./...
  go tool cover -func={{COVERAGE_FILE}}

test-coverage-html: test-coverage
  go tool cover -html={{COVERAGE_FILE}}

lint:
  command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint is not installed. Install it from https://golangci-lint.run/usage/install/" >&2; exit 1; }
  golangci-lint run

fmt:
  go fmt ./...

vet:
  go vet ./...

tidy:
  go mod tidy
  go mod verify

deps: tidy
  go mod download

clean:
  rm -rf {{BIN_DIR}}
  rm -f {{COVERAGE_FILE}}

clean-all: clean
  go clean -cache -testcache -modcache
  rm -f *.test

build-all:
  mkdir -p {{BIN_DIR}}
  GOOS=darwin GOARCH=amd64 go build -o {{BIN_DIR}}/{{BINARY_NAME}}-darwin-amd64 {{MAIN_PKG}}
  GOOS=darwin GOARCH=arm64 go build -o {{BIN_DIR}}/{{BINARY_NAME}}-darwin-arm64 {{MAIN_PKG}}
  GOOS=linux GOARCH=amd64 go build -o {{BIN_DIR}}/{{BINARY_NAME}}-linux-amd64 {{MAIN_PKG}}
  GOOS=windows GOARCH=amd64 go build -o {{BIN_DIR}}/{{BINARY_NAME}}-windows-amd64.exe {{MAIN_PKG}}

server: build
  ./{{BIN_DIR}}/{{BINARY_NAME}} serve

server-dev: build
  PORT=3000 ./{{BIN_DIR}}/{{BINARY_NAME}} serve

check:
  just fmt
  just vet
  just lint
  just test

ci:
  just tidy
  just check
  just test-coverage

install:
  go install {{MAIN_PKG}}

bench:
  go test -bench=. -benchmem ./...

dev:
  @echo "Starting development mode..."
  @echo "Run 'just build' or 'just test' in another terminal"
  @echo "Or use a file watcher like 'watchexec' or 'entr'"
