#!/usr/bin/env node
/**
 * ci-watch.cjs —— 盯 CI 结果，绿了自动合并（可选打 tag）
 *
 * 用法：
 *   node scripts/devtools/gh/ci-watch.cjs <prNumber> <branch> [--merge] [--tag=v1.52.0] [--title=..]
 *   node scripts/devtools/gh/ci-watch.cjs --all            # 只看所有分支状态，不合并
 *
 * 环境变量：GH_TOKEN（必需） GH_OWNER / GH_REPO / GH_HOST_IP（可选）
 */
const https = require("https");

const TOKEN = process.env.GH_TOKEN || "";
const OWNER = process.env.GH_OWNER || "NDCCCCCC";
const REPO = process.env.GH_REPO || "xingran";
const HOST_IP = process.env.GH_HOST_IP || "";
const API = "api.github.com";

if (!TOKEN) {
  console.error("缺少 GH_TOKEN 环境变量");
  process.exit(2);
}

function api(method, p, body) {
  return new Promise((resolve) => {
    const pl = body ? JSON.stringify(body) : null;
    const headers = {
      Host: API,
      Authorization: `Bearer ${TOKEN}`,
      Accept: "application/vnd.github+json",
      "User-Agent": "xingran-ci-watch",
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
        timeout: 60000,
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
    req.on("error", (e) => resolve({ status: 0, json: { raw: e.message } }));
    if (pl) req.write(pl);
    req.end();
  });
}

const argv = process.argv.slice(2);
const all = argv.includes("--all");
const doMerge = argv.includes("--merge");
const tagArg = argv.find((a) => a.startsWith("--tag="));
const titleArg = argv.find((a) => a.startsWith("--title="));
const pr = argv.find((a) => /^#?\d+$/.test(a));
const branch = argv.find((a) => !a.startsWith("--") && !/^#?\d+$/.test(a));

async function latestCi(b) {
  const r = await api("GET", `/repos/${OWNER}/${REPO}/actions/runs?branch=${encodeURIComponent(b)}&per_page=3`);
  return (r.json.workflow_runs || []).find((x) => x.name === "ci");
}

async function main() {
  if (all) {
    const runs = await api("GET", `/repos/${OWNER}/${REPO}/actions/runs?per_page=12`);
    const map = {};
    for (const r of runs.json.workflow_runs || []) {
      if (r.name !== "ci") continue;
      if (!map[r.head_branch]) map[r.head_branch] = `${r.head_sha.slice(0, 7)} ${r.status}/${r.conclusion || "-"}`;
    }
    for (const [k, v] of Object.entries(map)) console.log(`  ${k}: ${v}`);
    return;
  }
  if (!pr || !branch) {
    console.error("用法: ci-watch.cjs <prNumber> <branch> [--merge] [--tag=vX.Y.Z]");
    process.exit(2);
  }

  for (let i = 0; i < 90; i++) {
    const run = await latestCi(branch);
    console.log(`[${i + 1}]`, run ? `${run.head_sha.slice(0, 7)} ${run.status}/${run.conclusion || "-"}` : "(无 run)");
    if (run && run.status === "completed") {
      if (run.conclusion === "success") {
        if (!doMerge) {
          console.log("✅ CI 通过（未指定 --merge，不执行合并）");
          return;
        }
        const title = titleArg ? titleArg.slice("--title=".length) : undefined;
        const m = await api("PUT", `/repos/${OWNER}/${REPO}/pulls/${pr}`, {});
        const body = { merge_method: "squash" };
        if (title) body.commit_title = title;
        const mm = await api("PUT", `/repos/${OWNER}/${REPO}/pulls/${pr}/merge`, body);
        console.log(`merge #${pr} → HTTP ${mm.status} ${mm.json.sha ? mm.json.sha.slice(0, 10) : JSON.stringify(mm.json).slice(0, 120)}`);
        if (mm.status === 200 && tagArg) {
          const tag = tagArg.slice("--tag=".length);
          const t = await api("POST", `/repos/${OWNER}/${REPO}/git/tags`, {
            tag,
            message: tag,
            object: mm.json.sha,
            type: "commit",
          });
          console.log(`tag ${tag} → HTTP ${t.status}`);
        }
      } else {
        console.log(`❌ CI 失败（${run.conclusion}）—— 用 ci-log.cjs 查失败 job`);
      }
      return;
    }
    await new Promise((s) => setTimeout(s, 60000));
  }
}

main();
