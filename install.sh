#!/bin/sh
set -e

# setup.sh — 安装 bin/reasonix 到 /usr/local/bin/reasonix
#
# 用法:
#   ./setup.sh
#
# 需要先执行 `make` 生成 bin/reasonix；写入 /usr/local/bin 需要 sudo 权限。
# 仅支持存在 /usr/local/bin 的类 Unix 系统（如 macOS、Linux）。

if [ ! -f bin/reasonix ]; then
    echo "error: bin/reasonix not found; run 'make' first" >&2
    exit 1
fi

if [ ! -d /usr/local/bin ]; then
    echo "error: /usr/local/bin does not exist; this system is not supported" >&2
    echo "error: setup.sh only supports Unix-like systems with /usr/local/bin" >&2
    exit 1
fi

sudo cp bin/reasonix /usr/local/bin/reasonix
sudo chmod +x /usr/local/bin/reasonix

echo "installed: /usr/local/bin/reasonix"
