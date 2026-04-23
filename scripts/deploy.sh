#!/usr/bin/env bash
#
# deploy.sh - Yearning 一键部署脚本（裸机/VM）
#
# 用法:
#   ./scripts/deploy.sh                    # 交互式部署
#   ./scripts/deploy.sh --auto             # 使用默认配置自动部署
#   ./scripts/deploy.sh --install-dir /opt/yearning  # 指定安装目录
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# ── 默认配置（可通过环境变量覆盖） ──
INSTALL_DIR="${INSTALL_DIR:-/opt/yearning}"
YEARNING_PORT="${YEARNING_PORT:-8000}"
MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_USER="${MYSQL_USER:-yearning}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-}"
MYSQL_DB="${MYSQL_DB:-yearning}"
SECRET_KEY="${SECRET_KEY:-dbcjqheupqjsuwsm}"
RPC_ADDR="${RPC_ADDR:-127.0.0.1:50001}"
AUTO_MODE=false

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log()  { echo -e "${GREEN}[DEPLOY]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
fail() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }
ask()  { echo -en "${CYAN}[INPUT]${NC} $1 [${2}]: "; }

for arg in "$@"; do
  case "$arg" in
    --auto) AUTO_MODE=true ;;
    --install-dir=*) INSTALL_DIR="${arg#*=}" ;;
    --help|-h)
      cat <<EOF
用法: $0 [选项]

选项:
  --auto                 使用默认/环境变量配置，跳过交互提示
  --install-dir=PATH     指定安装目录 (默认: /opt/yearning)
  -h, --help             显示帮助

环境变量:
  INSTALL_DIR            安装目录
  YEARNING_PORT          服务端口 (默认: 8000)
  MYSQL_HOST             MySQL 地址 (默认: 127.0.0.1)
  MYSQL_PORT             MySQL 端口 (默认: 3306)
  MYSQL_USER             MySQL 用户 (默认: yearning)
  MYSQL_PASSWORD         MySQL 密码
  MYSQL_DB               MySQL 数据库 (默认: yearning)
  SECRET_KEY             加密密钥 (16字符)

示例:
  MYSQL_PASSWORD=mypass ./scripts/deploy.sh --auto
EOF
      exit 0
      ;;
  esac
done

# ── 交互式收集配置 ──
collect_config() {
  if $AUTO_MODE; then
    if [ -z "$MYSQL_PASSWORD" ]; then
      fail "自动模式下必须设置 MYSQL_PASSWORD 环境变量"
    fi
    return
  fi

  echo ""
  echo -e "${CYAN}╔══════════════════════════════════════╗${NC}"
  echo -e "${CYAN}║     Yearning 部署配置向导            ║${NC}"
  echo -e "${CYAN}╚══════════════════════════════════════╝${NC}"
  echo ""

  ask "安装目录" "$INSTALL_DIR"
  read -r input; [ -n "$input" ] && INSTALL_DIR="$input"

  ask "服务端口" "$YEARNING_PORT"
  read -r input; [ -n "$input" ] && YEARNING_PORT="$input"

  echo ""
  echo -e "${CYAN}── MySQL 配置 ──${NC}"

  ask "MySQL 地址" "$MYSQL_HOST"
  read -r input; [ -n "$input" ] && MYSQL_HOST="$input"

  ask "MySQL 端口" "$MYSQL_PORT"
  read -r input; [ -n "$input" ] && MYSQL_PORT="$input"

  ask "MySQL 用户" "$MYSQL_USER"
  read -r input; [ -n "$input" ] && MYSQL_USER="$input"

  while [ -z "$MYSQL_PASSWORD" ]; do
    ask "MySQL 密码" "(必填)"
    read -rs input; echo
    MYSQL_PASSWORD="$input"
  done

  ask "MySQL 数据库" "$MYSQL_DB"
  read -r input; [ -n "$input" ] && MYSQL_DB="$input"

  echo ""
  ask "加密密钥 (16字符)" "$SECRET_KEY"
  read -r input; [ -n "$input" ] && SECRET_KEY="$input"

  echo ""
  log "配置确认:"
  echo "  安装目录:    $INSTALL_DIR"
  echo "  服务端口:    $YEARNING_PORT"
  echo "  MySQL:       $MYSQL_USER@$MYSQL_HOST:$MYSQL_PORT/$MYSQL_DB"
  echo "  SecretKey:   ${SECRET_KEY:0:4}************"
  echo ""

  if ! $AUTO_MODE; then
    ask "确认部署? (y/N)" "y"
    read -r confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]] && [ -n "$confirm" ]; then
      echo "已取消"; exit 0
    fi
  fi
}

# ── 构建 ──
do_build() {
  log "=== 开始构建 ==="
  bash "$SCRIPT_DIR/build.sh"
}

# ── 安装 ──
do_install() {
  log "=== 安装到 $INSTALL_DIR ==="
  mkdir -p "$INSTALL_DIR"

  cp "$ROOT_DIR/output/Yearning" "$INSTALL_DIR/Yearning"
  chmod +x "$INSTALL_DIR/Yearning"

  # 生成配置文件
  cat > "$INSTALL_DIR/conf.toml" <<TOML
[Mysql]
Db = "$MYSQL_DB"
Host = "$MYSQL_HOST"
Port = "$MYSQL_PORT"
Password = "$MYSQL_PASSWORD"
User = "$MYSQL_USER"

[General]
SecretKey = "$SECRET_KEY"
RpcAddr = "$RPC_ADDR"
LogLevel = "info"
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

  log "配置文件已生成: $INSTALL_DIR/conf.toml"
}

# ── 初始化数据库 ──
do_init_db() {
  log "=== 初始化数据库 ==="
  cd "$INSTALL_DIR"
  ./Yearning install
  log "数据库初始化完成"
}

# ── 创建 systemd 服务 ──
create_systemd_service() {
  local service_file="/etc/systemd/system/yearning.service"

  if [ "$(id -u)" -ne 0 ]; then
    warn "非 root 用户，跳过 systemd 服务创建"
    warn "手动创建请执行: sudo bash -c 'cat > $service_file' 并填入以下内容:"
    echo ""
    generate_service_unit
    echo ""
    return
  fi

  log "创建 systemd 服务..."
  generate_service_unit > "$service_file"
  systemctl daemon-reload
  systemctl enable yearning
  log "systemd 服务已创建并设为开机启动"
}

generate_service_unit() {
  cat <<EOF
[Unit]
Description=Yearning SQL Audit Platform
After=network.target mysql.service mariadb.service

[Service]
Type=simple
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/Yearning run --port $YEARNING_PORT
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF
}

# ── 启动 Juno SQL 引擎 ──
start_juno() {
  log "=== 启动 Juno SQL 引擎 ==="

  if ! command -v docker >/dev/null 2>&1; then
    warn "未找到 Docker，无法自动启动 Juno"
    echo ""
    echo -e "${YELLOW}请手动启动 Juno SQL 引擎:${NC}"
    echo ""
    echo "  docker run -d \\"
    echo "    --name yearning-juno \\"
    echo "    -e MYSQL_USER=$MYSQL_USER \\"
    echo "    -e MYSQL_PASSWORD=<your_password> \\"
    echo "    -e MYSQL_ADDR=$MYSQL_HOST:$MYSQL_PORT \\"
    echo "    -e MYSQL_DB=$MYSQL_DB \\"
    echo "    -p 50001:50001 \\"
    echo "    --restart always \\"
    echo "    yeelabs/juno"
    echo ""
    echo -e "${YELLOW}Juno 与 Yearning 共用同一个数据库${NC}"
    echo -e "${YELLOW}conf.toml 中 RpcAddr 需指向 Juno 地址 (默认 127.0.0.1:50001)${NC}"
    echo ""
    return
  fi

  local juno_mysql_addr="$MYSQL_HOST:$MYSQL_PORT"
  if [ "$MYSQL_HOST" = "127.0.0.1" ] || [ "$MYSQL_HOST" = "localhost" ]; then
    juno_mysql_addr="172.17.0.1:$MYSQL_PORT"
    warn "MySQL 在本机，Juno 容器将通过 docker0 网关 (172.17.0.1) 连接"
    warn "请确认 MySQL 的 bind-address 允许 172.17.0.1 连接"
  fi

  docker rm -f yearning-juno 2>/dev/null || true
  docker run -d \
    --name yearning-juno \
    -e MYSQL_USER="$MYSQL_USER" \
    -e MYSQL_PASSWORD="$MYSQL_PASSWORD" \
    -e MYSQL_ADDR="$juno_mysql_addr" \
    -e MYSQL_DB="$MYSQL_DB" \
    -p 50001:50001 \
    --restart always \
    yeelabs/juno

  sleep 2
  if docker ps --filter name=yearning-juno --format '{{.Status}}' | grep -q Up; then
    log "Juno 引擎已启动 (端口 50001)"
  else
    warn "Juno 启动可能失败，请检查: docker logs yearning-juno"
  fi
}

# ── 启动服务 ──
do_start() {
  log "=== 启动 Yearning ==="

  if [ "$(id -u)" -eq 0 ] && command -v systemctl >/dev/null 2>&1; then
    systemctl start yearning
    sleep 2
    if systemctl is-active --quiet yearning; then
      log "Yearning 已通过 systemd 启动"
    else
      fail "启动失败，请查看: journalctl -u yearning -f"
    fi
  else
    log "直接启动（前台模式）..."
    cd "$INSTALL_DIR"
    ./Yearning run --port "$YEARNING_PORT"
  fi
}

# ── Main ──
collect_config
do_build
do_install
do_init_db
start_juno
create_systemd_service

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║        Yearning 部署完成！                   ║${NC}"
echo -e "${GREEN}╠══════════════════════════════════════════════╣${NC}"
echo -e "${GREEN}║                                              ║${NC}"
echo -e "${GREEN}║  访问地址:  http://localhost:${YEARNING_PORT}             ║${NC}"
echo -e "${GREEN}║  默认账号:  admin                            ║${NC}"
echo -e "${GREEN}║  默认密码:  Yearning_admin                   ║${NC}"
echo -e "${GREEN}║                                              ║${NC}"
echo -e "${GREEN}║  服务:                                       ║${NC}"
echo -e "${GREEN}║    Yearning : 端口 ${YEARNING_PORT}                       ║${NC}"
echo -e "${GREEN}║    Juno 引擎: 端口 50001 (SQL检测/执行)      ║${NC}"
echo -e "${GREEN}║                                              ║${NC}"
echo -e "${GREEN}║  管理命令:                                   ║${NC}"
echo -e "${GREEN}║    启动: systemctl start yearning            ║${NC}"
echo -e "${GREEN}║    停止: systemctl stop yearning             ║${NC}"
echo -e "${GREEN}║    日志: journalctl -u yearning -f           ║${NC}"
echo -e "${GREEN}║    Juno: docker logs -f yearning-juno        ║${NC}"
echo -e "${GREEN}║                                              ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════╝${NC}"
echo ""

ask "是否立即启动? (Y/n)" "Y"
read -r start_now
if [[ "$start_now" =~ ^[Nn]$ ]]; then
  log "可稍后手动启动: systemctl start yearning"
else
  do_start
fi
