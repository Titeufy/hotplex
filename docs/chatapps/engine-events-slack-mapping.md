# Engine Events → Slack Block Kit 映射最佳实践

> **状态**: ✅ 已实现  
> **最后更新**: 2026-02-26  
> **相关 Issue**: [#38](https://github.com/hrygo/hotplex/issues/38)  
> **实现文件**: `chatapps/slack/block_builder.go` (900 行)

---

## 📋 目录

- [概述](#概述)
- [Engine Events 完整类型](#engine-events-完整类型)
- [Slack Block Kit 映射方案](#slack-block-kit-映射方案)
- [事件聚合策略](#事件聚合策略)
- [Block Builder API 参考](#block-builder-api-参考)
- [mrkdwn 格式转换](#mrkdwn-格式转换)
- [最佳实践与限制](#最佳实践与限制)
- [实现示例](#实现示例)

---

## 概述

本文档定义了 HotPlex Engine Events 到 Slack Block Kit 的完整映射方案，确保 AI 代理的执行过程在 Slack 中以最佳 UX/UI 形式展现。

### 核心设计原则

| 原则 | 说明 |
|------|------|
| **即时反馈** | Thinking、Error 等关键事件立即发送，不聚合 |
| **同类聚合** | Tool Use/Result 等相似事件可合并展示 |
| **流式更新** | Answer 事件使用 `chat.update` API 节流更新 |
| **丰富上下文** | 使用 Context Blocks 展示元数据（时长、Token 等） |
| **交互友好** | 长输出提供"View Full Output"按钮 |

### 架构位置

```
┌─────────────────────────────────────────────────────────┐
│  Engine Layer (provider/event.go)                       │
│  - EventTypeThinking, EventTypeAnswer, ...              │
└────────────────────┬────────────────────────────────────┘
                     │ ProviderEvent
                     ▼
┌─────────────────────────────────────────────────────────┐
│  ChatApps Layer (chatapps/engine_handler.go)            │
│  - StreamCallback.Handle()                              │
│  - handleThinking(), handleAnswer(), ...                │
└────────────────────┬────────────────────────────────────┘
                     │ EventWithMeta
                     ▼
┌─────────────────────────────────────────────────────────┐
│  Block Builder (chatapps/slack/block_builder.go)        │
│  - BuildThinkingBlock() → Context Block                 │
│  - BuildAnswerBlock() → Section Block                   │
│  - BuildToolUseBlock() → Section + Fields               │
│  - BuildToolResultBlock() → Section + Actions           │
│  - BuildErrorBlock() → Section (danger)                 │
│  - BuildSessionStatsBlock() → Header + Section          │
└────────────────────┬────────────────────────────────────┘
                     │ []map[string]any
                     ▼
┌─────────────────────────────────────────────────────────┐
│  Slack API (chat.postMessage / chat.update)             │
└─────────────────────────────────────────────────────────┘
```

---

## Engine Events 完整类型

### ProviderEventType 枚举 (provider/event.go)

| 事件类型 | 值 | 描述 | 触发时机 |
|---------|-----|------|---------|
| `EventTypeThinking` | `"thinking"` | AI 推理中 | CLI 输出 `type="thinking"` 或 `type="status"` |
| `EventTypeAnswer` | `"answer"` | AI 文本输出 | CLI 输出 `type="assistant"` 带 text blocks |
| `EventTypeToolUse` | `"tool_use"` | 工具调用开始 | CLI 输出 `type="tool_use"` |
| `EventTypeToolResult` | `"tool_result"` | 工具执行结果 | CLI 输出 `type="tool_result"` |
| `EventTypeError` | `"error"` | 错误发生 | CLI 输出 `type="error"` 或 Provider 解析错误 |
| `EventTypeResult` | `"result"` | Turn 完成 | CLI 输出 `type="result"` (最终统计) |
| `EventTypeSystem` | `"system"` | 系统消息 | CLI 系统通知 (通常过滤) |
| `EventTypeUser` | `"user"` | 用户消息 | CLI 反射用户输入 (通常过滤) |
| `EventTypeStepStart` | `"step_start"` | 步骤开始 | OpenCode `Part.Type="step-start"` |
| `EventTypeStepFinish` | `"step_finish"` | 步骤完成 | OpenCode `Part.Type="step-finish"` |
| `EventTypeRaw` | `"raw"` | 原始输出 | 非 JSON 格式输出 (fallback) |
| `EventTypePermissionRequest` | `"permission_request"` | 权限请求 | Claude Code 请求用户审批 |

### 扩展事件类型 (Engine 内部)

| 事件类型 | 值 | 描述 | 来源 |
|---------|-----|------|------|
| `danger_block` | `"danger_block"` | 安全 WAF 拦截 | `internal/security/detector.go` |
| `session_stats` | `"session_stats"` | 会话统计 | `event.SessionStatsData` |

### EventMeta 字段结构 (event/events.go)

```go
type EventMeta struct {
    // 时长
    DurationMs      int64 `json:"duration_ms"`       // 事件执行时长
    TotalDurationMs int64 `json:"total_duration_ms"` // 总会话时长
    
    // 工具信息
    ToolName string `json:"tool_name"` // 工具名称 (e.g., "Bash", "Editor")
    ToolID   string `json:"tool_id"`   // 工具调用唯一 ID
    Status   string `json:"status"`    // "running" | "success" | "error"
    ErrorMsg string `json:"error_msg"` // 错误消息 (status=error 时)
    
    // Token 使用
    InputTokens      int32 `json:"input_tokens"`
    OutputTokens     int32 `json:"output_tokens"`
    CacheWriteTokens int32 `json:"cache_write_tokens"`
    CacheReadTokens  int32 `json:"cache_read_tokens"`
    
    // 摘要 (用于展示)
    InputSummary  string `json:"input_summary"`  // 输入的人类可读摘要
    OutputSummary string `json:"output_summary"` // 输出截断预览
    
    // 文件操作
    FilePath  string `json:"file_path"`  // 影响文件路径
    LineCount int32  `json:"line_count"` // 影响行数
    
    // 进度追踪
    Progress    int32 `json:"progress"`     // 0-100
    TotalSteps  int32 `json:"total_steps"`  // 总步骤数
    CurrentStep int32 `json:"current_step"` // 当前步骤
}
```

---

## Slack Block Kit 映射方案

### 1. Thinking 事件

**Block 类型**: `context`  
**聚合策略**: ❌ 不聚合 - 立即发送  
**UI 目标**: 即时反馈，让用户知道 AI 正在思考

```json
{
  "type": "context",
  "elements": [{
    "type": "mrkdwn",
    "text": ":brain: _Thinking..._"
  }]
}
```

**Go 实现** (`block_builder.go:256-271`):

```go
func (b *BlockBuilder) BuildThinkingBlock(content string) []map[string]any {
    displayText := content
    if displayText == "" {
        displayText = "Thinking..."
    }
    return []map[string]any{
        {
            "type": "context",
            "elements": []map[string]any{
                mrkdwnText(fmt.Sprintf(":brain: _%s_", displayText)),
            },
        },
    }
}
```

**设计说明**:
- 使用 `context` block 而非 `section`，视觉权重较低
- Emoji `:brain:` 提供即时语义识别
- 斜体 `_text_` 表示临时状态
- 支持流式更新（多次 thinking 事件更新同一消息）

---

### 2. Tool Use 事件

**Block 类型**: `section` + `fields`  
**聚合策略**: ✅ 同类聚合 - 多个工具可合并  
**UI 目标**: 清晰展示工具名称和输入参数

```json
{
  "type": "section",
  "text": {
    "type": "mrkdwn",
    "text": ":hammer_and_wrench: *Using tool:* `Bash`"
  },
  "fields": [{
    "type": "mrkdwn",
    "text": "*Input:*\n```ls -la```"
  }]
}
```

**Go 实现** (`block_builder.go:276-294`):

```go
func (b *BlockBuilder) BuildToolUseBlock(toolName, input string, truncated bool) []map[string]any {
    formattedInput := fmt.Sprintf("```%s```", input)
    if truncated {
        formattedInput += "\n*_Output truncated..._*"
    }
    return []map[string]any{
        {
            "type": "section",
            "text": mrkdwnText(fmt.Sprintf(":hammer_and_wrench: *Using tool:* `%s`", toolName)),
            "fields": []map[string]any{
                mrkdwnText("*Input:*\n" + formattedInput),
            },
        },
    }
}
```

**设计说明**:
- Emoji `:hammer_and_wrench:` 直观标识工具操作
- 工具名用反引号包裹，突出显示
- 输入内容用代码块展示，保持格式
- 超过 100 字符自动截断，避免消息过长

---

### 3. Tool Result 事件

**Block 类型**: `section` + `context` + `actions` (可选)  
**聚合策略**: ✅ 同类聚合  
**UI 目标**: 展示执行状态、时长、输出预览

#### 成功结果

```json
{
  "type": "section",
  "text": {
    "type": "mrkdwn",
    "text": ":white_check_mark: *Completed*"
  },
  "fields": [{
    "type": "mrkdwn",
    "text": "*Output:*\n```\ncommand output...\n```"
  }]
},
{
  "type": "context",
  "elements": [{
    "type": "mrkdwn",
    "text": ":timer_clock: *Duration:* 1.2s"
  }]
},
{
  "type": "actions",
  "elements": [{
    "type": "button",
    "text": {"type": "plain_text", "text": "View Full Output"},
    "action_id": "view_tool_output",
    "value": "expand_output"
  }]
}
```

#### 失败结果

```json
{
  "type": "section",
  "text": {
    "type": "mrkdwn",
    "text": ":x: *Failed*"
  },
  "fields": [{
    "type": "mrkdwn",
    "text": "*Error:*\n```\nerror message...\n```"
  }]
}
```

**Go 实现** (`block_builder.go:299-356`):

```go
func (b *BlockBuilder) BuildToolResultBlock(
    success bool, 
    durationMs int64, 
    output string, 
    hasButton bool,
) []map[string]any {
    var blocks []map[string]any
    
    // 1. Status block
    statusEmoji := ":white_check_mark:"
    statusText := "*Completed*"
    if !success {
        statusEmoji = ":x:"
        statusText = "*Failed*"
    }
    
    resultBlock := map[string]any{
        "type": "section",
        "text": mrkdwnText(fmt.Sprintf("%s %s", statusEmoji, statusText)),
    }
    
    // 2. Output preview (truncated to 300 chars)
    if output != "" {
        previewLen := 300
        preview := output
        if len(output) > previewLen {
            preview = output[:previewLen] + "..."
        }
        resultBlock["fields"] = []map[string]any{
            mrkdwnText("*Output:*\n```\n" + preview + "\n```"),
        }
    }
    blocks = append(blocks, resultBlock)
    
    // 3. Duration context
    if durationMs > 0 {
        blocks = append(blocks, map[string]any{
            "type": "context",
            "elements": []map[string]any{
                mrkdwnText(fmt.Sprintf(":timer_clock: *Duration:* %s", formatDuration(durationMs))),
            },
        })
    }
    
    // 4. Action button (optional)
    if hasButton && success {
        blocks = append(blocks, map[string]any{
            "type": "actions",
            "elements": []map[string]any{
                {
                    "type": "button",
                    "text": plainText("View Full Output"),
                    "action_id": "view_tool_output",
                    "value": "expand_output",
                },
            },
        })
    }
    
    return blocks
}
```

**设计说明**:
- 成功/失败使用不同 Emoji，快速视觉识别
- 输出预览限制 300 字符，避免消息过长
- 时长信息放在 `context` block，视觉层次分明
- "View Full Output" 按钮需配合交互式组件实现

---

### 4. Answer 事件

**Block 类型**: `section` (mrkdwn)  
**聚合策略**: ✅ 流式更新 - 使用 `chat.update` API  
**UI 目标**: 展示 AI 的最终回答，支持 Markdown 格式

```json
{
  "type": "section",
  "text": {
    "type": "mrkdwn",
    "text": "Here's the solution:\n\n```go\nfunc Hello() { ... }\n```\n\nThis code..."
  },
  "expand": true
}
```

**Go 实现** (`block_builder.go:389-402`):

```go
func (b *BlockBuilder) BuildAnswerBlock(content string) []map[string]any {
    formatter := NewMrkdwnFormatter()
    formattedContent := formatter.Format(content)
    return []map[string]any{
        {
            "type": "section",
            "text": mrkdwnText(formattedContent),
            "expand": true, // Enable expand for AI Assistant apps
        },
    }
}
```

**Markdown → mrkdwn 转换** (`block_builder.go:54-239`):

| Markdown | Slack mrkdwn | 实现方法 |
|---------|--------------|---------|
| `**bold**` | `*bold*` | `convertBold()` |
| `*italic*` | `_italic_` | `convertItalic()` |
| `~~strikethrough~~` | `~strikethrough~` | `convertStrikethrough()` |
| `[text](url)` | `<url|text>` | `convertLinks()` |
| ``` `code` ``` | ``` `code` ``` | 原生支持 |
| ``` ```lang\ncode\n``` ``` | ``` ```lang\ncode\n``` ``` | `FormatCodeBlock()` |

**流式更新策略** (`engine_handler.go:576-636`):

```go
func (s *StreamState) updateThrottled(
    ctx context.Context, 
    adapters *AdapterManager, 
    platform, sessionID, content string, 
    blockBuilder *slack.BlockBuilder, 
    metadata map[string]any,
) error {
    s.mu.Lock()
    
    // 节流：每秒最多更新 1 次
    if time.Since(s.LastUpdated) < time.Second {
        s.mu.Unlock()
        return nil
    }
    s.LastUpdated = time.Time{} // Mark as updating
    s.mu.Unlock()
    
    // Build blocks with content
    blocks := blockBuilder.BuildAnswerBlock(content)
    
    // Send update via chat.update API
    // ...
}
```

**设计说明**:
- Markdown 转 mrkdwn 保持格式一致性
- `expand: true` 启用 Slack AI Assistant 的展开功能
- 节流更新 (1 次/秒) 避免触发 Slack rate limit
- 使用 `chat.update` 而非重复发送，保持消息线程整洁

---

### 5. Error 事件

**Block 类型**: `section` (danger style)  
**聚合策略**: ❌ 不聚合 - 立即发送  
**UI 目标**: 醒目的错误提示

```json
{
  "type": "section",
  "text": {
    "type": "mrkdwn",
    "text": "*:warning: Execution Error*\n\n> Error message details..."
  }
}
```

**Go 实现** (`block_builder.go:361-384`):

```go
func (b *BlockBuilder) BuildErrorBlock(message string, isDangerBlock bool) []map[string]any {
    var blocks []map[string]any
    
    headerEmoji := ":warning:"
    if isDangerBlock {
        headerEmoji = ":x:"
    }
    
    // 1. Header block
    headerBlock := map[string]any{
        "type": "section",
        "text": mrkdwnText(fmt.Sprintf("*%s Execution Error*", headerEmoji)),
    }
    blocks = append(blocks, headerBlock)
    
    // 2. Error message (quoted style)
    errorBlock := map[string]any{
        "type": "section",
        "text": mrkdwnText(fmt.Sprintf("> %s", message)),
    }
    blocks = append(blocks, errorBlock)
    
    return blocks
}
```

**设计说明**:
- `:warning:` 用于普通错误，`:x:` 用于 danger_block
- 错误消息使用 `>` 引用格式，视觉区分
- 不使用 `header` block (不支持 mrkdwn)
- 立即发送，不等待聚合

---

### 6. Danger Block 事件

**Block 类型**: `section` (danger style)  
**聚合策略**: ❌ 不聚合 - 立即发送  
**UI 目标**: 安全 WAF 拦截警告

```json
{
  "type": "section",
  "text": {
    "type": "mrkdwn",
    "text": "*:x: Security Blocked*\n\n> Command `rm -rf /` is not allowed"
  }
}
```

**Go 实现**: 同 `BuildErrorBlock()`，传入 `isDangerBlock=true`

**触发条件** (`internal/security/detector.go`):
- `rm -rf /`
- `mkfs`
- `dd if=/dev/zero`
- 其他危险命令模式

---

### 7. Session Stats 事件

**Block 类型**: `header` + `section` + `context`  
**聚合策略**: 最后发送 - 会话总结  
**UI 目标**: 丰富的统计信息卡片

#### 样式选项

```go
type SessionStatsStyle string

const (
    StatsStyleCompact  SessionStatsStyle = "compact"  // 单行摘要
    StatsStyleCard     SessionStatsStyle = "card"     // 卡片式 (推荐)
    StatsStyleDetailed SessionStatsStyle = "detailed" // 完整报告
)
```

#### Card Style (推荐)

```json
{
  "type": "header",
  "text": {"type": "plain_text", "text": "✅ Session Complete"}
},
{
  "type": "section",
  "fields": [
    {"type": "mrkdwn", "text": "*⏱️ Duration*\n12.5s"},
    {"type": "mrkdwn", "text": "*📊 Tokens*\n1234 in / 567 out"},
    {"type": "mrkdwn", "text": "*💰 Cost*\n$0.02"},
    {"type": "mrkdwn", "text": "*🤖 Model*\nclaude-sonnet-4-20250514"}
  ]
},
{
  "type": "context",
  "elements": [{
    "type": "mrkdwn",
    "text": "📦 *Cache:* read 1234 tokens"
  }]
}
```

**Go 实现** (`block_builder.go:471-658`):

```go
func (b *BlockBuilder) BuildSessionStatsBlock(
    stats *event.SessionStatsData, 
    style SessionStatsStyle,
) []map[string]any {
    switch style {
    case StatsStyleCompact:
        return b.buildCompactStats(stats)
    case StatsStyleDetailed:
        return b.buildDetailedStats(stats)
    case StatsStyleCard:
        fallthrough
    default:
        return b.buildCardStats(stats)
    }
}
```

**SessionStatsData 字段** (`event/events.go:79-99`):

```go
type SessionStatsData struct {
    SessionID            string
    TotalDurationMs      int64
    InputTokens, OutputTokens, TotalTokens int32
    CacheReadTokens, CacheWriteTokens int32
    TotalCostUSD         float64
    ToolCallCount        int32
    ToolsUsed            []string
    FilesModified        int32
    FilePaths            []string
    ModelUsed            string
    IsError              bool
    ErrorMessage         string
}
```

**设计说明**:
- Card Style 平衡信息密度和可读性 (推荐默认)
- Compact 用于移动端或频繁更新场景
- Detailed 用于调试或审计场景
- 2 列 `fields` 布局优化空间使用

---

### 8. Permission Request 事件

**Block 类型**: `header` + `section` + `actions`  
**聚合策略**: ❌ 不聚合 - 需要用户立即决策  
**UI 目标**: 权限审批交互

```json
{
  "type": "header",
  "text": {"type": "plain_text", "text": "⚠️ Permission Request"}
},
{
  "type": "section",
  "text": {"type": "mrkdwn", "text": "*Tool:* `Bash`"}
},
{
  "type": "section",
  "text": {"type": "mrkdwn", "text": "*Command:*\n```\nls -la\n```"}
},
{
  "type": "context",
  "elements": [{
    "type": "mrkdwn",
    "text": "Session: `session-123`"
  }]
},
{
  "type": "actions",
  "elements": [
    {
      "type": "button",
      "text": {"type": "plain_text", "text": "✅ Allow"},
      "action_id": "perm_allow",
      "style": "primary",
      "value": "allow:session-123:message-id"
    },
    {
      "type": "button",
      "text": {"type": "plain_text", "text": "🚫 Deny"},
      "action_id": "perm_deny",
      "style": "danger",
      "value": "deny:session-123:message-id"
    }
  ]
}
```

**Go 实现** (`block_builder.go:783-857`):

```go
func BuildPermissionRequestBlocks(
    req *provider.PermissionRequest, 
    sessionID string,
) []map[string]any {
    tool, input := req.GetToolAndInput()
    
    // Truncate long commands
    displayInput := input
    if len(displayInput) > 500 {
        displayInput = displayInput[:497] + "..."
    }
    
    blocks := []map[string]any{}
    
    // 1. Header
    blocks = append(blocks, map[string]any{
        "type": "header",
        "text": plainText("⚠️ Permission Request"),
    })
    
    // 2. Tool information
    if tool != "" {
        blocks = append(blocks, map[string]any{
            "type": "section",
            "text": mrkdwnText(fmt.Sprintf("*Tool:* `%s`", tool)),
        })
    }
    
    // 3. Command preview
    if displayInput != "" {
        blocks = append(blocks, map[string]any{
            "type": "section",
            "text": mrkdwnText(fmt.Sprintf("*Command:*\n```\n%s\n```", displayInput)),
        })
    }
    
    // 4. Session info
    blocks = append(blocks, map[string]any{
        "type": "context",
        "elements": []map[string]any{
            mrkdwnText(fmt.Sprintf("Session: `%s`", sessionID)),
        },
    })
    
    // 5. Action buttons
    blocks = append(blocks, map[string]any{
        "type":     "actions",
        "block_id": fmt.Sprintf("perm_%s", req.MessageID),
        "elements": []map[string]any{
            {
                "type":      "button",
                "text":      plainText("✅ Allow"),
                "action_id": "perm_allow",
                "style":     "primary",
                "value":     fmt.Sprintf("allow:%s:%s", sessionID, req.MessageID),
            },
            {
                "type":      "button",
                "text":      plainText("🚫 Deny"),
                "action_id": "perm_deny",
                "style":     "danger",
                "value":     fmt.Sprintf("deny:%s:%s", sessionID, req.MessageID),
            },
        },
    })
    
    return blocks
}
```

**审批结果展示**:

```go
// 审批通过
BuildPermissionApprovedBlocks(tool, input)
// 审批拒绝
BuildPermissionDeniedBlocks(tool, input, reason)
```

**设计说明**:
- 两个按钮：Allow (primary) / Deny (danger)
- `block_id` 包含 MessageID，用于回调路由
- `value` 编码 operation:sessionID:messageID，便于解析
- 命令截断到 500 字符，避免超出限制

---

## 事件聚合策略

### 策略矩阵

| 事件类型 | Block 类型 | 聚合策略 | 发送时机 | 说明 |
|---------|-----------|---------|---------|------|
| `thinking` | context | ❌ 不聚合 | 立即 | 用户需要即时反馈 |
| `tool_use` | section | ✅ 同类聚合 | 节流 | 多工具可合并展示 |
| `tool_result` | section+actions | ✅ 同类聚合 | 节流 | 可合并 + 展开按钮 |
| `answer` | section | ✅ 流式更新 | 1 次/秒 | 使用 `chat.update` |
| `error` | section | ❌ 不聚合 | 立即 | 关键错误信息 |
| `danger_block` | section | ❌ 不聚合 | 立即 | 安全拦截警告 |
| `session_stats` | header+section | 最后发送 | 单次 | 会话总结 |
| `permission_request` | header+actions | ❌ 不聚合 | 立即 | 需要用户决策 |

### 流式更新实现 (`engine_handler.go:296-303`)

```go
func (c *StreamCallback) handleAnswer(data any) error {
    // Clear thinking state on first non-thinking event
    if c.thinkingSent {
        c.thinkingSent = false
        c.logger.Debug("Clearing thinking state for answer")
    }
    
    var answerContent string
    switch v := data.(type) {
    case string:
        answerContent = v
    case *event.EventWithMeta:
        answerContent = v.EventData
    default:
        answerContent = fmt.Sprintf("%v", data)
    }
    
    if answerContent == "" {
        return nil
    }
    
    // Use throttled streaming update if we have a message to update
    if c.streamState != nil {
        return c.streamState.updateThrottled(
            c.ctx, c.adapters, c.platform, c.sessionID, 
            answerContent, c.blockBuilder, c.metadata,
        )
    }
    
    // Otherwise send as new message
    blocks := c.blockBuilder.BuildAnswerBlock(answerContent)
    return c.sendBlockMessage(string(provider.EventTypeAnswer), blocks, false)
}
```

---

## Block Builder API 参考

### 构造函数

```go
// 创建 BlockBuilder 实例
blockBuilder := slack.NewBlockBuilder()
```

### 核心方法

| 方法 | 参数 | 返回 | 用途 |
|------|------|------|------|
| `BuildThinkingBlock(content string)` | thinking 文本内容 | `[]map[string]any` | thinking 事件 |
| `BuildToolUseBlock(toolName, input string, truncated bool)` | 工具名、输入、是否截断 | `[]map[string]any` | tool_use 事件 |
| `BuildToolResultBlock(success bool, durationMs int64, output string, hasButton bool)` | 成功标志、时长、输出、是否显示按钮 | `[]map[string]any` | tool_result 事件 |
| `BuildErrorBlock(message string, isDangerBlock bool)` | 错误消息、是否危险拦截 | `[]map[string]any` | error/danger_block 事件 |
| `BuildAnswerBlock(content string)` | Markdown 内容 | `[]map[string]any` | answer 事件 |
| `BuildSessionStatsBlock(stats *event.SessionStatsData, style SessionStatsStyle)` | 统计数据、样式 | `[]map[string]any` | session_stats 事件 |
| `BuildPermissionRequestBlocks(req *provider.PermissionRequest, sessionID string)` | 权限请求、会话 ID | `[]map[string]any` | permission_request 事件 |
| `BuildDividerBlock()` | 无 | `[]map[string]any` | 分隔线 |

### 辅助方法

```go
// mrkdwn 格式化器
formatter := slack.NewMrkdwnFormatter()
formatted := formatter.Format(markdownText)

// 代码块格式化
codeBlock := formatter.FormatCodeBlock(code, "go")

// 文本截断
truncated := slack.TruncateText(longText, 300)
```

---

## mrkdwn 格式转换

### Slack mrkdwn 语法

Slack 的 mrkdwn 格式支持有限的 Markdown 子集：

| 格式 | Markdown | Slack mrkdwn |
|------|---------|-------------|
| 粗体 | `**text**` | `*text*` |
| 斜体 | `*text*` 或 `_text_` | `_text_` |
| 删除线 | `~~text~~` | `~text~` |
| 行内代码 | `` `code` `` | `` `code` `` |
| 代码块 | ` ```lang\ncode\n``` ` | ` ```lang\ncode\n``` ` |
| 链接 | `[text](url)` | `<url\|text>` |
| 引用 | `> text` | `> text` |

### MrkdwnFormatter 实现

```go
// 完整的 Markdown → mrkdwn 转换
formatter := slack.NewMrkdwnFormatter()
mrkdwn := formatter.Format(markdown)

// 示例
input := "**Bold** and *italic* with `code`"
output := formatter.Format(input)
// 输出："*Bold* and _italic_ with `code`"
```

### 特殊字符转义

以下字符在 mrkdwn 中有特殊含义，需要转义：

| 字符 | 转义 | 说明 |
|------|------|------|
| `&` | `&amp;` | HTML 实体 |
| `<` | `&lt;` | 避免被解析为链接 |
| `>` | `&gt;` | 避免被解析为引用 |

---

## 最佳实践与限制

### Slack Block Kit 限制

| 限制类型 | 数值 | 说明 |
|---------|------|------|
| 单消息最大字符数 | 4000 | 包括所有 blocks 的文本内容 |
| 单消息最大 Blocks 数 | 50 | 超过会返回错误 |
| Section block fields 最大数 | 10 | 2 列布局，最多 5 行 |
| 按钮 value 最大长度 | 2000 | URL 编码后 |
| 代码块最大字符数 | 3000 | 包括 ``` 标记 |
| chat.update 速率限制 | ~1 次/秒 | 超过会返回 `rate_limited` |

### 消息发送最佳实践

#### 1. 使用 `chat.postMessage` vs `chat.update`

```go
// 新消息 - 使用 chat.postMessage
func (a *Adapter) sendBlocks(...) {
    payload := map[string]any{
        "channel": channelID,
        "text":    fallbackText, // 必需：纯文本回退
        "blocks":  blocks,
    }
    // POST https://slack.com/api/chat.postMessage
}

// 更新消息 - 使用 chat.update
func (a *Adapter) updateBlocks(...) {
    payload := map[string]any{
        "channel": channelID,
        "ts":      messageTS, // 必需：消息时间戳
        "text":    fallbackText,
        "blocks":  blocks,
    }
    // POST https://slack.com/api/chat.update
}
```

#### 2. 长消息分块策略

超过 4000 字符的消息需要分块发送：

```go
// ProcessorChain 中的 ChunkProcessor
if len(content) > 4000 {
    chunks := splitIntoChunks(content, 4000)
    for i, chunk := range chunks {
        // 发送多个消息
        sendBlockMessage(chunk)
    }
}
```

#### 3. 线程消息 vs 主频道

```go
// 在主频道发送
metadata["channel_type"] = "channel"

// 在线程中发送 (推荐用于多轮对话)
metadata["thread_ts"] = parentMessageTS
```

### 错误处理

```go
func (c *StreamCallback) sendBlockMessage(...) error {
    if c.adapters == nil {
        c.logger.Debug("No adapters, skipping message send")
        return nil
    }
    
    msg := &ChatMessage{...}
    processedMsg, err := c.processor.Process(c.ctx, msg)
    if err != nil {
        c.logger.Error("Message processing failed", "error", err)
        processedMsg = msg // Fallback to original
    }
    
    if processedMsg == nil {
        c.logger.Debug("Message dropped by processor")
        return nil
    }
    
    return c.adapters.SendMessage(c.ctx, c.platform, c.sessionID, processedMsg)
}
```

---

## 实现示例

### 完整事件处理流程

```go
package main

import (
    "context"
    "log/slog"
    "github.com/hrygo/hotplex/chatapps"
    "github.com/hrygo/hotplex/chatapps/slack"
    "github.com/hrygo/hotplex/event"
)

func main() {
    logger := slog.Default()
    
    // 1. 创建 BlockBuilder
    blockBuilder := slack.NewBlockBuilder()
    
    // 2. 创建 StreamCallback
    callback := chatapps.NewStreamCallback(
        context.Background(),
        "session-123",
        "slack",
        adapters,
        logger,
        map[string]any{"channel_id": "C123456"},
    )
    
    // 3. 模拟事件流
    // Thinking event
    callback.Handle("thinking", &event.EventWithMeta{
        EventType: "thinking",
        EventData: "Analyzing your request...",
    })
    
    // Tool use event
    callback.Handle("tool_use", &event.EventWithMeta{
        EventType: "tool_use",
        EventData: "ls -la",
        Meta: &event.EventMeta{
            ToolName: "Bash",
            ToolID:   "tool_123",
        },
    })
    
    // Tool result event
    callback.Handle("tool_result", &event.EventWithMeta{
        EventType: "tool_result",
        EventData: "total 1234\ndrwxr-xr-x ...",
        Meta: &event.EventMeta{
            ToolName:   "Bash",
            Status:     "success",
            DurationMs: 1234,
        },
    })
    
    // Answer event
    callback.Handle("answer", &event.EventWithMeta{
        EventType: "answer",
        EventData: "Here's what I found...",
    })
    
    // Session stats event
    callback.Handle("session_stats", &event.SessionStatsData{
        TotalDurationMs: 5678,
        InputTokens:     1234,
        OutputTokens:    567,
        TotalTokens:     1801,
        ToolsUsed:       []string{"Bash", "Editor"},
        FilesModified:   2,
    })
    
    // Error event
    callback.Handle("error", &event.EventWithMeta{
        EventType: "error",
        EventData: "Failed to execute command",
        Meta: &event.EventMeta{
            ErrorMsg: "Permission denied",
        },
    })
}
```

### 自定义 Block 样式

```go
// 使用 Compact 样式发送 session stats
statsBlock := blockBuilder.BuildSessionStatsBlock(
    statsData,
    slack.StatsStyleCompact,
)

// 使用 Detailed 样式发送详细报告
detailedBlock := blockBuilder.BuildSessionStatsBlock(
    statsData,
    slack.StatsStyleDetailed,
)
```

### 交互式按钮回调处理

```go
// 在 Slack adapter 中处理按钮回调
func handleInteractive(w http.ResponseWriter, r *http.Request) {
    // 解析 payload
    var payload slack.InteractionCallback
    json.Unmarshal(body, &payload)
    
    // 根据 action_id 路由
    switch payload.ActionCallback.BlockActions[0].ActionID {
    case "perm_allow":
        // 处理允许
        value := strings.Split(payload.ActionCallback.BlockActions[0].Value, ":")
        sessionID := value[1]
        messageID := value[2]
        // 调用 Engine 允许权限
        
    case "perm_deny":
        // 处理拒绝
        // ...
        
    case "view_tool_output":
        // 展开完整输出 (使用 modal 或 thread)
        // ...
    }
}
```

---

## 相关资源

### 官方文档

- [Slack Block Kit 文档](https://api.slack.com/block-kit)
- [Block Kit Builder](https://app.slack.com/block-kit-builder)
- [mrkdwn 格式参考](https://api.slack.com/reference/surfaces/formatting)
- [Go SDK (slack-go/slack)](https://github.com/slack-go/slack)

### 内部文件

| 文件 | 说明 |
|------|------|
| `chatapps/slack/block_builder.go` | Block Builder 完整实现 (900 行) |
| `chatapps/engine_handler.go` | 事件处理核心逻辑 |
| `chatapps/slack/adapter.go` | Slack API 封装 |
| `provider/event.go` | ProviderEventType 枚举定义 |
| `event/events.go` | EventMeta, SessionStatsData 定义 |

### 相关 Issue

- [#38 Engine Events → Slack Block Kit 最佳展现映射](https://github.com/hrygo/hotplex/issues/38)
- [#39 Permission Request 支持](https://github.com/hrygo/hotplex/issues/39)
- [#43 事件感知聚合优化](https://github.com/hrygo/hotplex/issues/43)

---

## 版本历史

| 版本 | 日期 | 变更 |
|------|------|------|
| 1.0 | 2026-02-26 | 初始版本，完成全部 8 种事件映射 |

---

**维护者**: HotPlex Team  
**最后审查**: 2026-02-26
