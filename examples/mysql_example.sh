#!/bin/bash

# MySQL 数据库分析示例脚本

# 配置 - 请根据实际情况修改
DB_HOST="localhost"
DB_PORT="3306"
DB_USER="your_username"
DB_PASS="your_password"
DB_SCHEMA="your_database"
OUTPUT_DIR="./mysql_analysis_$(date +%Y%m%d_%H%M%S)"

echo "🔍 开始分析 MySQL 数据库..."
echo "主机: $DB_HOST:$DB_PORT"
echo "Schema: $DB_SCHEMA"
echo ""

# 构建连接字符串
CONN_STR="$DB_USER:$DB_PASS@tcp($DB_HOST:$DB_PORT)/$DB_SCHEMA"

# 运行分析
./schema-analyzer scan \
  --type mysql \
  --conn "$CONN_STR" \
  --schema "$DB_SCHEMA" \
  --output "$OUTPUT_DIR" \
  --sample 1000

echo ""
echo "✅ 分析完成！"
echo "📁 结果保存在: $OUTPUT_DIR"
