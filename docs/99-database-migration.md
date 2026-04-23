# 99 - 数据库迁移方案

## 概述

本文档汇总所有功能增强涉及的数据库变更，按优先级排序。所有迁移脚本放在 `Yearning-next/migration/` 目录，通过 `Yearning migrate` 命令执行。

## 变更总览

| 功能 | 变更类型 | 表名 | 说明 |
|------|---------|------|------|
| 03 多数据源批量同步 | 新增表 | `core_batch_orders` | 批量工单关联 |
| 02 MFA 登录 | 新增字段 | `core_accounts` | mfa_secret, mfa_enabled |
| 05 工单复制 | 新增字段 | `core_sql_orders` | copy_from |
| 10 C 端直连 | 新增字段 | `core_data_sources` | direct_connect |
| 其他功能 | 无 DDL | - | 配置存储在 JSON 字段中 |

## 迁移 SQL

### Phase 1: P0 功能 (优先执行)

```sql
-- ============================================================
-- Migration: 001_mfa_support.sql
-- Feature: 02 - MFA 登录认证
-- ============================================================

ALTER TABLE core_accounts
    ADD COLUMN mfa_secret VARCHAR(100) DEFAULT '' COMMENT 'TOTP 密钥';

ALTER TABLE core_accounts
    ADD COLUMN mfa_enabled TINYINT(1) DEFAULT 0 COMMENT 'MFA 启用状态';
```

### Phase 2: P1 功能

```sql
-- ============================================================
-- Migration: 002_batch_orders.sql
-- Feature: 03 - 多数据源批量同步
-- ============================================================

CREATE TABLE IF NOT EXISTS core_batch_orders (
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    batch_id   VARCHAR(50) NOT NULL COMMENT '批次 ID',
    work_ids   JSON COMMENT '关联工单 ID 列表',
    username   VARCHAR(50) NOT NULL COMMENT '提交人',
    date       VARCHAR(50) NOT NULL COMMENT '提交时间',
    status     TINYINT(2) NOT NULL DEFAULT 2 COMMENT '状态: 2进行中 1完成 0部分失败',
    INDEX batch_idx (batch_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='批量工单关联表';

-- ============================================================
-- Migration: 003_order_copy.sql
-- Feature: 05 - 工单复制
-- ============================================================

ALTER TABLE core_sql_orders
    ADD COLUMN copy_from VARCHAR(50) DEFAULT '' COMMENT '复制来源工单 ID';
```

### Phase 3: P3 功能

```sql
-- ============================================================
-- Migration: 004_direct_connect.sql
-- Feature: 10 - C 端直连查询
-- ============================================================

ALTER TABLE core_data_sources
    ADD COLUMN direct_connect TINYINT(1) DEFAULT 0 COMMENT '是否允许直连查询';
```

## 无需 DDL 的功能

以下功能的新增配置存储在现有 JSON 字段中，无需数据库结构变更:

| 功能 | 存储位置 | 说明 |
|------|---------|------|
| 01 批量审批 | 无新增存储 | 复用现有模型 |
| 04 自定义消息推送 | `core_global_configurations.message` | Message JSON 扩展 |
| 07 外键审核 | `core_rules.audit_role` 或 `core_global_configurations.audit_role` | AuditRole JSON 扩展 |
| 08 表名前缀 | 同上 | AuditRole JSON 扩展 |
| 09 移动端审核 | 无新增存储 | 前端页面 + 复用 API |
| 11 企业微信群 | `core_global_configurations.message` | Message JSON 扩展 |

## GORM 自动迁移

在 `Yearning-next/src/service/migrate.go` 中注册新模型，确保 `Yearning install` 和 `Yearning migrate` 命令能自动创建/更新表结构:

```go
func migration() {
    sqlDB.AutoMigrate(
        // ... 现有模型
        &model.CoreBatchOrder{}, // 新增
    )
}
```

对于 ALTER TABLE 变更，GORM AutoMigrate 会自动处理新增字段（不会删除已有字段）。

## 回滚方案

如需回滚迁移:

```sql
-- 回滚 004
ALTER TABLE core_data_sources DROP COLUMN direct_connect;

-- 回滚 003
ALTER TABLE core_sql_orders DROP COLUMN copy_from;

-- 回滚 002
DROP TABLE IF EXISTS core_batch_orders;

-- 回滚 001
ALTER TABLE core_accounts DROP COLUMN mfa_enabled;
ALTER TABLE core_accounts DROP COLUMN mfa_secret;
```

## 注意事项

1. **执行顺序**: 严格按 Phase 1 -> 2 -> 3 顺序执行
2. **备份**: 执行前务必备份数据库
3. **兼容性**: 所有新增字段都有默认值，不影响现有功能
4. **JSON 字段**: JSON 中新增的配置项在代码中需要处理零值/缺失情况
5. **索引**: `core_batch_orders.batch_idx` 按需创建，初期数据量不大时可跳过
