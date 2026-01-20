#!/bin/bash

# U8 数据库分析示例脚本

# 配置
DB_SERVER="192.168.1.100"
DB_USER="sa"
DB_PASS="YourPassword"
DB_NAME="UFDATA_001_2023"
OUTPUT_DIR="./u8_analysis_$(date +%Y%m%d_%H%M%S)"

echo "🔍 开始分析 U8 数据库..."
echo "服务器: $DB_SERVER"
echo "数据库: $DB_NAME"
echo ""

# 构建连接字符串
CONN_STR="server=$DB_SERVER;user id=$DB_USER;password=$DB_PASS;database=$DB_NAME"

# 运行分析
./schema-analyzer scan \
  --type sqlserver \
  --conn "$CONN_STR" \
  --output "$OUTPUT_DIR" \
  --sample 2000

echo ""
echo "✅ 分析完成！"
echo "📁 结果保存在: $OUTPUT_DIR"
echo ""
echo "查看结果："
echo "  - 数据字典: cat $OUTPUT_DIR/dict.md"
echo "  - ER 图: 复制 $OUTPUT_DIR/er.mmd 到 https://mermaid.live/"
echo "  - JSON 数据: cat $OUTPUT_DIR/schema.json | jq"
