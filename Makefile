# Build variables
BINARY_BRIDGE=api-bridge
BINARY_OFFRAMP=api-offramp
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# Target architectures
ARCHS=amd64 arm64

.PHONY: all clean build-all build-bridge build-offramp

all: build-all

clean:
	rm -rf build/

build-all: clean
	@for arch in $(ARCHS); do \
		echo "Building for $$arch..."; \
		mkdir -p build/$$arch; \
		GOOS=linux GOARCH=$$arch go build $(LDFLAGS) -o build/$$arch/$(BINARY_BRIDGE) ./api-bridge; \
		GOOS=linux GOARCH=$$arch go build $(LDFLAGS) -o build/$$arch/$(BINARY_OFFRAMP) ./api-offramp; \
	done

build-bridge:
	@for arch in $(ARCHS); do \
		echo "Building bridge for $$arch..."; \
		mkdir -p build/$$arch; \
		GOOS=linux GOARCH=$$arch go build $(LDFLAGS) -o build/$$arch/$(BINARY_BRIDGE) ./api-bridge; \
	done

build-offramp:
	@for arch in $(ARCHS); do \
		echo "Building offramp for $$arch..."; \
		mkdir -p build/$$arch; \
		GOOS=linux GOARCH=$$arch go build $(LDFLAGS) -o build/$$arch/$(BINARY_OFFRAMP) ./api-offramp; \
	done

# Help target
help:
	@echo "Available targets:"
	@echo "  all          - Build all components for all architectures"
	@echo "  clean        - Remove build directory"
	@echo "  build-all    - Build both bridge and offramp for all architectures"
	@echo "  build-bridge - Build only bridge for all architectures"
	@echo "  build-offramp - Build only offramp for all architectures"
	@echo ""
	@echo "Build artifacts will be placed in build/<arch>/ directory" 