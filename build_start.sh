#!/bin/bash
# filepath: /root/workspace/examination-papers/deploy.sh

set -e

APP_NAME=exam-papers
BUILD_DIR=./build

echo "==> 开始构建 Go 项目..."
go mod tidy
mkdir -p $BUILD_DIR
go build -o $BUILD_DIR/$APP_NAME main.go

echo "==> 停止旧进程（如有）..."
pkill -f "$BUILD_DIR/$APP_NAME" || true

echo "==> 后台启动新进程..."
nohup $BUILD_DIR/$APP_NAME > $BUILD_DIR/app.log 2>&1 &

echo "==> 部署完成，日志输出在 $BUILD_DIR/app.log"