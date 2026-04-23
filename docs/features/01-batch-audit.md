# 01 - 批量审批

> **优先级**: P0 | **预估工期**: 3-5 天 | **依赖**: 无

## 一、需求背景

当前工单审批接口 `AuditOrderState` (`Yearning-next/src/handler/order/audit/audit.go`) 每次只能处理一个 `work_id` 的 agree/reject/undo 操作。在批量变更场景下（如多个数据源同时发起的 DDL 工单），审批人需要逐个打开工单进行审批，效率低下。

## 二、现状分析

### 2.1 当前审批流程

```go
// audit.go - AuditOrderState
// 接收单个 Confirm 结构体
type Confirm struct {
    WorkId   string `json:"work_id"`
    Page     int    `json:"page"`
    Flag     int    `json:"flag"`
    Text     string `json:"text"`
    Tp       string `json:"tp"`       // agree / reject / undo
    SourceId string `json:"source_id"`
    Delay    string `json:"delay"`
}
```

- `agree` 调用 `MultiAuditOrder(u, user)` -> 检查流程步骤 -> 推进或执行
- `reject` 调用 `RejectOrder(u, user)` -> 更新状态为 0 + 记录驳回理由
- `undo` 直接更新状态为 6

### 2.2 局限

- API 入口只接受单个 `work_id`
- 前端审核列表无多选机制
- 无法一次性处理多个工单

## 三、技术方案

### 3.1 后端改动

#### 3.1.1 新增批量审批请求结构

**文件**: `Yearning-next/src/handler/order/audit/impl.go`

```go
type BatchConfirm struct {
    WorkIds  []string `json:"work_ids"`
    Tp       string   `json:"tp"`        // agree / reject
    Text     string   `json:"text"`      // 驳回理由 (reject 时使用)
}

type BatchResult struct {
    Success []string         `json:"success"`
    Failed  []BatchFailItem  `json:"failed"`
}

type BatchFailItem struct {
    WorkId string `json:"work_id"`
    Error  string `json:"error"`
}
```

#### 3.1.2 新增批量审批 Handler

**文件**: `Yearning-next/src/handler/order/audit/audit.go`

```go
func BatchAuditOrderState(c yee.Context) (err error) {
    u := new(BatchConfirm)
    user := new(factory.Token).JwtParse(c)
    if err = c.Bind(u); err != nil {
        return c.JSON(http.StatusOK, common.ERR_COMMON_TEXT_MESSAGE(...))
    }

    result := BatchResult{}

    for _, workId := range u.WorkIds {
        confirm := &Confirm{
            WorkId:   workId,
            Tp:       u.Tp,
            Text:     u.Text,
        }
        // 获取 source_id 和 flag
        var order model.CoreSqlOrder
        model.DB().Select("source_id, current_step").
            Where("work_id = ?", workId).First(&order)
        confirm.SourceId = order.SourceId
        confirm.Flag = order.CurrentStep

        switch u.Tp {
        case "agree":
            resp := MultiAuditOrder(confirm, user.Username)
            if resp.Code != 0 {
                result.Failed = append(result.Failed, BatchFailItem{
                    WorkId: workId, Error: resp.Text,
                })
            } else {
                result.Success = append(result.Success, workId)
            }
        case "reject":
            resp := RejectOrder(confirm, user.Username)
            if resp.Code != 0 {
                result.Failed = append(result.Failed, BatchFailItem{
                    WorkId: workId, Error: resp.Text,
                })
            } else {
                result.Success = append(result.Success, workId)
            }
        }
    }

    return c.JSON(http.StatusOK, common.SuccessPayload(result))
}
```

#### 3.1.3 注册路由

**文件**: `Yearning-next/src/router/router.go`

在 `audit` 路由组内新增:

```go
audit.POST("/order/batch", audit2.BatchAuditOrderState)
```

### 3.2 前端改动

#### 3.2.1 新增 API

**文件**: `gemini-next-next/src/apis/orderPostApis.ts`

```typescript
export const BatchAuditOrder = (params: {
    work_ids: string[]
    tp: 'agree' | 'reject'
    text?: string
}) => axios.post('/api/v2/audit/order/batch', params)
```

#### 3.2.2 审核列表添加多选

**文件**: `gemini-next-next/src/views/server/order/list.vue`

改动要点:
- 表格增加 `row-selection` 属性启用 checkbox 多选
- 新增 `selectedRowKeys` 响应式变量跟踪选中项
- 表格上方增加批量操作工具栏:
  - 显示已选数量
  - "批量同意" 按钮
  - "批量驳回" 按钮 (需弹窗输入驳回理由)
  - "取消选择" 按钮

#### 3.2.3 批量操作交互

```
┌─────────────────────────────────────────────┐
│ 已选择 5 项  [批量同意] [批量驳回] [取消选择] │
├─────────────────────────────────────────────┤
│ ☑ │ 工单号  │ 提交人 │ 数据源 │ 状态 │ ...  │
│ ☑ │ WK-001  │ zhangsan│ prod-1│ 待审 │ ...  │
│ ☐ │ WK-002  │ lisi   │ prod-2│ 已执行│ ...  │
│ ☑ │ WK-003  │ wangwu │ dev-1 │ 待审 │ ...  │
└─────────────────────────────────────────────┘
```

- 仅状态为"待审核" (status=2) 且当前用户为当前步骤审批人的工单可被选中
- 批量操作完成后显示结果摘要: 成功 N 个, 失败 M 个 (展示失败详情)

## 四、数据模型

无需新增表或字段。批量操作复用现有 `CoreSqlOrder` 和 `CoreWorkflowDetail` 模型。

## 五、接口定义

### POST /api/v2/audit/order/batch

**请求**:
```json
{
    "work_ids": ["WK-20260422-001", "WK-20260422-002", "WK-20260422-003"],
    "tp": "agree",
    "text": ""
}
```

**响应 (成功)**:
```json
{
    "code": 1200,
    "payload": {
        "success": ["WK-20260422-001", "WK-20260422-003"],
        "failed": [
            {
                "work_id": "WK-20260422-002",
                "error": "当前用户无此工单审批权限"
            }
        ]
    }
}
```

## 六、测试要点

1. 批量同意 3 个处于不同审批步骤的工单，验证各自步骤正确推进
2. 批量驳回 + 驳回理由，验证所有工单状态变为 0 且评论记录生成
3. 混合场景: 部分工单当前用户无权审批，验证 partial success 结果
4. 并发安全: 两个审批人同时批量操作同一批工单
5. 消息推送: 每个工单的状态变更都应触发独立的通知
