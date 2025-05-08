# Build variables
BINARY_BRIDGE=api-bridge
BINARY_OFFRAMP=api-offramp
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# Target architectures
ARCHS=amd64 arm64
DARWIN_ARCHS=arm64

.PHONY: all clean build-all build-bridge build-offramp build-darwin

all: build-all

clean:
	rm -rf build/

build-all: clean build-linux build-darwin

build-linux:
	@for arch in $(ARCHS); do \
		echo "Building for linux/$$arch..."; \
		mkdir -p build/linux/$$arch; \
		GOOS=linux GOARCH=$$arch go build -v $(LDFLAGS) -o build/linux/$$arch/$(BINARY_BRIDGE) ./api-bridge/main.go; \
		GOOS=linux GOARCH=$$arch go build -v $(LDFLAGS) -o build/linux/$$arch/$(BINARY_OFFRAMP) ./api-offramp/main.go; \
	done

build-darwin:
	@for arch in $(DARWIN_ARCHS); do \
		echo "Building for darwin/$$arch..."; \
		mkdir -p build/darwin/$$arch; \
		GOOS=darwin GOARCH=$$arch go build -v $(LDFLAGS) -o build/darwin/$$arch/$(BINARY_BRIDGE) ./api-bridge/main.go; \
		GOOS=darwin GOARCH=$$arch go build -v $(LDFLAGS) -o build/darwin/$$arch/$(BINARY_OFFRAMP) ./api-offramp/main.go; \
	done

build-bridge:
	@for arch in $(ARCHS); do \
		echo "Building bridge for linux/$$arch..."; \
		mkdir -p build/linux/$$arch; \
		GOOS=linux GOARCH=$$arch go build -v $(LDFLAGS) -o build/linux/$$arch/$(BINARY_BRIDGE) ./api-bridge/main.go; \
	done
	@for arch in $(DARWIN_ARCHS); do \
		echo "Building bridge for darwin/$$arch..."; \
		mkdir -p build/darwin/$$arch; \
		GOOS=darwin GOARCH=$$arch go build -v $(LDFLAGS) -o build/darwin/$$arch/$(BINARY_BRIDGE) ./api-bridge/main.go; \
	done

build-offramp:
	@for arch in $(ARCHS); do \
		echo "Building offramp for linux/$$arch..."; \
		mkdir -p build/linux/$$arch; \
		GOOS=linux GOARCH=$$arch go build -v $(LDFLAGS) -o build/linux/$$arch/$(BINARY_OFFRAMP) ./api-offramp/main.go; \
	done
	@for arch in $(DARWIN_ARCHS); do \
		echo "Building offramp for darwin/$$arch..."; \
		mkdir -p build/darwin/$$arch; \
		GOOS=darwin GOARCH=$$arch go build -v $(LDFLAGS) -o build/darwin/$$arch/$(BINARY_OFFRAMP) ./api-offramp/main.go; \
	done

# Help target
help:
	@echo "Available targets:"
	@echo "  all          - Build all components for all architectures (Linux and Darwin)"
	@echo "  clean        - Remove build directory"
	@echo "  build-all    - Build both bridge and offramp for all architectures"
	@echo "  build-linux  - Build for Linux architectures (amd64, arm64)"
	@echo "  build-darwin - Build for Darwin/ARM64"
	@echo "  build-bridge - Build only bridge for all architectures"
	@echo "  build-offramp - Build only offramp for all architectures"
	@echo ""
	@echo "Build artifacts will be placed in build/<os>/<arch>/ directory" 