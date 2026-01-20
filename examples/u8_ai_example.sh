#!/bin/bash

# U8 数据库 AI 增强分析示例

# 配置
DB_SERVER="192.168.1.100"
DB_USER="sa"
DB_PASS="YourPassword"
DB_NAME="UFDATA_001_2023"
OUTPUT_DIR="./u8_ai_analysis_$(date +%Y%m%d_%H%M%S)"

# 阿里云 API Key（从环境变量或直接设置）
# export DASHSCOPE_API_KEY="sk-xxxxx"

echo "🔍 开始 AI 增强分析 U8 数据库..."
echo "服务器: $DB_SERVER"
echo "数据库: $DB_NAME"
echo ""

# 检查 API Key
if [ -z "$DASHSCOPE_API_KEY" ]; then
    echo "⚠️  警告：未设置 DASHSCOPE_API_KEY 环境变量"
    echo "   AI 功能将无法使用"
    echo ""
    read -p "是否继续？(y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# 构建连接字符串
CONN_STR="server=$DB_SERVER;user id=$DB_USER;password=$DB_PASS;database=$DB_NAME"

# 运行 AI 增强分析
./schema-analyzer scan \
  --type sqlserver \
  --conn "$CONN_STR" \
  --output "$OUTPUT_DIR" \
  --sample 2000 \
  --enable-ai

echo ""
echo "✅ 分析完成！"
echo "📁 结果保存在: $OUTPUT_DIR"
echo ""
echo "查看结果："
echo "  - AI 增强字典: cat $OUTPUT_DIR/dict.md"
echo "  - ER 图: 复制 $OUTPUT_DIR/er.mmd 到 https://mermaid.live/"
echo "  - JSON 数据: cat $OUTPUT_DIR/schema.json | jq"
echo ""
echo "AI 分析说明："
echo "  🤖标准 - AI 直接识别的 U8 标准字段（如 cDepCode、cInvCode）"
echo "  🔍推断 - AI 基于关联推断的自定义字段（如 cFree1、cDefine1）"
echo "  🔗关联 - 仅基于数据关系推断的字段"
