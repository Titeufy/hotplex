# 🚀 HotPlex Slack 机器人全功能手册

本手册旨在引导你从零开始完成 **HotPlex Slack 适配器** 的集成。内容涵盖了从权限申请到日常运维的所有细节，请按顺序执行。

---

## 🗝️ 第一步：获取权限密钥 (Tokens)

访问 [Slack API 控制台](https://api.slack.com/apps)，创建一个 **From scratch** 应用。

| 变量名             | 推荐格式   | 获取路径              | 作用说明                                                                  |
| :----------------- | :--------- | :-------------------- | :------------------------------------------------------------------------ |
| **Bot Token**      | `xoxb-...` | `OAuth & Permissions` | **核心令牌**：用于发送消息、上传文件和更新 UI。                           |
| **App Token**      | `xapp-...` | `Basic Information`   | **Socket 令牌**：启用 Socket Mode 必需，需包含 `connections:write` 权限。 |
| **Signing Secret** | 字符串     | `Basic Information`   | **安全验证**：HTTP 模式下用于验证 Slack 请求的合法性。                    |

---

## 🛠️ 第二步：应用核心配置 (必做)

请根据以下清单严格配置 Slack App 后面板，否则机器人将无法正常工作。

### 1. 权限范围 (Scopes)
在 `OAuth & Permissions` -> `Bot Token Scopes` 中**必须**勾选以下 8 项：
- `app_mentions:read`：检测在频道中被 @。
- `chat:write`：拥有发送消息的权限。
- `reactions:write`：给用户消息加状态表情（📥/🧠等）。
- `im:history`：允许在私聊中读取上下文。
- `channels:history` / `groups:history`：读取公开/私有频道历史。
- `files:write`：允许发送诊断文件或代码生成物。
- `commands`：支持 Slash Commands 运维指令。

### 2. 交互窗口 (App Home)
必须在 `App Home` 页面进行以下勾选：
- [x] **Show Tabs** -> **Messages Tab** (开启聊天框)。
- [x] **Allow users to send Slash commands and messages from the messages tab** (允许私聊)。

### 3. 事件订阅 (Events)
在 `Event Subscriptions` 中开启 **Enable Events** 并订阅以下内容：
- `app_mention` (频道被呼叫)
- `message.im` (私聊窗口消息)
- `message.channels / groups` (群组消息适配)

---

## 📡 第三步：运行模式配置

HotPlex 支持两种通信模式，请根据你的网络环境选择：

### 模式 A：Socket Mode (强烈推荐)
**最适合本地测试或内网环境。**
- **原理**：基于 WebSocket 长连接，无需公网 IP 和 Webhook 配置。
- **配置**：
  1. 在 `Socket Mode` 页面将其设为 **Enable**。
  2. 生成 App Token (xapp) 时确保包含了 `connections:write` 权限。
  3. 修改 `.env`：`SLACK_MODE=socket`, `SLACK_APP_TOKEN=xapp-...`。

### 模式 B：HTTP Mode (传统 Webhook)
**适合具备公网 IP/域名的生产环境。**
- **配置**：
  1. 在 `Event Subscriptions` -> `Request URL` 填写：`https://你的域名/webhook/slack/events`。
  2. 修改 `.env`：`SLACK_MODE=http`, `SLACK_SIGNING_SECRET=...`。

---

## ⌨️ 第四步：运维快捷指令 (Slash Commands)

请在 Slack 控制面板的 `Slash Commands` 页面手动添加以下指令，用于日常管理。此外，HotPlex 为了保障全场景可用性，特别支持了 **`#` 前缀** 的文本指令作为等效替代。

| 指令 (Slash) | 文本替代 (Text) | 作用描述           | 使用场景                                                 |
| :----------- | :-------------- | :----------------- | :------------------------------------------------------- |
| **`/reset`** | **`#reset`**    | **重置当前会话**   | 当 AI 记忆混乱、逻辑陷入死循环或需要开启全新任务时使用。 |
| **`/dc`**    | **`#dc`**       | **终止并保留进度** | 强制终止当前的 CLI 任务进程，但保留环境进度供下次恢复。  |

> [!IMPORTANT]
> **关于 `#` 指令的设计说明：**
> 由于 Slack 官方限制，用户在 **Thread (回复列/消息列)** 中无法直接通过输入框发送并触发标准的 Slash Commands。为了解决这一痛点，HotPlex 适配器会对以 `#` 开头的普通消息进行拦截解析，其执行逻辑与对应的 `/` 指令完全一致，确保你在任何对话环境下都能进行运维操作。

> [!TIP]
> 即使在 **Socket Mode** 下，`/` 指令仍需在 Slack 后台先进行“声明”，机器人才能在主输入框中提供自动补全提示。

---

## ✨ 交互反馈指南：如何查看机器人进度

为了让你清楚 AI 正在做什么，HotPlex 提供了多层级的视觉反馈：

### 1. 表情反馈 (Reactions)
机器人会通过点按你消息下的表情来告知即时状态：
- 📥 (`:inbox:`)：消息已接收，系统正在检索上下文并分配合算力。
- 🧠 (`:brain:`)：深度思考中，正在进行复杂的逻辑分析或规划。
- ✅ (`:white_check_mark:`)：任务顺利完成！
- ❌ (`:x:`)：执行期间发生了错误，请查看最后一条报错详情。

### 2. 消息区域化展示 (Zone Rendering)
为了防止消息内容被工具日志淹没，你会看到消息在四个固定区域内跳动：
- **思考区**：展示推理心路历程（Zone 0）。
- **行动区**：实时滚动的工具调用日志（Zone 1），执行完后会自动折叠。
- **展示区**：AI 的核心文本回复，支持极致流畅的流式打字效果（Zone 2）。
- **统计区**：底部的小字，告知本次交互的耗时与 Token 成本（Zone 3）。

---

## ✅ 配置文件说明 (slack.yaml)

在代码库的 `chatapps/configs/slack.yaml` 中，你可以自定义以下高级参数：

- `bot_user_id`: 机器人的成员 ID。*强烈建议从 Slack “查看个人档案”中复制并填入。*
- `dm_policy`: 私聊策略。`pairing` 表示仅允许在群聊中与之交互过的用户发起私聊。
- `aggregator`: 控制行动日志的保留条数。

---

## 🚑 常见故障排查

1. **机器人没反应？**
   - 检查机器人是否在频道内（`/invite @机器人名字`）。
   - 确认 `App Home` 里的聊天开关是否勾选。
2. **私聊发不出消息？**
   - 重新前往 `OAuth & Permissions` 点击 **Reinstall App to Workspace**。
3. **指令提示“Dispatch failed”？**
   - 检查 `.env` 中的模式（socket 或 http）是否与 Slack 后台配置一致。

---

## 📚 相关参考
- [Slack UX 事件列表与渲染建议](./chatapps-architecture.md#6-事件类型映射-event-types)
- [Slack 区域化交互 (Zone) 架构详情](./chatapps-slack-architecture.md#3-交互分层架构-zone-architecture)
- [ChatApps 插件化流水线原理](./chatapps-architecture.md#3-消息处理流水线-message-processor-pipeline)
- [ChatApps 接入层核心协议](./chatapps-architecture.md)
