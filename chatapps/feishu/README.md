# Feishu (Lark) Adapter for HotPlex

飞书（Lark）适配器，为 HotPlex 提供中国企业 IM 集成能力。

## 快速开始

### 1. 配置环境变量

```bash
export FEISHU_APP_ID=cli_a1b2c3d4e5f6g7h8
export FEISHU_APP_SECRET=xxxxxxxxxxxxxxxx
export FEISHU_VERIFICATION_TOKEN=xxxxxxxx
export FEISHU_ENCRYPT_KEY=xxxxxxxxxxxxxxxx
export FEISHU_SERVER_ADDR=:8082
```

### 2. 创建适配器实例

```go
import "github.com/hrygo/hotplex/chatapps/feishu"

config := &feishu.Config{
    AppID:             os.Getenv("FEISHU_APP_ID"),
    AppSecret:         os.Getenv("FEISHU_APP_SECRET"),
    VerificationToken: os.Getenv("FEISHU_VERIFICATION_TOKEN"),
    EncryptKey:        os.Getenv("FEISHU_ENCRYPT_KEY"),
    ServerAddr:        os.Getenv("FEISHU_SERVER_ADDR"),
}

adapter, err := feishu.NewAdapter(config, logger)
if err != nil {
    log.Fatal(err)
}

// 设置消息处理器
adapter.SetHandler(myHandler)

// 启动适配器
if err := adapter.Start(ctx); err != nil {
    log.Fatal(err)
}
```

### 3. 飞书开发者后台配置

1. 创建企业自建应用
2. 配置事件订阅：
   - 订阅事件：`im.message.receive_v1`
   - 消息读取状态：启用
3. 配置机器人：
   - 添加 `/reset`, `/dc` 命令
4. 配置服务器：
   - 请求地址：`https://your-domain.com/feishu/events`

## 配置说明

| 配置项 | 必填 | 说明 | 默认值 |
|--------|------|------|--------|
| AppID | ✅ | 飞书应用 App ID | - |
| AppSecret | ✅ | 飞书应用 App Secret | - |
| VerificationToken | ✅ | 事件订阅验证 Token | - |
| EncryptKey | ✅ | 消息加密 Key | - |
| ServerAddr | ❌ | Webhook 服务器地址 | `:8082` |
| MaxMessageLen | ❌ | 单消息最大长度 | `4096` |
| SystemPrompt | ❌ | 系统提示词 | - |

## 错误处理

```go
if err != nil {
    var apiErr *feishu.APIError
    if errors.As(err, &apiErr) {
        // API 错误，查看错误码
        log.Printf("API error: code=%d, msg=%s", apiErr.Code, apiErr.Msg)
    } else if errors.Is(err, feishu.ErrInvalidSignature) {
        // 签名验证失败
        log.Println("Invalid signature")
    }
}
```

## 常见错误码

| 错误码 | 说明 | 解决方案 |
|--------|------|----------|
| 99991663 | app access token invalid | 检查 AppID/AppSecret |
| 99991668 | Invalid access token | Token 过期，等待自动刷新 |
| 99991671 | No permission | 检查应用权限配置 |

## 测试

```bash
go test ./chatapps/feishu/... -v
go test ./chatapps/feishu/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 参考文档

- [飞书开放平台](https://open.feishu.cn/)
- [事件订阅机制](https://open.feishu.cn/document/ukTMukTMukTM/uYjNwUjL2YDM14iN2ATN)
- [消息发送 API](https://open.feishu.cn/document/ukTMukTMukTM/uYjNwUjL2YDM14iN2ATN)
- [互动卡片指南](https://open.feishu.cn/document/ukTMukTMukTM/uQjNwUjLyYDM14iO2ATN)

## 状态

- [x] Phase 1: 基础通信（Adapter + API Client）
- [ ] Phase 2: 交互增强（卡片构建器）
- [ ] Phase 3: 生产就绪（文档 + 压测）
