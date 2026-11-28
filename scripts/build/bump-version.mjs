#!/usr/bin/env node
/**
 * 版本号工具：仓库根 VERSION 文件是唯一事实来源。
 *
 * 口径：基础版号 1.0.0，每次构建成功后 patch 段 +1（前端 npm postbuild /
 * 纯后端构建脚本收尾各 +1；一体化「前端+后端」脚本用 SKIP_AUTO_BUMP=1 抑制
 * postbuild 递增，末尾统一 +1，保证前后端产物同版本）。
 *
 * 用法：
 *   node scripts/build/bump-version.mjs --get    输出当前版本（无换行，供 ldflags 注入）
 *   node scripts/build/bump-version.mjs --bump   patch +1 写回并打印变更
 *
 * VERSION 文件被 .gitignore 忽略，纯本地；文件缺失时按基础版号 1.0.0 起。
 */
import { existsSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const VERSION_FILE = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  "..",
  "..",
  "VERSION",
);
const BASELINE = "1.0.0";

function readVersion() {
  if (!existsSync(VERSION_FILE)) return BASELINE;
  const raw = readFileSync(VERSION_FILE, "utf8").trim();
  if (!/^\d+\.\d+\.\d+$/.test(raw)) {
    console.error(`[bump-version] VERSION 内容非法: "${raw}"，期望 x.y.z`);
    process.exit(1);
  }
  return raw;
}

const arg = process.argv[2] ?? "--get";
const current = readVersion();

if (arg === "--get") {
  process.stdout.write(current);
} else if (arg === "--bump") {
  if (process.env.SKIP_AUTO_BUMP === "1") {
    console.log("[bump-version] SKIP_AUTO_BUMP=1，跳过递增（由一体化脚本末尾统一处理）");
  } else {
    const [maj, min, pat] = current.split(".").map(Number);
    const next = `${maj}.${min}.${pat + 1}`;
    writeFileSync(VERSION_FILE, `${next}\n`, "utf8");
    console.log(`[bump-version] ${current} -> ${next}`);
  }
} else {
  console.error(`[bump-version] 未知参数: ${arg}（支持 --get / --bump）`);
  process.exit(1);
}
