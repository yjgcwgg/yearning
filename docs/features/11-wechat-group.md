# 11 - 企业微信群集成

> **优先级**: P3 | **预估工期**: 3-5 天 | **依赖**: 自定义消息推送 (04)

## 一、需求背景

团队日常沟通使用企业微信，需要将工单通知推送到企业微信群，并支持群内快捷链接跳转到审批页面。

## 二、现状分析

消息推送功能 04 (自定义消息推送) 已设计了企业微信 Webhook 的基础推送能力。本方案在此基础上增强:

1. 支持多群配置 (按数据源/环境分组推送)
2. 消息卡片交互 (附带审批链接)
3. 可选: 群内回调交互

## 三、技术方案

### 3.1 后端改动

#### 3.1.1 多群配置模型

扩展 `Message` 或新增独立配置:

```go
type WechatGroup struct {
    Name      string   `json:"name"`       // 群名称
    Webhook   string   `json:"webhook"`    // Webhook URL
    SourceIds []string `json:"source_ids"` // 关联的数据源 ID
    Events    []string `json:"events"`     // 订阅事件: submit/agree/reject/execute
}
```

在 `Message` 中新增:

```go
type Message struct {
    // ... 现有字段 + 04 方案新增字段

    // 企业微信多群配置 (新增)
    WechatGroups []WechatGroup `json:"wechat_groups"`
}
```

#### 3.1.2 消息卡片格式

**文件**: `Yearning-next/src/lib/pusher/wechat.go`

企业微信支持 markdown 消息:

```go
func buildWechatCard(vars TemplateVars) string {
    return fmt.Sprintf(`{
        "msgtype": "markdown",
        "markdown": {
            "content": "## Yearning 工单%s通知\n> **工单号:** %s\n> **数据源:** %s\n> **提交人:** <font color=\"info\">%s</font>\n> **说明:** %s\n> **审批人:** <font color=\"warning\">%s</font>\n> **状态:** <font color=\"comment\">%s</font>\n\n[PC端审批](%s/front/#/server/order/audit) | [移动端审批](%s/front/#/mobile/detail?work_id=%s)"
        }
    }`, vars.Status, vars.WorkId, vars.Source, vars.Username,
        vars.Text, vars.Assigned, vars.Status,
        vars.Domain, vars.Domain, vars.WorkId)
}
```

#### 3.1.3 按数据源路由推送

修改 `Push()` 方法，在企业微信推送时按群配置路由:

```go
func (tpl *OrderTPL) pushToWechatGroups(vars TemplateVars) {
    sourceId := tpl.orderInfo.SourceId
    eventType := tpl.eventType

    for _, group := range model.GloMessage.WechatGroups {
        // 检查数据源是否匹配
        if !containsSource(group.SourceIds, sourceId) {
            continue
        }
        // 检查事件是否订阅
        if !containsEvent(group.Events, eventType) {
            continue
        }
        // 推送到该群
        content := buildWechatCard(vars)
        go pushToWebhook(group.Webhook, content)
    }
}
```

#### 3.1.4 群通知测试

```go
// POST /api/v2/manage/setting/test?test=wechat_group
// body: { webhook: "https://qyapi.weixin.qq.com/..." }
```

### 3.2 前端改动

#### 3.2.1 设置页

**文件**: `gemini-next-next/src/views/manager/setting/setting.vue`

在消息推送配置中新增"企业微信群"区块:

```
企业微信群通知
├── [+ 添加群]
│
├── 群 1: 生产环境通知群
│   ├── Webhook: [https://qyapi.weixin.qq.com/cgi-bin/...]
│   ├── 关联数据源: [prod-mysql-01] [prod-mysql-02] (多选)
│   ├── 订阅事件: [✓提交] [✓同意] [✓驳回] [✓执行] [✓失败]
│   └── [测试] [删除]
│
├── 群 2: 开发环境通知群
│   ├── Webhook: [https://qyapi.weixin.qq.com/cgi-bin/...]
│   ├── 关联数据源: [dev-mysql-01] (多选)
│   ├── 订阅事件: [✓提交] [ 同意] [ 驳回] [✓执行] [✓失败]
│   └── [测试] [删除]
│
└── 说明: 不同群可关联不同数据源和事件，实现精准通知
```

#### 3.2.2 交互说明

- 支持动态增删群配置
- 数据源多选下拉: 可选"全部"或指定具体数据源
- 事件复选: 可选择性订阅特定事件类型
- 测试按钮: 向指定群发送测试消息

## 四、数据库迁移

无需 DDL 变更。`WechatGroups` 存储在 `CoreGlobalConfiguration.message` JSON 中。

## 五、企业微信 Webhook 接口参考

### 发送 Markdown 消息

```
POST https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx

{
    "msgtype": "markdown",
    "markdown": {
        "content": "## 标题\n> 引用\n**加粗** <font color=\"info\">绿色</font>"
    }
}
```

### 支持的 Markdown 语法

| 语法 | 说明 |
|------|------|
| `# 标题` | 标题 (h1-h6) |
| `**加粗**` | 加粗文本 |
| `> 引用` | 引用块 |
| `[链接](url)` | 超链接 |
| `<font color="info">` | 绿色文字 |
| `<font color="warning">` | 橙色文字 |
| `<font color="comment">` | 灰色文字 |

## 六、消息效果预览

```
┌──────────────────────────────────┐
│ Yearning 工单已提交通知           │
│                                  │
│ 工单号: WK-20260422-001          │
│ 数据源: prod-mysql-01            │
│ 提交人: 张三                     │
│ 说明: 用户表增加头像字段          │
│ 审批人: 李四                     │
│ 状态: 已提交                     │
│                                  │
│ PC端审批 | 移动端审批              │
└──────────────────────────────────┘
```

## 七、测试要点

1. 单群推送: 消息格式正确、链接可点击
2. 多群路由: 同一事件按数据源正确路由到对应群
3. 事件过滤: 未订阅的事件不推送
4. Webhook 失效时的错误处理 (不阻塞主流程)
5. 自定义模板在企业微信中的渲染效果
6. 移动端审批链接在企业微信内置浏览器中正常打开
