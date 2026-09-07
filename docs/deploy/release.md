# 发布与部署（Release）

> 发布流程（打 tag → 构建镜像）的唯一权威文档。
> 版本号在运行时的展示契约（`/api/v1/version`、CLI、dev 占位规则）见 [Meta API](../api/meta.md)，本文只约定构建与部署流程。

## 1. 发布流程：打 tag → release.sh → compose

版本的单一来源是 git tag（命名 `vX.Y.Z`）。构建镜像时版本经 `--build-arg VERSION` 注入 server 二进制（`internal/infra/buildinfo`），前端「关于」页经 API 读取，web 构建不在前端重复注入。

```bash
# 1) 确认工作区干净、HEAD 即要发布的内容，打 annotated tag
git tag -a v0.0.1 -m "LexiLoop v0.0.1"

# 2) 构建发布镜像（运行时自动探测，docker 优先；产物 lexi-loop:v0.0.1 与 lexi-loop:latest）
deploy/release.sh

# 3) 部署（使用 release.sh 产出的 lexi-loop:latest，不会重复构建）
docker compose up -d
```

## 2. release.sh 用法

```text
deploy/release.sh [选项] [版本]
```

| 项 | 说明 |
| --- | --- |
| `[版本]` | 可选。必须是已存在且指向当前 HEAD 的 git tag；缺省用 `git describe --tags --exact-match` 自动取 |
| `-r, --runtime` | 容器运行时：`docker` 或 `podman`；缺省自动探测（先 docker 后 podman），也可用环境变量 `CONTAINER_RUNTIME` 指定 |
| `-i, --image` | 镜像名，可含 registry 前缀；缺省 `lexi-loop`（与 docker-compose.yml 的 image 名一致），也可用环境变量 `LEXI_IMAGE` |
| `--push` | 构建成功后 push 上述全部镜像 tag |
| `-h, --help` | 帮助 |

脚本的前置检查（不满足即失败退出、不做任何构建）：

1. 工作区干净——发布内容必须可追溯；
2. 版本对应的 tag 存在且指向当前 HEAD——防止拿未发布的内容打出正确的版本号。

示例：

```bash
deploy/release.sh -r podman                              # 用 podman 构建
deploy/release.sh -i ghcr.io/dongwlin/lexi-loop --push   # 推送到 registry
```

## 3. 部署运行

仓库根的 `docker-compose.yml` 编排 app + `postgres:18-alpine`（app 依赖 db 健康检查，宿主端口 `${LEXI_HTTP_PORT:-8080}`）：

- **使用发布镜像**（推荐）：先跑 `deploy/release.sh`，再 `docker compose up -d`——compose 直接使用已存在的 `lexi-loop:latest`，不重复构建。
- **compose 自行构建**：`VERSION=v0.0.1 docker compose build`（build 段透传 `VERSION`；未注入时为 `dev` 占位，见 Meta API 契约）。

镜像内的运行结构：Caddy 托管 web 静态产物并反代 `/api`、`/healthz` 到同容器的 Go server（`deploy/Caddyfile`）；入口脚本 `deploy/docker-entrypoint.sh` 先显式执行 `lexi-loop migrate up` 再并行启动两者，并转发 SIGTERM 保持优雅关闭。

**podman**：镜像构建用 `deploy/release.sh -r podman`（Dockerfile 的 `RUN --mount` 缓存挂载在 buildah 下同样可用）。运行编排可用 `podman-compose` 或 `podman kube play`，行为以各自实现为准。

## 4. 核对发布版本

```bash
# CLI：输出版本号 / 构建时间 / Go 版本
docker run --rm --entrypoint /app/lexi-loop lexi-loop:v0.0.1 version -b

# HTTP：部署后核对（前端「关于」页数据来源）
curl http://127.0.0.1:8080/api/v1/version
```
