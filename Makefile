.PHONY: help build test race bench examples load-test vuln clean

# Default target
all: help

# Show help
help:
	@echo "Flux Framework Development"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "Targets:"
	@echo "  make build      - Verify core library compiles"
	@echo "  make test       - Run standard unit tests"
	@echo "  make race       - Run tests with concurrency race detector"
	@echo "  make bench       - Run the elite performance benchmark (requires Docker)"
	@echo "  make load-test   - Run local Load Test (requires 'ab')"
	@echo "  make examples    - Run end-to-end example validation suite"
	@echo "  make vuln       - Run security vulnerability scan (govulncheck)"
	@echo "  make clean      - Remove build artifacts and benchmark results"

# Verify core library compiles
build:
	go build -v ./...

# Run standard tests
test:
	go test -v ./...

# Run tests with the race detector enabled
race:
	go test -v -race ./...

# Run the elite performance benchmark (requires Docker)
bench:
	docker build -t flux-bench -f benchmarks/Dockerfile .
	docker run --rm flux-bench

# Run end-to-end example validation suite
examples:
	bash scripts/run_examples.sh

# Run a local load test (requires 'ab')
# Usage: make load-test ADDR=:8080 PATH=/users/1
load-test:
	@echo "🚀 Starting load test on $(ADDR)$(PATH)..."
	sh -c "ab -n 100000 -c 100 http://localhost$(ADDR)$(PATH)"

# Check for security vulnerabilities
vuln:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

# Clean up build artifacts
clean:
	rm -rf benchmarks/result_*
	go clean
