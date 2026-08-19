# 自动提交（mybatis-go 项目配置）

项目已启用「修改后自动提交仓库」机制，配置全部存于仓库内（可版本化）。日常开发无需关心；仅在配置 hooks、排查 push 失败或新克隆仓库时需要阅读本文件。

- **hooks 目录**：`.githooks/`，通过 `git config core.hooksPath .githooks` 指向（本机已配置）
  - `.githooks/post-commit` — 每次 commit 成功后自动 `git push` 到远程（失败静默降级，不影响本地提交）
- **监视脚本**：`scripts/auto-commit.sh` — 检测工作区变更后自动 `git add -A` + commit（提交信息按变更文件名自动生成）
  - `bash scripts/auto-commit.sh` — 后台监视，默认每 60s 检查一次（Ctrl+C 停止）
  - `bash scripts/auto-commit.sh 10` — 每 10s 检查一次
  - `bash scripts/auto-commit.sh 0` — 单次检查后立即退出
  - commit 后自动触发 post-commit hook 完成 push
- 新克隆仓库需执行一次启用 hooks：`git config core.hooksPath .githooks`
