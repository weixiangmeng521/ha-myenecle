#!/bin/bash
# ============================================================
# build.sh - Cross compile Go program for Home Assistant Add-on
# ============================================================

# Go 程序入口文件
MAIN_FILE="main.go"

# 输出目录
OUTPUT_DIR="build"
mkdir -p "$OUTPUT_DIR"

# 支持架构列表
ARCHS=("amd64" "arm64")   # amd64 -> Intel/AMD, arm64 -> Raspberry Pi 4 / aarch64

# GOOS
GOOS=linux
CGO_ENABLED=0   # 静态编译，不依赖 libc

echo "Starting cross-compilation..."

for ARCH in "${ARCHS[@]}"; do
    OUT_FILE="${OUTPUT_DIR}/enecle-${GOOS}-${ARCH}"
    echo "Building for ${GOOS}/${ARCH} -> ${OUT_FILE}"
    GOARCH=$ARCH go build -o "$OUT_FILE" .
    if [ $? -ne 0 ]; then
        echo "Build failed for ${ARCH}"
        exit 1
    fi
done

echo "Build finished! Binaries are in ${OUTPUT_DIR}/"
