*Read this in other languages: [English](production-guide.md), [简体中文](production-guide_zh.md).*

# 生产环境部署指南

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│                     负载均衡器 (LB)                          │
│                  (nginx / 云厂商 LB)                         │
└─────────────────────────────────────────────────────────────┘
                            │
         ┌──────────────────┼──────────────────┐
         ▼                  ▼                  ▼
   ┌──────────┐       ┌──────────┐       ┌──────────┐
   │ HotPlex  │       │ HotPlex  │       │ HotPlex  │
   │  节点 1  │       │  节点 2  │       │  节点 3  │
   └──────────┐       └──────────┘       └──────────┘
         │                  │                  │
         └──────────────────┴──────────────────┘
                            │
         ┌──────────────────┼──────────────────┐
         ▼                  ▼                  ▼
   ┌──────────┐       ┌──────────┐       ┌──────────┐
   │ Prometheus│       │  Jaeger  │       │  Loki    │
   │  (指标)   │       │  (追踪)  │       │  (日志)  │
   └──────────┐       └──────────┘       └──────────┘
```

## 扩容建议

| 并发用户数 | 实例数量 | 单实例 CPU | 单实例内存 |
| ---------- | -------- | ---------- | ---------- |
| 1-100      | 1        | 0.5 核     | 512MB      |
| 100-500    | 2-3      | 1 核       | 1GB        |
| 500-2000   | 5-10     | 2 核       | 2GB        |
| 2000+      | 10+      | 2-4 核     | 2-4GB      |

## 监控栈

### Prometheus 配置

```yaml
scrape_configs:
  - job_name: 'hotplex'
    static_configs:
      - targets: ['hotplex:8080']
    metrics_path: /metrics
```

### Grafana 仪表盘

关键面板：
- 活动会话数 (Active Sessions)
- 请求延迟 (p50, p95, p99)
- 错误率
- 工具调用频率

### 告警规则

```yaml
groups:
- name: hotplex
  rules:
  - alert: HighErrorRate
    expr: rate(hotplex_sessions_errors[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: 检测到高错误率
  
  - alert: SessionPoolExhausted
    expr: hotplex_sessions_active > 800
    for: 2m
    labels:
      severity: critical
```

## 安全检查清单

- [ ] 在负载均衡器启用 TLS 终止
- [ ] 配置网络策略 (Network Policies)
- [ ] 配置频率限制 (Rate Limiting)
- [ ] 启用身份认证 (Authentication)
- [ ] 配置资源限额 (Resource Limits)
- [ ] 启用审计日志 (Audit Logging)

## 备份与恢复

### 会话状态

会话是短暂的 (Ephemeral)，无需备份持久化状态。

### 配置信息

```bash
kubectl get configmap hotplex-config -o yaml > hotplex-config-backup.yaml
```

## 故障分析排查

### 内存占用过高

```bash
kubectl exec -it hotplex-xxx -- curl localhost:8080/debug/pprof/heap
```

### 请求响应变慢

在 Jaeger 中检查追踪，找出瓶颈所在的 Spans。

### 会话泄漏

```bash
curl http://hotplex:8080/metrics | grep hotplex_sessions_active
```
