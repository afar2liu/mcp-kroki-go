.PHONY: build clean install test run

# Binary name
BINARY_NAME=mcp-kroki-go

# Build the project
build:
	go build -o $(BINARY_NAME) .

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)

# Install dependencies
install:
	go mod tidy
	go mod download

# Run tests
test:
	go test -v ./...

# Run the server
run: build
	./$(BINARY_NAME)

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe .
