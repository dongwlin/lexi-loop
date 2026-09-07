#!/usr/bin/env bash
# deploy/dict/fetch.sh — 获取 pinned 版本的 ECDICT 词典数据（项目依赖）。
#
# 定位：项目 clone 后的标准初始化步骤之一（与 pnpm install、go mod download
# 同级）；产物 ecdict.csv / manifest.json 属本地数据产物，已加入 .gitignore
# 不进 git。Docker 构建期把本目录 COPY 进镜像（/app/data/），构建零网络依赖。
#
# 行为：下载 pinned commit 的 ecdict.csv → 校验期望 sha256 → 按实际文件
# 生成 manifest.json（version / source / sha256 / rowsTotal）。产物已存在且
# 校验通过时幂等跳过下载，可安全重复执行。
#
# 环境要求：bash + curl + coreutils（sha256sum / awk）。
# 词典数据版本升级 = 重新 pin（见下方 PIN 区），并重建镜像。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# ----------------------------- PIN 区（勿手改数据，只换 pin） -----------------------------
# 数据集：ECDICT ecdict.csv @ skywind3000/ECDICT，pinned 到该文件最后一次
# 变更的 commit（2025-01-02「删除 exchange 字段中冗余的 b/f/z 内容」），
# 不用 master——raw 地址必须可复现。版本号取 pinned commit 短 SHA。
# sha256 为该 commit 下文件的确定性内容摘要，是本脚本唯一的完整性判据。
ECDICT_COMMIT="82c9872576b23118d7c42e920c11beb77f510ae2"
ECDICT_VERSION="82c9872"
EXPECTED_SHA256="1a6947e04785db63613a92e14903cdae7954f7e84860b10e68e5c7cbb3f9c3cf"
CSV_URL="https://raw.githubusercontent.com/skywind3000/ECDICT/${ECDICT_COMMIT}/ecdict.csv"
# --------------------------------------------------------------------------------------------

CSV_FILE="ecdict.csv"
MANIFEST_FILE="manifest.json"

info() { printf '[dict] %s\n' "$1"; }
die() { printf 'fetch.sh: %s\n' "$1" >&2; exit 1; }

sha256_of() { sha256sum "$1" 2>/dev/null | awk '{print $1}'; }

command -v curl >/dev/null 2>&1 || die "需要 curl 下载词典数据"
command -v sha256sum >/dev/null 2>&1 || die "需要 sha256sum（coreutils）校验数据"

if [[ -f "$CSV_FILE" ]] && [[ "$(sha256_of "$CSV_FILE")" == "$EXPECTED_SHA256" ]]; then
	info "ecdict.csv 已存在且 sha256 校验通过，跳过下载（幂等可重跑）"
else
	if [[ -f "$CSV_FILE" ]]; then
		info "已有 ecdict.csv 与期望 sha256 不符，重新下载"
	fi
	tmp="$CSV_FILE.part"
	info "下载 ECDICT 数据: $CSV_URL"
	curl -fL --retry 3 --retry-connrefused -o "$tmp" "$CSV_URL" ||
		die "下载失败：$CSV_URL"
	actual="$(sha256_of "$tmp")"
	[[ "$actual" == "$EXPECTED_SHA256" ]] ||
		die "sha256 校验失败：期望 $EXPECTED_SHA256，实际 $actual（数据源被改动？不要手动改 pin，须重新定位 commit）"
	mv "$tmp" "$CSV_FILE"
	info "ecdict.csv 下载并校验通过（sha256=$EXPECTED_SHA256）"
fi

# rowsTotal 是数据行数（不含表头），由 pinned 文件内容决定（sha256 相同则
# 行数必然相同）；manifest 每次运行重新生成，保证与文件一致。
rows_total="$(awk 'END{print NR-1}' "$CSV_FILE")"
[[ "$rows_total" -gt 0 ]] || die "数据行数为 0，CSV 不完整"

cat > "$MANIFEST_FILE" <<EOF
{
  "version": "$ECDICT_VERSION",
  "source": "ecdict",
  "sha256": "$EXPECTED_SHA256",
  "rowsTotal": $rows_total
}
EOF
info "manifest.json 已生成（version=$ECDICT_VERSION，rowsTotal=$rows_total）"
info "完成。执行 pnpm/docker 构建即可把词典数据打进镜像（见 docs/deploy/release.md）"
