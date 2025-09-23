#!/bin/bash

set -e

echo "🧪 Testing Goryu CLI..."

TEST_DIR=$(mktemp -d)
echo "📁 Using test directory: $TEST_DIR"

echo "🔨 Building goryu..."
go build -o goryu ./cmd/goryu

cleanup() {
    echo "🧹 Cleaning up test directory..."
    rm -rf "$TEST_DIR"
    rm -f ./goryu
}
trap cleanup EXIT

GORYU_PATH=$(realpath ./goryu)

cd "$TEST_DIR"

echo "✅ Testing init with basic template..."
"$GORYU_PATH" init test-basic --template=basic
if [ ! -f "test-basic/cmd/server/main.go" ]; then
    echo "❌ Basic template init failed"
    exit 1
fi

echo "✅ Testing init with api template..."
"$GORYU_PATH" init test-api --template=api
if [ ! -f "test-api/cmd/server/main.go" ]; then
    echo "❌ API template init failed"
    exit 1
fi

echo "✅ Testing generate handler with hyphenated name..."
cd test-api
"$GORYU_PATH" generate handler user-profile --type=basic
if [ ! -f "internal/handlers/user-profile.go" ]; then
    echo "❌ Handler generation failed"
    exit 1
fi

if grep -q "func.*-.*(" internal/handlers/user-profile.go; then
    echo "❌ Found hyphens in Go function names"
    exit 1
fi

echo "✅ Testing generate model with hyphenated name..."
"$GORYU_PATH" generate model user-data --type=basic
if [ ! -f "internal/models/user-data.go" ]; then
    echo "❌ Model generation failed"
    exit 1
fi

if grep -q "type.*-.*struct" internal/models/user-data.go; then
    echo "❌ Found hyphens in Go type names"
    exit 1
fi

echo "✅ Testing generate GORM model..."
"$GORYU_PATH" generate model product --type=db --db-tool=gorm
if [ ! -f "internal/models/product.go" ]; then
    echo "❌ GORM model generation failed"
    exit 1
fi

echo "✅ Testing invalid db-tool validation..."
if "$GORYU_PATH" generate model test --type=db --db-tool=invalid 2>/dev/null; then
    echo "❌ Invalid db-tool should have failed but didn't"
    exit 1
fi

echo "✅ Testing generate middleware..."
"$GORYU_PATH" generate middleware rate-limiter
if [ ! -f "internal/middleware/rate-limiter.go" ]; then
    echo "❌ Middleware generation failed"
    exit 1
fi

cd ..

echo "🎉 All CLI tests passed!"
echo "📊 Test Summary:"
echo "  ✅ Init basic template"
echo "  ✅ Init API template"
echo "  ✅ Generate handler (hyphenated name)"
echo "  ✅ Generate model (hyphenated name)"
echo "  ✅ Generate GORM model"
echo "  ✅ Invalid db-tool validation"
echo "  ✅ Generate middleware"