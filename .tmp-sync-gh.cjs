#!/usr/bin/env node
/**
 * 增量同步本地代码到 GitHub NDCCCCCC/xingran
 *
 * 策略：以 GitHub main 的 tree 为 base_tree，只上传本地有差异的文件，
 * 因此 GitHub 独有的资产（.github CI/CD、UserOUService 等）自动保留。
 *
 * 用法：node /tmp/sync-gh.cjs [--dry-run]
 */
const { execSync } = require("child_process");
const fs = require("fs");
const crypto = require("crypto");
const path = require("path");

const REPO = "NDCCCCCC/xingran";
const BRANCH = process.env.SYNC_BRANCH || "main";
const DRY = process.argv.includes("--dry-run");
const CWD = "D:\\CODE\\ClaudeCode\\ruoyi-go-backend";

// 需要从远端删除的文件（本地已不存在 / 与本地重构冲突）
const DELETE_PREFIX = [
  "xingran-react-frontend/.planning/",
  "xingran-react-frontend/coverage/",
];
const DELETE_EXACT = new Set([
  "xingran-react-frontend/14-04-SUMMARY.md",
  "xingran-react-frontend/14-05b-SUMMARY.md",
  "xingran-react-frontend/FRONTEND_OPTIMIZATION_SUMMARY.md",
  "xingran-react-frontend/FRONTEND_PERFORMANCE_AUDIT.md",
  "xingran-react-frontend/FRONTEND_PERFORMANCE_RE-EVALUATION.md",
  // 本地已在 Phase 62-01 将其 fixtures 迁至 test_fixtures_test.go，
  // 保留会与本地 test_fixtures_test.go 的同名常量重复声明（CI typecheck 冲突）
  "internal/services/addomain/user_ou_service_test.go",
]);

process.chdir(CWD);
const TOKEN = execSync("gh auth token", { encoding: "utf8" }).trim();

const EXCLUDE_PREFIX = [
  ".planning/",
  ".git/",
  "node_modules/",
  "dist/",
  "uploads/",
  ".claude/",
  ".trae/",
  ".cursor/",
  ".vscode/",
  // 历史 wip 提交误放的重复嵌套路径（9f2f02a2），非产品代码
  "internal/services/system/internal/",
];
const EXCLUDE_EXACT = new Set([
  "PR_BODY.md",
  "QUICK_FIX_PATCH.md",
  "ruoyi-backend-linux",
  "xingran-backend-linux",
  ".github/workflows/frontend-build.yml",
]);
const EXCLUDE_RE = [
  /^ruoyi-backend(-linux)?$/,
  /^xingran-backend(-linux)?(\.exe)?$/,
  /\.stackdump$/,
  /^add_asset_columns\.sql$/,
  /^check_asset_types\.sql$/,
  /^debug_asset_filter\.sql$/,
  /^build_linux\.sh$/,
  /^check_vdi_menu_fix\.sh$/,
];
const BINARY_EXT = new Set([
  ".png", ".jpg", ".jpeg", ".gif", ".ico", ".webp", ".bmp", ".svg",
  ".woff", ".woff2", ".ttf", ".eot", ".otf", ".mp3", ".mp4", ".zip", ".gz",
  ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".exe", ".dll", ".so", ".dylib",
  ".db", ".sqlite", ".stackdump",
]);

function ghApi(method, endpoint, body) {
  const url = `https://api.github.com${endpoint}`;
  return fetch(url, {
    method,
    headers: {
      Authorization: `Bearer ${TOKEN}`,
      Accept: "application/vnd.github+json",
      "X-GitHub-Api-Version": "2022-11-28",
      "Content-Type": "application/json",
      "User-Agent": "xingran-sync",
    },
    body: body ? JSON.stringify(body) : undefined,
  });
}

function gitBlobSha(buf) {
  const header = Buffer.from(`blob ${buf.length}\0`, "utf8");
  return crypto.createHash("sha1").update(header).update(buf).digest("hex");
}

/** 本地路径 → GitHub 路径 */
function mapPath(rel) {
  return rel.replace(/^xingran-react-frontend\//, "xingran-react-frontend/");
}

/** 文本内容做目录映射替换（二进制跳过） */
function transformContent(rel, buf) {
  const ext = path.extname(rel).toLowerCase();
  if (BINARY_EXT.has(ext)) return buf;
  const text = buf.toString("utf8");
  if (!text.includes("xingran-react-frontend")) return buf;
  return Buffer.from(text.replaceAll("xingran-react-frontend", "xingran-react-frontend"), "utf8");
}

async function main() {
  console.log(`[sync] target=${REPO}@${BRANCH} dryRun=${DRY}`);

  // 1. 远端 main 引用
  const refRes = await ghApi("GET", `/repos/${REPO}/git/ref/heads/${BRANCH}`);
  if (!refRes.ok) throw new Error(`ref fetch failed: ${refRes.status} ${await refRes.text()}`);
  const ref = await refRes.json();
  const headSha = ref.object.sha;
  console.log(`[sync] remote head = ${headSha}`);

  // 2. 远端完整 tree
  const treeRes = await ghApi("GET", `/repos/${REPO}/git/trees/${headSha}?recursive=1`);
  if (!treeRes.ok) throw new Error(`tree fetch failed: ${treeRes.status}`);
  const tree = await treeRes.json();
  const remoteBlobs = new Map();
  for (const e of tree.tree) if (e.type === "blob") remoteBlobs.set(e.path, e.sha);
  console.log(`[sync] remote blobs = ${remoteBlobs.size}`);

  // 3. 本地跟踪文件
  const raw = execSync("git ls-files -z", { cwd: CWD, maxBuffer: 256 * 1024 * 1024 }).toString();
  const statRaw = execSync("git ls-files -s -z", { cwd: CWD, maxBuffer: 256 * 1024 * 1024 }).toString();
  const modeByPath = new Map();
  for (const rec of statRaw.split("\0").filter(Boolean)) {
    const m = rec.match(/^(\d+)\s+[0-9a-f]+\s+\d+\t([\s\S]+)$/);
    if (m) modeByPath.set(m[2], m[1] === "100755" ? "100755" : "100644");
  }

  const toUpload = [];
  let unchanged = 0;
  let skipped = 0;
  for (const rel of raw.split("\0").filter(Boolean)) {
    if (
      EXCLUDE_PREFIX.some((p) => rel.startsWith(p)) ||
      EXCLUDE_EXACT.has(rel) ||
      EXCLUDE_RE.some((re) => re.test(path.basename(rel)))
    ) {
      skipped++;
      continue;
    }
    const ghPath = mapPath(rel);
    const buf = transformContent(rel, fs.readFileSync(path.join(CWD, rel)));
    const sha = gitBlobSha(buf);
    if (remoteBlobs.get(ghPath) === sha) {
      unchanged++;
      continue;
    }
    toUpload.push({ rel, ghPath, sha, buf, mode: modeByPath.get(rel) || "100644" });
  }
  console.log(
    `[sync] scanned=${raw.split("\0").filter(Boolean).length} unchanged=${unchanged} ` +
      `skipped=${skipped} toUpload=${toUpload.length}`
  );
  const newFiles = toUpload.filter((f) => !remoteBlobs.has(f.ghPath));
  const modifiedFiles = toUpload.filter((f) => remoteBlobs.has(f.ghPath));
  console.log(`[sync]   new=${newFiles.length} modified=${modifiedFiles.length}`);

  if (DRY) {
    console.log("\n[dry-run] 新增文件样例:");
    newFiles.slice(0, 20).forEach((f) => console.log(`  + ${f.ghPath}`));
    console.log("\n[dry-run] 修改文件样例:");
    modifiedFiles.slice(0, 20).forEach((f) => console.log(`  M ${f.ghPath}`));
    const ghOnly = [...remoteBlobs.keys()].filter(
      (p) => !new Set(toUpload.map((u) => u.ghPath)).has(p) && !unchangedSet.has(p)
    );
    console.log(`\n[dry-run] 远端保留(本地无对应或未变) 文件数: ${remoteBlobs.size - unchanged}`);
    return;
  }

  // 4. 上传 blob（或复用：blob sha + 内容确定性 = 本地可算出 GitHub 侧同一 sha）
  const SKIP_UPLOAD = process.argv.includes("--skip-upload");
  const CONCURRENCY = 3;
  const MAX_ATTEMPTS = 6;
  const uploaded = [];
  let failCount = 0;
  if (SKIP_UPLOAD) {
    for (const f of toUpload) uploaded.push({ ...f }); // f.sha 已是本地算出的 git blob sha
    console.log(`[sync] skip-upload: reuse ${uploaded.length} locally-derived blob shas`);
  } else {
    for (let i = 0; i < toUpload.length; i += CONCURRENCY) {
      const batch = toUpload.slice(i, i + CONCURRENCY);
      const results = await Promise.all(
        batch.map(async (f) => {
          for (let attempt = 1; attempt <= MAX_ATTEMPTS; attempt++) {
            let res;
            try {
              res = await ghApi("POST", `/repos/${REPO}/git/blobs`, {
                content: f.buf.toString("base64"),
                encoding: "base64",
              });
            } catch (netErr) {
              res = null;
            }
            if (res && res.ok) {
              const j = await res.json();
              return { ...f, sha: j.sha };
            }
            if (attempt === MAX_ATTEMPTS) {
              const txt = res ? await res.text() : "(network error)";
              console.error(
                `BLOB FAIL ${f.ghPath}: ${res ? res.status : "n/a"} ${txt.slice(0, 160)}`
              );
              failCount++;
              return null;
            }
            const wait = Math.min(30000, 1500 * Math.pow(2, attempt - 1));
            await new Promise((r) => setTimeout(r, wait));
          }
          return null;
        })
      );
      results.filter(Boolean).forEach((r) => uploaded.push(r));
      if (i % (CONCURRENCY * 20) === 0) {
        console.log(`[sync] blobs ${uploaded.length}/${toUpload.length}`);
      }
      await new Promise((r) => setTimeout(r, 120));
    }
  }
  if (failCount > 0) throw new Error(`${failCount} blob uploads failed`);
  console.log(`[sync] blobs uploaded = ${uploaded.length}`);

  // 5. 创建新 tree（base_tree 保留远端未提及的条目；小块 + 重试，GitHub 对超大 tree 会 502）
  const entries = uploaded.map((f) => ({
    path: f.ghPath,
    mode: f.mode,
    type: "blob",
    sha: f.sha,
  }));
  // 删除项：sha=null 表示从 base_tree 移除
  const toDelete = [...remoteBlobs.keys()].filter(
    (p) => DELETE_PREFIX.some((pre) => p.startsWith(pre)) || DELETE_EXACT.has(p)
  );
  for (const p of toDelete) entries.push({ path: p, mode: "100644", type: "blob", sha: null });
  if (toDelete.length > 0) {
    console.log(`[sync] will delete ${toDelete.length} remote path(s)`);
    toDelete.slice(0, 15).forEach((p) => console.log(`  - ${p}`));
  }
  let newTreeSha = tree.sha;
  const TREE_CHUNK = 150;
  let chunkNo = 0;
  for (let i = 0; i < entries.length; i += TREE_CHUNK) {
    const chunk = entries.slice(i, i + TREE_CHUNK);
    chunkNo++;
    let ok = false;
    for (let attempt = 1; attempt <= 6 && !ok; attempt++) {
      const res = await ghApi("POST", `/repos/${REPO}/git/trees`, {
        base_tree: newTreeSha,
        tree: chunk,
      });
      if (res.ok) {
        const j = await res.json();
        newTreeSha = j.sha;
        ok = true;
        console.log(`[sync] tree chunk ${chunkNo} (${chunk.length}) -> ${newTreeSha}`);
      } else {
        const txt = await res.text();
        console.error(`[sync] tree chunk ${chunkNo} attempt ${attempt} failed ${res.status}: ${txt.slice(0, 200)}`);
        await new Promise((r) => setTimeout(r, Math.min(30000, 2000 * Math.pow(2, attempt - 1))));
      }
    }
    if (!ok) throw new Error(`tree chunk ${chunkNo} failed after retries`);
    await new Promise((r) => setTimeout(r, 300));
  }

  // 6. 创建 commit
  const message = `feat(xingran): 同步 v1.25 权限治理与样式收敛 + 全量去 RuoYi 化

从本地 v1.25 里程碑成果同步（Phase 84-93，22/22 需求达成）：
- 权限线: 133 处写路由动作级拆分、76 页面三态门控全覆盖、72 个 F 型权限码播种
- 样式线: 453 条暗色手写规则清零、!important 452(≤455)、hex 26(≤60)、index.css 3194 行(≤3200)
- 拆分: room-devices 896→342 行解耦
- 去 RuoYi 化: Go module → xingran-next、Redis 前缀 xingran:、前端 token xingranBrand、
  包名 xingran-react-frontend、品牌 XingRan-Next

保留 GitHub 侧资产（CI/CD、coverage 脚本、规划文件等）不变。`;
  const commitRes = await ghApi("POST", `/repos/${REPO}/git/commits`, {
    message,
    tree: newTreeSha,
    parents: [headSha],
  });
  if (!commitRes.ok) throw new Error(`commit failed: ${commitRes.status} ${await commitRes.text()}`);
  const commit = await commitRes.json();
  console.log(`[sync] commit = ${commit.sha}`);

  // 7. 更新 ref（fast-forward）
  const updRes = await ghApi("PATCH", `/repos/${REPO}/git/refs/heads/${BRANCH}`, {
    sha: commit.sha,
    force: false,
  });
  if (!updRes.ok) throw new Error(`ref update failed: ${updRes.status} ${await updRes.text()}`);
  console.log(`[sync] ✅ pushed ${BRANCH} -> ${commit.sha}`);
}

const unchangedSet = new Set();
main().catch((e) => {
  console.error("[sync] FATAL:", e.message);
  process.exit(1);
});
