#!/bin/bash

# Build for different platforms
echo "Building MCP SQLite Server for multiple platforms..."

# Windows
GOOS=windows GOARCH=amd64 go build -o mcp-sqlite-server.exe
echo "✓ Windows build complete: mcp-sqlite-server.exe"

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o mcp-sqlite-server-darwin-amd64
echo "✓ macOS Intel build complete: mcp-sqlite-server-darwin-amd64"

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o mcp-sqlite-server-darwin-arm64
echo "✓ macOS Apple Silicon build complete: mcp-sqlite-server-darwin-arm64"

# Linux
GOOS=linux GOARCH=amd64 go build -o mcp-sqlite-server-linux
echo "✓ Linux build complete: mcp-sqlite-server-linux"

echo "All builds complete!"