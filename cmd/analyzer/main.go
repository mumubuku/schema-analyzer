package main

import (
	"fmt"
	"log"
	"os"
	"schema-analyzer/internal/adapter"
	"schema-analyzer/internal/ai"
	"schema-analyzer/internal/analyzer"
	"schema-analyzer/internal/graph"
	"schema-analyzer/internal/renderer"

	"github.com/spf13/cobra"
)

var (
	dbType     string
	connStr    string
	schema     string
	outputDir  string
	sampleSize int
	enableAI   bool
	aiAPIKey   string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "schema-analyzer",
		Short: "通用数据库结构分析器",
		Long:  "自动分析数据库结构，推断表关系，生成数据字典和 ER 图",
	}

	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "扫描数据库并分析结构",
		Run:   runScan,
	}

	scanCmd.Flags().StringVar(&dbType, "type", "sqlserver", "数据库类型 (sqlserver/mysql)")
	scanCmd.Flags().StringVar(&connStr, "conn", "", "连接字符串")
	scanCmd.Flags().StringVar(&schema, "schema", "", "数据库 schema (MySQL 必需)")
	scanCmd.Flags().StringVar(&outputDir, "output", "./output", "输出目录")
	scanCmd.Flags().IntVar(&sampleSize, "sample", 1000, "采样大小")
	scanCmd.Flags().BoolVar(&enableAI, "enable-ai", false, "启用 AI 增强（需要 API Key）")
	scanCmd.Flags().StringVar(&aiAPIKey, "ai-key", "", "AI API Key（或使用环境变量 DASHSCOPE_API_KEY）")
	scanCmd.MarkFlagRequired("conn")

	rootCmd.AddCommand(scanCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runScan(cmd *cobra.Command, args []string) {
	fmt.Println("🔍 开始扫描数据库...")

	// 创建适配器
	var dbAdapter adapter.DBAdapter
	var err error

	switch dbType {
	case "sqlserver":
		dbAdapter, err = adapter.NewSQLServerAdapter(connStr)
	case "mysql":
		if schema == "" {
			log.Fatal("MySQL 需要指定 --schema 参数")
		}
		dbAdapter, err = adapter.NewMySQLAdapter(connStr, schema)
	default:
		log.Fatalf("不支持的数据库类型: %s", dbType)
	}

	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer dbAdapter.Close()

	fmt.Println("✓ 数据库连接成功")

	// 1. 获取元数据
	fmt.Println("\n📊 获取数据库元数据...")
	meta, err := dbAdapter.IntrospectSchema()
	if err != nil {
		log.Fatalf("获取元数据失败: %v", err)
	}
	fmt.Printf("✓ 发现 %d 个表\n", len(meta.Tables))

	// 2. 构建 Schema Graph
	fmt.Println("\n🔨 构建 Schema Graph...")
	g := graph.NewSchemaGraph()

	// 创建规则引擎解释器
	ruleExplainer := analyzer.NewRuleBasedExplainer()

	// 添加表和列节点
	for _, table := range meta.Tables {
		// 表节点
		tableNode := &graph.Node{
			ID:   table.Name,
			Type: graph.NodeTypeTable,
			Name: table.Name,
			Properties: map[string]interface{}{
				"schema": table.Schema,
			},
		}
		g.AddNode(tableNode)

		// 列节点
		for _, col := range table.Columns {
			// 采样统计
			stats, _ := dbAdapter.SampleColumnStats(table.Name, col.Name, sampleSize)
			
			nullRatio := 0.0
			distinctRate := 0.0
			if stats != nil && stats.TotalRows > 0 {
				nullRatio = float64(stats.NullCount) / float64(stats.TotalRows)
				distinctRate = float64(stats.DistinctCount) / float64(stats.TotalRows)
			}

			// 使用规则引擎解释字段
			explanation := ruleExplainer.Explain(table.Name, col.Name, col.DataType, stats)

			colNode := &graph.Node{
				ID:   fmt.Sprintf("%s.%s", table.Name, col.Name),
				Type: graph.NodeTypeColumn,
				Name: col.Name,
				Properties: map[string]interface{}{
					"table":               table.Name,
					"data_type":           col.DataType,
					"length":              col.Length,
					"nullable":            col.Nullable,
					"is_primary_key":      col.IsPrimaryKey,
					"null_ratio":          nullRatio,
					"distinct_rate":       distinctRate,
					"ai_chinese_name":     explanation.ChineseName,
					"ai_description":      explanation.Description,
					"ai_business_meaning": explanation.BusinessMeaning,
					"ai_confidence":       explanation.Confidence,
					"ai_source":           "rule_based",
				},
			}
			g.AddNode(colNode)
		}
	}

	fmt.Println("✓ Graph 构建完成（已添加字段解释）")

	// 3. AI 增强分析（可选）
	if enableAI {
		runAIEnhancedAnalysis(dbAdapter, meta, g)
	}

	// 4. 推断关系
	fmt.Println("\n🔗 推断表间关系...")
	inferer := analyzer.NewRelationshipInferer(dbAdapter)
	edges, err := inferer.InferRelationships(meta)
	if err != nil {
		log.Printf("推断关系时出错: %v", err)
	} else {
		for _, edge := range edges {
			g.AddEdge(edge)
		}
		fmt.Printf("✓ 发现 %d 个推断关系\n", len(edges))
	}

	// 4. 检测枚举表
	fmt.Println("\n📋 检测枚举/码表...")
	enumDetector := analyzer.NewEnumDetector(dbAdapter)
	enumTables, err := enumDetector.DetectEnumTables(meta)
	if err != nil {
		log.Printf("检测枚举表时出错: %v", err)
	} else {
		fmt.Printf("✓ 发现 %d 个可能的枚举表\n", len(enumTables))
		for _, et := range enumTables {
			fmt.Printf("  - %s (行数: %d, 置信度: %.2f)\n", et.Name, et.RowCount, et.Confidence)
		}
	}

	// 6. 输出结果
	fmt.Println("\n📝 生成输出文件...")
	os.MkdirAll(outputDir, 0755)

	// JSON
	jsonData, _ := g.ToJSON()
	os.WriteFile(fmt.Sprintf("%s/schema.json", outputDir), jsonData, 0644)
	fmt.Printf("✓ %s/schema.json\n", outputDir)

	// Markdown 字典
	var mdContent string
	if enableAI {
		// 使用增强版渲染器
		mdRenderer := renderer.NewEnhancedMarkdownRenderer()
		mdContent = mdRenderer.Render(g)
	} else {
		mdRenderer := renderer.NewMarkdownRenderer()
		mdContent = mdRenderer.Render(g)
	}
	os.WriteFile(fmt.Sprintf("%s/dict.md", outputDir), []byte(mdContent), 0644)
	fmt.Printf("✓ %s/dict.md\n", outputDir)

	// Mermaid ER 图
	mermaidRenderer := renderer.NewMermaidRenderer()
	mermaidContent := mermaidRenderer.Render(g)
	os.WriteFile(fmt.Sprintf("%s/er.mmd", outputDir), []byte(mermaidContent), 0644)
	fmt.Printf("✓ %s/er.mmd\n", outputDir)

	fmt.Println("\n✅ 分析完成！")
}


// runAIEnhancedAnalysis 运行 AI 增强分析
func runAIEnhancedAnalysis(dbAdapter adapter.DBAdapter, meta *adapter.SchemaMetadata, g *graph.SchemaGraph) {
	fmt.Println("\n🤖 启用 AI 增强分析...")
	
	// 获取 API Key
	apiKey := aiAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("DASHSCOPE_API_KEY")
	}
	
	if apiKey == "" {
		fmt.Println("⚠️  未提供 API Key，跳过 AI 分析")
		fmt.Println("   提示：使用 --ai-key 或设置环境变量 DASHSCOPE_API_KEY")
		return
	}
	
	// 创建 AI 客户端
	aiClient := ai.NewAlibabaClient(apiKey)
	
	// 创建混合分析器
	hybridAnalyzer := analyzer.NewHybridAnalyzer(dbAdapter, aiClient)
	
	// 执行 AI 增强分析
	enhanced, err := hybridAnalyzer.AnalyzeWithAI(meta)
	if err != nil {
		fmt.Printf("⚠️  AI 分析失败: %v\n", err)
		return
	}
	
	// 将 AI 解释添加到 Graph 节点
	for tableName, table := range enhanced.Tables {
		for colName, col := range table.Columns {
			if col.Explanation != nil {
				nodeID := fmt.Sprintf("%s.%s", tableName, colName)
				if node := g.GetNode(nodeID); node != nil {
					node.Properties["ai_chinese_name"] = col.Explanation.ChineseName
					node.Properties["ai_description"] = col.Explanation.Description
					node.Properties["ai_business_meaning"] = col.Explanation.BusinessMeaning
					node.Properties["ai_confidence"] = col.Explanation.Confidence
					node.Properties["ai_source"] = col.Explanation.Source
				}
			}
		}
	}
	
	fmt.Println("✓ AI 分析完成")
	
	// 统计
	standardCount := 0
	inferredCount := 0
	relationCount := 0
	
	for _, table := range enhanced.Tables {
		for _, col := range table.Columns {
			if col.Explanation != nil {
				switch col.Explanation.Source {
				case "ai_standard":
					standardCount++
				case "ai_inferred":
					inferredCount++
				case "relation":
					relationCount++
				}
			}
		}
	}
	
	fmt.Printf("  - AI 直接识别: %d 个标准字段\n", standardCount)
	fmt.Printf("  - AI 推断: %d 个自定义字段\n", inferredCount)
	fmt.Printf("  - 关系推断: %d 个字段\n", relationCount)
}
