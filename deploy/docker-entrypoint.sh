#!/bin/sh
# 容器入口：先执行数据库迁移，再同时运行 Go server 与 Caddy。
# 本脚本作为 PID 1，须显式转发 SIGTERM/SIGINT，保留 server 的优雅关闭。
set -e

/app/lexi-loop migrate up

/app/lexi-loop serve &
SERVER_PID=$!

caddy run --config /etc/caddy/Caddyfile --adapter caddyfile &
CADDY_PID=$!

term() {
	kill -TERM "$SERVER_PID" 2>/dev/null || true
	kill -TERM "$CADDY_PID" 2>/dev/null || true
}
trap term TERM INT

# 任一进程退出即结束容器：通知另一个进程后收尾
while kill -0 "$SERVER_PID" 2>/dev/null && kill -0 "$CADDY_PID" 2>/dev/null; do
	sleep 1
done

term
wait
