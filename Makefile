# Immich Importer Bootstrap - Build Configuration
#
# This Makefile builds bootstrap binaries for all supported platforms.
# The resulting binaries contain placeholder strings that the Immich server
# patches at download time with the user's specific server URL and setup token.

BINARY_NAME = bootstrap
OUTPUT_DIR = dist

# Build flags for small binaries
LDFLAGS = -s -w

# All target platforms
PLATFORMS = darwin-arm64 darwin-amd64 linux-amd64 windows-amd64

.PHONY: all clean $(PLATFORMS)

all: $(PLATFORMS)
	@echo ""
	@echo "Build complete! Binaries are in $(OUTPUT_DIR)/"
	@ls -lh $(OUTPUT_DIR)/

darwin-arm64:
	@mkdir -p $(OUTPUT_DIR)
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-arm64 .
	@echo "Built: $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-arm64"

darwin-amd64:
	@mkdir -p $(OUTPUT_DIR)
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64 .
	@echo "Built: $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64"

linux-amd64:
	@mkdir -p $(OUTPUT_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64 .
	@echo "Built: $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64"

windows-amd64:
	@mkdir -p $(OUTPUT_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Built: $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe"

clean:
	rm -rf $(OUTPUT_DIR)

# Install binaries to Immich server assets directory
install: all
	@if [ -z "$(IMMICH_SERVER_DIR)" ]; then \
		echo "Error: IMMICH_SERVER_DIR not set"; \
		echo "Usage: make install IMMICH_SERVER_DIR=/path/to/immich-fork/server"; \
		exit 1; \
	fi
	mkdir -p $(IMMICH_SERVER_DIR)/assets/bootstrap
	cp $(OUTPUT_DIR)/* $(IMMICH_SERVER_DIR)/assets/bootstrap/
	@echo ""
	@echo "Installed to $(IMMICH_SERVER_DIR)/assets/bootstrap/"

# Verify placeholder strings exist in binaries
verify:
	@echo "Verifying placeholder strings in binaries..."
	@for platform in $(PLATFORMS); do \
		binary="$(OUTPUT_DIR)/$(BINARY_NAME)-$$platform"; \
		if [ "$$platform" = "windows-amd64" ]; then binary="$$binary.exe"; fi; \
		if grep -q "__IMMICH_SERVER_URL_PLACEHOLDER_" "$$binary" 2>/dev/null; then \
			echo "  $$binary: OK"; \
		else \
			echo "  $$binary: MISSING PLACEHOLDERS!"; \
			exit 1; \
		fi; \
	done
	@echo "All binaries contain valid placeholder strings."
