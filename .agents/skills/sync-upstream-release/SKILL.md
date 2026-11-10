---
name: sync-upstream-release
description: Synchronize the siyuan-ce fork with a requested siyuan-note release, resolve fork-specific merge conflicts, and verify the integration. Also use for explicitly requested local macOS packaging or CE release commits and tags.
---

# 同步 SiYuan CE 上游版本

SiYuan CE 的 Go 模块为 `github.com/siyuan-community/siyuan/kernel`，官方上游使用 `github.com/siyuan-note/siyuan/kernel`。同步时既要处理 import 冲突，也要改写上游新增文件的模块路径，并在上游重构后移植 CE 定制。

## 范围与授权

- 「更新到 X.Y.Z」默认完成源码合并、验证和暂存。提交、打包、创建发布标签、推送按用户明确授权分别执行；已有授权直接沿用，不重复确认。读取本技能本身不扩大授权。
- 用户随后说「commit」时，只提交当前已验证的改动；不因此补建空的发布提交或标签。仅处理冲突时，不自动继续打包、发布或开启另一轮合并。
- 本地标签清理、打包或发布准备按 [packaging-and-release.md](references/packaging-and-release.md) 执行，仅在请求包含这些动作时读取。

## 确定合并目标

1. 按仓库 `AGENTS.md` 核实路径、CE 工作分支、`origin`、`upstream`、HEAD 和工作区。官方 `upstream/dev` 不是 CE 的开发分支。保留未跟踪文件；与上游路径重名时报告阻塞，不覆盖。
2. **已有 `MERGE_HEAD`：** 记录 HEAD 和 MERGE_HEAD 的完整 SHA，核实目标符合用户请求，直接进入「解决冲突并保留 CE 行为」，跳过本节后续步骤。目标不符时保留现场并报告，不自行重定向合并。不要因合并产生的已跟踪改动而要求清空工作区，也不要重新选择最新上游。
3. **尚无合并：** 合并前已跟踪文件须干净。按仓库规则核实 CE 基线；获取相关远端时使用 `--no-tags`，防止把官方同名发布标签混入 CE 本地标签：

   ```text
   git fetch origin --prune --no-tags
   git fetch upstream --prune --no-tags
   ```

4. 指定版本时，从上游直接读取该版本，而非信任可能属于 CE 的本地同名标签。下面的版本号替换为用户要求的值；refspec **只有源，没有目标**，因此不改写本地标签：

   ```text
   git fetch --no-tags upstream refs/tags/vX.Y.Z
   git rev-parse 'FETCH_HEAD^{commit}'
   ```

   立即保存结果为本次目标 SHA，并用 `git show <目标SHA>:app/package.json`、`git show <目标SHA>:kernel/util/working.go` 核实版本。目标不存在或版本不符时报告，不能改用最新 master 或手工改版本号冒充升级。只有用户要求同步最新上游而未指定版本时，才使用已获取的 `upstream/master` SHA。
5. 用 `git rev-list --left-right --count HEAD...<目标SHA>` 记录 CE 与目标各自独有的提交数。目标已是 HEAD 的祖先时核对当前版本，报告已包含的目标及版本差异，不制造空合并。否则执行 `git merge --no-commit --no-ff <目标SHA>`，即使已获提交授权也先停在可验证状态。后续基线始终使用保存的 SHA 或 MERGE_HEAD，而不是可能推进的远端分支。

## 解决冲突并保留 CE 行为

在仓库根目录运行：

```text
node .agents/skills/sync-upstream-release/scripts/resolve-go-imports.mjs
```

脚本从 HEAD 与 MERGE_HEAD 读取模块路径，改写全部已跟踪 `.go` 文件，包括没有冲突的上游新增文件。对 import 冲突，用「上游完整 import 块 + CE 相对合并基线新增的导入」重建，避免留下上游已移除的导入；仅格式化改动文件，上游原本未格式化的文件保持原格式。退出码 `0` 表示自动处理结束，`2` 表示仍需手工处理，`1` 表示前提不满足；这些退出码不代表已完成暂存或验证。

手工处理剩余项时，对照 `git show :1:<路径>`、`:2:<路径>`、`:3:<路径>`，以及 `git diff <合并基线SHA> HEAD -- <路径>`。保留上游功能和 CE 的实质定制，单纯格式差异采用上游版本。

- **CD 工作流：** 保留官方仓库 owner 守卫、CE 资产名与上游 changelog 来源。`create_release` 不依赖被 CE 跳过的 Android 作业；合入上游新增的其他前置作业和条件，并同步工作流回归测试，不照抄旧的 `needs` 列表。
- **删除或重命名：** CE 已删除的工作流继续保持删除。上游删除的 Go 文件若只有模块路径差异，确认没有调用者后接受删除。含实质定制的文件先查找上游替代实现；能保持行为时移植代码和测试后删除旧文件，只有存在未解决的兼容性或产品取舍时才询问用户。
- **Electron 重构：** 以当前 `app/package.json` 的入口和 `app/electron/README.md` 定位源码，不继续修改已废弃的 `main.js`。检查新 TypeScript 模块中的远端代理参数、连接切换参数、下载地址和 IPC 表。`--proxy` 应由主进程消费、切换连接后沿用，并保留代理失败/超时继续加载及本机网络设置；Apple silicon 下载继续使用 CE 仓库和 CE 资产名。渲染常量与 `shared/channels.ts` 保持一致。
- **模块与依赖：** 活跃 Go 源码和构建脚本使用 CE 模块路径；不要替换 `github.com/siyuan-note/dejavu` 等独立依赖。按文档专供历史上游提交使用的 `.go.txt` 生成器、历史密文和清单保持原样，说明例外，不能全局替换或用当前写入器重建旧样本。`go.sum` 冲突结合最终 `go.mod` 处理，必要时运行 `go mod tidy` 并审查差异，不手写校验和。

## 验证

按 [verification.md](references/verification.md) 检查导入、格式、类型、合约和受影响行为。修复合并引入的错误；对已证实的上游失败单独报告，不能宣称全套通过，也不能删掉断言来掩盖失败。验证和回归范围随实际改动确定，不因脚本退出 `0` 或上游发布成功而省略。

## 暂存与提交

1. 检查工作区没有冲突标记或未处理删除，再用 `git add -u` 暂存本次合并改动；不要用 `git add -A` 带入 `.agents/` 等原有未跟踪内容。新建的集成回归文件按路径显式添加。
2. 审查暂存差异，确认冲突项清空、没有意外 lint 修复或生成产物。未授权提交时保留可审查的合并状态，报告已暂存及剩余验证限制。
3. 已明确授权提交且当前存在 MERGE_HEAD 时，再核实当前分支，执行 `git commit --no-edit --cleanup=strip` 完成合并。保留准确的默认合并主题，不套用普通功能提交的 gitmoji，也不把合并到固定 SHA 描述成同步了更新的 master。`--cleanup=strip` 会去掉 MERGE_MSG 的 `# Conflicts:` 注释。
4. 提交后核对父提交、工作树和相对 `origin` 的领先数；如实汇报目标 SHA、原始两侧独有提交数、保留的 CE 定制、验证结果及提交 SHA。若当前没有 MERGE_HEAD，按仓库普通提交规则处理用户明确要求提交的改动，不再运行合并提交命令。
