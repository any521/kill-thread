#!/usr/bin/env bash
# 构建脚本：产出 Windows exe + Linux 二进制
set -euo pipefail
cd "$(dirname "$0")"
echo "==> build windows amd64 exe"
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o kill-port.exe .
echo "==> build linux amd64"
GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o kill-port-linux .
ls -la kill-port.exe kill-port-linux
