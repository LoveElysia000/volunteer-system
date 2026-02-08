
---

# 环保志愿者平台 AI 功能集成实施计划

## 📅 项目阶段规划

| 阶段 | 重点任务 | 目标产出 | 预计耗时 |
| --- | --- | --- | --- |
| **P1: 基础设施** | 引入 AI SDK，封装通用客户端 | `pkg/ai` 基础包完成 | 0.5 天 |
| **P2: 文案助手** | 实现组织方发布活动时的文案生成 | `/ai/generate-desc` 接口上线 | 1 天 |
| **P3: 数据改造** | 数据库支持向量存储，存量数据清洗 | `activities` 表结构变更 | 0.5 天 |
| **P4: 推荐引擎** | 实现基于 Embedding 的相似度匹配 | `/activities/recommend` 接口上线 | 1-2 天 |

---

## 🛠️ 第一阶段：基础设施建设 (Infrastructure)

### 1. 依赖安装

在项目根目录执行：

```bash
go get github.com/sashabaranov/go-openai

```

### 2. 配置文件更新

编辑 `config/config.yaml`，增加 AI 服务配置：

```yaml
ai:
  provider: "openai" # 或 deepseek
  api_key: "sk-xxxxxxxxxxxxxxxx"
  base_url: "https://api.openai.com/v1" # 国内模型请填对应 BaseUrl
  model_chat: "gpt-3.5-turbo"           # 或 deepseek-chat
  model_embedding: "text-embedding-ada-002"

```

### 3. 封装 AI 客户端 (`pkg/ai/client.go`)

封装一个单例客户端，用于统一管理调用。

```go
package ai

import (
    "context"
    openai "github.com/sashabaranov/go-openai"
)

type Client struct {
    c *openai.Client
}

// NewClient 初始化 AI 客户端
func NewClient(apiKey, baseURL string) *Client {
    cfg := openai.DefaultConfig(apiKey)
    if baseURL != "" {
        cfg.BaseURL = baseURL
    }
    return &Client{c: openai.NewClientWithConfig(cfg)}
}

// Chat 生成文本（用于文案）
func (a *Client) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
    resp, err := a.c.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model: openai.GPT3Dot5Turbo,
        Messages: []openai.ChatCompletionMessage{
            {Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
            {Role: openai.ChatMessageRoleUser, Content: userPrompt},
        },
    })
    if err != nil {
        return "", err
    }
    return resp.Choices[0].Message.Content, nil
}

// GetEmbedding 生成向量（用于推荐）
func (a *Client) GetEmbedding(ctx context.Context, text string) ([]float32, error) {
    resp, err := a.c.CreateEmbeddings(ctx, openai.EmbeddingRequest{
        Input: []string{text},
        Model: openai.AdaEmbeddingV2,
    })
    if err != nil {
        return nil, err
    }
    return resp.Data[0].Embedding, nil
}

```

---

## 🚀 第二阶段：活动文案辅助生成 (Copywriting)

### 1. 业务逻辑设计

* **输入**：活动标题、类型（如植树）、核心关键词。
* **Prompt 策略**：设定 AI 为“资深公益策划师”，要求语气热情、结构清晰。
* **输出**：一段 200-300 字的格式化文案。

### 2. Service 层实现 (`internal/service/ai_service.go`)

```go
func (s *AIService) GenerateActivityCopy(ctx context.Context, title, actType, keywords string) (string, error) {
    systemPrompt := `你是一名经验丰富的环保公益活动策划师。请根据用户提供的活动信息，撰写一段吸引人的活动详情介绍。
    要求：
    1. 语气热情、积极向上。
    2. 包含【活动背景】、【活动内容】、【期待您的加入】三个部分。
    3. 字数控制在 300 字以内。`

    userPrompt := fmt.Sprintf("活动标题：%s\n活动类型：%s\n关键词：%s", title, actType, keywords)

    return s.aiClient.Chat(ctx, systemPrompt, userPrompt)
}

```

### 3. Handler 层接入 (`internal/handler/ai_handler.go`)

```go
// POST /api/v1/ai/activity-copy
func GenerateCopy(ctx context.Context, c *app.RequestContext) {
    var req struct {
        Title    string `json:"title"`
        Type     string `json:"type"`
        Keywords string `json:"keywords"`
    }
    if err := c.BindAndValidate(&req); err != nil {
        c.JSON(400, utils.ErrorResponse(err))
        return
    }

    content, err := aiService.GenerateActivityCopy(ctx, req.Title, req.Type, req.Keywords)
    if err != nil {
        c.JSON(500, utils.ErrorResponse(err))
        return
    }

    c.JSON(200, utils.SuccessResponse(map[string]string{"content": content}))
}

```

---

## 🧠 第三阶段：智能推荐系统 (Recommendation)

此阶段无需引入向量数据库（如 Milvus），直接利用 **MySQL 8.0+ 的 JSON 存储** + **Go 内存计算**，适合 10 万级以下数据量，轻量高效。

### 1. 数据库变更 (Schema Change)

执行 SQL 脚本，为活动表增加向量字段：

```sql
ALTER TABLE activities 
ADD COLUMN embedding_vector JSON COMMENT 'AI特征向量(1536维)';

```

### 2. GORM 模型调整 (`internal/model/activity.go`)

使用 GORM 的 `serializer` 自动处理 JSON 序列化。

```go
type Activity struct {
    gorm.Model
    Title       string
    Description string
    // ... 其他字段
    
    // 新增字段：不需要手动解析 JSON，GORM 会自动处理
    EmbeddingVector []float32 `gorm:"type:json;serializer:json" json:"-"` 
}

```

### 3. 核心逻辑：发布活动时生成向量 (Write Path)

修改 `CreateActivity` 逻辑。

```go
// internal/service/activity_service.go

func (s *ActivityService) CreateActivity(ctx context.Context, act *model.Activity) error {
    // 1. 拼接用于向量化的文本 (标题 + 描述权重最高)
    inputText := fmt.Sprintf("%s。%s", act.Title, act.Description)
    
    // 2. 调用 AI 生成向量
    // 注意：实际生产中建议放入消息队列异步处理，防止阻塞 HTTP 响应
    vector, err := s.aiClient.GetEmbedding(ctx, inputText)
    if err == nil {
        act.EmbeddingVector = vector
    }
    
    // 3. 存入 MySQL
    return s.dao.CreateActivity(act)
}

```

### 4. 核心逻辑：获取推荐 (Read Path)

使用余弦相似度（Cosine Similarity）算法。

**工具函数 (`pkg/utils/math.go`)**:

```go
func CosineSimilarity(a, b []float32) float64 {
    if len(a) != len(b) || len(a) == 0 { return 0 }
    var dot, normA, normB float64
    for i := range a {
        dot += float64(a[i] * b[i])
        normA += float64(a[i] * a[i])
        normB += float64(b[i] * b[i])
    }
    if normA == 0 || normB == 0 { return 0 }
    return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

```

**推荐 Service (`internal/service/recommend_service.go`)**:

```go
func (s *RecommendService) RecommendActivities(ctx context.Context, userId uint) ([]*model.Activity, error) {
    // 1. 获取用户画像（可以是用户填写的兴趣标签，或者是用户过去参加过的活动标题拼接）
    userProfile := s.dao.GetUserInterestTags(userId) // e.g., "植树 海洋保护 周末"
    
    // 2. 生成用户当前的兴趣向量
    userVector, _ := s.aiClient.GetEmbedding(ctx, userProfile)
    
    // 3. 拉取所有已发布的活动（只查 ID 和 向量 以减少内存消耗）
    // 性能优化：如果数据量大，加上 Where("created_at > ?", lastMonth)
    var candidates []model.Activity
    s.db.Select("id", "title", "embedding_vector").Where("status = ?", "published").Find(&candidates)
    
    // 4. 内存计算相似度
    type Result struct {
        Act   *model.Activity
        Score float64
    }
    var results []Result
    
    for i := range candidates {
        if len(candidates[i].EmbeddingVector) == 0 { continue }
        score := utils.CosineSimilarity(userVector, candidates[i].EmbeddingVector)
        results = append(results, Result{Act: &candidates[i], Score: score})
    }
    
    // 5. 排序并取 Top 10
    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })
    
    // 6. 组装最终返回数据 (根据ID回查完整详情)
    // ...
    return topActivities, nil
}

```

---

## 📈 部署与优化建议

### 1. 环境变量安全

不要将 API Key 提交到 Git。在生产环境中，使用 Docker 环境变量注入：

```bash
docker run -e AI_API_KEY="sk-xxxx" my-volunteer-app

```

### 2. 存量数据处理

项目上线后，数据库里旧的活动没有向量数据。需要编写一个 `Task` 脚本：

1. 遍历 `activities` 表中 `embedding_vector` 为 NULL 的记录。
2. 循环调用 Embedding API。
3. 更新数据库。

### 3. 成本控制

Embedding 接口非常便宜，但 Chat 接口（文案生成）较贵。

* **缓存**：对于相同的输入参数，将 AI 的生成结果存入 Redis (TTL 24小时)，避免重复扣费。

### 4. 国内网络问题

如果服务器在国内，OpenAI 的 API 会超时。建议：

* 使用 **DeepSeek (深度求索)** 或 **阿里通义千问** 的 API（它们兼容 OpenAI SDK，只需换 BaseURL 和 Key）。
* 或者配置 HTTP 代理。