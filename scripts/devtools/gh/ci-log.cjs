#!/usr/bin/env node
/**
 * ci-log.cjs —— 下载失败 job 的日志并抽出关键错误行
 *
 * 用法：
 *   node scripts/devtools/gh/ci-log.cjs <branch>            # 查该分支最新 ci run 的失败 job
 *   node scripts/devtools/gh/ci-log.cjs --job=<jobId>       # 查指定 job
 *   node scripts/devtools/gh/ci-log.cjs --branch=main --tail=80
 *
 * 环境变量：GH_TOKEN（必需） GH_OWNER / GH_REPO / GH_HOST_IP（可选）
 */
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

function apiRaw(p) {
  return new Promise((r) => {
    const req = https.request(
      {
        host: HOST_IP || API,
        servername: API,
        port: 443,
        path: p,
        method: "GET",
        headers: {
          Host: API,
          Authorization: `Bearer ${TOKEN}`,
          Accept: "application/vnd.github+json",
          "User-Agent": "xingran-ci-log",
        },
        timeout: 120000,
      },
      (res) => {
        const chunks = [];
        res.on("data", (d) => chunks.push(d));
        res.on("end", () => r(Buffer.concat(chunks).toString()));
      }
    );
    req.on("error", (e) => r(String(e.message)));
    req.end();
  });
}

function api(method, p, body) {
  return new Promise((resolve) => {
    const pl = body ? JSON.stringify(body) : null;
    const headers = {
      Host: API,
      Authorization: `Bearer ${TOKEN}`,
      Accept: "application/vnd.github+json",
      "User-Agent": "xingran-ci-log",
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
    req.on("error", (e) => resolve({ status: 0, json: { raw: e.message } }));
    if (pl) req.write(pl);
    req.end();
  });
}

const argv = process.argv.slice(2);
const jobArg = argv.find((a) => a.startsWith("--job="));
const branchArg = argv.find((a) => a.startsWith("--branch="));
const tailArg = argv.find((a) => a.startsWith("--tail="));
const plain = argv.find((a) => !a.startsWith("--"));
const branch = branchArg ? branchArg.slice("--branch=".length) : plain;
const tail = tailArg ? parseInt(tailArg.slice("--tail=".length), 10) : 40;

const KEY = /(--- FAIL|FAIL\s+github|undefined:|cannot use|build failed|Error:|diff coverage|coverage gate|Test Files|Tests\s|not ok|Found multiple|Unable to find)/i;

async function main() {
  let jobId = jobArg ? jobArg.slice("--job=".length) : null;
  let runMeta = null;

  if (!jobId) {
    if (!branch) {
      console.error("用法: ci-log.cjs <branch> | --job=<id>");
      process.exit(2);
    }
    const runs = await api("GET", `/repos/${OWNER}/${REPO}/actions/runs?branch=${encodeURIComponent(branch)}&per_page=3`);
    const run = (runs.json.workflow_runs || []).find((x) => x.name === "ci");
    if (!run) {
      console.log("无 ci run");
      return;
    }
    runMeta = run;
    const jobs = await api("GET", `/repos/${OWNER}/${REPO}/actions/runs/${run.id}/jobs`);
    for (const j of jobs.json.jobs || []) console.log(`  ${j.name}: ${j.conclusion}`);
    const bad = (jobs.json.jobs || []).find((j) => j.conclusion === "failure");
    if (!bad) {
      console.log("无失败 job");
      return;
    }
    jobId = bad.id;
    console.log(`失败 job: ${bad.name} (${jobId})`);
  }

  // 日志取用 gh CLI：`gh run view --job <id> --log`。
  // ⚠️ 不用 GitHub 的 job logs API —— 该端点在 job 刚结束时可能返回空/未就绪
  //   （实测多次拿到 0 行），而 gh CLI 走同一份数据但稳定可用。
  let text = "";
  try {
    text = execFileSync("gh", ["run", "view", "--job", String(jobId), "--log"], {
      encoding: "utf8",
      maxBuffer: 256 * 1024 * 1024,
      stdio: ["ignore", "pipe", "ignore"],
    });
  } catch (e) {
    console.error(`取日志失败（gh run view 退出码 ${e.status}）`);
  }
  const lines = text.split("\n");
  console.log(`总行数 ${lines.length}`);
  const hits = lines.filter((l) => KEY.test(l) && !/npm error invalid/.test(l));
  console.log(`--- 关键行（最后 ${Math.min(hits.length, tail)} 条）---`);
  hits.slice(-tail).forEach((l) => console.log(" " + l.replace(/\x1b\[[0-9;]*m/g, "").slice(0, 260)));
  if (!hits.length) {
    console.log("--- 无匹配关键行，输出末尾 40 行 ---");
    lines
      .filter((l) => l.trim())
      .slice(-40)
      .forEach((l) => console.log(" " + l.replace(/\x1b\[[0-9;]*m/g, "").slice(0, 260)));
  }
}

main();
