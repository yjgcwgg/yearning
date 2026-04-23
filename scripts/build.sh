#!/usr/bin/env bash
#
# build.sh - 从源码构建 Yearning（前端 + 后端）
#
# 用法:
#   ./scripts/build.sh              # 构建前后端
#   ./scripts/build.sh --backend    # 仅构建后端（跳过前端）
#   ./scripts/build.sh --frontend   # 仅构建前端
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$ROOT_DIR/Yearning-next"
FRONTEND_DIR="$ROOT_DIR/gemini-next-next"
EMBED_DIST="$BACKEND_DIR/src/service/dist"
EMBED_CHAT="$BACKEND_DIR/src/service/chat/server/app"
OUTPUT_DIR="$ROOT_DIR/output"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[BUILD]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
fail() { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

BUILD_FRONTEND=true
BUILD_BACKEND=true

for arg in "$@"; do
  case "$arg" in
    --backend)  BUILD_FRONTEND=false ;;
    --frontend) BUILD_BACKEND=false ;;
    --help|-h)
      echo "用法: $0 [--backend|--frontend]"
      echo "  无参数    构建前端 + 后端"
      echo "  --backend  仅构建后端"
      echo "  --frontend 仅构建前端"
      exit 0
      ;;
  esac
done

# ── 检查依赖 ──
check_deps() {
  if $BUILD_FRONTEND; then
    command -v node >/dev/null 2>&1 || fail "未找到 node，请先安装 Node.js >= 16"
    command -v npm  >/dev/null 2>&1 || fail "未找到 npm"
    log "Node $(node -v) / npm $(npm -v)"
  fi
  if $BUILD_BACKEND; then
    command -v go >/dev/null 2>&1 || fail "未找到 go，请先安装 Go >= 1.22"
    log "Go $(go version | awk '{print $3}')"
  fi
}

# ── 构建前端 ──
build_frontend() {
  log "=== 构建前端 ==="
  cd "$FRONTEND_DIR"

  if [ ! -d "node_modules" ]; then
    log "安装前端依赖..."
    npm install --legacy-peer-deps
  fi

  log "执行 Vite 构建..."
  npm run build

  if [ ! -d "dist" ]; then
    fail "前端构建失败，dist/ 目录不存在"
  fi

  log "前端构建完成: $FRONTEND_DIR/dist/"
}

# ── 准备 embed 目录 ──
prepare_embed() {
  log "=== 准备 embed 资源 ==="

  # 复制前端产出到后端 embed 位置
  rm -rf "$EMBED_DIST"
  cp -r "$FRONTEND_DIR/dist" "$EMBED_DIST"
  log "前端产出已复制到 $EMBED_DIST"

  # 确保 chat embed 目录存在（go:embed 要求目录非空）
  if [ ! -f "$EMBED_CHAT/index.html" ]; then
    mkdir -p "$EMBED_CHAT"
    echo '<!DOCTYPE html><html><body><p>Chat module placeholder</p></body></html>' > "$EMBED_CHAT/index.html"
    warn "chat 模块占位文件已创建（如有实际 chat 构建产物请替换）"
  fi
}

# ── 构建后端 ──
build_backend() {
  log "=== 构建后端 ==="
  cd "$BACKEND_DIR"

  log "整理 Go 依赖..."
  go mod tidy

  GOOS=${GOOS:-$(go env GOOS)}
  GOARCH=${GOARCH:-$(go env GOARCH)}
  log "目标平台: ${GOOS}/${GOARCH}"

  log "编译中..."
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
    go build -trimpath -ldflags="-s -w" -o "$OUTPUT_DIR/Yearning" .

  log "后端构建完成: $OUTPUT_DIR/Yearning"
}

# ── 打包产出 ──
package_output() {
  log "=== 打包产出 ==="
  mkdir -p "$OUTPUT_DIR"

  # 复制配置模板
  cp "$BACKEND_DIR/conf.toml.template" "$OUTPUT_DIR/conf.toml"
  log "配置文件模板已复制到 $OUTPUT_DIR/conf.toml"

  log ""
  log "========================================="
  log " 构建完成！产出目录: $OUTPUT_DIR/"
  log "========================================="
  ls -lh "$OUTPUT_DIR/"
}

# ── Main ──
check_deps

if $BUILD_FRONTEND; then
  build_frontend
fi

if $BUILD_BACKEND; then
  prepare_embed
  build_backend
fi

package_output

log ""
log "下一步:"
log "  1. 编辑 $OUTPUT_DIR/conf.toml 配置数据库连接"
log "  2. 首次运行: cd $OUTPUT_DIR && ./Yearning install"
log "  3. 启动服务: cd $OUTPUT_DIR && ./Yearning run"
