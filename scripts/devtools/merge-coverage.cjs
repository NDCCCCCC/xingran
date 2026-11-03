#!/usr/bin/env node
/**
 * merge-coverage.cjs —— 合并多个 vitest shard 的 coverage-final.json
 *
 * 为什么不用 `vitest merge-reports`：
 *   该子命令在 vitest 4 下的参数与文档/直觉不一致（实测 `--reportsDirectory`
 *   报 "Unknown option"，连续两轮 CI 因此失败）。
 *
 * 为什么不用 istanbul-lib-coverage：
 *   v8 provider 产出的 json 虽然叫 coverage-final.json、且文档称与 istanbul
 *   兼容，但与该库的字段期望是否完全吻合无法在本地用真实数据验证（本地
 *   node_modules 装不全）⇒ 不把「合并不报错」押在未验证的库行为上。
 *
 * 本脚本按 istanbul coverage-final.json 的结构做**透明合并**：
 *   - 每个文件路径下有 statementMap / fnMap / branchMap（结构定义）与
 *     s / f / b（执行计数）四组数据；
 *   - **同一文件在不同 shard 里的 map key 是天然对齐的** —— key 由源码位置
 *     决定，而分片只切分「测试文件」，不切分「源码文件」⇒ 同一源码文件的
 *     语句划分在各 shard 中完全一致；
 *   - 故合并 = 对同一路径下同 key 的计数**相加**，map 结构取任一 shard 的。
 *
 * ⚠️ 为防「key 实际不对齐」被静默合并成错误数据，脚本会**校验各 shard 对同一
 * 文件的 map key 集是否一致**，不一致即以非 0 退出（fail-closed）——
 * 这是「合并后分母不得变小」这条正确性闸的前置检查。
 *
 * 用法：node scripts/devtools/merge-coverage.cjs <shardsDir> <outputFile>
 */
const fs = require("fs");
const path = require("path");

const shardsDir = process.argv[2] || "coverage/shards";
const outputFile = process.argv[3] || "coverage/coverage-final.json";

function collect(dir) {
  const out = [];
  if (!fs.existsSync(dir)) return out;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, entry.name);
    if (entry.isDirectory()) out.push(...collect(p));
    else if (entry.name === "coverage-final.json") out.push(p);
  }
  return out;
}

const files = collect(shardsDir).sort();
if (files.length === 0) {
  console.error(`[merge] 在 ${shardsDir} 下未找到任何 coverage-final.json —— 合并失败（fail-closed）`);
  process.exit(1);
}

const COUNT_KEYS = ["s", "f", "b"];
const MAP_KEYS = ["statementMap", "fnMap", "branchMap"];

const merged = new Map(); // path -> { maps, counts }
let totalInputFiles = 0;

for (const file of files) {
  let data;
  try {
    data = JSON.parse(fs.readFileSync(file, "utf8"));
  } catch (e) {
    console.error(`[merge] 解析失败 ${file}: ${e.message}`);
    process.exit(1);
  }
  const paths = Object.keys(data);
  totalInputFiles += paths.length;
  console.log(`[merge] + ${file}（${paths.length} 个文件）`);

  for (const p of paths) {
    const entry = data[p] || {};
    let slot = merged.get(p);
    if (!slot) {
      slot = { maps: {}, counts: { s: {}, f: {}, b: {} }, keysets: {} };
      merged.set(p, slot);
    }
    for (const mk of MAP_KEYS) {
      const m = entry[mk];
      if (!m) continue;
      if (!slot.maps[mk]) {
        slot.maps[mk] = m;
        slot.keysets[mk] = new Set(Object.keys(m));
      } else {
        // 校验 key 集是否一致（分片不应改变同一源码文件的语句/函数/分支划分）
        const prev = slot.keysets[mk];
        for (const k of Object.keys(m)) {
          if (!prev.has(k)) {
            console.error(
              `[merge] ❌ key 不对齐：${p} 的 ${mk} 在不同 shard 中出现了不同的 key（${k}）`,
            );
            console.error("[merge]    ⇒ 分片改变了同一源码文件的语句划分 ⇒ 合并结果不可信，拒绝合并。");
            process.exit(1);
          }
        }
      }
    }
    for (const ck of COUNT_KEYS) {
      const c = entry[ck];
      if (!c) continue;
      if (!slot.counts[ck]) slot.counts[ck] = {};
      for (const [k, v] of Object.entries(c)) {
        slot.counts[ck][k] = (slot.counts[ck][k] || 0) + (typeof v === "number" ? v : 0);
      }
    }
  }
}

// 输出合并后的 coverage-final.json（与 vitest 产出的同构）
const out = {};
for (const [p, slot] of merged) {
  const entry = { path: p };
  for (const mk of MAP_KEYS) if (slot.maps[mk]) entry[mk] = slot.maps[mk];
  for (const ck of COUNT_KEYS) entry[ck] = slot.counts[ck] || {};
  out[p] = entry;
}

fs.mkdirSync(path.dirname(outputFile), { recursive: true });
fs.writeFileSync(outputFile, JSON.stringify(out));

// 对照数据：这是「分片后覆盖率分母不得变小」这条正确性闸的数据来源
let st = 0;
let stCovered = 0;
for (const entry of Object.values(out)) {
  for (const v of Object.values(entry.s || {})) {
    st += 1;
    if (v > 0) stCovered += 1;
  }
}
console.log(`[merge] ✅ 合并 ${files.length} 个分片 → ${outputFile}`);
console.log(`[merge]    输入文件计数(累计): ${totalInputFiles}  |  去重后文件数: ${Object.keys(out).length}`);
console.log(`[merge]    statements: ${stCovered}/${st} = ${st ? ((stCovered / st) * 100).toFixed(2) : "0.00"}%`);
