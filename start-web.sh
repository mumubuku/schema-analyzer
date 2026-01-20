#!/bin/bash

echo "🚀 启动 Schema Analyzer Web 服务器..."
echo ""

# 检查是否已构建
if [ ! -f "schema-analyzer-server" ]; then
    echo "📦 首次运行，正在构建..."
    go build -o schema-analyzer-server cmd/server/main.go
    if [ $? -ne 0 ]; then
        echo "❌ 构建失败"
        exit 1
    fi
    echo "✅ 构建完成"
    echo ""
fi

# 设置端口
PORT=${PORT:-8080}

echo "📡 服务地址: http://localhost:$PORT"
echo "📊 打开浏览器访问上述地址开始分析"
echo ""
echo "💡 提示："
echo "  - 按 Ctrl+C 停止服务"
echo "  - 设置环境变量 PORT 可更改端口"
echo "  - 设置环境变量 DASHSCOPE_API_KEY 可启用 AI"
echo ""

# 启动服务
PORT=$PORT ./schema-analyzer-server
