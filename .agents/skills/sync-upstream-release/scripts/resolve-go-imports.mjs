#!/usr/bin/env node
// 在 git merge upstream/master 停下之后、提交之前，于仓库根目录运行：
// 1. 把工作区所有 .go 文件中的上游模块路径改写为 fork 的模块路径（取自 HEAD 与 MERGE_HEAD 的 kernel/go.mod）；
// 2. 冲突落在 import 块内的 Go 文件，用「上游的 import 块 + fork 相对合并基线新增的导入」重建整个块，
//    fork 重排过的导入行会在冲突块之外被干净合入，只解冲突块会留下上游已删除的导入；
// 3. 对改动过的文件执行 gofmt，上游原文件本身未通过 gofmt 的保留上游格式，避免下次同步产生无谓冲突；
// 4. 列出仍需手工处理的冲突。全部处理完返回 0，仍有待处理项返回 2，前置条件不满足返回 1。
import {execFileSync} from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const repo = execFileSync("git", ["rev-parse", "--show-toplevel"], {encoding: "utf8"}).trim();
const git = (args) => execFileSync("git", args, {cwd: repo, encoding: "utf8", maxBuffer: 256 * 1024 * 1024});
const tryGit = (args) => {
    try {
        return git(args);
    } catch {
        return null;
    }
};

if (tryGit(["rev-parse", "-q", "--verify", "MERGE_HEAD"]) === null) {
    console.error("未处于合并状态（找不到 MERGE_HEAD），请先执行 git merge upstream/master");
    process.exit(1);
}

const readModule = (text) => ((text || "").match(/^module\s+(\S+)/m) || [])[1];
const forkModule = readModule(tryGit(["show", "HEAD:kernel/go.mod"]));
const upstreamModule = readModule(tryGit(["show", "MERGE_HEAD:kernel/go.mod"]));
if (!forkModule || !upstreamModule) {
    console.error("无法从 HEAD 或 MERGE_HEAD 的 kernel/go.mod 读取模块路径");
    process.exit(1);
}
if (forkModule === upstreamModule) {
    console.log("两侧模块路径相同（" + forkModule + "），无需处理");
    process.exit(0);
}

// 只匹配完整的模块路径前缀，避免误伤 github.com/siyuan-note/dejavu 等其他上游模块
const upstreamRe = new RegExp(upstreamModule.replaceAll(".", "\\.") + "(?=[/\"])", "g");
const rewrite = (text) => text.replace(upstreamRe, () => forkModule);

const SPEC_RE = /^(?:([A-Za-z_]\w*|\.|_)\s+)?"([^"]+)"\s*(?:\/\/.*)?$/;
const MARKER_RE = /^(<<<<<<<|\|\|\|\|\|\|\||=======|>>>>>>>)( |$)/;

const specOf = (line) => {
    const m = line.trim().match(SPEC_RE);
    return m ? {key: (m[1] ? m[1] + " " : "") + "\"" + m[2] + "\"", importPath: m[2]} : null;
};

// 返回第一个 import ( ... ) 块的起止行号（指向括号所在行），没有则返回 null
const findImportBlock = (lines) => {
    const start = lines.findIndex((line) => line.trim() === "import (");
    if (start < 0) {
        return null;
    }
    for (let i = start + 1; i < lines.length; i++) {
        if (lines[i].trim() === ")") {
            return {start, end: i};
        }
    }
    return null;
};

// 块内的冲突标记必须完整成组，否则说明冲突跨出了 import 块，交给人工处理
const markersBalanced = (lines, block) => {
    let depth = 0;
    let found = false;
    for (let i = block.start + 1; i < block.end; i++) {
        const m = lines[i].match(MARKER_RE);
        if (!m) {
            continue;
        }
        found = true;
        if (m[1] === "<<<<<<<") {
            if (depth !== 0) {
                return false;
            }
            depth = 1;
        } else if (m[1] === ">>>>>>>") {
            if (depth !== 1) {
                return false;
            }
            depth = 0;
        } else if (depth !== 1) {
            return false;
        }
    }
    return found && depth === 0;
};

const blockSpecs = (text) => {
    const lines = text.split("\n");
    const block = findImportBlock(lines);
    const specs = new Map();
    if (!block) {
        return specs;
    }
    for (const line of lines.slice(block.start + 1, block.end)) {
        const spec = specOf(line);
        if (spec && !specs.has(spec.key)) {
            specs.set(spec.key, {line, importPath: spec.importPath});
        }
    }
    return specs;
};

// 以上游的 import 块为准，追加 fork 新增的导入：标准库放第一组末尾，其余放最后一组末尾，组内排序交给 gofmt
const composeImportBody = (upstreamText, oursText, baseText) => {
    const upstreamLines = upstreamText.split("\n");
    const upstreamBlock = findImportBlock(upstreamLines);
    const body = upstreamLines.slice(upstreamBlock.start + 1, upstreamBlock.end);
    const upstreamSpecs = blockSpecs(upstreamText);
    const baseSpecs = blockSpecs(baseText);
    const added = [...blockSpecs(oursText)].filter(([key]) => !baseSpecs.has(key) && !upstreamSpecs.has(key)).map(([, spec]) => spec);
    const std = added.filter((spec) => !spec.importPath.split("/")[0].includes(".")).map((spec) => spec.line);
    const others = added.filter((spec) => spec.importPath.split("/")[0].includes(".")).map((spec) => spec.line);
    const firstBlank = body.findIndex((line) => line.trim() === "");
    if (firstBlank < 0) {
        return [...body, ...std, ...others];
    }
    return [...body.slice(0, firstBlank), ...std, ...body.slice(firstBlank), ...others];
};

const unmerged = new Map();
for (const entry of git(["ls-files", "-u", "-z"]).split("\0").filter(Boolean)) {
    const [meta, file] = entry.split("\t");
    if (!unmerged.has(file)) {
        unmerged.set(file, new Set());
    }
    unmerged.get(file).add(meta.split(" ")[2]);
}

const goFiles = [...new Set(git(["ls-files", "-z"]).split("\0").filter((file) => file.endsWith(".go")))];
const changed = [];
const recomposed = [];
const manual = [];
for (const file of goFiles) {
    const abs = path.join(repo, file);
    if (!fs.existsSync(abs)) {
        continue;
    }
    const original = fs.readFileSync(abs, "utf8");
    let text = rewrite(original);
    const stages = unmerged.get(file);
    if (stages && (!stages.has("2") || !stages.has("3"))) {
        manual.push([file, stages.has("2") ? "上游删除了该文件，fork 修改过它" : "fork 删除了该文件，上游修改过它"]);
    } else if (stages) {
        let lines = text.split("\n");
        const block = findImportBlock(lines);
        const upstreamText = rewrite(git(["show", ":3:" + file]));
        if (block && markersBalanced(lines, block) && findImportBlock(upstreamText.split("\n"))) {
            const baseText = stages.has("1") ? rewrite(git(["show", ":1:" + file])) : "";
            const body = composeImportBody(upstreamText, rewrite(git(["show", ":2:" + file])), baseText);
            lines = [...lines.slice(0, block.start + 1), ...body, ...lines.slice(block.end)];
            text = lines.join("\n");
            recomposed.push(file);
        }
        const left = lines.findIndex((line) => MARKER_RE.test(line));
        if (left >= 0) {
            manual.push([file, "第 " + (left + 1) + " 行起仍有冲突标记（不在 import 块内）"]);
        }
    }
    if (text !== original) {
        fs.writeFileSync(abs, text);
        changed.push(file);
    }
}

const pending = new Set(manual.map(([file]) => file));
const formatTargets = [];
const keptFormat = [];
for (const file of changed) {
    if (pending.has(file)) {
        continue;
    }
    const upstream = tryGit(["show", "MERGE_HEAD:" + file]);
    if (upstream !== null) {
        let unformatted;
        try {
            // 必须检查上游原文：改写路径后导入顺序本来就会变，gofmt 必然报未格式化
            unformatted = execFileSync("gofmt", ["-l"], {input: upstream, encoding: "utf8"}).trim() !== "";
        } catch {
            unformatted = true;
        }
        if (unformatted) {
            keptFormat.push(file);
            continue;
        }
    }
    formatTargets.push(file);
}
if (formatTargets.length > 0) {
    try {
        execFileSync("gofmt", ["-w", ...formatTargets], {cwd: repo, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"]});
    } catch (e) {
        console.error("gofmt 失败：\n" + (e.stderr || e.message));
        process.exit(1);
    }
}

for (const [file, stages] of unmerged) {
    if (!file.endsWith(".go")) {
        const kind = !stages.has("2") ? "fork 删除了该文件，上游修改过它" : !stages.has("3") ? "上游删除了该文件，fork 修改过它" : "内容冲突";
        manual.push([file, kind]);
    }
}

console.log("模块路径：" + upstreamModule + " -> " + forkModule);
console.log("改写路径或重建 import 块的 Go 文件：" + changed.length + "（其中重建 import 块 " + recomposed.length + " 个）");
console.log("已 gofmt：" + formatTargets.length);
if (keptFormat.length > 0) {
    console.log("上游原文件未通过 gofmt，保留上游格式：\n  " + keptFormat.join("\n  "));
}
if (manual.length > 0) {
    console.log("仍需手工处理：");
    for (const [file, reason] of manual) {
        console.log("  " + file + "  " + reason);
    }
    process.exit(2);
}
console.log("冲突已全部处理");
