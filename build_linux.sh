#!/bin/bash
echo "Building Linux binary..."
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0
go build -ldflags="-s -w" -o ruoyi-backend-linux ./cmd/main.go
if [ $? -eq 0 ]; then
    echo "Build successful: ruoyi-backend-linux"
    ls -lh ruoyi-backend-linux
else
    echo "Build failed"
    exit 1
fi
