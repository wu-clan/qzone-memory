#!/usr/bin/env bash
# 跨平台构建脚本。
#
# 注意：本项目使用 CGO 版 SQLite（mattn/go-sqlite3），跨平台编译需要对应目标平台的
# C 交叉工具链。没有工具链时，相应平台会自动跳过——建议直接在各目标平台本地执行
# `make release`，或用 CI 矩阵分别构建。

set -u

APP="qzone-memory"
MAIN="./cmd/server/main.go"
mkdir -p dist

build() {
  local goos="$1" goarch="$2" ext="$3"
  echo ">> 构建 ${goos}/${goarch}"
  if CGO_ENABLED=1 GOOS="$goos" GOARCH="$goarch" \
      go build -trimpath -ldflags "-s -w" \
      -o "dist/${APP}-${goos}-${goarch}${ext}" "$MAIN" 2>/dev/null; then
    echo "   完成：dist/${APP}-${goos}-${goarch}${ext}"
  else
    echo "   跳过（缺少 ${goos}/${goarch} 的 CGO 交叉工具链）"
  fi
}

build "$(go env GOOS)" "$(go env GOARCH)" ""   # 当前平台（一定成功）
build darwin  arm64 ""
build darwin  amd64 ""
build linux   amd64 ""
build windows amd64 ".exe"

echo "产物位于 dist/"
