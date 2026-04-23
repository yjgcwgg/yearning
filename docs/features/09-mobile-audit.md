# 09 - 移动端审核

> **优先级**: P2 | **预估工期**: 7-10 天 | **依赖**: 批量审批 (01)

## 一、需求背景

审批人经常不在电脑前，收到工单通知后无法及时审批，导致流程阻塞。需要支持通过手机浏览器/企业微信/钉钉内嵌 H5 进行审批操作。

## 二、方案选型

| 方案 | 优点 | 缺点 |
|------|------|------|
| 原生 App (iOS/Android) | 体验最佳 | 开发/维护成本极高 |
| 小程序 | 入口便捷 | 需审核上架, 两端开发 |
| **响应式 H5** | 一套代码, 无需安装 | 体验略差于原生 |

**选择**: 响应式 H5 方案。复用现有 Vue3 技术栈，新增 `/mobile` 路由前缀下的轻量页面，针对移动端优化交互。

## 三、技术方案

### 3.1 后端改动

#### 3.1.1 移动端专用 API

新增精简接口，减少数据传输量:

```go
// GET /api/v2/mobile/orders - 待审核工单列表 (精简字段)
func MobileOrderList(c yee.Context) error {
    user := new(factory.Token).JwtParse(c)
    page := c.QueryParam("page")

    var orders []model.CoreSqlOrder
    model.DB().Select("work_id, username, real_name, text, source, "+
        "status, date, type, current_step").
        Where("status = 2").
        Where("FIND_IN_SET(?, assigned)", user.Username).
        Order("date DESC").
        Offset((page-1)*20).Limit(20).
        Find(&orders)

    return c.JSON(200, common.SuccessPayload(orders))
}

// GET /api/v2/mobile/order/:work_id - 工单详情
func MobileOrderDetail(c yee.Context) error {
    workId := c.Params("work_id")

    var order model.CoreSqlOrder
    model.DB().Where("work_id = ?", workId).First(&order)

    var details []model.CoreWorkflowDetail
    model.DB().Where("work_id = ?", workId).
        Order("id ASC").Find(&details)

    var comments []model.CoreOrderComment
    model.DB().Where("work_id = ?", workId).
        Order("id ASC").Find(&comments)

    return c.JSON(200, common.SuccessPayload(map[string]interface{}{
        "order":    order,
        "flow":     details,
        "comments": comments,
    }))
}

// POST /api/v2/mobile/audit - 审批操作 (复用现有逻辑)
func MobileAudit(c yee.Context) error {
    // 复用 AuditOrderState 内部逻辑
    return AuditOrderState(c)
}
```

#### 3.1.2 JWT 移动端适配

移动端 Token 延长有效期至 24h:

```go
func JwtAuthMobile(h Token) (string, error) {
    token := jwt.New(jwt.SigningMethodHS256)
    claims := token.Claims.(jwt.MapClaims)
    claims["name"] = h.Username
    claims["real_name"] = h.RealName
    claims["is_record"] = h.IsRecord
    claims["platform"] = "mobile"
    claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
    return token.SignedString([]byte(model.C.General.SecretKey))
}
```

#### 3.1.3 路由注册

```go
mobile := r.Group("/mobile")
mobile.GET("/orders", MobileOrderList)
mobile.GET("/order/:work_id", MobileOrderDetail)
mobile.POST("/audit", MobileAudit)
```

#### 3.1.4 消息推送嵌入直链

修改推送模板，附加移动端审批链接:

```
移动端审批: ${domain}/front/#/mobile/detail?work_id=xxx
```

### 3.2 前端改动

#### 3.2.1 新增移动端页面

**新增文件结构**:

```
gemini-next-next/src/views/mobile/
├── login.vue      - 移动端登录 (简化布局)
├── orders.vue     - 待审核列表 (卡片布局)
└── detail.vue     - 工单详情 + 审批操作
```

#### 3.2.2 路由配置

**文件**: `gemini-next-next/src/router.ts`

```typescript
{
    path: '/mobile',
    name: 'mobile',
    component: () => import('@/views/mobile/layout.vue'),
    children: [
        {
            path: '/mobile/login',
            name: 'mobile/login',
            component: () => import('@/views/mobile/login.vue'),
        },
        {
            path: '/mobile/orders',
            name: 'mobile/orders',
            meta: { title: '待审核工单' },
            component: () => import('@/views/mobile/orders.vue'),
        },
        {
            path: '/mobile/detail',
            name: 'mobile/detail',
            meta: { title: '工单详情' },
            component: () => import('@/views/mobile/detail.vue'),
        },
    ],
},
```

#### 3.2.3 待审核列表 (orders.vue)

卡片式布局，触控友好:

```
┌──────────────────────────┐
│ 待审核工单 (5)            │
├──────────────────────────┤
│ ┌────────────────────┐   │
│ │ WK-20260422-001    │   │
│ │ 张三 · dev-mysql   │   │
│ │ 用户表增加头像字段   │   │
│ │ DDL · 2026-04-22   │   │
│ │ ───────────────── │   │
│ │ [驳回]      [同意] │   │
│ └────────────────────┘   │
│                          │
│ ┌────────────────────┐   │
│ │ WK-20260422-002    │   │
│ │ 李四 · prod-mysql  │   │
│ │ 订单表增加索引      │   │
│ │ DDL · 2026-04-22   │   │
│ │ ───────────────── │   │
│ │ [驳回]      [同意] │   │
│ └────────────────────┘   │
│                          │
│      [加载更多...]        │
└──────────────────────────┘
```

- 支持下拉刷新 + 上拉加载更多
- 左滑卡片显示"驳回"按钮 (可选手势)
- 快速操作: 直接在列表卡片上同意/驳回

#### 3.2.4 工单详情 (detail.vue)

```
┌──────────────────────────┐
│ ← 工单详情                │
├──────────────────────────┤
│ WK-20260422-001          │
│ 状态: 待审核 · 步骤 2/3   │
├──────────────────────────┤
│ 提交人: 张三              │
│ 数据源: dev-mysql-01      │
│ 数据库: app_db            │
│ 类型: DDL                 │
│ 时间: 2026-04-22 10:30   │
├──────────────────────────┤
│ 工单说明:                 │
│ 用户表增加头像字段         │
├──────────────────────────┤
│ SQL:                      │
│ ┌──────────────────────┐ │
│ │ ALTER TABLE users    │ │
│ │ ADD COLUMN avatar    │ │
│ │ VARCHAR(255);        │ │
│ └──────────────────────┘ │
├──────────────────────────┤
│ 审批流程:                 │
│ ✓ 张三 提交 04-22 10:30  │
│ → 李四 待审核 (当前)      │
│ ○ 王五 待执行             │
├──────────────────────────┤
│                          │
│ 驳回理由 (可选):          │
│ [____________________]   │
│                          │
│ [  驳回  ]  [  同意  ]   │
└──────────────────────────┘
```

- SQL 区域可折叠展开
- 审批流程以时间线形式展示
- 底部固定操作按钮

#### 3.2.5 样式适配

- 使用 CSS `@media` 响应式断点
- 视口 meta: `<meta name="viewport" content="width=device-width, initial-scale=1">`
- 最小触控区域: 44x44px
- 字体大小: 最小 14px
- 安全区域适配 (iPhone 底部)

## 四、数据库迁移

无需新增表或字段。

## 五、接口定义

### GET /api/v2/mobile/orders?page=1

**响应**:
```json
{
    "code": 1200,
    "payload": [
        {
            "work_id": "WK-20260422-001",
            "username": "zhangsan",
            "real_name": "张三",
            "text": "用户表增加头像字段",
            "source": "dev-mysql-01",
            "status": 2,
            "date": "2026-04-22 10:30",
            "type": 0,
            "current_step": 2
        }
    ]
}
```

### GET /api/v2/mobile/order/:work_id

**响应**:
```json
{
    "code": 1200,
    "payload": {
        "order": { "work_id": "...", "sql": "ALTER TABLE ...", "..." : "..." },
        "flow": [
            { "username": "zhangsan", "action": "已提交", "time": "..." },
            { "username": "lisi", "action": "待审核", "time": "" }
        ],
        "comments": []
    }
}
```

## 六、测试要点

1. 移动端登录 (含 MFA 流程)
2. 待审核列表仅展示当前用户需审批的工单
3. 同意/驳回操作正确触发后端逻辑
4. 消息推送中的移动端链接可正确打开
5. 多分辨率适配 (375px / 414px / 768px)
6. 弱网环境下的加载体验
7. 企业微信/钉钉内嵌浏览器兼容性
