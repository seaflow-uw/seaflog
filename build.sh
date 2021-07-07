#!/bin/bash
# Build seaflog command-line tool for 64-bit MacOS and Linux

VERSION=$(git describe --long --dirty --tags)
GOOS=darwin GOARCH=amd64 go build -o "seaflog-${VERSION}-darwin-amd64" cmd/seaflog/main.go || exit 1
GOOS=linux GOARCH=amd64 go build -o "seaflog-${VERSION}-linux-amd64" cmd/seaflog/main.go || exit 1
