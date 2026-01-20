package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"schema-analyzer/internal/adapter"
	"schema-analyzer/internal/ai"
	"schema-analyzer/internal/analyzer"
	"schema-analyzer/internal/graph"
	"schema-analyzer/internal/renderer"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// AnalysisRequest 分析请求
type AnalysisRequest struct {
	DBType     string `json:"db_type"`     // sqlserver/mysql
	Host       string `json:"host"`        // 主机地址
	Port       string `json:"port"`        // 端口
	Username   string `json:"username"`    // 用户名
	Password   string `json:"password"`    // 密码
	Database   string `json:"database"`    // 数据库名
	Schema     string `json:"schema"`      // Schema（MySQL需要）
	SampleSize int    `json:"sample_size"` // 采样大小
	EnableAI   bool   `json:"enable_ai"`   // 是否启用AI
	APIKey     string `json:"api_key"`     // AI API Key
}

// AnalysisTask 分析任务
type AnalysisTask struct {
	ID        string
	Request   AnalysisRequest
	Status    string // pending/running/completed/failed
	Progress  int    // 0-100
	Message   string
	Result    *AnalysisResult
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	SchemaJSON string            `json:"schema_json"`
	DictMD     string            `json:"dict_md"`
	ErMermaid  string            `json:"er_mermaid"`
	Stats      map[string]int    `json:"stats"`
}

var (
	tasks   = make(map[string]*AnalysisTask)
	tasksMu sync.RWMutex
)

func main() {
	// 静态文件
	http.Handle("/", http.FileServer(http.Dir("web/static")))
	
	// API 路由
	http.HandleFunc("/api/analyze", handleAnalyze)
	http.HandleFunc("/api/task/", handleTaskStatus)
	http.HandleFunc("/api/ws", handleWebSocket)
	http.HandleFunc("/api/test-connection", handleTestConnection)
	http.HandleFunc("/api/list-databases", handleListDatabases)
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	fmt.Printf("🚀 Schema Analyzer Web Server\n")
	fmt.Printf("📡 服务地址: http://localhost:%s\n", port)
	fmt.Printf("📊 打开浏览器访问上述地址开始分析\n\n")
	
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// handleAnalyze 处理分析请求
func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req AnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// 创建任务
	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
	task := &AnalysisTask{
		ID:        taskID,
		Request:   req,
		Status:    "pending",
		Progress:  0,
		Message:   "任务已创建，等待执行...",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	tasksMu.Lock()
	tasks[taskID] = task
	tasksMu.Unlock()
	
	// 异步执行分析
	go runAnalysis(task)
	
	// 返回任务ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"task_id": taskID,
		"status":  "pending",
	})
}

// handleTaskStatus 查询任务状态
func handleTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskID := filepath.Base(r.URL.Path)
	
	tasksMu.RLock()
	task, exists := tasks[taskID]
	tasksMu.RUnlock()
	
	if !exists {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// handleWebSocket WebSocket 连接
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()
	
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		return
	}
	
	// 持续推送任务状态
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	for range ticker.C {
		tasksMu.RLock()
		task, exists := tasks[taskID]
		tasksMu.RUnlock()
		
		if !exists {
			break
		}
		
		if err := conn.WriteJSON(task); err != nil {
			break
		}
		
		if task.Status == "completed" || task.Status == "failed" {
			break
		}
	}
}

// runAnalysis 执行分析
func runAnalysis(task *AnalysisTask) {
	updateTask := func(status string, progress int, message string) {
		tasksMu.Lock()
		task.Status = status
		task.Progress = progress
		task.Message = message
		task.UpdatedAt = time.Now()
		tasksMu.Unlock()
	}
	
	updateTask("running", 10, "正在连接数据库...")
	
	// 构建连接字符串
	var connStr string
	var dbAdapter adapter.DBAdapter
	var err error
	
	req := task.Request
	
	switch req.DBType {
	case "sqlserver":
		connStr = fmt.Sprintf("server=%s;port=%s;user id=%s;password=%s;database=%s",
			req.Host, req.Port, req.Username, req.Password, req.Database)
		dbAdapter, err = adapter.NewSQLServerAdapter(connStr)
	case "mysql":
		connStr = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?timeout=30s&readTimeout=30s&writeTimeout=30s",
			req.Username, req.Password, req.Host, req.Port, req.Database)
		dbAdapter, err = adapter.NewMySQLAdapter(connStr, req.Schema)
	default:
		updateTask("failed", 0, "不支持的数据库类型")
		return
	}
	
	if err != nil {
		updateTask("failed", 0, fmt.Sprintf("连接失败: %v", err))
		return
	}
	defer dbAdapter.Close()
	
	updateTask("running", 20, "获取数据库元数据...")
	
	// 获取元数据
	meta, err := dbAdapter.IntrospectSchema()
	if err != nil {
		updateTask("failed", 20, fmt.Sprintf("获取元数据失败: %v", err))
		return
	}
	
	updateTask("running", 40, fmt.Sprintf("发现 %d 个表，构建 Schema Graph...", len(meta.Tables)))
	
	// 构建 Graph
	g := graph.NewSchemaGraph()
	
	sampleSize := req.SampleSize
	if sampleSize == 0 {
		sampleSize = 1000
	}
	
	for i, table := range meta.Tables {
		progress := 40 + int(float64(i)/float64(len(meta.Tables))*20)
		updateTask("running", progress, fmt.Sprintf("分析表 %s (%d/%d)...", table.Name, i+1, len(meta.Tables)))
		
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
			stats, _ := dbAdapter.SampleColumnStats(table.Name, col.Name, sampleSize)
			
			nullRatio := 0.0
			distinctRate := 0.0
			if stats != nil && stats.TotalRows > 0 {
				nullRatio = float64(stats.NullCount) / float64(stats.TotalRows)
				distinctRate = float64(stats.DistinctCount) / float64(stats.TotalRows)
			}
			
			colNode := &graph.Node{
				ID:   fmt.Sprintf("%s.%s", table.Name, col.Name),
				Type: graph.NodeTypeColumn,
				Name: col.Name,
				Properties: map[string]interface{}{
					"table":          table.Name,
					"data_type":      col.DataType,
					"length":         col.Length,
					"nullable":       col.Nullable,
					"is_primary_key": col.IsPrimaryKey,
					"null_ratio":     nullRatio,
					"distinct_rate":  distinctRate,
				},
			}
			g.AddNode(colNode)
		}
	}
	
	// AI 增强
	if req.EnableAI && req.APIKey != "" {
		updateTask("running", 60, "AI 增强分析中...")
		
		aiClient := ai.NewAlibabaClient(req.APIKey)
		hybridAnalyzer := analyzer.NewHybridAnalyzer(dbAdapter, aiClient)
		
		enhanced, err := hybridAnalyzer.AnalyzeWithAI(meta)
		if err == nil {
			// 更新节点
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
		}
	}
	
	updateTask("running", 70, "推断表间关系...")
	
	// 推断关系
	inferer := analyzer.NewRelationshipInferer(dbAdapter)
	edges, _ := inferer.InferRelationships(meta)
	for _, edge := range edges {
		g.AddEdge(edge)
	}
	
	updateTask("running", 85, "检测枚举表...")
	
	// 检测枚举表
	enumDetector := analyzer.NewEnumDetector(dbAdapter)
	enumTables, _ := enumDetector.DetectEnumTables(meta)
	
	updateTask("running", 95, "生成输出...")
	
	// 生成输出
	schemaJSON, _ := g.ToJSON()
	
	mdRenderer := renderer.NewEnhancedMarkdownRenderer()
	dictMD := mdRenderer.Render(g)
	
	mermaidRenderer := renderer.NewMermaidRenderer()
	erMermaid := mermaidRenderer.Render(g)
	
	// 保存结果
	result := &AnalysisResult{
		SchemaJSON: string(schemaJSON),
		DictMD:     dictMD,
		ErMermaid:  erMermaid,
		Stats: map[string]int{
			"tables":      len(meta.Tables),
			"relations":   len(edges),
			"enum_tables": len(enumTables),
		},
	}
	
	tasksMu.Lock()
	task.Result = result
	tasksMu.Unlock()
	
	updateTask("completed", 100, "分析完成！")
}


// handleTestConnection 测试数据库连接
func handleTestConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		DBType   string `json:"db_type"`
		Host     string `json:"host"`
		Port     string `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	var connStr string
	var db *sql.DB
	var err error
	
	switch req.DBType {
	case "sqlserver":
		connStr = fmt.Sprintf("server=%s;port=%s;user id=%s;password=%s",
			req.Host, req.Port, req.Username, req.Password)
		db, err = sql.Open("sqlserver", connStr)
	case "mysql":
		connStr = fmt.Sprintf("%s:%s@tcp(%s:%s)/?timeout=10s",
			req.Username, req.Password, req.Host, req.Port)
		db, err = sql.Open("mysql", connStr)
	default:
		http.Error(w, "Unsupported database type", http.StatusBadRequest)
		return
	}
	
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("连接失败: %v", err),
		})
		return
	}
	defer db.Close()
	
	if err := db.Ping(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("连接失败: %v", err),
		})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "连接成功！",
	})
}

// handleListDatabases 列出所有数据库
func handleListDatabases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		DBType   string `json:"db_type"`
		Host     string `json:"host"`
		Port     string `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	var connStr string
	var db *sql.DB
	var err error
	
	switch req.DBType {
	case "sqlserver":
		connStr = fmt.Sprintf("server=%s;port=%s;user id=%s;password=%s",
			req.Host, req.Port, req.Username, req.Password)
		db, err = sql.Open("sqlserver", connStr)
	case "mysql":
		connStr = fmt.Sprintf("%s:%s@tcp(%s:%s)/?timeout=10s",
			req.Username, req.Password, req.Host, req.Port)
		db, err = sql.Open("mysql", connStr)
	default:
		http.Error(w, "Unsupported database type", http.StatusBadRequest)
		return
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()
	
	var query string
	if req.DBType == "mysql" {
		query = "SHOW DATABASES"
	} else {
		query = "SELECT name FROM sys.databases WHERE name NOT IN ('master', 'tempdb', 'model', 'msdb')"
	}
	
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			continue
		}
		// 过滤系统数据库
		if req.DBType == "mysql" {
			if dbName == "information_schema" || dbName == "mysql" || 
			   dbName == "performance_schema" || dbName == "sys" {
				continue
			}
		}
		databases = append(databases, dbName)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"databases": databases,
	})
}
