.PHONY: build run test clean lint deps

# Build the application
build:
	go build -o bin/chunker cmd/chunker/main.go

# Run the application
run: build
	./bin/chunker

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Run linter
lint:
	golangci-lint run

# Download dependencies
deps:
	go mod download
	go mod tidy

# Run with specific port
run-port: build
	PORT=$(PORT) ./bin/chunker

# Build for multiple platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build -o bin/chunker-darwin-amd64 cmd/chunker/main.go
	GOOS=linux GOARCH=amd64 go build -o bin/chunker-linux-amd64 cmd/chunker/main.go
	GOOS=windows GOARCH=amd64 go build -o bin/chunker-windows-amd64.exe cmd/chunker/main.go