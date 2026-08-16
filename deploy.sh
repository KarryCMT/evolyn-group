#!/usr/bin/env bash

# frog-web / frog-core 交互式 SSH 部署脚本。
# 密码由 ssh 自己在终端中读取，脚本不会保存或打印密码。

set -Eeuo pipefail

readonly SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
readonly WEB_DIR="$SCRIPT_DIR/apps/frog-web"
readonly CORE_DIR="$SCRIPT_DIR/apps/frog-core"
readonly WEB_ARTIFACT="$WEB_DIR/dist"
readonly CORE_ARTIFACT="$CORE_DIR/temp/frog"
# 服务器连接信息和部署目录固定，避免每次部署时误填。
readonly SERVER_HOST="114.132.172.237"
readonly SERVER_PORT="22"
readonly REMOTE_WEB_DIR="/www/wwwroot/frog/web/dist"
readonly REMOTE_CORE_DIR="/www/wwwroot/frog/go"
readonly REMOTE_CORE_SCRIPT_DIR="$REMOTE_CORE_DIR/scripts"

CONTROL_DIR=""
CONTROL_SOCKET=""
SERVER_TARGET=""

info() {
  printf '\n[%s] %s\n' "INFO" "$1"
}

fail() {
  printf '\n[ERROR] %s\n' "$1" >&2
  exit 1
}

prompt_default() {
  local label="$1"
  local default_value="$2"
  local value=""

  read -r -p "$label [$default_value]: " value
  printf '%s' "${value:-$default_value}"
}

cleanup() {
  # 关闭复用连接后，仅移除本次部署创建的空临时目录。
  if [[ -n "$CONTROL_SOCKET" && -S "$CONTROL_SOCKET" && -n "$SERVER_TARGET" ]]; then
    ssh -S "$CONTROL_SOCKET" -O exit "$SERVER_TARGET" >/dev/null 2>&1 || true
  fi
  if [[ -n "$CONTROL_DIR" && -d "$CONTROL_DIR" ]]; then
    rmdir "$CONTROL_DIR" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT INT TERM

for command_name in ssh scp mktemp; do
  command -v "$command_name" >/dev/null 2>&1 ||
    fail "缺少命令：$command_name"
done

printf 'frog SSH 部署\n'
printf '  服务器：%s:%s\n' "$SERVER_HOST" "$SERVER_PORT"
printf '  1) frog-web  -> %s\n' "$REMOTE_WEB_DIR"
printf '  2) frog-core -> %s\n' "$REMOTE_CORE_DIR"
printf '  3) frog-web + frog-core\n'

while true; do
  read -r -p "请选择部署内容 [3]: " DEPLOY_CHOICE
  DEPLOY_CHOICE="${DEPLOY_CHOICE:-3}"
  [[ "$DEPLOY_CHOICE" =~ ^[123]$ ]] && break
  printf '请输入 1、2 或 3\n' >&2
done

DEPLOY_WEB=false
DEPLOY_CORE=false
case "$DEPLOY_CHOICE" in
  1) DEPLOY_WEB=true ;;
  2) DEPLOY_CORE=true ;;
  3)
    DEPLOY_WEB=true
    DEPLOY_CORE=true
    ;;
esac

# 先完成所有选中项目的本地构建，任一构建失败都不会连接服务器。
if [[ "$DEPLOY_WEB" == true ]]; then
  command -v pnpm >/dev/null 2>&1 || fail "缺少命令：pnpm"
  info "本地打包 frog-web"
  (cd -- "$WEB_DIR" && pnpm run build)
  [[ -d "$WEB_ARTIFACT" && -f "$WEB_ARTIFACT/index.html" ]] ||
    fail "frog-web 打包完成后未找到产物：$WEB_ARTIFACT"
fi

if [[ "$DEPLOY_CORE" == true ]]; then
  command -v gf >/dev/null 2>&1 || fail "缺少命令：gf"
  info "本地打包 frog-core"
  (cd -- "$CORE_DIR" && gf build)
  [[ -f "$CORE_ARTIFACT" ]] ||
    fail "frog-core 打包完成后未找到产物：$CORE_ARTIFACT"
fi

SERVER_USER="$(prompt_default "SSH 用户名" "root")"
[[ "$SERVER_USER" =~ ^[A-Za-z_][A-Za-z0-9._-]*$ ]] ||
  fail "SSH 用户名格式无效"

SERVER_TARGET="$SERVER_USER@$SERVER_HOST"
CONTROL_DIR="$(mktemp -d "${TMPDIR:-/tmp}/frog-deploy.XXXXXX")"
CONTROL_SOCKET="$CONTROL_DIR/ssh.sock"

SSH_ARGS=(
  -p "$SERVER_PORT"
  -o "ControlPath=$CONTROL_SOCKET"
  -o "ControlMaster=auto"
  -o "ControlPersist=120"
  -o "ServerAliveInterval=30"
  -o "ServerAliveCountMax=3"
)
SCP_ARGS=(
  -P "$SERVER_PORT"
  -o "ControlPath=$CONTROL_SOCKET"
  -o "ControlMaster=auto"
  -o "ControlPersist=120"
  -C
)

info "正在连接 $SERVER_TARGET:$SERVER_PORT"
printf '如未配置 SSH 密钥，请按终端提示手动输入服务器密码。\n'

# 建立一个可复用的主连接，后续上传和执行脚本不需要重复输入密码。
ssh \
  -p "$SERVER_PORT" \
  -o "ControlPath=$CONTROL_SOCKET" \
  -o "ControlMaster=yes" \
  -o "ControlPersist=120" \
  -o "ServerAliveInterval=30" \
  -o "ServerAliveCountMax=3" \
  -N -f "$SERVER_TARGET"

if [[ "$DEPLOY_WEB" == true ]]; then
  info "上传 frog-web/dist 内容到 $REMOTE_WEB_DIR"
  ssh "${SSH_ARGS[@]}" "$SERVER_TARGET" "mkdir -p -- '$REMOTE_WEB_DIR'"
  scp "${SCP_ARGS[@]}" -r "$WEB_ARTIFACT/." "$SERVER_TARGET:$REMOTE_WEB_DIR/"
fi

if [[ "$DEPLOY_CORE" == true ]]; then
  readonly CORE_UPLOAD_NAME=".frog.uploading.$$"

  # 替换运行中的 Go 程序前先停服，并确认启停脚本都存在。
  info "执行 $REMOTE_CORE_SCRIPT_DIR/stop.sh"
  ssh \
    -t \
    "${SSH_ARGS[@]}" \
    "$SERVER_TARGET" \
    "cd -- '$REMOTE_CORE_SCRIPT_DIR' && { test -f 'stop.sh' || { echo '找不到脚本：$REMOTE_CORE_SCRIPT_DIR/stop.sh' >&2; exit 1; }; } && { test -f 'start.sh' || { echo '找不到脚本：$REMOTE_CORE_SCRIPT_DIR/start.sh' >&2; exit 1; }; } && sh 'stop.sh'"

  info "上传 frog-core/temp/frog 到 $REMOTE_CORE_DIR/frog"
  ssh "${SSH_ARGS[@]}" "$SERVER_TARGET" "mkdir -p -- '$REMOTE_CORE_DIR'"

  # 先上传临时文件再原子替换，避免传输中断损坏服务器上的现有程序。
  scp \
    "${SCP_ARGS[@]}" \
    "$CORE_ARTIFACT" \
    "$SERVER_TARGET:$REMOTE_CORE_DIR/$CORE_UPLOAD_NAME"
  ssh \
    "${SSH_ARGS[@]}" \
    "$SERVER_TARGET" \
    "chmod 755 '$REMOTE_CORE_DIR/$CORE_UPLOAD_NAME' && mv -f -- '$REMOTE_CORE_DIR/$CORE_UPLOAD_NAME' '$REMOTE_CORE_DIR/frog'"

  info "执行 $REMOTE_CORE_SCRIPT_DIR/start.sh"
  ssh \
    -t \
    "${SSH_ARGS[@]}" \
    "$SERVER_TARGET" \
    "cd -- '$REMOTE_CORE_SCRIPT_DIR' && sh 'start.sh'"
fi

info "部署完成"
