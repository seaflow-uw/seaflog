#!/bin/bash
# Build seaflog command-line tool for 64-bit MacOS and Linux

VERSION=$(git describe --dirty --tags)
GOOS=darwin GOARCH=amd64 go build -o "build/seaflog.${VERSION}.darwin-amd64" cmd/seaflog/main.go || exit 1
GOOS=linux GOARCH=amd64 go build -o "build/seaflog.${VERSION}.linux-amd64" cmd/seaflog/main.go || exit 1
GOOS=darwin GOARCH=arm64 go build -o "build/seaflog.${VERSION}.darwin-arm64" cmd/seaflog/main.go || exit 1
openssl dgst -sha256 build/*.${VERSION}.* | sed -e 's|build/||g'
