#!/bin/bash

# 生成gRPC代码脚本
set -e

# 检查protoc是否安装
if ! command -v protoc &> /dev/null; then
    echo "protoc未安装，请先安装Protocol Buffers编译器"
    exit 1
fi

# 检查Go插件是否安装
if ! command -v protoc-gen-go &> /dev/null; then
    echo "正在安装protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "正在安装protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# 创建输出目录
mkdir -p proto/agent

# 生成Go代码
echo "生成gRPC Go代码..."
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/agent.proto

echo "gRPC代码生成完成！"