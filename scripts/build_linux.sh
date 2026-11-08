#!/bin/bash
set -e
OUT_DIR="out"
OUT_FILE="$OUT_DIR/xingran-backend-linux"
mkdir -p "$OUT_DIR"
echo "Building Linux binary -> $OUT_FILE ..."
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0
go build -ldflags="-s -w" -o "$OUT_FILE" ./cmd/main.go
echo "Build successful: $OUT_FILE"
ls -lh "$OUT_FILE"
