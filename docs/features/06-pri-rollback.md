# 06 - 基于主键生成回滚语句

> **优先级**: P2 | **预估工期**: 7-10 天 | **依赖**: RPC 引擎改动

## 一、需求背景

DML 工单执行后，如果发现数据异常需要回滚。当前 `CoreRollback` 模型和 `AuditRole.PRIRollBack` 开关已存在，但未实现基于主键的回滚语句自动生成逻辑。

## 二、现状分析

### 2.1 已有基础

```go
// modal.go
type CoreRollback struct {
    ID     uint   `gorm:"primary_key;AUTO_INCREMENT"`
    WorkId string `gorm:"type:varchar(50);not null;index:workId_idx"`
    SQL    string `gorm:"type:longtext;not null"`
}

// engine.go
type AuditRole struct {
    // ...
    PRIRollBack bool  // 开关已定义但未使用
}
```

### 2.2 当前执行流程

```
审批通过 -> Engine.Exec (RPC) -> 目标库执行 SQL -> 记录 CoreSqlRecord
```

执行阶段不会读取原始数据，无法生成回滚语句。

## 三、技术方案

### 3.1 回滚生成策略

| DML 类型 | 回滚策略 | 生成方式 |
|----------|---------|---------|
| `UPDATE` | 还原为原始值 | 执行前 `SELECT * WHERE <条件>` 备份原始行，生成 `UPDATE ... SET col=old_val WHERE pk=xxx` |
| `DELETE` | 恢复被删行 | 执行前 `SELECT * WHERE <条件>` 备份，生成 `INSERT INTO ... VALUES (...)` |
| `INSERT` | 删除新增行 | 执行后记录自增 ID/主键值，生成 `DELETE FROM ... WHERE pk=xxx` |

### 3.2 后端改动

#### 3.2.1 RPC 引擎扩展

在 RPC 引擎的 `Engine.Exec` 中新增回滚生成逻辑（伪代码）:

```go
func (e *Engine) execWithRollback(tx *sql.Tx, stmt DMLStatement,
    rule AuditRole) (rollbackSQL string, err error) {

    if !rule.PRIRollBack {
        return "", nil
    }

    // 1. 获取表的主键信息
    pkCols := getTablePrimaryKey(stmt.Table)
    if len(pkCols) == 0 {
        return "", nil // 无主键表不生成回滚
    }

    switch stmt.Type {
    case UPDATE:
        // 2a. 备份原始数据
        rows := query("SELECT * FROM %s WHERE %s", stmt.Table, stmt.Where)
        // 3a. 执行 UPDATE
        exec(stmt.SQL)
        // 4a. 生成回滚: UPDATE ... SET col=old_val WHERE pk=xxx
        for _, row := range rows {
            setClauses := buildSetClauses(row, pkCols)
            wherePK := buildPKWhere(row, pkCols)
            rollback += fmt.Sprintf("UPDATE `%s` SET %s WHERE %s;\n",
                stmt.Table, setClauses, wherePK)
        }

    case DELETE:
        // 2b. 备份将被删除的行
        rows := query("SELECT * FROM %s WHERE %s", stmt.Table, stmt.Where)
        // 3b. 执行 DELETE
        exec(stmt.SQL)
        // 4b. 生成回滚: INSERT INTO ... VALUES (...)
        for _, row := range rows {
            cols, vals := buildInsertParts(row)
            rollback += fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s);\n",
                stmt.Table, cols, vals)
        }

    case INSERT:
        // 2c. 执行 INSERT
        result := exec(stmt.SQL)
        lastId := result.LastInsertId()
        // 3c. 生成回滚: DELETE FROM ... WHERE pk=xxx
        rollback = fmt.Sprintf("DELETE FROM `%s` WHERE `%s` = %d;\n",
            stmt.Table, pkCols[0], lastId)
    }

    return rollback, nil
}
```

#### 3.2.2 保存回滚语句

在 RPC 执行结束后，将生成的回滚 SQL 写入 `CoreRollback` 表:

```go
if rollbackSQL != "" {
    model.DB().Create(&model.CoreRollback{
        WorkId: order.WorkId,
        SQL:    rollbackSQL,
    })
}
```

#### 3.2.3 查询回滚语句 API

**文件**: `Yearning-next/src/handler/order/audit/audit.go`

```go
// GET /api/v2/audit/order/rollback?work_id=xxx
func FetchRollbackSQL(c yee.Context) error {
    workId := c.QueryParam("work_id")
    var rollbacks []model.CoreRollback
    model.DB().Where("work_id = ?", workId).Find(&rollbacks)

    var allSQL []string
    for _, r := range rollbacks {
        allSQL = append(allSQL, r.SQL)
    }
    return c.JSON(200, common.SuccessPayload(map[string]interface{}{
        "work_id":      workId,
        "rollback_sql": strings.Join(allSQL, "\n"),
        "count":        len(rollbacks),
    }))
}
```

#### 3.2.4 执行回滚 API

```go
// POST /api/v2/audit/order/rollback/exec
type RollbackExecReq struct {
    WorkId string `json:"work_id"`
}

func ExecRollback(c yee.Context) error {
    req := new(RollbackExecReq)
    c.Bind(req)
    user := new(factory.Token).JwtParse(c)

    // 1. 查询原工单获取数据源信息
    var order model.CoreSqlOrder
    model.DB().Where("work_id = ?", req.WorkId).First(&order)

    // 2. 查询回滚 SQL
    var rollbacks []model.CoreRollback
    model.DB().Where("work_id = ?", req.WorkId).Find(&rollbacks)

    // 3. 通过 RPC 执行回滚 SQL (复用 Engine.Exec)
    // 创建一个回滚工单记录
    rollbackOrder := &model.CoreSqlOrder{
        WorkId:   factory.GenWorkId(),
        SQL:      combinedRollbackSQL,
        SourceId: order.SourceId,
        // ...
    }

    // 4. 记录回滚操作
    model.DB().Create(&model.CoreWorkflowDetail{
        WorkId:   order.WorkId,
        Username: user.Username,
        Action:   "执行回滚",
        Time:     time.Now().Format("2006-01-02 15:04"),
    })

    return c.JSON(200, common.SuccessPayLoadToMessage("回滚执行成功"))
}
```

#### 3.2.5 路由注册

```go
audit.GET("/order/rollback", audit2.FetchRollbackSQL)
audit.POST("/order/rollback/exec", audit2.ExecRollback)
```

### 3.3 前端改动

#### 3.3.1 工单详情页

在工单详情页增加"回滚"标签页:

```
┌───────┬───────┬───────┬───────┐
│ SQL   │ 流程  │ 记录  │ 回滚  │
└───────┴───────┴───────┴───────┘

回滚语句:
┌─────────────────────────────────────────┐
│ -- UPDATE 回滚                           │
│ UPDATE `users` SET `name`='old_name'     │
│   WHERE `id` = 123;                      │
│ UPDATE `users` SET `name`='old_name2'    │
│   WHERE `id` = 456;                      │
│                                          │
│ -- DELETE 回滚                           │
│ INSERT INTO `users` (`id`,`name`,`email`)│
│   VALUES (789,'zhangsan','z@test.com');   │
└─────────────────────────────────────────┘

[复制 SQL]  [执行回滚]

⚠️ 执行回滚将在目标数据源执行上述 SQL，请确认无误后操作
```

- "执行回滚" 按钮需二次确认 (输入工单号确认)
- 仅已执行 (status=1) 或执行失败 (status=4) 的工单显示回滚标签
- 回滚 SQL 使用 Monaco Editor 只读模式展示

#### 3.3.2 新增 API

```typescript
export const FetchRollbackSQL = (workId: string) =>
    axios.get('/api/v2/audit/order/rollback', { params: { work_id: workId } })

export const ExecRollback = (params: { work_id: string }) =>
    axios.post('/api/v2/audit/order/rollback/exec', params)
```

## 四、数据库迁移

无需新增表。`CoreRollback` 表已存在，复用即可。

## 五、注意事项

1. **大表保护**: 备份原始数据前检查 `MaxAffectRows`，超过阈值的 DML 不生成回滚（避免内存溢出）
2. **无主键表**: 无主键表跳过回滚生成，在审核报告中提示
3. **BLOB/TEXT**: 大字段使用 hex 编码存储回滚值
4. **事务一致性**: 备份和执行在同一事务中完成
5. **回滚有效期**: 表结构变更后回滚语句可能失效，需要提示用户

## 六、测试要点

1. UPDATE 单行/多行 -> 回滚语句能还原所有行的原始值
2. DELETE 单行/多行 -> 回滚语句能恢复所有被删行
3. INSERT 单行/批量 -> 回滚语句能删除新增行
4. 复合主键表的回滚语句 WHERE 条件正确
5. NULL 值字段的正确处理
6. 超大影响行数时跳过回滚生成
7. 执行回滚 API 的权限控制
