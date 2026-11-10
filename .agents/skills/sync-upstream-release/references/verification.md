# 合并验证与失败定位

以下命令从 CE 仓库执行，子目录命令注明工作目录。遵守 `AGENTS.md` 的产品构建边界；验证不编译或重启正在使用的内核，也不运行 `pnpm dev`、`pnpm build`。

## 静态与生成检查

- `git grep -n 'github.com/siyuan-note/siyuan/kernel'`：逐项分类。活跃源码、构建命令不得残留官方内核模块路径；历史生成器的文档化用途不改写。
- `rg -n '^(<<<<<<< |=======|>>>>>>> )' kernel app/src app/electron .github`：检查每个命中，避免把正文或分隔线误当作冲突。
- `gofmt -l kernel`：只处理本次改动。对上游原有格式问题，用 `git show <目标SHA>:<路径>` 的原文核实，不对整个目录执行 `gofmt -w`。
- `go vet ./...`（`kernel/`）：检查导入和类型，不产出产品内核二进制。需要时对实际使用的构建标签补充检查。
- 锁文件变化时先 `pnpm install --frozen-lockfile`（`app/`），再 `pnpm run lint`。审查自动修复产生的差异。镜像缺少已锁定公开包时，核实来源并遵守现有 registry 约束；允许切换时可为单次安装传入 `--registry=https://registry.npmjs.org`，不改全局配置、降级依赖或解冻锁文件来绕过错误。
- 导入合约变动后运行 `pnpm run api:check`（`app/`）；额外修改合约或公开声明时，按 `docs/API-CONTRACTS.md` 处理生成与 Petal 同步，不手改生成文件。语言文件变化运行 `python3 scripts/check-lang-keys.py`，用户指南变化验证 `.sy` JSON 与四种语言的对应内容。
- 分别查看相对合并前 HEAD、相对固定上游目标的差异。`git diff --check` 中既有的上游空白或 CE 文件头问题应核实来源并记录；不为消除告警而改写第三方许可或 CE 著作权。

## 行为回归

- 先运行 `pnpm run test:changed --explain`（`app/`）检查选测范围。源码横跨大量模块或共享测试夹具变化时运行 `pnpm run test`；集中改动用 `pnpm run test:affected` 或 `pnpm run test:focused <测试文件> ...`。向 `pnpm test` 传文件不会缩小范围。
- Go 测试按 CE 定制与冲突覆盖范围选择，SQLite 行为使用仓库现有 `fts5,sqlcipher` 标签。至少检查涉及的更新包、Bazaar、导出或其他定制行为；合约变化覆盖 `go test ./apicontract/...`。不把 vet 或 lint 当作行为测试。
- 工作流断言须反映 CE 跳过 Android 等真实差异，同时保留上游新增的门禁覆盖。发现断言失配时核对运行行为，不能只改预期让测试通过。
- macOS 临时路径若因 `/var` 的符号链接触发路径保护，确认原因后仅对测试命令指定物理目录，例如 `env TMPDIR=/private/tmp go test ...`；不放宽产品路径校验。
- Electron 用例成片报 `app`、`enable` 或 `initialize` 缺失时，先核对实际 Node 版本、仓库 engines 和模块 mock 参数。需要时修复测试兼容层并复测，不据此认定大量生产模块都已损坏。

## 基线失败与最终结果

出现失败时先修复本次引入的集成问题。其余失败用固定上游 SHA 的源码、相同依赖、工具版本和运行参数在独立临时目录复现；带齐夹具与资源，缺文件的对比不构成基线证明。必要的测试环境兼容调整应在两侧保持一致并披露。不要重置工作区来切换基线或移动发布标签来配合验证。

保存完整日志并汇总通过、失败、跳过及未执行的检查；不把中断的测试算作完成。修改之后至少复跑受影响测试：若没有再跑全套，区分原全量结果与后续选测结果，不能合成为一次虚构的全量通过。已经通过且未受后续修改影响的检查无需重复。
