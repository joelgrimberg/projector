.PHONY: build build-all clean test install

# Binary name
BINARY_NAME=projector

# Version (you can override this with make VERSION=1.0.0)
VERSION ?= 0.1.0

# Build flags
LDFLAGS=-ldflags "-X main.Version=${VERSION}"

# Build for current platform
build:
	go build ${LDFLAGS} -o ${BINARY_NAME} .

# Build for all platforms
build-all: clean
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-arm64 .
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-linux-arm64 .

# Create distribution directory
dist:
	mkdir -p dist

# Clean build artifacts
clean:
	rm -rf dist/
	rm -f ${BINARY_NAME}

# Run tests
test:
	go test ./...

# Install locally
install:
	go install ${LDFLAGS} .

# Create checksums for releases
checksums: build-all
	cd dist && shasum -a 256 * > checksums.txt

# Help
help:
	@echo "Available targets:"
	@echo "  build      - Build for current platform"
	@echo "  build-all  - Build for all platforms (darwin/linux, amd64/arm64)"
	@echo "  clean      - Remove build artifacts"
	@echo "  test       - Run tests"
	@echo "  install    - Install locally"
	@echo "  checksums  - Create SHA256 checksums for releases"
	@echo "  help       - Show this help"
