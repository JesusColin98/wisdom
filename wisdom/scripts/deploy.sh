#!/bin/bash
set -e

# Project Wisdom: Deployment & Initialization Script

echo "🌌 Starting Wisdom Deployment Cycle..."

# 1. Verification
echo "🧪 Running full test suite..."
go test -v ./...

# 2. Build
echo "🏗️ Building high-performance Go binary..."
go build -o wisdom_engine cmd/wisdom/main.go

# 3. Infrastructure Prep
echo "💾 Initializing semantic substrate (SQLite)..."
if [ ! -f "wisdom.db" ]; then
    # Create empty DB and apply schema
    touch wisdom.db
    sqlite3 wisdom.db < pkg/cortex/schema.sql
    echo "✅ Substrate initialized."
else
    echo "ℹ️ Substrate already exists, skipping initialization."
fi

# 4. Port Check
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null ; then
    echo "⚠️ Port 8080 is already in use. Please stop other services."
    exit 1
fi

echo "🚀 Wisdom Engine is ready. Run './wisdom_engine' to start."
