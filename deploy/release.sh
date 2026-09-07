#!/usr/bin/env bash
# lexi-loop 发布镜像构建：打 tag 后用它构建注入版本号的容器镜像。
# 发布流程与约定见 docs/deploy/release.md；版本运行时契约见 docs/api/meta.md。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

IMAGE="${LEXI_IMAGE:-lexi-loop}"
RUNTIME="${CONTAINER_RUNTIME:-}"
PUSH=0
VERSION=""

usage() {
	cat <<'EOF'
用法: deploy/release.sh [选项] [版本]

构建发布镜像（先打 tag，再运行本脚本）。版本号经 --build-arg VERSION 注入
server 二进制（infra/buildinfo，经 /api/v1/version 与 lexi-loop version 查询），
镜像同时打 <image>:<版本> 与 <image>:latest（latest 供 docker-compose 直接使用）。

选项:
  -r, --runtime RUNTIME  容器运行时：docker 或 podman；缺省自动探测（先 docker 后
                         podman），也可用环境变量 CONTAINER_RUNTIME 指定
  -i, --image NAME       镜像名，可含 registry 前缀（缺省 lexi-loop，环境变量 LEXI_IMAGE）
      --push             构建成功后 push 上述全部镜像 tag
  -h, --help             显示本帮助

参数:
  版本                   可选。必须是已存在且指向当前 HEAD 的 git tag；缺省自动取
                         当前 HEAD 上的 tag（git describe --tags --exact-match）。

前置检查（不满足即失败退出，不做任何构建）:
  1. 工作区干净（git status 无未提交改动）；
  2. 版本对应的 tag 存在且指向当前 HEAD——防止拿未发布的内容打出正确的版本号。

示例:
  git tag -a v0.0.1 -m "LexiLoop v0.0.1"
  deploy/release.sh                                    # 自动探测运行时（docker 优先）
  deploy/release.sh -r podman                          # 用 podman 构建
  deploy/release.sh -i ghcr.io/dongwlin/lexi-loop --push
EOF
}

die() { printf 'release.sh: %s\n' "$1" >&2; exit 1; }
info() { printf '[release] %s\n' "$1"; }

while [[ $# -gt 0 ]]; do
	case "$1" in
	-r | --runtime)
		[[ $# -ge 2 ]] || die "选项 $1 缺少参数（--help 查看用法）"
		RUNTIME="$2"
		shift 2
		;;
	-i | --image)
		[[ $# -ge 2 ]] || die "选项 $1 缺少参数（--help 查看用法）"
		IMAGE="$2"
		shift 2
		;;
	--push)
		PUSH=1
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	--)
		shift
		;;
	-*)
		die "未知选项: $1（--help 查看用法）"
		;;
	*)
		[[ -z "$VERSION" ]] || die "多余的参数: $1（版本只能指定一次）"
		VERSION="$1"
		shift
		;;
	esac
done

resolve_runtime() {
	local r="$RUNTIME"
	if [[ -z "$r" ]]; then
		local cand
		for cand in docker podman; do
			if command -v "$cand" >/dev/null 2>&1; then
				r="$cand"
				break
			fi
		done
		[[ -n "$r" ]] || die "未找到可用的容器运行时（docker / podman）；请用 --runtime 指定或先安装"
	fi
	command -v "$r" >/dev/null 2>&1 || die "容器运行时 '$r' 不可用：命令不存在"
	printf '%s' "$r"
}

cd "$REPO_ROOT"
[[ -f Dockerfile ]] || die "未找到 Dockerfile（脚本须位于仓库的 deploy/ 下）"
command -v git >/dev/null 2>&1 || die "构建发布镜像需要 git"

[[ -z "$(git status --porcelain)" ]] ||
	die "工作区有未提交改动，发布构建要求干净的工作区（git status 查看）"

if [[ -z "$VERSION" ]]; then
	VERSION="$(git describe --tags --exact-match 2>/dev/null)" ||
		die "当前 HEAD 不在任何 tag 上；先打 tag（git tag -a vX.Y.Z -m ...）或显式传入版本"
else
	tag_commit="$(git rev-parse -q --verify "refs/tags/${VERSION}^{commit}" 2>/dev/null || true)"
	[[ -n "$tag_commit" ]] || die "tag '${VERSION}' 不存在；先打 tag（git tag -a ${VERSION} -m ...）"
	[[ "$tag_commit" == "$(git rev-parse HEAD)" ]] ||
		die "tag '${VERSION}' 不指向当前 HEAD；请 checkout 到该 tag 再构建，或在当前 HEAD 上重新打 tag"
fi

RUNTIME="$(resolve_runtime)"

info "运行时: ${RUNTIME} | 版本: ${VERSION} | 镜像: ${IMAGE}:${VERSION} + :latest"
# podman 默认产出 OCI 格式镜像，Dockerfile 的 HEALTHCHECK 会被静默丢弃；
# 固定 --format docker 保持两种运行时产物一致。
build_args=(build --build-arg "VERSION=${VERSION}"
	-t "${IMAGE}:${VERSION}" -t "${IMAGE}:latest")
[[ "$RUNTIME" == "podman" ]] && build_args+=(--format docker)
"${RUNTIME}" "${build_args[@]}" .

if [[ "$PUSH" -eq 1 ]]; then
	for tag in "${VERSION}" latest; do
		info "push ${IMAGE}:${tag}"
		"${RUNTIME}" push "${IMAGE}:${tag}"
	done
fi

info "完成。可用 ${IMAGE}:${VERSION} 部署（docker compose up -d 直接使用 latest），" \
	"版本核对: ${RUNTIME} run --rm --entrypoint /app/lexi-loop ${IMAGE}:${VERSION} version -b"
