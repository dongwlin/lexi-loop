# syntax=docker/dockerfile:1

# ---------- web 构建 ----------
FROM docker.io/library/node:24-alpine AS web-builder

WORKDIR /repo

RUN corepack enable && corepack prepare pnpm@11.13.1 --activate

# 先只拷贝依赖清单安装依赖，利用层缓存
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/web/package.json apps/web/
COPY packages/api-client/package.json packages/api-client/

RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
	pnpm install --frozen-lockfile

# 拷贝源码并构建；VITE_API_BASE_URL 留空 = 同源相对路径，由 Caddy 反代 /api
COPY apps/web apps/web
COPY packages/api-client packages/api-client

RUN pnpm -F @lexi-loop/web build

# ---------- server 构建 ----------
FROM docker.io/library/golang:1.26-alpine AS server-builder

WORKDIR /src

COPY apps/server/go.mod apps/server/go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
	go mod download

WORKDIR /src/apps/server

COPY apps/server ./

# VERSION 由构建方注入；发布构建直接用 deploy/release.sh（见 docs/deploy/release.md），
# 手动构建可传 --build-arg VERSION="$(git describe --tags --always --dirty)"；
# 未注入时为开发构建占位 dev，构建时间取镜像构建时刻（infra/buildinfo）。
ARG VERSION=dev

RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 go build -trimpath \
	-ldflags="-s -w \
	-X github.com/dongwlin/lexi-loop/apps/server/internal/infra/buildinfo.Version=${VERSION} \
	-X github.com/dongwlin/lexi-loop/apps/server/internal/infra/buildinfo.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
	-o /out/lexi-loop .

# ---------- 运行镜像：Caddy 托管 web 并反代 server ----------
FROM docker.io/library/caddy:2-alpine

RUN apk add --no-cache tzdata

COPY --from=server-builder /out/lexi-loop /app/lexi-loop
COPY --from=web-builder /repo/apps/web/dist /srv
COPY deploy/Caddyfile /etc/caddy/Caddyfile
COPY --chmod=0755 deploy/docker-entrypoint.sh /app/docker-entrypoint.sh

ENV GIN_MODE=release

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=15s --retries=3 \
	CMD wget -qO- http://127.0.0.1/healthz >/dev/null || exit 1

ENTRYPOINT ["/app/docker-entrypoint.sh"]
