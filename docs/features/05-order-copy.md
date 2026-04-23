# 05 - 工单复制 (多环境流转)

> **优先级**: P1 | **预估工期**: 3-5 天 | **依赖**: 无

## 一、需求背景

在多环境（dev -> staging -> prod）部署场景中，同一 SQL 变更需要在多个环境依次执行。当前每个环境都需要手动创建工单、填写 SQL、选择数据源。需要支持将已有工单快速复制到其他环境，并可在复制时微调 SQL。

## 二、现状分析

### 2.1 工单模型

```go
type CoreSqlOrder struct {
    WorkId      string  // 唯一工单号
    SourceId    string  // 数据源 ID
    Source      string  // 数据源名称
    IDC         string  // 环境标识
    SQL         string  // SQL 内容
    Text        string  // 工单说明
    Type        int     // 0: DDL, 1: DML
    Status      uint    // 工单状态
    Username    string  // 提交人
    Assigned    string  // 当前审批人
    CurrentStep int     // 当前步骤
    // ...
}
```

工单没有记录来源/复制关系的字段。

## 三、技术方案

### 3.1 后端改动

#### 3.1.1 数据模型扩展

**文件**: `Yearning-next/src/model/modal.go`

`CoreSqlOrder` 新增字段:

```go
CopyFrom string `gorm:"type:varchar(50);default:''" json:"copy_from"`
```

用于追溯工单的复制来源。

#### 3.1.2 复制工单 API

**新增文件**: `Yearning-next/src/handler/personal/copy.go`

```go
type CopyOrderReq struct {
    WorkId         string `json:"work_id"`          // 源工单 ID
    TargetSourceId string `json:"target_source_id"` // 目标数据源 ID
    SQL            string `json:"sql"`              // 可选: 修改后的 SQL
    Text           string `json:"text"`             // 可选: 修改后的说明
}

func OrderCopy(c yee.Context) error {
    req := new(CopyOrderReq)
    c.Bind(req)
    user := new(factory.Token).JwtParse(c)

    // 1. 查询源工单
    var srcOrder model.CoreSqlOrder
    model.DB().Where("work_id = ?", req.WorkId).First(&srcOrder)

    // 2. 构建新工单
    newOrder := &model.CoreSqlOrder{
        SourceId: req.TargetSourceId,
        SQL:      srcOrder.SQL,
        Text:     srcOrder.Text,
        Type:     srcOrder.Type,
        Backup:   srcOrder.Backup,
        DataBase: srcOrder.DataBase,
        Table:    srcOrder.Table,
        Delay:    "none",
        CopyFrom: srcOrder.WorkId,
    }

    // 允许修改 SQL 和说明
    if req.SQL != "" {
        newOrder.SQL = req.SQL
    }
    if req.Text != "" {
        newOrder.Text = req.Text
    }

    // 3. 权限检查
    if !permission.NewPermissionService(model.DB()).Equal(&permission.Control{
        User: user.Username, Kind: newOrder.Type, SourceId: req.TargetSourceId,
    }) {
        return c.JSON(200, ERR("无目标数据源操作权限"))
    }

    // 4. 关联流程并创建
    step, err := wrapperPostOrderInfo(newOrder, c)
    if err != nil {
        return c.JSON(200, ERR(err))
    }
    newOrder.ID = 0
    model.DB().Create(newOrder)

    // 5. 记录流程和推送
    model.DB().Create(&model.CoreWorkflowDetail{
        WorkId:   newOrder.WorkId,
        Username: user.Username,
        Action:   fmt.Sprintf("从工单 %s 复制", srcOrder.WorkId),
        Time:     time.Now().Format("2006-01-02 15:04"),
    })
    pusher.NewMessagePusher(newOrder.WorkId).Order().
        OrderBuild(pusher.SummitStatus).Push()

    return c.JSON(200, common.SuccessPayload(map[string]interface{}{
        "work_id":  newOrder.WorkId,
        "copy_from": srcOrder.WorkId,
    }))
}
```

#### 3.1.3 路由注册

```go
r.POST("/common/order/copy", personal.OrderCopy)
```

### 3.2 前端改动

#### 3.2.1 工单列表

在工单列表的操作列增加"复制"按钮:

```
┌──────────────────────────────────────────────────────┐
│ 工单号  │ 提交人 │ 数据源 │ 状态 │ 操作             │
│ WK-001  │ zhangsan│ dev-1 │ 已完成│ [详情] [复制]    │
└──────────────────────────────────────────────────────┘
```

#### 3.2.2 复制弹窗

点击"复制"按钮弹出 Modal:

```
┌─────────────────────────────────────────┐
│ 复制工单到其他环境                        │
│                                         │
│ 源工单: WK-001 (dev-mysql)              │
│                                         │
│ 目标数据源: [下拉选择 - 按环境分组]       │
│   ├── DEV                               │
│   │   └── dev-mysql-01 (已选)           │
│   ├── STAGING                           │
│   │   └── staging-mysql-01              │
│   └── PROD                              │
│       └── prod-mysql-01                 │
│                                         │
│ SQL 内容: (可编辑)                       │
│ ┌─────────────────────────────────────┐ │
│ │ ALTER TABLE users ADD COLUMN ...    │ │
│ └─────────────────────────────────────┘ │
│                                         │
│ 工单说明: (可编辑)                       │
│ [____________________________________] │
│                                         │
│           [取消]  [确认复制]             │
└─────────────────────────────────────────┘
```

#### 3.2.3 工单详情

在工单详情页展示复制来源信息:

```
复制来源: WK-001 (点击跳转)
```

#### 3.2.4 新增 API

**文件**: `gemini-next-next/src/apis/orderPostApis.ts`

```typescript
export const CopyOrder = (params: {
    work_id: string
    target_source_id: string
    sql?: string
    text?: string
}) => axios.post('/api/v2/common/order/copy', params)
```

## 四、数据库迁移

```sql
ALTER TABLE core_sql_orders ADD COLUMN copy_from VARCHAR(50) DEFAULT ''
    COMMENT '复制来源工单 ID';
```

## 五、接口定义

### POST /api/v2/common/order/copy

**请求**:
```json
{
    "work_id": "WK-20260422-001",
    "target_source_id": "uuid-prod-mysql-01",
    "sql": "ALTER TABLE users ADD COLUMN avatar VARCHAR(255) DEFAULT '';",
    "text": "用户表增加头像字段 (从 dev 复制)"
}
```

**响应**:
```json
{
    "code": 1200,
    "payload": {
        "work_id": "WK-20260422-005",
        "copy_from": "WK-20260422-001"
    }
}
```

## 六、测试要点

1. 从已完成工单复制到另一环境，验证新工单创建正确
2. 复制时修改 SQL 内容，验证新工单使用修改后的 SQL
3. 权限校验: 用户对目标数据源无权限时复制失败
4. `copy_from` 追溯链: A -> B -> C 的级联复制
5. 复制后的新工单走目标数据源对应的审批流程
6. 流程详情中记录"从工单 xxx 复制"
