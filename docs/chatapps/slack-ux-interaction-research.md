# Slack UX 交互体验优化调研报告

**调研日期**: 2026-02-27
**目标**: 优化 Slack 交互体验 - 展示 thinking 状态、tool 信息、tool result 状态（忽略内容）、执行耗时，消除"黑洞感"

---

## 1. 需求澄清

### 1.1 核心目标
- **过滤低价值噪音**：不展示过长的 tool result 内容
- **消除黑洞感**：让用户知道 AI 正在做什么

### 1.2 策略选择
- **更新节奏**：关键节点更新（B），而非实时流式（A）
- **事件展示内容**：
  - Thinking: 显示状态（thinking/analyzing/planning）
  - Tool Use: 工具名 + 命令预览
  - Tool Result: **只看状态 + 耗时**，不显示内容
  - Answer: 最终回复
  - Error: 友好错误信息
  - Authorization: 权限请求（已有实现）
- **Answer 分块**：代码块完整保留 + chat.update 追加

---

## 2. 当前 HotPlex 实现分析

### 2.1 事件处理流程

| 事件类型 | 当前处理 | Block 类型 |
|---------|---------|-----------|
| thinking | 显示 "Thinking..." 或自定义内容 | Context Block |
| tool_use | 显示工具名 + Emoji + 输入预览 | Section Block |
| tool_result | 显示状态 + 输出预览(300字符) + 耗时 | Section + Context |
| answer | Mrkdwn 格式化 + 流式更新 | Section Block |
| error | 错误消息（引用格式） | Section Block |
| permission | 权限请求 + 按钮 | Header + Actions |

### 2.2 现有机制

**分块机制** (`chunker.go`):
- 触发阈值: 4000 字符
- 策略: 保留代码块完整，按段落/换行分割

**流式更新** (`engine_handler.go`):
- 节流: 最小 1 秒间隔
- 使用 `chat.update` 原地更新

### 2.3 存在差距

| 项目 | 当前 | 期望 |
|-----|------|------|
| Tool Result | 显示 300 字符预览 | ✅ 只看状态 + 耗时 |
| Thinking | 静态状态 | 显示实际状态 |
| Answer 分块 | 简单按字符 | 代码块完整 + 追加更新 |

---

## 3. Slack 官方最佳实践

### 3.1 Rate Limit 规范

| API | 限制 |
|-----|------|
| `chat.postMessage` | 1 message/second/channel |
| `chat.update` | **<= 1 次/3 秒** |
| HTTP 429 | Retry-After header |

**关键发现**: `chat.update` 官方建议最多 **每 3 秒一次**，HotPlex 当前 1 秒间隔可能过于激进。

### 3.2 Assistant Status API

Slack 提供专门的 AI Assistant 状态 API：

```json
// 设置状态
POST /slack/api/assistant.threads.setStatus
{
  "status": "thinking...",
  "loading_messages": [
    "Teaching the hamsters to type faster…",
    "Untangling the internet cables…",
    "Convincing the AI to stop overthinking…"
  ]
}
```

**特点**:
- 显示在消息输入框下方
- 支持旋转加载消息
- 回复后自动清除

### 3.3 Block Kit 限制

| Block 类型 | 字符限制 |
|-----------|---------|
| section | 3000 |
| context | 3000 |
| markdown | **12000** |
| 整体消息 | 40000 |

---

## 4. 竞品分析

### 4.1 Coder Blink (AI Agent 平台)

**关键模式**:

```typescript
// 1. 创建消息
const { message } = await slack.createMessageFromEvent({
  client: app.client,
  event,
});

// 2. 设置状态（打字指示器）
await app.client.assistant.threads.setStatus({
  channel_id: event.channel,
  status: "is typing...",
  thread_ts: event.thread_ts ?? event.ts,
});

// 3. 处理聊天
await agent.chat.sendMessages(chat.id, [message]);

// 4. 清除状态（回复后）
await app.client.assistant.threads.setStatus({
  channel_id: channel,
  thread_ts: thread_ts,
  status: "",  // 清空
});
```

**学习点**:
- 使用 `assistant.threads.setStatus` 代替自定义 thinking 消息
- 支持旋转加载消息增加趣味性

### 4.2 OpenClaw

| 特性 | OpenClaw 实现 |
|-----|--------------|
| 流式模式 | partial/block/progress 三种 |
| Ack Reaction | 👀 emoji |
| Thinking 级别 | off/minimal/low/medium/high/xhigh |
| 分块模式 | paragraph-first (`chunkMode: "newline"`) |

---

## 5. 优化方案

### 5.1 事件展示优化

#### Thinking 状态
- **当前**: Context Block 显示 "Thinking..."
- **优化**: 使用 `assistant.threads.setStatus` + 可选 Context Block 显示详细状态

#### Tool Use
- **当前**: 工具名 + Emoji + 输入预览（code block）
- **优化**: 保持，显示命令预览即可

#### Tool Result ⚠️ 核心改动
- **当前**: 显示状态 + 输出预览(300字符) + 耗时 + 路径
- **优化后**:
  ```
  ✅ Bash completed (150ms)
  ```
  - 只显示: 工具名 + 状态(成功/失败) + 执行耗时
  - **不显示**: 输出内容

#### Answer
- **当前**: Mrkdwn 格式化 + 流式更新
- **优化**:
  - 代码块完整保留
  - 使用 `chat.update` 追加（不是新建消息）
  - 考虑使用 `markdown` block (12000 字符限制)

### 5.2 节流策略调整

```go
// 推荐配置
const (
    MinUpdateInterval = 3 * time.Second  // Slack 官方建议
    MinCharDelta      = 50                // 最小字符变化
)
```

### 5.3 新增 Slack Status 集成

```go
// 使用 assistant.threads.setStatus
func (a *Adapter) SetThinkingStatus(channelID, threadTS, status string, loadingMessages []string) error {
    return a.client.AssistantThreadsSetStatus(&AssistantThreadsSetStatusParams{
        ChannelID:       channelID,
        ThreadTS:        threadTS,
        Status:          status,
        LoadingMessages: loadingMessages,
    })
}
```

---

## 6. 实现优先级

### Phase 1: 快速修复 (1-2 天)
1. **Tool Result 简化** - 只显示状态 + 耗时，不显示内容
2. **节流调整** - chat.update 改为 3 秒间隔

### Phase 2: 体验增强 (1 周)
1. **Assistant Status 集成** - 使用 Slack 原生状态 API
2. **Answer 追加更新** - chat.update 改为追加而非替换

### Phase 3: 高级特性 (2+ 周)
1. **旋转加载消息** - 增加趣味性
2. **Thinking 级别配置** - 参考 OpenClaw

---

## 7. 参考资源

- [Slack Block Kit 文档](https://api.slack.com/block-kit)
- [Rate Limits](https://api.slack.com/apis/web-api/rate-limits)
- [Assistant Threads API](https://api.slack.com/assistant)
- [Coder Blink](https://github.com/coder/blink)
- [OpenClaw](https://github.com/otherguy/openclaw)

---

*Report generated for HotPlex Slack UX optimization*
