#!/usr/bin/env bash
# 构建：Windows GUI 版 + Windows CLI 版 + Linux 版
set -euo pipefail
cd "$(dirname "$0")"
echo "==> kill-port.exe (Windows GUI, 无黑框)"
GOOS=windows GOARCH=amd64 go build -trimpath -tags gui -ldflags "-s -w -H windowsgui" -o kill-port.exe .
echo "==> kill-port-cli.exe (Windows 命令行)"
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o kill-port-cli.exe .
echo "==> kill-port (Linux)"
go build -trimpath -ldflags "-s -w" -o kill-port-linux .
ls -la kill-port.exe kill-port-cli.exe kill-port-linux
