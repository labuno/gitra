#!/bin/bash
# 双击本文件即可启动 gitra 图形界面（macOS）。
cd "$(dirname "$0")" || exit 1
if [ ! -x bin/gitra ] || [ -n "$(find cmd internal -newer bin/gitra -name '*.go' -print -quit 2>/dev/null)" ]; then
  echo "首次运行需要构建 gitra，请稍候…"
  go build -o bin/gitra ./cmd/gitra || { echo "构建失败：请确认已安装 Go 工具链。"; read -r -p "按回车关闭…"; exit 1; }
fi
exec bin/gitra
