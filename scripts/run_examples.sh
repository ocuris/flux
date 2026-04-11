#!/usr/bin/env bash
# Flux Examples — Quick Start
# Run each example in a separate terminal window.
# Requires Go 1.21+.

set -euo pipefail

print_separator() { printf '\n%s\n\n' '──────────────────────────────────────────────────────'; }

cat <<'EOF'

  ███████╗██╗     ██╗   ██╗██╗  ██╗
  ██╔════╝██║     ██║   ██║╚██╗██╔╝
  █████╗  ██║     ██║   ██║ ╚███╔╝
  ██╔══╝  ██║     ██║   ██║ ██╔██╗
  ██║     ███████╗╚██████╔╝██╔╝ ██╗
  ╚═╝     ╚══════╝ ╚═════╝ ╚═╝  ╚═╝

  Examples Quick Start — https://github.com/ocuris/flux

EOF

print_separator

echo "1️⃣  BASIC CRUD (port :8000)"
echo "   cd example/basic && go run main.go"
echo ""
echo "   Quick tests:"
echo "   curl http://localhost:8000/health"
echo "   curl http://localhost:8000/users"
echo "   curl -X POST http://localhost:8000/users \\"
echo "        -H 'Content-Type: application/json' \\"
echo "        -d '{\"name\":\"Alice\",\"email\":\"alice@example.com\",\"age\":28}'"
echo "   open http://localhost:8000/docs"

print_separator

echo "2️⃣  JWT AUTHENTICATION (port :8001)"
echo "   JWT_SECRET=my-dev-secret cd example/auth && go run main.go"
echo ""
echo "   Quick tests:"
echo "   curl http://localhost:8001/health"
echo "   curl -X POST http://localhost:8001/login \\"
echo "        -H 'Content-Type: application/json' \\"
echo "        -d '{\"email\":\"john@example.com\",\"password\":\"password123\"}'"
echo "   # Copy token from response, then:"
echo "   TOKEN=<paste-token-here>"
echo "   curl -H \"Authorization: Bearer \$TOKEN\" http://localhost:8001/profile"
echo "   curl -H \"Authorization: Bearer \$TOKEN\" http://localhost:8001/users"
echo "   open http://localhost:8001/docs"

print_separator

echo "3️⃣  INPUT VALIDATION (port :8002)"
echo "   cd example/validation && go run main.go"
echo ""
echo "   Quick tests:"
echo "   # Valid product"
echo "   curl -X POST http://localhost:8002/products \\"
echo "        -H 'Content-Type: application/json' \\"
echo "        -d '{\"name\":\"Laptop\",\"sku\":\"LPT-001\",\"price\":999.99,\"category\":\"electronics\",\"stock\":50}'"
echo "   # Intentional validation errors"
echo "   curl -X POST http://localhost:8002/products \\"
echo "        -H 'Content-Type: application/json' \\"
echo "        -d '{\"name\":\"x\",\"sku\":\"bad sku!\",\"price\":-1,\"category\":\"unknown\",\"stock\":-5}'"
echo "   open http://localhost:8002/docs"

print_separator

echo "4️⃣  HIGH-THROUGHPUT CACHE (port :8003)"
echo "   cd example/high-throughput && go run main.go"
echo ""
echo "   Quick tests:"
echo "   curl http://localhost:8003/users/1        # cache miss (~1 ms)"
echo "   curl http://localhost:8003/users/1        # cache hit  (<0.1 ms)"
echo "   curl http://localhost:8003/metrics"
echo "   curl -X DELETE http://localhost:8003/cache/1"
echo ""
echo "   Load test (requires Apache Bench):"
echo "   for i in {1..50}; do curl -s http://localhost:8003/users/1 > /dev/null; done"
echo "   ab -n 10000 -c 100 http://localhost:8003/users/1"
echo "   curl http://localhost:8003/metrics"

print_separator

echo "5️⃣  STRUCTURED ERROR HANDLING (port :8004)"
echo "   cd example/advanced && go run main.go"
echo ""
echo "   Quick tests:"
echo "   curl http://localhost:8004/articles/1"
echo "   curl http://localhost:8004/demo/validation"
echo "   curl http://localhost:8004/demo/not-found"
echo "   curl http://localhost:8004/demo/conflict"
echo "   curl http://localhost:8004/demo/panic        # triggers 500 (Recover middleware)"
echo "   open http://localhost:8004/docs"

print_separator

echo "📖  DOCUMENTATION"
echo "   README.md           — framework reference"
echo "   API_DOCUMENTATION.md — DocBuilder reference"
echo ""
echo "✨  Happy building with Flux!"
echo ""
