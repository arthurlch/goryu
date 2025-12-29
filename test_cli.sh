#!/bin/bash

# Goryu CLI Comprehensive Test Suite
# This script tests the end-to-end functionality of the CLI.

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color


pass() {
    echo -e "${GREEN}✓ $1${NC}"
}

fail() {
    echo -e "${RED}✗ $1${NC}"
    exit 1
}

echo "🚀 Starting Goryu CLI Test Suite..."

# Setup
TEST_DIR=$(mktemp -d)
echo "📂 Using temp directory: $TEST_DIR"

# Use go from PATH (works in CI and local environments)
GO_BIN="go"

echo "🔨 Building goryu binary..."
if $GO_BIN build -o goryu ./cmd/goryu; then
    pass "Built goryu binary"
else
    fail "Failed to build goryu binary"
fi

GORYU_PATH=$(realpath ./goryu)

cleanup() {
    echo "🧹 Cleaning up..."
    rm -rf "$TEST_DIR"
    rm -f ./goryu
}
trap cleanup EXIT

GORYU_ROOT=$(pwd)

cd "$TEST_DIR" || fail "Could not cd to test dir"

# Helper to add local replace
add_replace() {
    $GO_BIN mod edit -replace github.com/arthurlch/goryu="$GORYU_ROOT"
}

# --- Test 1: Init Basic Template ---
echo "------------------------------------------------"
echo "🧪 Test: Init Basic Template"
"$GORYU_PATH" init test-basic --template=basic

if [ -d "test-basic" ]; then
    cd test-basic
    
    echo "   Running go mod tidy..."
    add_replace
    $GO_BIN mod tidy > /dev/null 2>&1
    
    echo "   Verifying compilation..."
    if $GO_BIN build ./... > /dev/null; then
        pass "Basic template compiles"
    else
        fail "Basic template failed to compile"
    fi
    cd ..
else
    fail "Failed to create test-basic directory"
fi

# --- Test 2: Init API Template & Generators ---
echo "------------------------------------------------"
echo "🧪 Test: Init API Template & Generators"
"$GORYU_PATH" init test-api --template=api
cd test-api

echo "   Running go mod tidy..."
add_replace
$GO_BIN mod tidy > /dev/null 2>&1

# Generate Handler
echo "   Generating handler..."
"$GORYU_PATH" generate handler user --type=crud
if [ -f "internal/handlers/user.go" ]; then pass "Handler generated"; else fail "Handler missing"; fi

# Generate Model
echo "   Generating model..."
"$GORYU_PATH" generate model product --type=basic
if [ -f "internal/models/product.go" ]; then pass "Model generated"; else fail "Model missing"; fi

# Generate Middleware
echo "   Generating middleware..."
"$GORYU_PATH" generate middleware auth-check
if [ -f "internal/middleware/auth-check.go" ]; then pass "Middleware generated"; else fail "Middleware missing"; fi

# Generate Route
echo "   Generating route..."
"$GORYU_PATH" generate route api --group="/v1"
if [ -f "internal/routes/api.go" ]; then pass "Route generated"; else fail "Route missing"; fi

# Generate Stub Handlers for API Route (since generating route doesn't auto-generate handlers)
cat > internal/handlers/api.go <<EOF
package handlers

import "github.com/arthurlch/goryu"

func HandleApiGet(c *goryu.Context) {}
func HandleApiPost(c *goryu.Context) {}
func HandleApiPut(c *goryu.Context) {}
func HandleApiDelete(c *goryu.Context) {}
EOF

# Verify Compilation of Generated Code
echo "   Verifying compilation of generated code..."
$GO_BIN mod tidy > /dev/null 2>&1
if $GO_BIN build ./... > /dev/null; then
    pass "Generated code compiles"
else
    fail "Generated code failed to compile"
fi

cd ..

# --- Test 3: Scaffold API ---
echo "------------------------------------------------"
echo "🧪 Test: Scaffold API"
mkdir scaffold-test
cd scaffold-test
$GO_BIN mod init scaffold-test
add_replace
# Provide necessary structure/files if scaffold expects them, 
# but usually scaffold is run inside an existing project. 
# We'll treat this as a fresh start for simplicity or reuse test-api?
# Let's reuse test-api for a real integration test.
cd ../test-api

echo "   Scaffolding 'order' resource..."
# We assume test-api has the structure.
# Force --db=false to avoid needing real DB setup/drivers for compilation if they require external deps not in go.mod automatically
"$GORYU_PATH" scaffold api order --fields="amount:float,customer:string" --db=false

echo "   Verifying scaffold compilation..."
$GO_BIN mod tidy > /dev/null 2>&1
if $GO_BIN build ./... > /dev/null; then
    pass "Scaffolded API compiles"
else
    fail "Scaffolded API failed to compile"
fi

cd ..

# --- Test 4: Goryu Build Command ---
echo "------------------------------------------------"
echo "🧪 Test: Goryu Build Command"
cd test-api
if "$GORYU_PATH" build --output=server; then
    if [ -f "server" ]; then
        pass "Build command produced binary"
    else
        fail "Build command failed to produce binary"
    fi
else
    fail "Build command failed"
fi
cd ..


echo "------------------------------------------------"
echo -e "${GREEN}✨ All Tests Passed! The CLI is rock solid.${NC}"
exit 0