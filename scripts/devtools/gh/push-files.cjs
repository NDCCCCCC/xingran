#!/usr/bin/env node
/**
 * push-files.cjs —— GitHub 远端分支文件推送器（全量对齐语义）
 *
 * 解决的问题：按 commit 增量生成推送清单，会产生「文件版本漂移」
 * —— 某文件的最新版从未进过远端，远端留着旧版，导致 CI 编译失败。
 * 本脚本默认「全量对齐」：对传入的每个文件，先比对远端 blob sha，
 * 不同则推相同内容；相同则跳过。绝不依赖增量清单的正确性。
 *
 * 用法：
 *   node scripts/devtools/gh/push-files.cjs <branch> <fileList.json> <msgFile> [--allow-dirty]
 *
 * fileList.json: JSON 数组，元素为「相对仓库根的文件路径」
 * msgFile:       含 commit message 的文件
 *
 * 环境变量：
 *   GH_TOKEN    必需，GitHub token
 *   GH_OWNER    默认 NDCCCCCC
 *   GH_REPO     默认 xingran
 *   GH_HOST_IP  可选，api.github.com 的解析 IP（本机网络受限时用）
 */
const fs = require("fs");
const https = require("https");
const crypto = require("crypto");

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
if (!branch || !listFile || !msgFile) {
  console.error("用法: push-files.cjs <branch> <fileList.json> <msgFile>");
  process.exit(2);
}

const files = JSON.parse(fs.readFileSync(listFile, "utf8"));
const message = fs.readFileSync(msgFile, "utf8");

function api(method, p, body, raw) {
  return new Promise((resolve, reject) => {
    const pl = body ? JSON.stringify(body) : null;
    const headers = {
      Host: API,
      Authorization: `Bearer ${TOKEN}`,
      Accept: "application/vnd.github+json",
      "User-Agent": "xingran-push-files",
    };
    if (pl) {
      headers["Content-Type"] = "application/json";
      headers["Content-Length"] = Buffer.byteLength(pl);
    }
    const opts = {
      host: HOST_IP || API,
      servername: API,
      port: 443,
      path: p,
      method,
      headers,
      timeout: 120000,
    };
    const req = https.request(opts, (res) => {
      const chunks = [];
      res.on("data", (d) => chunks.push(d));
      res.on("end", () => {
        const buf = Buffer.concat(chunks);
        if (raw) return resolve({ status: res.statusCode, buf });
        const t = buf.toString();
        try {
          resolve({ status: res.statusCode, json: JSON.parse(t) });
        } catch {
          resolve({ status: res.statusCode, json: { raw: t.slice(0, 200) } });
        }
      });
    });
    req.on("error", reject);
    if (pl) req.write(pl);
    req.end();
  });
}

const gitBlob = (p) =>
  new Promise((resolve, reject) => {
    const { execFile } = require("child_process");
    execFile("git", ["hash-object", "--", p], { encoding: "utf8" }, (err, stdout) => {
      if (err) reject(err);
      else resolve(stdout.trim());
    });
  });

async function main() {
  // 1. 取远端分支当前 tree
  const ref = await api("GET", `/repos/${OWNER}/${REPO}/git/ref/heads/${encodeURIComponent(branch)}`);
  if (ref.status !== 200) {
    console.error(`取远端分支失败 HTTP ${ref.status}: ${JSON.stringify(ref.json).slice(0, 200)}`);
    process.exit(1);
  }
  const baseCommit = ref.json.object.sha;
  const commitRes = await api("GET", `/repos/${OWNER}/${REPO}/git/commits/${baseCommit}`);
  let baseTree = commitRes.json.tree.sha;

  // 2. 读远端当前 tree，建立 path→sha 映射
  const remoteMap = new Map();
  const remoteBlobs = new Set();
  let truncated = false;
  const walk = async (sha) => {
    const t = await api("GET", `/repos/${OWNER}/${REPO}/git/trees/${sha}?recursive=1`);
    if (t.json.truncated) truncated = true;
    for (const e of t.json.tree || []) if (e.type === "blob") {
      remoteMap.set(e.path, e.sha);
      remoteBlobs.add(e.sha);
    }
  };
  await walk(baseTree);
  if (truncated) console.warn("[warn] tree 响应被截断，远端路径映射不完整");

  // GitHub Trees API 只接受**仓库中已存在**的 blob sha ⇒ 新文件必须先 POST /git/blobs 上传
  const uploaded = new Map();
  async function ensureBlob(localSha, file) {
    if (remoteBlobs.has(localSha)) return localSha;
    if (uploaded.has(localSha)) return uploaded.get(localSha);
    const content = fs.readFileSync(file).toString("base64");
    const bl = await api("POST", `/repos/${OWNER}/${REPO}/git/blobs`, { content, encoding: "base64" });
    if (bl.status !== 201) {
      console.error(`上传 blob 失败 HTTP ${bl.status}: ${JSON.stringify(bl.json).slice(0, 200)}`);
      process.exit(1);
    }
    uploaded.set(localSha, bl.json.sha);
    return bl.json.sha;
  }

  // 3. 逐文件比对：本地 blob sha vs 远端 blob sha
  const toUpdate = [];
  const skipped = [];
  for (const f of files) {
    if (!fs.existsSync(f)) {
      console.error(`[abort] 文件不存在: ${f}`);
      process.exit(1);
    }
    const localSha = await gitBlob(f);
    const remoteSha = remoteMap.get(f);
    if (localSha === remoteSha) skipped.push(f);
    else toUpdate.push({ path: f, sha: await ensureBlob(localSha, f), mode: "100644" });
  }

  console.log(`[push] 目标分支 ${branch}`);
  console.log(`[push] 文件 ${files.length}：需更新 ${toUpdate.length}，内容一致跳过 ${skipped.length}`);
  for (const f of skipped) console.log(`  = ${f}`);

  if (toUpdate.length === 0) {
    console.log("[push] 远端已是最新，无需推送");
    return;
  }

  // 4. 组装 tree：只改指定路径，其余保持 baseTree
  const entries = toUpdate.map((u) => ({ path: u.path, mode: u.mode, type: "blob", sha: u.sha }));
  const newTree = await api("POST", `/repos/${OWNER}/${REPO}/git/trees`, {
    base_tree: baseTree,
    tree: entries,
  });
  if (newTree.status !== 201) {
    console.error(`建 tree 失败 HTTP ${newTree.status}: ${JSON.stringify(newTree.json).slice(0, 300)}`);
    process.exit(1);
  }
  baseTree = newTree.json.sha;

  // 5. 建 commit
  const commit = await api("POST", `/repos/${OWNER}/${REPO}/git/commits`, {
    message,
    tree: baseTree,
    parents: [baseCommit],
  });
  if (commit.status !== 201) {
    console.error(`建 commit 失败 HTTP ${commit.status}: ${JSON.stringify(commit.json).slice(0, 300)}`);
    process.exit(1);
  }

  // 6. 更新 ref
  const upd = await api("PATCH", `/repos/${OWNER}/${REPO}/git/refs/heads/${encodeURIComponent(branch)}`, {
    sha: commit.json.sha,
    force: false,
  });
  if (upd.status !== 200) {
    console.error(`更新 ref 失败 HTTP ${upd.status}: ${JSON.stringify(upd.json).slice(0, 300)}`);
    process.exit(1);
  }
  console.log(`[push] ✅ ${branch} → ${commit.json.sha.slice(0, 10)}（tree ${baseTree.slice(0, 10)}）`);
  console.log(
    "[push] ⚠️ 注意：GitHub API 更新 ref **不触发** CI 的 synchronize 事件。\n" +
      "       必须让 PR 看到新 commit 才会重新跑 CI ⇒ 用 pr-retrigger.cjs 重建分支或 close/reopen PR。",
  );
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
