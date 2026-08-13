#!/usr/bin/env bash
# ============================================================
# auto-commit.sh — 修改后自动提交（后台监视）
#
# 用法：
#   bash scripts/auto-commit.sh            # 默认每 60 秒检查一次（Ctrl+C 停止）
#   bash scripts/auto-commit.sh 10         # 每 10 秒检查一次
#   bash scripts/auto-commit.sh 0          # 单次检查后立即退出
#
# 行为：
#   检测到工作区有变更 → git add -A → 按变更文件名自动生成提交信息 → commit
#   commit 后由 .githooks/post-commit 自动 push 到远程
# ============================================================
set -u

INTERVAL="${1:-60}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

changed_files() {
    {
        git diff --name-only
        git diff --cached --name-only
        git ls-files --others --exclude-standard
    } | sed 's|.*/||' | sort -u | head -5 | paste -sd, -
}

auto_commit() {
    if git diff --quiet && git diff --cached --quiet && [ -z "$(git ls-files --others --exclude-standard)" ]; then
        return 0
    fi
    msg="auto commit: $(changed_files)"
    if git add -A && git commit -q -m "$msg"; then
        echo "[auto-commit] $(date '+%H:%M:%S') committed: ${msg}"
    fi
}

if [ "$INTERVAL" = "0" ]; then
    auto_commit
    exit 0
fi

echo "[auto-commit] watching changes every ${INTERVAL}s (Ctrl+C to stop)"
while true; do
    auto_commit
    sleep "$INTERVAL"
done
