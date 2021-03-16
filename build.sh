#!/bin/bash
# Build seaflog command-line tool for 64-bit MacOS and Linux

VERSION=$(git describe --long --dirty)
GOOS=darwin GOARCH=amd64 go build -o "seaflog.${VERSION}.darwin-amd64/seaflog" cmd/seaflog/main.go || exit 1
GOOS=linux GOARCH=amd64 go build -o "seaflog.${VERSION}.linux-amd64/seaflog" cmd/seaflog/main.go || exit 1
zip -q -r "seaflog.${VERSION}.darwin-amd64.zip" "seaflog.${VERSION}.darwin-amd64" || exit 1
zip -q -r "seaflog.${VERSION}.linux-amd64.zip" "seaflog.${VERSION}.linux-amd64"|| exit 1
