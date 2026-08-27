#!/usr/bin/env bash
# Cline Go Proxy 一键启动脚本
# 用法: ./start.sh [端口号]   (默认 3457)
# 前台运行，Ctrl+C 停止；日志同时输出到终端和 data/start.log

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

DEFAULT_PORT=3457
PORT=""

# ---- 参数解析：仅接受一个可选的数字端口 ----
case "${1:-}" in
  "") PORT="$DEFAULT_PORT" ;;
  -h|--help)
    echo "用法: ./start.sh [端口号] (默认 $DEFAULT_PORT)"
    exit 0
    ;;
  *)
    if [[ "$1" =~ ^[0-9]+$ ]] && [ "$1" -ge 1 ] && [ "$1" -le 65535 ]; then
      PORT="$1"
    else
      echo "错误: 无效端口 '$1'（应为 1..65535 的数字）" >&2
      exit 1
    fi
    ;;
esac
# 超过一个参数时拒绝
if [ $# -gt 1 ]; then
  echo "错误: 参数过多，只接受一个可选端口号" >&2
  exit 1
fi

# ---- 端口占用检查（不关闭占用者）----
if lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then
  echo "错误: 端口 $PORT 已被占用，占用进程如下（本脚本不会关闭它）："
  lsof -nP -iTCP:"$PORT" -sTCP:LISTEN
  echo "可换一个端口: ./start.sh $((PORT + 1))"
  exit 1
fi

# ---- 运行方式：本机有 Go 则直接编译运行，否则回退到 Docker ----
mkdir -p data
LOG_FILE="data/start.log"

if command -v go >/dev/null 2>&1; then
  echo "==> 使用本机 Go 构建..."
  go build -o cline-proxy . || { echo "错误: go build 失败" >&2; exit 1; }
  echo "==> 启动 cline-proxy (host 0.0.0.0, port $PORT)"
  echo "==> 管理后台: http://127.0.0.1:$PORT/admin/"
  echo "==> 停止方式: Ctrl+C；日志文件: $LOG_FILE"
  exec ./cline-proxy -host 0.0.0.0 -port "$PORT" 2>&1 | tee "$LOG_FILE"
else
  if ! command -v docker >/dev/null 2>&1; then
    echo "错误: 未找到 Go，也未找到 Docker，无法启动。" >&2
    echo "修复方式（二选一）：" >&2
    echo "  1) brew install go" >&2
    echo "  2) 安装 Docker Desktop: https://www.docker.com/products/docker-desktop/" >&2
    exit 1
  fi
  if ! docker info >/dev/null 2>&1; then
    echo "错误: Docker 已安装但守护进程未运行，请先启动 Docker Desktop。" >&2
    exit 1
  fi
  echo "==> 本机无 Go，使用 Docker 构建..."
  docker build -t cline-proxy-local . >/dev/null || { echo "错误: docker build 失败" >&2; exit 1; }
  echo "==> 启动容器 (host 端口 $PORT -> 容器 3457)"
  echo "==> 管理后台: http://127.0.0.1:$PORT/admin/"
  echo "==> 停止方式: Ctrl+C；日志文件: $LOG_FILE"
  DOCKER_ARGS=(
    --rm -i --name cline-proxy-local
    -p "$PORT:3457"
    -v "$ROOT_DIR/data:/app/data"
  )
  # override.md 存在才挂载，避免 Docker 把它创建成目录
  if [ -f override.md ]; then
    DOCKER_ARGS+=(-v "$ROOT_DIR/override.md:/app/override.md:ro")
  fi
  # 前台跟随容器，sig-proxy 默认转发信号，Ctrl+C 即停止容器，不留后台进程
  exec docker run "${DOCKER_ARGS[@]}" cline-proxy-local -host 0.0.0.0 -port 3457 2>&1 | tee "$LOG_FILE"
fi
