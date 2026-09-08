#!/usr/bin/env bash
# lexi-loop 发版编排：bump workspace 版本 → 提交 → push 分支 → 打 tag → push tag。
# 推送版本 tag 后由 GitHub Actions（release.yml）构建并发布镜像；镜像构建的
# 本地备用路径见 deploy/release.sh。流程约定见 docs/deploy/release.md。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# workspace 版本号所在文件；新增带 version 的 workspace 包时同步维护此处
# 与 release.yml 的版本校验。
VERSION_FILES=(
	package.json
	apps/web/package.json
	packages/api-client/package.json
)

usage() {
	cat <<'EOF'
用法: deploy/tag-release.sh vX.Y.Z

一条命令完成发版：更新 workspace 版本号（根、apps/web、packages/api-client
三处 package.json）→ 提交 → push 当前分支 → 打 annotated tag → push tag 触发
Release container 工作流。镜像构建由 GitHub Actions 完成，本地备用构建见
deploy/release.sh。

前置检查（不满足即失败退出，不做任何改动）:
  1. 工作区干净（git status 无未提交改动）；
  2. 当前分支为 main；
  3. 版本号符合稳定版 vX.Y.Z 格式，且 tag 在本地与远程均不存在；
  4. 三处 package.json 的 version 当前值一致。

示例:
  deploy/tag-release.sh v0.0.4
EOF
}

die() { printf 'tag-release.sh: %s\n' "$1" >&2; exit 1; }
info() { printf '[tag-release] %s\n' "$1"; }
run() { printf '[tag-release] + %s\n' "$*"; "$@"; }

VERSION=""
while [[ $# -gt 0 ]]; do
	case "$1" in
	-h | --help)
		usage
		exit 0
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
[[ -n "$VERSION" ]] || { usage; exit 1; }
[[ "$VERSION" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] ||
	die "版本号 '$VERSION' 不符合稳定版 vX.Y.Z 格式"
NEW_VERSION="${VERSION#v}"

cd "$REPO_ROOT"
command -v git >/dev/null 2>&1 || die "需要 git"
[[ -f Dockerfile ]] || die "未找到 Dockerfile（脚本须位于仓库的 deploy/ 下）"

[[ -z "$(git status --porcelain)" ]] ||
	die "工作区有未提交改动，先提交或暂存（git status 查看）"

branch="$(git rev-parse --abbrev-ref HEAD)"
[[ "$branch" == "main" ]] || die "当前分支为 '$branch'，发版须从 main 执行"

if git rev-parse -q --verify "refs/tags/${VERSION}" >/dev/null 2>&1; then
	die "tag '${VERSION}' 已存在（本地）；发版须使用未发布过的新版本号"
fi
if git ls-remote --tags origin "refs/tags/${VERSION}" | grep -q .; then
	die "tag '${VERSION}' 已存在（远程）；发版须使用未发布过的新版本号"
fi

current_version=""
for f in "${VERSION_FILES[@]}"; do
	[[ -f "$f" ]] || die "未找到 $f"
	v="$(sed -n 's/^  "version": "\(.*\)",$/\1/p' "$f")"
	[[ -n "$v" ]] || die "$f 未匹配到顶层 version 字段，请人工检查该文件的缩进格式"
	if [[ -z "$current_version" ]]; then
		current_version="$v"
	elif [[ "$current_version" != "$v" ]]; then
		die "workspace 版本不一致：$f 为 $v，其余为 $current_version；先人工对齐再发版"
	fi
done
info "workspace 版本: ${current_version} → ${NEW_VERSION}（tag ${VERSION}）"

for f in "${VERSION_FILES[@]}"; do
	sed -i "s/^  \"version\": \"[^\"]*\",$/  \"version\": \"${NEW_VERSION}\",/" "$f"
done
changed="$(git diff --numstat -- "${VERSION_FILES[@]}" | wc -l)"
[[ "$changed" -eq "${#VERSION_FILES[@]}" ]] || {
	git checkout -- "${VERSION_FILES[@]}"
	die "版本号替换后预期 ${#VERSION_FILES[@]} 个文件变更，实际 ${changed} 个；已还原，请人工检查"
}

run git add -- "${VERSION_FILES[@]}"
run git commit -m "chore: bump workspace version to ${NEW_VERSION}"
run git push origin "$branch"
run git tag -a "$VERSION" -m "LexiLoop $VERSION"
run git push origin "$VERSION"

info "已推送 ${VERSION}，Release container 工作流开始构建镜像，进度见仓库 Actions 页面"
info "发布成功后部署: LEXI_IMAGE_TAG=${VERSION} docker compose pull app && docker compose up -d --no-build"
