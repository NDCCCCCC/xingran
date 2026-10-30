#!/usr/bin/env node
/** 带公共 DNS 绕过的 GitHub API 查询工具（本机内网 DNS 不解析 github.com） */
const dns = require("dns");
const { execSync } = require("child_process");

dns.setServers(["8.8.8.8", "8.8.4.4", "114.114.114.114", "223.5.5.5"]);
// 内网 DNS 不解析 github.com，且公共 DNS 偶发超时：解析失败时回落到已知 IP。
const KNOWN_IP = {
  "api.github.com": ["20.205.243.168", "140.82.113.6", "140.82.112.6"],
};
const originalLookup = dns.lookup;
const cache = new Map();
dns.lookup = function (hostname, options, callback) {
  let opts = options;
  let cb = callback;
  if (typeof opts === "function") {
    cb = opts;
    opts = {};
  }
  const fallback =
    KNOWN_IP[hostname] && KNOWN_IP[hostname].length > 0 ? KNOWN_IP[hostname][0] : null;
  const finish = (addr) => {
    if (opts && opts.all) return cb(null, [{ address: addr, family: 4 }]);
    cb(null, addr, 4);
  };
  if (opts && opts.all) {
    dns.resolve4(hostname, (err, addrs) => {
      if (err || !addrs || !addrs.length) {
        if (fallback) return finish(fallback);
        return originalLookup(hostname, opts, cb);
      }
      cb(null, addrs.map((a) => ({ address: a, family: 4 })));
    });
    return;
  }
  if (cache.has(hostname)) return finish(cache.get(hostname));
  dns.resolve4(hostname, (err, addrs) => {
    if (err || !addrs || !addrs.length) {
      if (fallback) {
        cache.set(hostname, fallback);
        return finish(fallback);
      }
      return originalLookup(hostname, opts, cb);
    }
    cache.set(hostname, addrs[0]);
    finish(addrs[0]);
  });
};

// undici(Node fetch) 走 dns.promises.lookup，与回调版 lookup 是两套实现，必须都 patch。
const originalPromiseLookup = dns.promises.lookup.bind(dns.promises);
dns.promises.lookup = async function (hostname, options = {}) {
  const fallback = KNOWN_IP[hostname] && KNOWN_IP[hostname][0];
  try {
    return await originalPromiseLookup(hostname, options);
  } catch (e) {
    if (!fallback) throw e;
    if (options && options.all) return [{ address: fallback, family: 4 }];
    return { address: fallback, family: 4 };
  }
};

const TOKEN = execSync("gh auth token", { encoding: "utf8" }).trim();
const endpoint = process.argv[2];

(async () => {
  const https = require("https");
  const API_HOST = "api.github.com";
  const API_IP = "20.205.243.168";
  const doReq = () =>
    new Promise((resolve, reject) => {
      const req = https.request(
        {
          host: API_IP,
          servername: API_HOST,
          port: 443,
          path: endpoint,
          method: "GET",
          headers: {
            Host: API_HOST,
            Authorization: `Bearer ${TOKEN}`,
            Accept: "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
            "User-Agent": "xingran-q",
          },
          timeout: 60000,
        },
        (res) => {
          const chunks = [];
          res.on("data", (c) => chunks.push(c));
          res.on("end", () => {
            resolve({ status: res.statusCode, text: Buffer.concat(chunks).toString("utf8") });
          });
        }
      );
      req.on("timeout", () => req.destroy(new Error("timeout")));
      req.on("error", reject);
      req.end();
    });

  let out;
  let lastErr;
  for (let attempt = 1; attempt <= 5; attempt++) {
    try {
      out = await doReq();
      lastErr = null;
      break;
    } catch (e) {
      lastErr = e;
      await new Promise((r) => setTimeout(r, 1500 * attempt));
    }
  }
  if (!out) {
    console.error("ERR request failed after retries:", lastErr && lastErr.message);
    process.exit(1);
  }
  const res = { ok: out.status >= 200 && out.status < 300, status: out.status };
  const text = out.text;
  if (!res.ok) {
    console.error(`HTTP ${res.status}: ${text.slice(0, 300)}`);
    process.exit(1);
  }
  // 支持 --jq 简单投影：用 node 表达式路径，如 ".workflow_runs[0].status"
  const jqPath = process.argv[4];
  if (process.argv[3] === "--pick" && jqPath) {
    let obj = JSON.parse(text);
    for (const seg of jqPath.replace(/^\./, "").split(".")) {
      const m = seg.match(/^(\w+)\[(\d+)\]$/);
      if (m) obj = obj?.[m[1]]?.[Number(m[2])];
      else obj = obj?.[seg];
    }
    console.log(typeof obj === "object" ? JSON.stringify(obj) : obj);
  } else {
    console.log(text);
  }
})().catch((e) => {
  console.error("ERR", e.message);
  process.exit(1);
});
