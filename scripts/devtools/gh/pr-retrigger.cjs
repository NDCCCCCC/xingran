#!/usr/bin/env node
/**
 * pr-retrigger.cjs —— 解决「API 更新远端分支不触发 CI」的根本问题
 *
 * 背景：GitHub 的 `synchronize` 事件只在**推送**（git push）时触发，
 * 通过 Git Data API 更新 ref 不触发 ⇒ CI 不会跑新 commit。
 * 反复踩坑后固化为工具：把当前分支内容重建到一条**新分支**，
 * 并对目标 PR 走 close→reopen（reopen 会触发 `reopened` 事件，
 * 部分仓库配置下也会重新排 CI）。两条路都走，幂等。
 *
 * 用法：
 *   node scripts/devtools/gh/pr-retrigger.cjs <prNumber> <newBranch> <fileList.json> <msgFile> [--title=..] [--body-file=..]
 *
 * 环境变量：GH_TOKEN（必需） GH_OWNER / GH_REPO / GH_HOST_IP（可选）
 */
const fs = require("fs");
const https = require("https");
const { execFileSync } = require("child_process");

const TOKEN = process.env.GH_TOKEN || "";
const OWNER = process.env.GH_OWNER || "NDCCCCCC";
const REPO = process.env.GH_REPO || "xingran";
const HOST_IP = process.env.GH_HOST_IP || "";
const API = "api.github.com";

if (!TOKEN) {
  console.error("缺少 GH_TOKEN 环境变量");
  process.exit(2);
}

const argv = process.argv.slice(2);
const pr = argv[0];
const newBranch = argv[1];
const listFile = argv[2];
const msgFile = argv[3];
const titleArg = argv.find((a) => a.startsWith("--title="));
const bodyArg = argv.find((a) => a.startsWith("--body-file="));

function api(method, p, body) {
  return new Promise((resolve, reject) => {
    const pl = body ? JSON.stringify(body) : null;
    const headers = {
      Host: API,
      Authorization: `Bearer ${TOKEN}`,
      Accept: "application/vnd.github+json",
      "User-Agent": "xingran-pr-retrigger",
    };
    if (pl) {
      headers["Content-Type"] = "application/json";
      headers["Content-Length"] = Buffer.byteLength(pl);
    }
    const req = https.request(
      {
        host: HOST_IP || API,
        servername: API,
        port: 443,
        path: p,
        method,
        headers,
        timeout: 120000,
      },
      (res) => {
        const chunks = [];
        res.on("data", (d) => chunks.push(d));
        res.on("end", () => {
          const t = Buffer.concat(chunks).toString();
          try {
            resolve({ status: res.statusCode, json: JSON.parse(t) });
          } catch {
            resolve({ status: res.statusCode, json: { raw: t.slice(0, 200) } });
          }
        });
      }
    );
    req.on("error", reject);
    if (pl) req.write(pl);
    req.end();
  });
}

const git = (args) => execFileSync("git", args, { encoding: "utf8" }).trim();

async function main() {
  const files = JSON.parse(fs.readFileSync(listFile, "utf8"));
  const message = fs.readFileSync(msgFile, "utf8");
  const title = titleArg ? titleArg.slice("--title=".length) : message.split("\n")[0];
  const body = bodyArg ? fs.readFileSync(bodyArg.slice("--body-file=".length), "utf8") : message;

  // 1. 找 base 分支（PR 的 base）
  const prInfo = await api("GET", `/repos/${OWNER}/${REPO}/pulls/${pr}`);
  if (prInfo.status !== 200) {
    console.error(`取 PR #${pr} 失败 HTTP ${prInfo.status}`);
    process.exit(1);
  }
  const base = prInfo.json.base.ref;
  console.log(`[retrigger] PR #${pr} base=${base} → 新分支 ${newBranch}`);

  // 2. 以 base 最新 commit 建新分支
  const baseRef = await api("GET", `/repos/${OWNER}/${REPO}/git/ref/heads/${encodeURIComponent(base)}`);
  const baseSha = baseRef.json.object.sha;
  const baseCommit = await api("GET", `/repos/${OWNER}/${REPO}/git/commits/${baseSha}`);
  let tree = baseCommit.json.tree.sha;

  // 远端 tree 映射（用于跳过一致文件）
  const remoteMap = new Map();
  const t0 = await api("GET", `/repos/${OWNER}/${REPO}/git/trees/${tree}?recursive=1`);
  for (const e of t0.json.tree || []) if (e.type === "blob") remoteMap.set(e.path, e.sha);

  const entries = [];
  for (const f of files) {
    if (!fs.existsSync(f)) {
      console.error(`[abort] 文件不存在: ${f}`);
      process.exit(1);
    }
    const sha = git(["hash-object", "--", f]);
    if (remoteMap.get(f) === sha) continue;
    entries.push({ path: f, mode: "100644", type: "blob", sha });
  }
  if (entries.length === 0) {
    console.log("[retrigger] base 已包含全部目标内容，无需新建分支");
    return;
  }

  const newTree = await api("POST", `/repos/${OWNER}/${REPO}/git/trees`, { base_tree: tree, tree: entries });
  tree = newTree.json.sha;
  const commit = await api("POST", `/repos/${OWNER}/${REPO}/git/commits`, {
    message,
    tree,
    parents: [baseSha],
  });

  // 3. 建新分支 ref
  const create = await api("POST", `/repos/${OWNER}/${REPO}/git/refs`, {
    ref: `refs/heads/${newBranch}`,
    sha: commit.json.sha,
  });
  if (create.status !== 201) {
    console.error(`建分支失败 HTTP ${create.status}: ${JSON.stringify(create.json).slice(0, 200)}`);
    process.exit(1);
  }
  console.log(`[retrigger] 新分支 ${newBranch} → ${commit.json.sha.slice(0, 10)}`);

  // 4. 把 PR 的 head 切到新分支
  const setHead = await api("PATCH", `/repos/${OWNER}/${REPO}/pulls/${pr}`, { head: newBranch });
  console.log(`[retrigger] PR head 切到 ${newBranch} → HTTP ${setHead.status}`);

  // 5. close → reopen（触发 PR 事件，排 CI）
  const closed = await api("PATCH", `/repos/${OWNER}/${REPO}/pulls/${pr}`, { state: "closed" });
  console.log(`[retrigger] close → HTTP ${closed.status}`);
  await new Promise((s) => setTimeout(s, 1500));
  const reopened = await api("PATCH", `/repos/${OWNER}/${REPO}/pulls/${pr}`, { state: "open" });
  console.log(`[retrigger] reopen → HTTP ${reopened.status}`);
  console.log("[retrigger] ✅ 已触发 CI 重排；用 ci-watch.cjs 盯结果");
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
