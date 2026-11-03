#!/usr/bin/env node
/**
 * pr-publish.cjs —— 一条龙：建分支 → 全量对齐推文件 → 建 PR
 *
 * 把「推文件」和「建 PR」绑成一步，避免出现「推了但没触发 CI」的中间态。
 *
 * 用法：
 *   node scripts/devtools/gh/pr-publish.cjs <branch> <fileList.json> <msgFile> \
 *        [--title=..] [--body-file=..] [--base=main] [--delete=path1,path2]
 *
 * fileList.json: JSON 数组，相对仓库根的文件路径；文件不存在时视为「从远端删除」
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
const branch = argv[0];
const listFile = argv[1];
const msgFile = argv[2];
const getArg = (k) => {
  const a = argv.find((x) => x.startsWith(`--${k}=`));
  return a ? a.slice(k.length + 3) : null;
};
const title = getArg("title") || fs.readFileSync(msgFile, "utf8").split("\n")[0];
const bodyFile = getArg("body-file");
const base = getArg("base") || "main";
const deleteArg = getArg("delete");

function api(method, p, body) {
  return new Promise((resolve, reject) => {
    const pl = body ? JSON.stringify(body) : null;
    const headers = {
      Host: API,
      Authorization: `Bearer ${TOKEN}`,
      Accept: "application/vnd.github+json",
      "User-Agent": "xingran-pr-publish",
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
  const message = fs.readFileSync(msgFile, "utf8");
  const body = bodyFile ? fs.readFileSync(bodyFile, "utf8") : message;
  const files = JSON.parse(fs.readFileSync(listFile, "utf8"));
  const deletes = deleteArg ? deleteArg.split(",").filter(Boolean) : [];

  // 1. base 最新 commit + tree
  const baseRef = await api("GET", `/repos/${OWNER}/${REPO}/git/ref/heads/${encodeURIComponent(base)}`);
  if (baseRef.status !== 200) {
    console.error(`取 base 分支失败 HTTP ${baseRef.status}`);
    process.exit(1);
  }
  const baseSha = baseRef.json.object.sha;
  const baseCommit = await api("GET", `/repos/${OWNER}/${REPO}/git/commits/${baseSha}`);
  let tree = baseCommit.json.tree.sha;

  const remoteMap = new Map();
  const t0 = await api("GET", `/repos/${OWNER}/${REPO}/git/trees/${tree}?recursive=1`);
  if (t0.json.truncated) console.warn("[warn] tree 被截断，路径映射不完整");
  for (const e of t0.json.tree || []) if (e.type === "blob") remoteMap.set(e.path, e.sha);

  // 2. 组装变更：新增/修改 + 删除
  const entries = [];
  const unchanged = [];
  for (const f of files) {
    if (!fs.existsSync(f)) {
      console.error(`[abort] 文件不存在（删除请用 --delete）: ${f}`);
      process.exit(1);
    }
    const sha = git(["hash-object", "--", f]);
    if (remoteMap.get(f) === sha) {
      unchanged.push(f);
      continue;
    }
    entries.push({ path: f, mode: "100644", type: "blob", sha });
  }
  for (const d of deletes) {
    if (remoteMap.has(d)) entries.push({ path: d, mode: "100644", type: "blob", sha: null });
    else console.log(`  - ${d}（远端本就没有，跳过删除）`);
  }

  console.log(`[publish] base=${base}@${baseSha.slice(0, 8)}  分支=${branch}`);
  console.log(`[publish] 变更 ${entries.length} 个；内容一致跳过 ${unchanged.length} 个`);
  for (const f of unchanged) console.log(`  = ${f}`);

  if (entries.length === 0) {
    console.log("[publish] 与 base 无差异，无需建 PR");
    return;
  }

  // 3. 建 tree + commit
  const newTree = await api("POST", `/repos/${OWNER}/${REPO}/git/trees`, { base_tree: tree, tree: entries });
  if (newTree.status !== 201) {
    console.error(`建 tree 失败 HTTP ${newTree.status}: ${JSON.stringify(newTree.json).slice(0, 300)}`);
    process.exit(1);
  }
  const commit = await api("POST", `/repos/${OWNER}/${REPO}/git/commits`, {
    message,
    tree: newTree.json.sha,
    parents: [baseSha],
  });
  if (commit.status !== 201) {
    console.error(`建 commit 失败 HTTP ${commit.status}: ${JSON.stringify(commit.json).slice(0, 300)}`);
    process.exit(1);
  }

  // 4. 建分支 ref（已存在则更新）
  const exist = await api("GET", `/repos/${OWNER}/${REPO}/git/ref/heads/${encodeURIComponent(branch)}`);
  let refStatus;
  if (exist.status === 200) {
    const upd = await api("PATCH", `/repos/${OWNER}/${REPO}/git/refs/heads/${encodeURIComponent(branch)}`, {
      sha: commit.json.sha,
      force: false,
    });
    refStatus = `更新 → HTTP ${upd.status}`;
  } else {
    const cr = await api("POST", `/repos/${OWNER}/${REPO}/git/refs`, {
      ref: `refs/heads/${branch}`,
      sha: commit.json.sha,
    });
    refStatus = cr.status === 201 ? "新建" : `创建失败 HTTP ${cr.status}`;
  }
  console.log(`[publish] 分支 ${branch}：${refStatus} @ ${commit.json.sha.slice(0, 10)}`);

  // 5. 建 PR
  const pr = await api("POST", `/repos/${OWNER}/${REPO}/pulls`, {
    title,
    head: branch,
    base,
    body,
  });
  if (pr.status !== 201) {
    console.error(`建 PR 失败 HTTP ${pr.status}: ${JSON.stringify(pr.json).slice(0, 300)}`);
    process.exit(1);
  }
  console.log(`[publish] ✅ PR #${pr.json.number} → ${pr.json.html_url}`);
  console.log(`[publish] 下一步: node scripts/devtools/gh/ci-watch.cjs ${pr.json.number} ${branch} --merge`);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
