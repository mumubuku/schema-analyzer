// 数据库类型切换
document.getElementById('dbType').addEventListener('change', function() {
    const port = document.getElementById('port');
    const schemaGroup = document.getElementById('schemaGroup');
    const schema = document.getElementById('schema');
    const database = document.getElementById('database');
    
    if (this.value === 'mysql') {
        port.value = '3306';
        schemaGroup.style.display = 'block';
        // 自动同步 schema 和 database
        schema.value = database.value;
    } else {
        port.value = '1433';
        schemaGroup.style.display = 'none';
    }
});

// MySQL 数据库名变化时，自动同步 schema
document.getElementById('database').addEventListener('input', function() {
    if (document.getElementById('dbType').value === 'mysql') {
        document.getElementById('schema').value = this.value;
    }
});

// AI 开关
document.getElementById('enableAI').addEventListener('change', function() {
    const apiKeyGroup = document.getElementById('apiKeyGroup');
    apiKeyGroup.style.display = this.checked ? 'block' : 'none';
});

// 表单提交
document.getElementById('analysisForm').addEventListener('submit', async function(e) {
    e.preventDefault();
    
    const formData = {
        db_type: document.getElementById('dbType').value,
        host: document.getElementById('host').value,
        port: document.getElementById('port').value,
        username: document.getElementById('username').value,
        password: document.getElementById('password').value,
        database: document.getElementById('database').value,
        schema: document.getElementById('schema').value,
        sample_size: parseInt(document.getElementById('sampleSize').value),
        enable_ai: document.getElementById('enableAI').checked,
        api_key: document.getElementById('apiKey').value
    };
    
    // 显示进度条
    document.getElementById('progressContainer').style.display = 'block';
    document.getElementById('resultContainer').style.display = 'none';
    document.getElementById('submitBtn').disabled = true;
    
    try {
        // 提交分析请求
        const response = await fetch('/api/analyze', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(formData)
        });
        
        const data = await response.json();
        const taskId = data.task_id;
        
        // 通过 WebSocket 监听进度
        monitorProgress(taskId);
        
    } catch (error) {
        alert('提交失败: ' + error.message);
        document.getElementById('submitBtn').disabled = false;
    }
});

// 监听进度
function monitorProgress(taskId) {
    const ws = new WebSocket(`ws://${window.location.host}/api/ws?task_id=${taskId}`);
    
    ws.onmessage = function(event) {
        const task = JSON.parse(event.data);
        
        // 更新进度条
        const progressFill = document.getElementById('progressFill');
        const progressMessage = document.getElementById('progressMessage');
        
        const progress = task.progress || 0;
        progressFill.style.width = progress + '%';
        progressFill.textContent = progress + '%';
        progressMessage.textContent = task.message || '处理中...';
        
        // 完成或失败
        if (task.status === 'completed') {
            displayResult(task.result);
            document.getElementById('submitBtn').disabled = false;
            ws.close();
        } else if (task.status === 'failed') {
            alert('分析失败: ' + task.message);
            document.getElementById('progressContainer').style.display = 'none';
            document.getElementById('submitBtn').disabled = false;
            ws.close();
        }
    };
    
    ws.onerror = function(error) {
        console.error('WebSocket error:', error);
        // 降级到轮询
        pollProgress(taskId);
    };
}

// 轮询进度（WebSocket 失败时的降级方案）
async function pollProgress(taskId) {
    const interval = setInterval(async () => {
        try {
            const response = await fetch(`/api/task/${taskId}`);
            const task = await response.json();
            
            // 更新进度
            const progressFill = document.getElementById('progressFill');
            const progressMessage = document.getElementById('progressMessage');
            
            const progress = task.progress || 0;
            progressFill.style.width = progress + '%';
            progressFill.textContent = progress + '%';
            progressMessage.textContent = task.message || '处理中...';
            
            if (task.status === 'completed') {
                clearInterval(interval);
                displayResult(task.result);
                document.getElementById('submitBtn').disabled = false;
            } else if (task.status === 'failed') {
                clearInterval(interval);
                alert('分析失败: ' + task.message);
                document.getElementById('progressContainer').style.display = 'none';
                document.getElementById('submitBtn').disabled = false;
            }
        } catch (error) {
            console.error('Poll error:', error);
        }
    }, 1000);
}

// 显示结果
function displayResult(result) {
    document.getElementById('resultContainer').style.display = 'block';
    
    // 统计信息
    const statsHTML = `
        <div class="stat-card">
            <h3>${result.stats.tables}</h3>
            <p>数据表</p>
        </div>
        <div class="stat-card">
            <h3>${result.stats.relations}</h3>
            <p>推断关系</p>
        </div>
        <div class="stat-card">
            <h3>${result.stats.enum_tables}</h3>
            <p>枚举表</p>
        </div>
    `;
    document.getElementById('stats').innerHTML = statsHTML;
    
    // 数据字典（Markdown 转 HTML）
    document.getElementById('dictResult').innerHTML = markdownToHTML(result.dict_md);
    
    // ER 图
    document.getElementById('erResult').textContent = result.er_mermaid;
    
    // JSON
    document.getElementById('jsonResult').textContent = JSON.stringify(JSON.parse(result.schema_json), null, 2);
    
    // 滚动到结果
    document.getElementById('resultContainer').scrollIntoView({ behavior: 'smooth' });
}

// 切换标签
function switchTab(tabName) {
    // 移除所有 active
    document.querySelectorAll('.tab').forEach(tab => tab.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));
    
    // 添加 active
    event.target.classList.add('active');
    document.getElementById(tabName + '-content').classList.add('active');
}

// 简单的 Markdown 转 HTML
function markdownToHTML(markdown) {
    let html = markdown;
    
    // 标题
    html = html.replace(/^### (.*$)/gim, '<h3>$1</h3>');
    html = html.replace(/^## (.*$)/gim, '<h2>$1</h2>');
    html = html.replace(/^# (.*$)/gim, '<h1>$1</h1>');
    
    // 粗体
    html = html.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
    
    // 代码块
    html = html.replace(/`([^`]+)`/g, '<code>$1</code>');
    
    // 表格（简单处理）
    html = html.replace(/\|/g, '</td><td>');
    html = html.replace(/<td>(.*?)<\/td>/g, function(match, p1) {
        if (p1.includes('---')) {
            return '';
        }
        return match;
    });
    
    // 换行
    html = html.replace(/\n/g, '<br>');
    
    return html;
}


// 测试连接
async function testConnection() {
    const data = {
        db_type: document.getElementById('dbType').value,
        host: document.getElementById('host').value,
        port: document.getElementById('port').value,
        username: document.getElementById('username').value,
        password: document.getElementById('password').value
    };
    
    try {
        const response = await fetch('/api/test-connection', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        
        const result = await response.json();
        
        if (result.success) {
            alert('✅ ' + result.message);
        } else {
            alert('❌ ' + result.message);
        }
    } catch (error) {
        alert('❌ 测试失败: ' + error.message);
    }
}

// 列出数据库
async function listDatabases() {
    const data = {
        db_type: document.getElementById('dbType').value,
        host: document.getElementById('host').value,
        port: document.getElementById('port').value,
        username: document.getElementById('username').value,
        password: document.getElementById('password').value
    };
    
    try {
        const response = await fetch('/api/list-databases', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        
        const result = await response.json();
        
        if (result.success && result.databases) {
            // 创建选择对话框
            let message = '请选择要分析的数据库：\n\n';
            result.databases.forEach((db, index) => {
                message += `${index + 1}. ${db}\n`;
            });
            message += '\n输入序号（1-' + result.databases.length + '）：';
            
            const choice = prompt(message);
            if (choice) {
                const index = parseInt(choice) - 1;
                if (index >= 0 && index < result.databases.length) {
                    const selectedDB = result.databases[index];
                    document.getElementById('database').value = selectedDB;
                    
                    // 如果是 MySQL，自动填充 schema
                    if (data.db_type === 'mysql') {
                        document.getElementById('schema').value = selectedDB;
                    }
                    
                    alert('✅ 已选择数据库: ' + selectedDB);
                }
            }
        } else {
            alert('❌ 获取数据库列表失败');
        }
    } catch (error) {
        alert('❌ 获取失败: ' + error.message);
    }
}
