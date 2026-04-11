# Use the latest stable Go image
FROM golang:latest

WORKDIR /app

# 1. Copy the core Flux framework
COPY . .

# 2. Set up the comparison benchmark directory
RUN mkdir -p /bench
COPY benchmarks/comparison/go.mod /bench/go.mod
COPY benchmarks/comparison/go.sum /bench/go.sum
COPY benchmarks/comparison/comparison_test.go /bench/comparison_test.go

WORKDIR /bench
# Use the local codebase for the replacement
RUN go mod edit -replace github.com/ocuris/flux=/app
# Download remaining dependencies (this is now much faster)
RUN go mod download


# 4. Run benchmarks on startup
# Using -benchtime=3s for stable results
ENTRYPOINT ["go", "test", "-bench=.", "-benchmem", "-benchtime=3s"]
