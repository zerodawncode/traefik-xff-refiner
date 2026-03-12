.PHONY: lint test yaegi_test vendor clean
export GO111MODULE=on

default: lint test

# Lint the Go code
lint:
	golangci-lint run

# Run standard Go tests
test:
	go test -v -cover ./...

# Run tests with Yaegi
# This assumes your plugin's .traefik.yml is in the root and tests are in *_test.go files.
# Yaegi needs to interpret the plugin configuration and the test files.
yaegi_test:
	yaegi test -v .

# Create vendor directory
vendor:
	go mod vendor

# Clean up build artifacts
clean:
	rm -rf ./vendor
	rm -f coverage.out

# Help target
help:
	@echo "Available targets:"
	@echo "  default    : Run lint and standard tests"
	@echo "  lint       : Lint the Go code"
	@echo "  test       : Run standard Go tests"
	@echo "  yaegi_test : Run tests with Yaegi"
	@echo "  vendor     : Create vendor directory"
	@echo "  clean      : Clean up build artifacts"
