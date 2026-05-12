#!/bin/bash
# Wisdom Development Environment Setup (Linux/macOS)

echo "Setting up Wisdom Development Environment..."

# 1. Install Protobuf Compiler
if ! command -v protoc &> /dev/null
then
    echo "protoc not found. Installing via apt-get (requires sudo)..."
    sudo apt-get update && sudo apt-get install -y protobuf-compiler
else
    echo "protoc is already installed."
fi

# 2. Install Go Protoc Plugins
echo "Installing/Updating Go Protoc Plugins..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 3. Add GOPATH bin to PATH
export PATH="$PATH:$(go env GOPATH)/bin"
echo "Updated PATH to include Go bin directory."

echo "Setup complete. You can now run 'protoc' to generate Go code from .proto files."
