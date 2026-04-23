# 08 - 表名前缀审核规则

> **优先级**: P2 | **预估工期**: 2-3 天 | **依赖**: RPC 引擎改动

## 一、需求背景

数据库表命名规范是团队协作的基础要求。需要在 DDL 审核阶段自动校验表名是否符合配置的前缀规则，如 `t_`, `tbl_` 等。

## 二、现状分析

### 2.1 已有字段

```go
// engine.go - AuditRole
DDLTablePrefix string // 字段已定义，类型为 string
```

该字段已在审核规则结构体中声明，但 RPC 引擎未实现具体校验逻辑。

## 三、技术方案

### 3.1 后端改动

#### 3.1.1 扩展审核规则

**文件**: `Yearning-next/src/engine/engine.go`

```go
type AuditRole struct {
    // ... 现有字段
    DDLTablePrefix     string // 逗号分隔的允许前缀列表 (已有)
    DDLTablePrefixMode int    // 0: 前缀匹配, 1: 正则匹配 (新增)
}
```

#### 3.1.2 RPC 引擎校验逻辑

```go
func checkTablePrefix(tableName string, rule AuditRole) *Record {
    if rule.DDLTablePrefix == "" {
        return nil // 未配置则跳过
    }

    prefixes := strings.Split(rule.DDLTablePrefix, ",")

    switch rule.DDLTablePrefixMode {
    case 0: // 前缀匹配
        for _, prefix := range prefixes {
            prefix = strings.TrimSpace(prefix)
            if strings.HasPrefix(tableName, prefix) {
                return nil // 匹配成功
            }
        }
        return &Record{
            Level: ERROR,
            Error: fmt.Sprintf(
                "表名 '%s' 不符合前缀规范，允许的前缀: %s",
                tableName, rule.DDLTablePrefix),
        }

    case 1: // 正则匹配
        for _, pattern := range prefixes {
            pattern = strings.TrimSpace(pattern)
            if matched, _ := regexp.MatchString(pattern, tableName); matched {
                return nil
            }
        }
        return &Record{
            Level: ERROR,
            Error: fmt.Sprintf(
                "表名 '%s' 不符合命名规范，允许的模式: %s",
                tableName, rule.DDLTablePrefix),
        }
    }

    return nil
}
```

应用于以下 DDL 语句:
- `CREATE TABLE <name>`
- `ALTER TABLE <old> RENAME TO <new>` (校验新表名)
- `RENAME TABLE <old> TO <new>` (校验新表名)

#### 3.1.3 审核调用点

在 RPC 引擎的 `Engine.Check` DDL 审核流程中，在解析出表名后调用 `checkTablePrefix`:

```go
func (e *Engine) checkDDL(stmt ast.StmtNode, rule AuditRole) []Record {
    // ... 现有检查
    tableName := extractTableName(stmt)
    if r := checkTablePrefix(tableName, rule); r != nil {
        records = append(records, *r)
    }
    // ...
}
```

### 3.2 前端改动

#### 3.2.1 审核规则配置页

**文件**: `gemini-next-next/src/views/manager/rules/index.vue`

在 DDL 规则配置中新增:

```
表名规范
├── 表名前缀: [t_,tbl_________]
│   提示: 多个前缀用逗号分隔
├── 匹配模式: ○ 前缀匹配  ○ 正则匹配
│   前缀匹配: 表名必须以指定前缀开头
│   正则匹配: 表名必须匹配指定正则表达式
└── 预览: 输入表名测试 → [test_users] ✓ 符合规范
```

#### 3.2.2 交互说明

- 配置为空表示不校验
- 前缀匹配为默认模式（简单直观）
- 正则模式适用于更复杂的场景（如 `^(t|tbl|tmp)_[a-z]+$`）
- 提供实时预览: 输入测试表名立即显示校验结果

## 四、数据库迁移

无需 DDL 变更。`DDLTablePrefixMode` 存储在审核规则 JSON 中。

## 五、配置示例

### 示例 1: 前缀匹配

```
DDLTablePrefix: "t_,tbl_"
DDLTablePrefixMode: 0
```

| 表名 | 结果 |
|------|------|
| `t_users` | 通过 |
| `tbl_orders` | 通过 |
| `users` | 拒绝: 不符合前缀规范，允许的前缀: t_,tbl_ |
| `tmp_data` | 拒绝 |

### 示例 2: 正则匹配

```
DDLTablePrefix: "^(t|tbl)_[a-z][a-z0-9_]*$"
DDLTablePrefixMode: 1
```

| 表名 | 结果 |
|------|------|
| `t_users` | 通过 |
| `tbl_order_items` | 通过 |
| `t_Users` | 拒绝: 大写字母不匹配 |
| `t_123` | 拒绝: 数字开头不匹配 |

## 六、审核报告示例

```
检查通过 ✓ CREATE TABLE t_users (...)
    └── ✓ 表名 t_users 符合前缀规范 (t_)

检查失败 ✗ CREATE TABLE users (...)
    └── ✗ 表名 'users' 不符合前缀规范，允许的前缀: t_,tbl_

检查失败 ✗ ALTER TABLE t_orders RENAME TO orders_bak
    └── ✗ 新表名 'orders_bak' 不符合前缀规范，允许的前缀: t_,tbl_
```

## 七、测试要点

1. 前缀匹配: 单前缀 / 多前缀 / 空前缀（跳过）
2. 正则匹配: 合法正则 / 非法正则（降级跳过）
3. CREATE TABLE 校验
4. RENAME TABLE 校验新表名
5. ALTER TABLE RENAME TO 校验新表名
6. 不同规则集 (CoreRules) 可配置不同前缀
