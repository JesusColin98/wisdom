# Wisdom Development Environment Setup (Windows/PowerShell)

Write-Host "Setting up Wisdom Development Environment..." -ForegroundColor Cyan

# 1. Install Protobuf Compiler (via Chocolatey or manual download is better, but we'll use a direct approach if possible)
# Note: This script assumes Go is already installed.

if (!(Get-Command protoc -ErrorAction SilentlyContinue)) {
    Write-Host "protoc not found. Please install the Protocol Buffers compiler from: https://github.com/protocolbuffers/protobuf/releases" -ForegroundColor Yellow
} else {
    Write-Host "protoc is already installed." -ForegroundColor Green
}

# 2. Install Go Protoc Plugins
Write-Host "Installing/Updating Go Protoc Plugins..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 3. Add GOPATH bin to PATH for the current session
$goPathBin = "$($(go env GOPATH))\bin"
if ($env:PATH -notlike "*$goPathBin*") {
    $env:PATH += ";$goPathBin"
    Write-Host "Added $goPathBin to PATH for this session." -ForegroundColor Green
}

Write-Host "Setup complete. You can now run 'protoc' to generate Go code from .proto files." -ForegroundColor Cyan
