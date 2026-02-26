# Claude Code 功能验证脚本

本目录包含两个验证脚本，用于验证 HotPlex 对 Claude Code 高级功能的支持情况。

## 📜 脚本列表

### 1. `verify_claude_features_offline.py` - 离线验证

**用途**: 分析代码库，检查实现完整性  
**前置条件**: 无需 Claude Code CLI，在 HotPlex 项目根目录运行即可

```bash
# 运行完整验证
python3 scripts/verify_claude_features_offline.py

# 验证单个功能
python3 scripts/verify_claude_features_offline.py --feature plan-mode
python3 scripts/verify_claude_features_offline.py --feature ask-user-question
python3 scripts/verify_claude_features_offline.py --feature output-styles
python3 scripts/verify_claude_features_offline.py --feature permission-request
```

**验证内容**:
- ✅ `provider/claude_provider.go` 事件解析逻辑
- ✅ `provider/event.go` 事件类型定义
- ✅ `docs/chatapps/engine-events-slack-mapping.md` 文档完整性

---

### 2. `verify_claude_features.py` - 在线验证

**用途**: 实际调用 Claude Code CLI，测试功能行为  
**前置条件**: 
- 已安装 Claude Code CLI: `npm install -g @anthropic-ai/claude-code`
- 已完成认证：`claude auth login`
- Node.js >= 18

```bash
# 运行完整验证
python3 scripts/verify_claude_features.py

# 验证单个功能
python3 scripts/verify_claude_features.py --feature plan-mode
python3 scripts/verify_claude_features.py --feature ask-user-question
python3 scripts/verify_claude_features.py --feature output-styles
python3 scripts/verify_claude_features.py --feature permission-request

# 指定工作目录
python3 scripts/verify_claude_features.py --work-dir /path/to/project
```

**测试方法**:
- 发送特定提示词触发目标功能
- 捕获 `stream-json` 输出
- 解析并验证事件类型和字段

---

## 🎯 验证的功能

### 1. Plan Mode (计划模式)

**功能描述**: Claude 只生成计划步骤，不执行任何工具调用

**识别特征**:
- `thinking` 事件的 `subtype == "plan_generation"`
- 无 `tool_use` 事件
- 显示步骤进度 (Step N/M)

**验证方法**:
```bash
claude -p "分析这个项目，制定重构计划，只输出计划不要执行" \
  --output-format stream-json
```

**期望输出**:
```json
{"type":"thinking","subtype":"plan_generation","status":"Planning..."}
```

---

### 2. AskUserQuestion (用户澄清问题)

**功能描述**: Claude 主动提问，用户通过交互式按钮回答

**识别特征**:
- `tool_use` 事件，`name == "AskUserQuestion"`
- 包含 `options` 数组 (可选答案列表)
- `questionType`: single-select | multi-select | custom

**验证方法**:
```bash
claude -p "我想添加新功能但不确定技术栈，请问我一些问题来明确需求" \
  --output-format stream-json
```

**期望输出**:
```json
{
  "type":"tool_use",
  "name":"AskUserQuestion",
  "input":{
    "question":"应该使用哪个测试框架？",
    "options":[
      {"label":"Jest","value":"jest"},
      {"label":"Vitest","value":"vitest"}
    ]
  }
}
```

---

### 3. Output Styles (输出风格)

**功能描述**: 教育性输出模式，包含解释性见解或协作学习

**类型**:
- `default`: 标准模式
- `explanatory`: 解释性模式 (提供教育性 Insights)
- `learning`: 学习模式 (添加 `TODO(human)` 标记)

**验证方法**:
```bash
# Explanatory
claude -p "解释这段代码，使用 explanatory output style" \
  --output-format stream-json

# Learning
claude -p "教我如何编写 HTTP 服务器，使用 learning output style" \
  --output-format stream-json
```

**期望输出 (Learning)**:
```
这里是实现示例...

TODO(human): 请你自己实现错误处理部分
```

---

### 4. Permission Request (权限请求)

**功能描述**: 危险操作前请求用户审批

**识别特征**:
- `permission_request` 事件类型
- 包含 `permission.name` 和 `permission.input`
- Slack UI 显示 Allow/Deny 按钮

**验证方法**:
```bash
claude -p "删除 temp 目录" \
  --permission-mode default \
  --output-format stream-json
```

**期望输出**:
```json
{
  "type":"permission_request",
  "permission":{
    "name":"Bash",
    "input":"rm -rf ./temp"
  }
}
```

---

## 📊 验证报告

运行验证脚本后，会生成详细的验证报告，包括:

- ✅ 已实现的功能清单
- ❌ 缺失的功能项
- 💡 实现建议和代码示例

完整报告见：[`docs/verification/claude-features-verification-report.md`](./claude-features-verification-report.md)

---

## 🔧 故障排查

### 问题：脚本提示 "claude 命令未找到"

**解决**:
```bash
# 安装 Claude Code CLI
npm install -g @anthropic-ai/claude-code

# 验证安装
claude --version
```

---

### 问题：脚本提示 "未认证"

**解决**:
```bash
# 进行认证
claude auth login

# 验证认证状态
claude auth status --text
```

---

### 问题：离线验证报告文件缺失

**解决**:
```bash
# 检查文件路径
ls -la docs/chatapps/engine-events-slack-mapping.md
ls -la provider/claude_provider.go
ls -la provider/event.go

# 确保在项目根目录运行
cd /path/to/HotPlex
python3 scripts/verify_claude_features_offline.py
```

---

### 问题：测试超时 (60s)

**解决**:
```bash
# 增加超时时间 (编辑脚本)
# 修改 subprocess.run 的 timeout 参数
timeout=120  # 秒
```

---

## 📚 相关资源

- [Claude Code 官方文档](https://code.claude.com/docs)
- [Plan Mode 指南](https://code.claude.com/docs/en/plan-mode)
- [AskUserQuestion 工具说明](https://claudelog.com/faqs/what-is-ask-user-question-tool-in-claude-code/)
- [Output Styles 文档](https://code.claude.com/docs/en/output-styles)
- [HotPlex Slack 映射文档](../docs/chatapps/engine-events-slack-mapping.md)

---

**维护者**: HotPlex Team  
**最后更新**: 2026-02-26
