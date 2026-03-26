# `internal/router/assistant.go`

## 路由

### `POST /api/assistant/sessions`

- 鉴权：是
- 身份：志愿者/组织管理者
- 功能：创建 AI 会话
- 请求体：`{ scene:string, title:string }`
- 返回 `data`：`{ session_id:int64 }`

### `POST /api/assistant/chat`

- 鉴权：是
- 身份：志愿者/组织管理者
- 功能：发起 AI 对话
- 请求体：`{ session_id:int64, message:string, stream:bool }`
- 返回 `data`：`AssistantChatResponse`

### `POST /api/assistant/chat/stream`

- 鉴权：是
- 身份：志愿者/组织管理者
- 功能：发起 AI 流式对话
- 请求体：`{ session_id:int64, message:string, stream:bool }`
- 返回：`text/event-stream`

### `GET /api/assistant/sessions/:id/messages`

- 鉴权：是
- 身份：志愿者/组织管理者
- 功能：查看会话历史
- 路径参数：`id:int64`
- 返回 `data`：`{ list:AssistantMessageItem[] }`

### `POST /api/assistant/actions/activity-draft`

- 鉴权：是
- 身份：志愿者/组织管理者
- 功能：用 AI 快速生成活动草案
- 请求体：`{ session_id:int64, topic:string, target_people:string, location:string }`
- 返回 `data`：`AssistantActivityDraftActionResponse`

## 数据结构

### 请求消息

### `AssistantCreateSessionRequest`

- `scene:string`
- `title:string`

### `AssistantChatRequest`

- `session_id:int64`
- `message:string`
- `stream:bool`

### `AssistantSessionMessagesRequest`

- `id:int64`

### `AssistantActivityDraftActionRequest`

- `session_id:int64`
- `topic:string`
- `target_people:string`
- `location:string`

### `AssistantCreateSessionResponse`

- `session_id:int64`

### `AssistantChatResponse`

- `reply:string`
- `tool_calls:AssistantToolCall[]`
- `usage:AssistantUsage`

### `AssistantToolCall`

- `tool_name:string`
- `success:bool`
- `error_code:string`
- `error_msg:string`
- `latency_ms:int32`
- `input:string`
- `output:string`

### `AssistantUsage`

- `model:string`
- `token_in:int32`
- `token_out:int32`
- `latency_ms:int32`

### `AssistantMessageItem`

- `id:int64`
- `session_id:int64`
- `seq_no:int32`
- `role:int32`
- `content:string`
- `model:string`
- `finish_reason:int32`
- `token_in:int32`
- `token_out:int32`
- `latency_ms:int32`
- `request_id:string`
- `created_at:string`

### `AssistantActivityDraftActionResponse`

- `session_id:int64`
- `result:AssistantChatResponse`

### `AssistantSessionMessagesResponse`

- `list:AssistantMessageItem[]`

## SSE 事件

- `start`：会话开始
- `message` / 业务事件：由 `service.BuildAssistantStreamEvents` 生成
- `error`：错误信息
- `done`：流结束
