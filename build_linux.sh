#!/bin/bash
echo "Building Linux binary..."
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0
go build -ldflags="-s -w" -o xingran-backend-linux ./cmd/main.go
if [ $? -eq 0 ]; then
    echo "Build successful: xingran-backend-linux"
    ls -lh xingran-backend-linux
else
    echo "Build failed"
    exit 1
fi
