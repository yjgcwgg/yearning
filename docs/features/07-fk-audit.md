# 07 - 外键审核规则

> **优先级**: P2 | **预估工期**: 5-7 天 | **依赖**: RPC 引擎改动

## 一、需求背景

当前审核规则中 `DDLEnableForeignKey` 仅为布尔开关（允许/禁止外键），缺少更细粒度的审核能力，如外键列索引检查、级联操作控制、命名规范校验等。

## 二、现状分析

### 2.1 已有字段

```go
// engine.go - AuditRole
DDLEnableForeignKey bool   // 是否允许创建外键 (仅此一项)
```

### 2.2 局限

- 只能全局允许或禁止外键
- 不检查外键列是否有索引（无索引的外键会严重影响性能）
- 不限制 CASCADE/SET NULL 等级联操作（误操作风险）
- 不校验外键命名是否规范

## 三、技术方案

### 3.1 后端改动

#### 3.1.1 扩展审核规则

**文件**: `Yearning-next/src/engine/engine.go`

在 `AuditRole` 中新增:

```go
type AuditRole struct {
    // ... 现有字段
    DDLEnableForeignKey      bool   // 是否允许创建外键 (已有)
    DDLForeignKeyMustIndexed bool   // 外键列必须有索引
    DDLForeignKeyCascade     bool   // 是否允许 CASCADE 操作
    DDLForeignKeySetNull     bool   // 是否允许 SET NULL 操作
    DDLForeignKeyNaming      string // 外键命名规范前缀 (如 "fk_")
}
```

#### 3.1.2 RPC 引擎审核逻辑

在 DDL 审核阶段 (`Engine.Check`)，解析 SQL 中的 `FOREIGN KEY` 子句，增加以下检查:

```go
func checkForeignKey(stmt *ast.CreateTableStmt, rule AuditRole) []Record {
    var records []Record

    for _, constraint := range stmt.Constraints {
        if constraint.Tp != ast.ConstraintForeignKey {
            continue
        }

        // 检查 1: 是否允许外键
        if !rule.DDLEnableForeignKey {
            records = append(records, Record{
                Level: ERROR,
                Error: "当前规则不允许创建外键",
            })
            continue
        }

        // 检查 2: 外键列必须有索引
        if rule.DDLForeignKeyMustIndexed {
            fkCols := constraint.Keys
            if !hasIndex(stmt, fkCols) {
                records = append(records, Record{
                    Level: WARNING,
                    Error: fmt.Sprintf("外键列 %v 缺少索引，建议添加索引以提升性能",
                        colNames(fkCols)),
                })
            }
        }

        // 检查 3: 级联操作
        if constraint.Refer.OnDelete != nil {
            action := constraint.Refer.OnDelete.ReferOpt
            if action == ast.ReferOptionCascade && !rule.DDLForeignKeyCascade {
                records = append(records, Record{
                    Level: ERROR,
                    Error: "当前规则不允许 ON DELETE CASCADE",
                })
            }
            if action == ast.ReferOptionSetNull && !rule.DDLForeignKeySetNull {
                records = append(records, Record{
                    Level: ERROR,
                    Error: "当前规则不允许 ON DELETE SET NULL",
                })
            }
        }

        // 检查 4: 命名规范
        if rule.DDLForeignKeyNaming != "" {
            fkName := constraint.Name
            if !strings.HasPrefix(fkName, rule.DDLForeignKeyNaming) {
                records = append(records, Record{
                    Level: WARNING,
                    Error: fmt.Sprintf("外键名 '%s' 不符合命名规范，"+
                        "应以 '%s' 为前缀", fkName, rule.DDLForeignKeyNaming),
                })
            }
        }

        // 检查 5: 引用表是否存在
        refTable := constraint.Refer.Table.Name
        if !tableExists(refTable) {
            records = append(records, Record{
                Level: ERROR,
                Error: fmt.Sprintf("外键引用表 '%s' 不存在", refTable),
            })
        }

        // 检查 6: 引用列类型是否匹配
        if !columnTypeMatch(constraint.Keys, constraint.Refer) {
            records = append(records, Record{
                Level: ERROR,
                Error: "外键列与引用列的数据类型不匹配",
            })
        }
    }

    return records
}
```

对 `ALTER TABLE ... ADD FOREIGN KEY` 语句做相同检查。

### 3.2 前端改动

#### 3.2.1 审核规则配置页

**文件**: `gemini-next-next/src/views/manager/rules/index.vue`

在 DDL 规则配置中新增"外键审核"区块:

```
外键审核
├── [✓] 允许创建外键 (DDLEnableForeignKey)
│   ├── [✓] 外键列必须有索引 (DDLForeignKeyMustIndexed)
│   ├── [ ] 允许 ON DELETE CASCADE (DDLForeignKeyCascade)
│   ├── [ ] 允许 ON DELETE SET NULL (DDLForeignKeySetNull)
│   └── 外键命名前缀: [fk_________] (DDLForeignKeyNaming)
```

交互说明:
- 当"允许创建外键"关闭时，子选项全部灰化
- "外键命名前缀" 为空表示不校验命名
- 提供常见配置预设 (如"严格模式": 允许外键 + 必须索引 + 禁止 CASCADE)

## 四、数据库迁移

无需 DDL 变更。新增字段存储在 `CoreRules.audit_role` JSON 或 `CoreGlobalConfiguration.audit_role` JSON 中。

## 五、审核报告示例

```
检查通过 ✓ CREATE TABLE orders (...)
    └── 外键 fk_orders_user_id:
        ✓ 外键列 user_id 存在索引
        ✓ 引用表 users 存在
        ✓ 列类型匹配 (INT)
        ✓ 命名规范符合 fk_ 前缀
        ⚠ ON DELETE CASCADE - 当前配置允许

检查失败 ✗ ALTER TABLE orders ADD FOREIGN KEY (user_id) REFERENCES ...
    └── 外键 (未命名):
        ✗ 外键名缺失，应以 fk_ 为前缀
        ✗ 外键列 user_id 缺少索引
```

## 六、测试要点

1. `DDLEnableForeignKey=false` 时，所有外键 DDL 被拒绝
2. `DDLForeignKeyMustIndexed=true` 时，无索引的外键列触发警告
3. `DDLForeignKeyCascade=false` 时，CASCADE 被拒绝
4. 命名规范校验: 正确前缀通过，错误前缀触发警告
5. 引用表不存在时触发错误
6. 列类型不匹配时触发错误 (如 INT 引用 VARCHAR)
7. `ALTER TABLE ADD FOREIGN KEY` 同样适用所有检查
