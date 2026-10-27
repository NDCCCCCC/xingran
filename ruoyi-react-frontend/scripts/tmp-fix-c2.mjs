// 修复 unknown 化后的类型错误（用后删）
import { readFileSync, writeFileSync } from "node:fs";

function sub(p, a, b, must = true) {
  let t = readFileSync(p, "utf8");
  if (!t.includes(a)) {
    if (must) console.log("MISS", p, JSON.stringify(a.slice(0, 60)));
    return false;
  }
  t = t.split(a).join(b);
  writeFileSync(p, t);
  console.log("OK", p);
  return true;
}

// 1. catch (error: unknown) + error.message → (error as Error).message（8 个 network 页面）
const catchFiles = [
  "src/pages/network/command/index.tsx",
  "src/pages/network/credentials/index.tsx",
  "src/pages/network/devices/index.tsx",
  "src/pages/network/discoveries/index.tsx",
  "src/pages/network/executions/index.tsx",
  "src/pages/network/mac/index.tsx",
  "src/pages/network/backups/index.tsx",
  "src/pages/network/templates/index.tsx",
];
for (const f of catchFiles) {
  let t = readFileSync(f, "utf8");
  const lines = t.split("\n");
  let changed = false;
  for (let i = 0; i < lines.length; i++) {
    if (lines[i].includes("catch (error: unknown) {")) {
      // 向后找配对 catch 块内的 error.message（最多 30 行，遇 "    }" 结束）
      for (let j = i + 1; j < Math.min(i + 30, lines.length); j++) {
        if (lines[j].includes("error.message")) {
          lines[j] = lines[j].replaceAll("error.message", "(error as Error).message");
          changed = true;
          break;
        }
        if (/^\s*}\s*$/.test(lines[j])) break;
      }
    }
  }
  if (changed) {
    writeFileSync(f, lines.join("\n"));
    console.log("OK catch", f);
  } else console.log("NOOP catch", f);
}

// 2. WidgetRenderer spread
sub(
  "src/components/dashboard/widgets/WidgetRenderer.tsx",
  "{...(componentProps as unknown)}",
  "{...(componentProps as Record<string, unknown>)}"
);

// 3. monitor/server handleTableChange antd 类型
sub(
  "src/pages/monitor/server/index.tsx",
  "const handleTableChange = (pagination: unknown, _filters: Record<string, unknown>, sorter: unknown) => {",
  "const handleTableChange = (\n    pagination: TablePaginationConfig,\n    _filters: Record<string, unknown>,\n    sorter: SorterResult<Record<string, unknown>>\n  ) => {"
);
