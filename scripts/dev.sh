#!/usr/bin/env bash
#
# dev.sh - 本地开发环境启动脚本
#
# 用法:
#   ./scripts/dev.sh              # 启动前端 dev server + 后端
#   ./scripts/dev.sh --frontend   # 仅启动前端
#   ./scripts/dev.sh --backend    # 仅启动后端
#   ./scripts/dev.sh --db         # 仅启动 MySQL (Docker)
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/Yearning-next"
FRONTEND_DIR="$ROOT_DIR/gemini-next-next"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[DEV]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }

# 用 Docker 启动 MySQL 开发数据库
start_dev_db() {
  log "启动开发数据库 (MySQL 8.0)..."
  docker run -d \
    --name yearning-dev-mysql \
    -e MYSQL_ROOT_PASSWORD=root123 \
    -e MYSQL_DATABASE=yearning \
    -e MYSQL_USER=yearning \
    -e MYSQL_PASSWORD=yearning123 \
    -p 3306:3306 \
    --health-cmd='mysqladmin ping -h localhost -u root -proot123' \
    --health-interval=5s \
    --health-retries=10 \
    mysql:8.0 \
    --character-set-server=utf8mb4 \
    --collation-server=utf8mb4_general_ci \
    --default-authentication-plugin=mysql_native_password \
    2>/dev/null || {
      warn "容器已存在，尝试启动..."
      docker start yearning-dev-mysql 2>/dev/null || true
    }

  log "等待 MySQL 就绪..."
  for i in $(seq 1 30); do
    if docker exec yearning-dev-mysql mysqladmin ping -h localhost -u root -proot123 >/dev/null 2>&1; then
      log "MySQL 已就绪"
      break
    fi
    sleep 1
  done
}

# 用 Docker 启动 Juno SQL 引擎
start_dev_juno() {
  log "启动 Juno SQL 引擎..."

  local mysql_addr
  if docker ps --format '{{.Names}}' | grep -q yearning-dev-mysql; then
    local mysql_ip
    mysql_ip=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' yearning-dev-mysql 2>/dev/null)
    if [ -n "$mysql_ip" ]; then
      mysql_addr="${mysql_ip}:3306"
      log "检测到开发 MySQL 容器，Juno 使用容器 IP: $mysql_addr"
    else
      mysql_addr="172.17.0.1:3306"
      log "MySQL 容器 IP 未获取到，使用 docker0 网关: $mysql_addr"
    fi
  else
    mysql_addr="172.17.0.1:3306"
    log "未检测到 MySQL 容器，使用 docker0 网关: $mysql_addr"
    warn "请确认宿主机 MySQL 监听了 0.0.0.0 或 172.17.0.1"
  fi

  docker rm -f yearning-dev-juno 2>/dev/null || true
  docker run -d \
    --name yearning-dev-juno \
    -e MYSQL_USER="${DEV_MYSQL_USER:-yearning}" \
    -e MYSQL_PASSWORD="${DEV_MYSQL_PASSWORD:-yearning123}" \
    -e MYSQL_ADDR="$mysql_addr" \
    -e MYSQL_DB="${DEV_MYSQL_DB:-yearning}" \
    -p 50001:50001 \
    yeelabs/juno \
    2>/dev/null || {
      warn "Juno 启动失败，尝试拉取镜像..."
      docker pull yeelabs/juno && docker run -d \
        --name yearning-dev-juno \
        -e MYSQL_USER="${DEV_MYSQL_USER:-yearning}" \
        -e MYSQL_PASSWORD="${DEV_MYSQL_PASSWORD:-yearning123}" \
        -e MYSQL_ADDR="$mysql_addr" \
        -e MYSQL_DB="${DEV_MYSQL_DB:-yearning}" \
        -p 50001:50001 \
        yeelabs/juno
    }

  sleep 1
  if docker ps --filter name=yearning-dev-juno --format '{{.Status}}' | grep -q Up; then
    log "Juno 引擎已启动 (端口 50001)"
  else
    warn "Juno 引擎启动可能失败，请检查: docker logs yearning-dev-juno"
  fi
}

# 生成开发配置文件
ensure_dev_config() {
  local conf="$BACKEND_DIR/conf.toml"
  if [ ! -f "$conf" ]; then
    log "生成开发配置文件..."
    cat > "$conf" <<TOML
[Mysql]
Db = "yearning"
Host = "127.0.0.1"
Port = "3306"
Password = "yearning123"
User = "yearning"

[General]
SecretKey = "dbcjqheupqjsuwsm"
RpcAddr = "127.0.0.1:50001"
LogLevel = "debug"
Lang = "zh_CN"

[Oidc]
Enable = false
ClientId = ""
ClientSecret = ""
Scope = "openid profile"
AuthUrl = ""
TokenUrl = ""
UserUrl = ""
RedirectUrL = ""
UserNameKey = "preferred_username"
RealNameKey = "name"
EmailKey = "email"
SessionKey = "session_state"
TOML
    log "配置文件: $conf"
  fi
}

# 启动后端
start_backend() {
  log "=== 启动后端 ==="
  cd "$BACKEND_DIR"
  ensure_dev_config

  # 确保 embed 目录存在
  mkdir -p src/service/chat/server/app
  [ -f src/service/chat/server/app/index.html ] || \
    echo '<!DOCTYPE html><html><body></body></html>' > src/service/chat/server/app/index.html
  mkdir -p src/service/dist
  [ -f src/service/dist/index.html ] || \
    echo '<!DOCTYPE html><html><body><script>location.href="http://localhost:5173"</script></body></html>' > src/service/dist/index.html

  log "初始化数据库（如需要）..."
  go run . install 2>/dev/null || true

  log "启动 Yearning 后端 (端口 8000)..."
  go run . run &
  BACKEND_PID=$!
  log "后端 PID: $BACKEND_PID"
}

# 启动前端 dev server
start_frontend() {
  log "=== 启动前端开发服务器 ==="
  cd "$FRONTEND_DIR"

  if [ ! -d "node_modules" ]; then
    log "安装依赖..."
    npm install --legacy-peer-deps
  fi

  log "启动 Vite dev server (端口 5173)..."
  npm run dev &
  FRONTEND_PID=$!
  log "前端 PID: $FRONTEND_PID"
}

# 清理
cleanup() {
  log "停止进程..."
  [ -n "${BACKEND_PID:-}" ]  && kill "$BACKEND_PID"  2>/dev/null || true
  [ -n "${FRONTEND_PID:-}" ] && kill "$FRONTEND_PID" 2>/dev/null || true
  exit 0
}

trap cleanup SIGINT SIGTERM

# ── Main ──
case "${1:-all}" in
  --db)
    start_dev_db
    start_dev_juno
    ;;
  --backend)
    start_backend
    wait
    ;;
  --frontend)
    start_frontend
    wait
    ;;
  --juno)
    start_dev_juno
    ;;
  --help|-h)
    cat <<EOF
用法: $0 [命令]

命令:
  (无参数)     启动前端 + 后端
  --frontend   仅启动前端 dev server
  --backend    仅启动后端
  --db         启动 MySQL + Juno (Docker)
  --juno       仅启动 Juno SQL 引擎 (Docker)
  -h, --help   显示帮助

开发流程:
  1. ./scripts/dev.sh --db          # 启动数据库 + Juno 引擎
  2. ./scripts/dev.sh               # 启动前后端
  3. 浏览器访问 http://localhost:5173

前端通过 Vite proxy 将 API 请求转发到后端 :8000
Juno SQL 引擎运行在 :50001，提供 SQL 检测和执行能力
EOF
    exit 0
    ;;
  *)
    start_backend
    sleep 3
    start_frontend
    echo ""
    log "开发环境已启动:"
    log "  前端: http://localhost:5173"
    log "  后端: http://localhost:8000"
    log "  Juno: 127.0.0.1:50001 (需先执行 --db 启动)"
    log ""
    log "按 Ctrl+C 停止所有服务"
    wait
    ;;
esac
