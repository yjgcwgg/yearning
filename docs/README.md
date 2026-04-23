# Yearning SQL 审核平台 - 功能增强技术方案

## 项目概述

Yearning 是一款 MySQL SQL 审核平台，提供工单审批、SQL 查询、自动化任务等核心功能。本次技术方案涵盖 11 项功能增强，旨在提升审批效率、增强安全性、扩展推送渠道并改善多环境协作体验。

## 技术栈

| 层级 | 技术选型 |
|------|---------|
| 后端 | Go 1.22, yee HTTP 框架, GORM + MySQL, JWT, RPC SQL 引擎 |
| 前端 | Vue 3 + TypeScript + Vite 3, Ant Design Vue, Monaco Editor, Vuex 4 |
| 数据库 | MySQL 5.7+ (元数据 + 目标数据源) |
| 消息推送 | 邮件 (gomail) + 钉钉 Webhook |
| 认证 | 本地账户 + LDAP + OIDC |

## 文档索引

### 项目分析

| 文档 | 说明 |
|------|------|
| [00-project-analysis.md](./00-project-analysis.md) | 项目现状分析、架构图、数据模型、痛点总结 |
| [99-database-migration.md](./99-database-migration.md) | 数据库迁移方案汇总 |

### 功能方案 (按优先级排序)

| 优先级 | 文档 | 功能 | 预估工期 |
|--------|------|------|---------|
| P0 | [01-batch-audit.md](./features/01-batch-audit.md) | 批量审批 | 3-5 天 |
| P0 | [02-mfa-login.md](./features/02-mfa-login.md) | MFA 登录认证 | 3-5 天 |
| P1 | [03-multi-datasource.md](./features/03-multi-datasource.md) | 多数据源批量同步 | 5-7 天 |
| P1 | [04-custom-pusher.md](./features/04-custom-pusher.md) | 自定义消息推送 | 5-7 天 |
| P1 | [05-order-copy.md](./features/05-order-copy.md) | 工单复制 (多环境流转) | 3-5 天 |
| P2 | [06-pri-rollback.md](./features/06-pri-rollback.md) | 基于主键生成回滚语句 | 7-10 天 |
| P2 | [07-fk-audit.md](./features/07-fk-audit.md) | 外键审核规则 | 5-7 天 |
| P2 | [08-table-prefix.md](./features/08-table-prefix.md) | 表名前缀审核规则 | 2-3 天 |
| P2 | [09-mobile-audit.md](./features/09-mobile-audit.md) | 移动端审核 | 7-10 天 |
| P3 | [10-direct-query.md](./features/10-direct-query.md) | C 端直连查询 | 5-7 天 |
| P3 | [11-wechat-group.md](./features/11-wechat-group.md) | 企业微信群集成 | 3-5 天 |

## 依赖关系

```
批量审批 ──────────────> 移动端审核
自定义消息推送 ─────────> 企业微信群集成
```

其余功能模块之间无强依赖，可并行开发。

## 代码仓库结构

```
yearning/
├── Yearning-next/           # Go 后端
│   ├── src/
│   │   ├── model/           # 数据模型 (GORM)
│   │   ├── router/          # 路由注册
│   │   ├── handler/         # 业务逻辑
│   │   │   ├── login/       # 登录认证
│   │   │   ├── personal/    # 用户工单操作
│   │   │   ├── order/audit/ # 工单审批
│   │   │   ├── manage/      # 后台管理
│   │   │   └── fetch/       # 数据获取/AI
│   │   ├── engine/          # SQL 审核引擎定义
│   │   ├── lib/
│   │   │   ├── pusher/      # 消息推送
│   │   │   ├── factory/     # JWT/工具函数
│   │   │   └── pool/        # (新增) 连接池
│   │   └── service/         # 启动/定时任务
│   └── migration/           # 数据库迁移
├── gemini-next-next/        # Vue3 前端
│   └── src/
│       ├── views/           # 页面组件
│       ├── apis/            # API 请求
│       ├── store/           # Vuex 状态
│       └── router.ts        # 路由配置
└── docs/                    # 技术文档 (本目录)
```
