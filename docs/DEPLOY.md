# Yearning 功能增强 - 部署说明

## 变更概览

本次发布包含以下三大功能增强：

| 功能 | 优先级 | 说明 |
|------|--------|------|
| 批量工单审核 | P0 | 审核人可一次勾选多个待审工单，批量同意或驳回 |
| MFA 登录认证 | P0 | 基于 TOTP 的多因子认证，支持 Google Authenticator 等应用 |
| 批量工单申请 (多数据源) | P1 | 同一 SQL 一次提交到多个数据源，自动生成独立工单 |

## 架构说明

Yearning 由三个组件组成：

| 组件 | 说明 | 端口 |
|------|------|------|
| Yearning | API 服务 + 前端 | 8000 |
| Juno | SQL 检测/执行引擎 (RPC) | 50001 |
| MySQL | 数据库（两者共用） | 3306 |

Juno 是 Yearning 的 SQL 引擎，通过 `net/rpc` 协议通信，提供 SQL 语法检测 (`Engine.Check`)、语句执行 (`Engine.Exec`)、查询 (`Engine.Query`) 等能力。**没有 Juno，SQL 检测和工单执行将不可用。**

> 参考: [Juno 调用文档](https://next.yearning.io/zh/development/16e5vp2y/)

## 快速开始

### 方式一：Docker 一键部署（推荐）

```bash
# 启动（自动构建镜像 + MySQL + Juno 引擎）
./scripts/docker-deploy.sh

# 查看日志
./scripts/docker-deploy.sh --logs

# 停止
./scripts/docker-deploy.sh --stop
```

### 方式二：裸机一键部署

```bash
# 交互式部署（会提示输入 MySQL 配置）
./scripts/deploy.sh

# 或自动模式（通过环境变量配置）
MYSQL_HOST=127.0.0.1 MYSQL_PASSWORD=your_pass ./scripts/deploy.sh --auto
```

### 方式三：仅构建

```bash
# 构建前后端，产出到 output/ 目录
./scripts/build.sh
```

### 开发环境

```bash
./scripts/dev.sh --db        # 先启动 MySQL
./scripts/dev.sh             # 启动前端 + 后端开发服务器
```

详细说明见下方各节。

## 变更文件清单

### 后端 (Yearning-next)

| 文件 | 变更 | 说明 |
|------|------|------|
| `go.mod` | 修改 | 新增 `github.com/pquerna/otp v1.5.0` 依赖 |
| `src/model/modal.go` | 修改 | `CoreAccount` 新增 MFA 字段；新增 `CoreBatchOrder` 模型 |
| `src/service/migrate.go` | 修改 | 自动迁移注册 `CoreBatchOrder` |
| `src/handler/order/audit/impl.go` | 修改 | 新增 `BatchConfirm`/`BatchResult` 类型 |
| `src/handler/order/audit/audit.go` | 修改 | 新增 `BatchAuditOrderState` 处理函数 |
| `src/handler/personal/post.go` | 修改 | 新增 `sqlBatchOrderPost`/`GetBatchOrderDetail` |
| `src/handler/personal/mfa.go` | **新增** | MFA 设置/验证/关闭/状态查询 |
| `src/handler/login/login.go` | 修改 | 普通登录和 LDAP 登录接入 MFA 验证 |
| `src/router/router.go` | 修改 | 注册批量审核、批量提交、MFA 路由 |

### 前端 (gemini-next-next)

| 文件 | 变更 | 说明 |
|------|------|------|
| `src/config/request.ts` | 修改 | 全局响应拦截器排除 MFA 1300 状态码 |
| `src/apis/orderPostApis.ts` | 修改 | 新增批量审核/批量提交 API |
| `src/apis/user.ts` | 修改 | 新增 MFA 管理 API |
| `src/apis/loginApi.ts` | 修改 | `LoginFrom` 增加 `mfa_code` 字段 |
| `src/components/table/table.vue` | 修改 | 透传 `rowSelection`/`rowKey` 属性 |
| `src/components/orderTable/orderTable.vue` | 修改 | 审核列表支持多选 + 批量操作 |
| `src/views/apply/order.vue` | 修改 | 支持多数据源选择和批量提交 |
| `src/views/login/login-form.vue` | 修改 | MFA 验证码输入框 |
| `src/views/home/profile.vue` | 修改 | MFA 设置管理界面 |
| `src/lang/zh-cn/order/index.ts` | 修改 | 批量操作中文 i18n |
| `src/lang/en-us/order/index.ts` | 修改 | 批量操作英文 i18n |
| `src/lang/zh-cn/common/index.ts` | 修改 | MFA 管理中文 i18n |
| `src/lang/en-us/common/index.ts` | 修改 | MFA 管理英文 i18n |

## 部署脚本说明

所有脚本位于 `scripts/` 目录：

| 脚本 | 说明 |
|------|------|
| `build.sh` | 从源码构建前端 + 后端，产出到 `output/` 目录 |
| `deploy.sh` | 裸机一键部署（构建 + 配置 + systemd 服务） |
| `docker-deploy.sh` | Docker 一键部署（构建镜像 + MySQL + 启动） |
| `docker-compose.yml` | Docker Compose 编排文件 |
| `Dockerfile` | 多阶段构建 Dockerfile（Node + Go + Alpine） |
| `dev.sh` | 本地开发环境启动脚本 |

## 部署方式一：Docker 容器化部署

### 前置条件

- Docker >= 20.10
- Docker Compose V2

### 启动

```bash
# 一键启动（自动构建 + 启动 MySQL + Juno + Yearning）
./scripts/docker-deploy.sh

# 自定义配置：编辑 scripts/.env 后重新启动
vim scripts/.env
./scripts/docker-deploy.sh --rebuild
```

Docker Compose 会启动三个服务：
- **yearning** — API 服务（端口 8000）
- **juno** — SQL 检测/执行引擎（端口 50001），使用 `yeelabs/juno` 镜像
- **mysql** — MySQL 8.0 数据库（端口 3307）

Yearning 通过 `RPC_ADDR=juno:50001` 环境变量连接同网络中的 Juno 容器。

### 管理

```bash
./scripts/docker-deploy.sh --logs      # 查看 Yearning + Juno 日志
./scripts/docker-deploy.sh --status    # 查看状态
./scripts/docker-deploy.sh --stop      # 停止
./scripts/docker-deploy.sh --destroy   # 停止并删除数据
```

### 自定义配置

编辑 `scripts/.env`（首次运行自动生成）：

```env
MYSQL_USER=yearning
MYSQL_PASSWORD=your_password
MYSQL_ROOT_PASSWORD=your_root_password
MYSQL_DB=yearning
SECRET_KEY=your16charSecret
YEARNING_PORT=8000
MYSQL_PORT=3307
JUNO_PORT=50001
```

## 部署方式二：裸机/VM 部署

### 前置条件

- Go >= 1.22
- Node.js >= 16
- MySQL >= 5.7（需提前安装并创建数据库）
- Docker（用于运行 Juno 引擎）或手动部署 Juno

### 交互式部署

```bash
./scripts/deploy.sh
```

脚本将引导你输入 MySQL 连接信息，然后自动完成：构建 → 安装 → 初始化数据库 → 创建 systemd 服务。

### 自动化部署

```bash
MYSQL_HOST=10.0.0.1 \
MYSQL_PASSWORD=your_pass \
MYSQL_DB=yearning \
INSTALL_DIR=/opt/yearning \
./scripts/deploy.sh --auto
```

### 仅构建

```bash
./scripts/build.sh                # 构建前后端
./scripts/build.sh --backend      # 仅构建后端
./scripts/build.sh --frontend     # 仅构建前端

# 交叉编译
GOOS=linux GOARCH=amd64 ./scripts/build.sh
```

构建产出在 `output/` 目录：

```
output/
├── Yearning       # 可执行文件（内嵌前端）
└── conf.toml      # 配置文件模板
```

### 手动部署步骤

如不使用脚本，手动操作步骤：

```bash
# 1. 构建前端
cd gemini-next-next
npm install --legacy-peer-deps
npm run build

# 2. 复制前端产出到后端 embed 位置
cp -r dist ../Yearning-next/src/service/dist

# 3. 构建后端
cd ../Yearning-next
mkdir -p src/service/chat/server/app
echo '<html></html>' > src/service/chat/server/app/index.html
go mod tidy
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o Yearning

# 4. 配置
cp conf.toml.template conf.toml
vim conf.toml    # 编辑数据库连接信息，确保 RpcAddr 指向 Juno 地址

# 5. 启动 Juno SQL 引擎（与 Yearning 共用同一数据库）
docker run -d \
  --name yearning-juno \
  -e MYSQL_USER=yearning \
  -e MYSQL_PASSWORD=your_password \
  -e MYSQL_ADDR=127.0.0.1:3306 \
  -e MYSQL_DB=yearning \
  -p 50001:50001 \
  --restart always \
  yeelabs/juno

# 6. 初始化并启动 Yearning
./Yearning install
./Yearning run --port 8000
```

> **重要**: Juno 引擎必须在 Yearning 之前启动，且 `conf.toml` 中的 `RpcAddr` 需指向 Juno 地址（默认 `127.0.0.1:50001`）。Juno 与 Yearning 共用同一个 MySQL 数据库。当前 Juno 镜像支持 amd64/arm64 架构。

### 数据库迁移

**自动迁移（推荐）**：`Yearning install` 启动时 GORM AutoMigrate 自动创建/更新表。

**手动 SQL**（如需提前变更）：

```sql
-- MFA 支持
ALTER TABLE core_accounts ADD COLUMN mfa_secret VARCHAR(100) DEFAULT '';
ALTER TABLE core_accounts ADD COLUMN mfa_enabled TINYINT(1) DEFAULT 0;

-- 批量工单
CREATE TABLE IF NOT EXISTS core_batch_orders (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    batch_id VARCHAR(50) NOT NULL,
    work_ids JSON,
    username VARCHAR(50) NOT NULL,
    date VARCHAR(50) NOT NULL,
    status TINYINT(2) NOT NULL DEFAULT 2,
    INDEX batch_idx (batch_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### Nginx 配置参考

无需额外路由变更，所有新 API 遵循现有 `/api/v2/` 前缀。Yearning 二进制文件内嵌前端，直接反代即可：

```nginx
server {
    listen 80;
    server_name yearning.example.com;

    location / {
        proxy_pass http://127.0.0.1:8000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## 开发环境

```bash
# 启动开发数据库 + Juno 引擎（Docker）
./scripts/dev.sh --db

# 启动前后端开发服务器
./scripts/dev.sh

# 仅启动前端（需后端已运行）
./scripts/dev.sh --frontend

# 仅启动后端
./scripts/dev.sh --backend

# 仅启动 Juno 引擎
./scripts/dev.sh --juno
```

`--db` 命令会同时启动 MySQL 和 Juno 引擎两个 Docker 容器。前端 Vite dev server 运行在 `:5173`，通过代理将 API 请求转发到后端 `:8000`。Juno 引擎运行在 `:50001`。

## 新增 API 端点

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/v2/audit/order/batch` | JWT | 批量审核工单 |
| POST | `/api/v2/common/batch_post` | JWT | 批量提交工单（多数据源） |
| GET | `/api/v2/common/batch` | JWT | 查询批量工单详情 |
| POST | `/api/v2/common/mfa/setup` | JWT | 生成 MFA 密钥和二维码 |
| POST | `/api/v2/common/mfa/verify` | JWT | 验证并启用 MFA |
| POST | `/api/v2/common/mfa/disable` | JWT | 验证并关闭 MFA |
| GET | `/api/v2/common/mfa/status` | JWT | 查询 MFA 启用状态 |

## 功能使用说明

### 批量工单审核

1. 进入 **工单审批** 页面
2. 待审工单（状态=审核中 且 当前步骤审核人包含自己）可勾选复选框
3. 勾选后顶部出现操作栏：显示已选数量、批量同意、批量驳回、取消选择
4. **批量同意**：弹出确认对话框，确认后逐一处理，结果汇总展示
5. **批量驳回**：弹出文本框填写驳回理由，提交后逐一处理

### MFA 多因子认证

**用户启用 MFA：**

1. 登录后进入 **个人详情** 页面
2. 页面底部找到「多因子认证 (MFA)」卡片
3. 点击「启用 MFA」，系统生成 TOTP 密钥和二维码
4. 使用 Google Authenticator / Microsoft Authenticator 等应用扫码
5. 输入应用显示的 6 位验证码，点击「确认绑定」

**MFA 登录流程：**

1. 输入用户名密码后点击登录
2. 如果账户启用了 MFA，出现验证码输入框
3. 输入 Authenticator 应用中的 6 位验证码
4. 再次点击登录完成验证

**关闭 MFA：**

1. 在 **个人详情** 页面的 MFA 卡片中
2. 输入当前 Authenticator 验证码
3. 点击「关闭 MFA」

### 批量工单申请（多数据源）

1. 进入 **工单申请** 页面，选择一个主数据源
2. 在工单填写页面，「批量数据源」下拉框中选择额外目标数据源
3. 填写 SQL、说明等信息
4. 提交按钮自动切换为「批量提交」
5. 系统为每个数据源创建独立工单，各自走审批流

## 回滚方案

如需回滚至升级前版本：

```sql
-- 回滚批量工单表
DROP TABLE IF EXISTS core_batch_orders;

-- 回滚 MFA 字段
ALTER TABLE core_accounts DROP COLUMN mfa_enabled;
ALTER TABLE core_accounts DROP COLUMN mfa_secret;
```

回滚后需重新部署旧版后端和前端代码。

## 兼容性说明

- 所有新增数据库字段均有默认值，不影响现有数据和功能
- MFA 默认关闭，用户自行选择启用
- 批量操作为增量功能，不影响原有的单条审核/提交流程
- 前端 i18n 支持中英文双语
