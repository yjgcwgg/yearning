#!/usr/bin/env bash
#
# docker-deploy.sh - Docker 一键部署 Yearning
#
# 用法:
#   ./scripts/docker-deploy.sh              # 构建并启动
#   ./scripts/docker-deploy.sh --rebuild    # 强制重新构建
#   ./scripts/docker-deploy.sh --stop       # 停止服务
#   ./scripts/docker-deploy.sh --destroy    # 停止并删除所有数据
#   ./scripts/docker-deploy.sh --logs       # 查看日志
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.yml"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[DOCKER]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
fail() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

# 检查 docker 和 docker compose
check_docker() {
  command -v docker >/dev/null 2>&1 || fail "未找到 docker，请先安装 Docker"

  if docker compose version >/dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
  elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE_CMD="docker-compose"
  else
    fail "未找到 docker compose，请安装 Docker Compose V2"
  fi

  log "使用: $($COMPOSE_CMD version 2>&1 | head -1)"
}

# 创建 .env 文件（如不存在）
ensure_env() {
  local env_file="$SCRIPT_DIR/.env"
  if [ ! -f "$env_file" ]; then
    log "生成默认 .env 配置..."
    cat > "$env_file" <<'EOF'
# Yearning Docker 部署配置
# 修改后执行 docker compose up -d 生效

MYSQL_USER=yearning
MYSQL_PASSWORD=Yearning_pass123
MYSQL_ROOT_PASSWORD=Yearning_root123
MYSQL_DB=yearning
SECRET_KEY=dbcjqheupqjsuwsm
YEARNING_PORT=8000
MYSQL_PORT=3307
JUNO_PORT=50001
EOF
    log ".env 文件已创建: $env_file"
    warn "建议修改 .env 中的密码后再启动"
  fi
}

# 启动
do_up() {
  local extra_args=""
  if [ "${REBUILD:-false}" = "true" ]; then
    extra_args="--build --force-recreate"
  fi

  log "=== 启动 Yearning ==="
  cd "$SCRIPT_DIR"
  $COMPOSE_CMD -f "$COMPOSE_FILE" up -d $extra_args

  log ""
  log "等待服务启动..."
  sleep 5

  local port
  port=$(grep YEARNING_PORT "$SCRIPT_DIR/.env" 2>/dev/null | cut -d= -f2 || echo "8000")
  [ -z "$port" ] && port="8000"

  echo ""
  echo -e "${GREEN}╔══════════════════════════════════════════════╗${NC}"
  echo -e "${GREEN}║      Yearning Docker 部署完成！              ║${NC}"
  echo -e "${GREEN}╠══════════════════════════════════════════════╣${NC}"
  echo -e "${GREEN}║                                              ║${NC}"
  echo -e "${GREEN}║  访问地址:  http://localhost:${port}             ║${NC}"
  echo -e "${GREEN}║  默认账号:  admin                            ║${NC}"
  echo -e "${GREEN}║  默认密码:  Yearning_admin                   ║${NC}"
  echo -e "${GREEN}║                                              ║${NC}"
  echo -e "${GREEN}║  服务:                                       ║${NC}"
  echo -e "${GREEN}║    Yearning : 端口 ${port}                       ║${NC}"
  echo -e "${GREEN}║    Juno 引擎: 端口 50001 (SQL检测/执行)      ║${NC}"
  echo -e "${GREEN}║                                              ║${NC}"
  echo -e "${GREEN}║  管理命令:                                   ║${NC}"
  echo -e "${GREEN}║    日志: $0 --logs           ║${NC}"
  echo -e "${GREEN}║    停止: $0 --stop           ║${NC}"
  echo -e "${GREEN}║    重建: $0 --rebuild        ║${NC}"
  echo -e "${GREEN}║                                              ║${NC}"
  echo -e "${GREEN}╚══════════════════════════════════════════════╝${NC}"
  echo ""

  log "查看实时日志: $COMPOSE_CMD -f $COMPOSE_FILE logs -f yearning juno"
}

# 停止
do_stop() {
  log "停止服务..."
  cd "$SCRIPT_DIR"
  $COMPOSE_CMD -f "$COMPOSE_FILE" down
  log "服务已停止"
}

# 销毁（含数据）
do_destroy() {
  warn "此操作将删除所有容器和数据卷！"
  echo -n "确认? (输入 YES 继续): "
  read -r confirm
  if [ "$confirm" != "YES" ]; then
    echo "已取消"
    exit 0
  fi
  cd "$SCRIPT_DIR"
  $COMPOSE_CMD -f "$COMPOSE_FILE" down -v --remove-orphans
  log "所有容器和数据已清除"
}

# 日志
do_logs() {
  cd "$SCRIPT_DIR"
  $COMPOSE_CMD -f "$COMPOSE_FILE" logs -f yearning juno
}

# 状态
do_status() {
  cd "$SCRIPT_DIR"
  $COMPOSE_CMD -f "$COMPOSE_FILE" ps
}

# ── Main ──
check_docker

case "${1:-up}" in
  --rebuild)
    ensure_env
    REBUILD=true do_up
    ;;
  --stop)
    do_stop
    ;;
  --destroy)
    do_destroy
    ;;
  --logs)
    do_logs
    ;;
  --status)
    do_status
    ;;
  --help|-h)
    cat <<EOF
用法: $0 [命令]

命令:
  (无参数)     构建并启动（首次会构建镜像）
  --rebuild    强制重新构建镜像
  --stop       停止所有服务
  --destroy    停止并删除所有数据（需确认）
  --logs       查看 Yearning 实时日志
  --status     查看容器状态
  -h, --help   显示帮助
EOF
    exit 0
    ;;
  *)
    ensure_env
    do_up
    ;;
esac
