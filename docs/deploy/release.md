# 发布与部署（Release）

> 发布流程（打 tag → 构建镜像）的唯一权威文档。
> 版本号在运行时的展示契约（`/api/v1/version`、CLI、dev 占位规则）见 [Meta API](../api/meta.md)，本文只约定构建与部署流程。

## 1. 发布流程：打 tag → GitHub Actions → GHCR → compose

版本的单一来源是 git tag（命名 `vX.Y.Z`）。构建镜像时版本经 `--build-arg VERSION` 注入 server 二进制（`internal/infra/buildinfo`），前端「关于」页经 API 读取，web 构建不在前端重复注入。

```bash
# 确认发布内容已提交，使用未发布过的新版本号；以下以 v0.0.3 为例
git tag -a v0.0.3 -m "LexiLoop v0.0.3"
git push origin main
git push origin v0.0.3
```

[Release container](../../.github/workflows/release.yml) 在推送版本 tag 后自动运行：校验稳定版本号 → 复用 CI → 下载并校验固定版本词典 → 构建 `linux/amd64` 镜像 → 发布到 `ghcr.io/dongwlin/lexi-loop`。镜像带版本 tag（如 `v0.0.3`）及 `latest`，`latest` 指向最近成功执行的发布（重跑旧版本也会更新它）；部署建议固定版本。非 `vX.Y.Z` 的 tag 与分支手动运行会被拒绝。失败后可重跑原工作流，或在 Actions 中选择已有版本 tag 手动运行；该 tag 必须已经包含工作流文件，旧 tag 不会自动补发。

发布使用 GitHub 自动提供的 `GITHUB_TOKEN`，仅发布 job 获得 `packages: write`，无需额外保存 registry 密钥。包会关联当前仓库，fork 自动使用自己的小写仓库名作为镜像名。首次发布后，在仓库 **Packages → lexi-loop → Package settings** 中确认可见性：GHCR 新包默认私有，需要匿名拉取时将其设为 Public；私有包部署前使用具有 `read:packages` 权限的凭据登录 `ghcr.io`。授权与可见性规则见 [GitHub GHCR 文档](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)。

[CI](../../.github/workflows/ci.yml) 在 main 推送、PR 和手动运行时检查：

- Go 格式、vet、build、race 测试（真实 PostgreSQL testcontainers；集成测试因环境故障被跳过时 CI 失败）、OpenAPI 生成物一致性。
- Node 24 + 根 `packageManager` 固定的 pnpm，锁文件安装、API client 生成物一致性、类型检查、ESLint、Vitest（含真实浏览器 Story / a11y）、Storybook 构建、Playwright 冒烟（自带生产构建）。

发布 job 依赖全部检查通过；PR 检查不登录 GHCR、不发布镜像。推送 tag 只发布容器，不自动变更运行中的部署。

## 2. 本地备用发布：release.sh

```text
deploy/release.sh [选项] [版本]
```

| 项 | 说明 |
| --- | --- |
| `[版本]` | 可选。必须是已存在且指向当前 HEAD 的 git tag；缺省用 `git describe --tags --exact-match` 自动取 |
| `-r, --runtime` | 容器运行时：`docker` 或 `podman`；缺省自动探测（先 docker 后 podman），也可用环境变量 `CONTAINER_RUNTIME` 指定 |
| `-i, --image` | 镜像名，可含 registry 前缀；缺省 `ghcr.io/dongwlin/lexi-loop`（与 docker-compose.yml 的 image 名一致），也可用环境变量 `LEXI_IMAGE` |
| `--push` | 构建成功后 push 上述全部镜像 tag |
| `--strict` | 严格模式：`deploy/dict/ecdict.csv` 缺失时构建失败；缺省只警告，产出「无词典数据」镜像 |
| `-h, --help` | 帮助 |

脚本的前置检查（不满足即失败退出、不做任何构建）：

1. 工作区干净——发布内容必须可追溯；
2. 版本对应的 tag 存在且指向当前 HEAD——防止拿未发布的内容打出正确的版本号；
3. 词典数据 `deploy/dict/ecdict.csv` 存在（缺失时警告；`--strict` 下失败）。

示例：

```bash
deploy/release.sh -r podman                              # 用 podman 构建
deploy/release.sh -i ghcr.io/dongwlin/lexi-loop --push   # 推送到 registry
```

## 3. 词典数据依赖（deploy/dict/）

ECDICT 词典数据（全量约 63 MB、77 万词条）定位为**项目依赖**：仓库内专用数据目录 `deploy/dict/`，`ecdict.csv` 与 `manifest.json` 属数据产物，已加入 `.gitignore` 不进 git；版本固定与校验职责全部在提交入库的 `deploy/dict/fetch.sh`。

```bash
deploy/dict/fetch.sh   # 下载 pinned ecdict.csv → sha256 校验 → 生成 manifest.json
```

- **版本 pin**：`fetch.sh` 内置 pinned 的 ECDICT commit SHA（不用 master）与期望 sha256；词典数据集升级 = 重新 pin 后重建镜像，运行期不做自动升级检查。
- **幂等**：产物已存在且 sha256 校验通过时跳过下载，可安全重复执行。
- **manifest.json**：`{ version, source, sha256, rowsTotal }`，`version` 取 pinned commit 短 SHA；serve 启动的自动导入按「`source_version` 达到 `rowsTotal`」判定词典完整性（契约见 [Meta API §3](../api/meta.md)）。
- **构建期**：Dockerfile 直接 `COPY deploy/dict/ → /app/data/`，构建零网络依赖；缺数据文件时仍可构建（产出「无词典数据」镜像，运行期自动导入自然跳过，行为同无词典）并输出构建警告，发布严格模式用 `release.sh --strict`（缺文件即失败）。
- 镜像运行体积：词典数据使运行镜像 86.5 MB → 约 150 MB；ECDICT 为 MIT 许可，随镜像分发无授权问题。

## 4. 部署运行

仓库根的 `docker-compose.yml` 编排 app + `postgres:18-alpine`（app 依赖 db 健康检查，宿主端口 `${LEXI_HTTP_PORT:-8080}`）：

- **使用 GHCR 发布镜像**（推荐）：等待 Actions 发布成功，设置镜像版本后显式拉取与启动，不在部署机编译：

  ```bash
  export LEXI_IMAGE_TAG=v0.0.3
  docker compose pull app
  docker compose up -d --no-build
  ```

  可将 `LEXI_IMAGE_TAG=v0.0.3` 写入部署目录的 `.env` 持久保存；省略时使用 `latest`。`LEXI_IMAGE` 可覆盖默认镜像仓库（例如 fork 的 GHCR 路径）。回滚镜像时改为此前版本再执行上述命令；数据库迁移兼容性需另行确认。
- **本机构建**：先执行 `deploy/dict/fetch.sh`，再 `deploy/release.sh --strict`，最后 `docker compose up -d --no-build`；脚本与 Compose 默认镜像名一致。若用 `-i` 自定义镜像名，Compose 也需设置相同的 `LEXI_IMAGE`。
- **compose 自行构建**：`VERSION=v0.0.1 docker compose build`（build 段透传 `VERSION`；未注入时为 `dev` 占位，见 Meta API 契约）。

镜像内的运行结构：Caddy 托管 web 静态产物并反代 `/api`、`/healthz` 到同容器的 Go server（`deploy/Caddyfile`）；入口脚本 `deploy/docker-entrypoint.sh` 先显式执行 `lexi-loop migrate up` 再并行启动两者，并转发 SIGTERM 保持优雅关闭。serve 即时可用：内置词典导入在 server 进程内异步执行（不阻塞启动，进度经 `GET /api/v1/dictionary-import` 暴露，见 [Meta API §3](../api/meta.md)）；环境开关 `LEXI_DICT_CSV`（缺省 `/app/data/ecdict.csv`）与 `LEXI_DICT_AUTOCHECK=0`（关闭）。

**podman**：镜像构建用 `deploy/release.sh -r podman`（Dockerfile 的 `RUN --mount` 缓存挂载在 buildah 下同样可用；脚本对 podman 固定 `--format docker`——OCI 镜像格式不支持 HEALTHCHECK，缺省会把它静默丢弃）。Dockerfile 与 compose 的基础镜像一律使用全限定名（`docker.io/library/...`）：podman 默认强制短名称解析交互确认，非限定短名在无 TTY 环境会直接失败，新增基础镜像时须保持全限定。运行编排可用 `podman-compose` 或 `podman kube play`，行为以各自实现为准。

两个运维事实：

- **docker 与 podman 的镜像存储相互独立**：`release.sh` 只填充当次调用的运行时，同一份发布产物在两边都要用时（如 docker 跑 compose、podman 日常验证），分别用各自运行时各跑一次脚本。
- **buildah 会把多阶段构建的中间 stage 留成悬空（`<none>`）镜像**，它们正是 podman 增量构建 `Using cache` 命中的载体；删掉它们下次构建回到冷路径。确需清理时用 `podman image prune`——注意它会连同存储里其它悬空镜像一起删除。

## 5. 核对发布版本

```bash
# CLI：输出版本号 / 构建时间 / Go 版本
docker run --rm --entrypoint /app/lexi-loop ghcr.io/dongwlin/lexi-loop:v0.0.3 version -b

# HTTP：部署后核对（前端「关于」页数据来源）
curl http://127.0.0.1:8080/api/v1/version
```
