// Phase 74 Wave C2：源码 any → unknown（精确位置，用后删）
import { readFileSync, writeFileSync } from "node:fs";
const spots = `src/components/dashboard/settings/EndpointSelector.tsx:49:15
src/components/dashboard/widgets/WidgetRenderer.tsx:41:47
src/components/operations/WorkstationDeviceTable/index.tsx:446:19
src/components/shared/ExcelExport.tsx:20:28
src/components/shared/ExcelExport.tsx:148:48
src/components/shared/ExcelImport.tsx:261:47
src/components/shared/NetworkExport.tsx:15:28
src/components/shared/NetworkExport.tsx:38:36
src/lib/api.ts:99:20
src/lib/api.ts:100:21
src/lib/api.ts:106:38
src/pages/knowledge/articles/index.tsx:164:32
src/pages/monitor/logs/index.tsx:192:33
src/pages/monitor/server/index.tsx:152:42
src/pages/monitor/server/index.tsx:152:72
src/pages/monitor/server/index.tsx:152:86
src/pages/network/backups/index.tsx:262:21
src/pages/network/command/index.tsx:141:21
src/pages/network/credentials/index.tsx:198:21
src/pages/network/devices/index.tsx:384:21
src/pages/network/discoveries/index.tsx:197:21
src/pages/network/executions/index.tsx:137:21
src/pages/network/mac/index.tsx:249:21
src/pages/network/templates/index.tsx:104:32
src/pages/network/templates/index.tsx:190:21
src/pages/network/templates/index.tsx:331:78
src/pages/operations/rpa/tasks/modals/EditModal.tsx:182:56
src/pages/operations/rpa/tasks/modals/EditModal.tsx:182:74
src/pages/system/config/index.tsx:152:74
src/pages/system/settings/default-theme.tsx:73:37
src/pages/system/settings/default-theme.tsx:79:25
src/utils/iconUtils.tsx:115:54
src/utils/sm-crypto.d.ts:21:21
src/utils/tableHelpers.tsx:122:17
src/utils/tableHelpers.tsx:122:25`
  .split("\n")
  .filter(Boolean);

const byFile = new Map();
for (const s of spots) {
  const parts = s.split(":");
  const col = parseInt(parts.pop());
  const line = parseInt(parts.pop());
  const file = parts.join(":");
  if (!byFile.has(file)) byFile.set(file, []);
  byFile.get(file).push({ line, col });
}
for (const [file, list] of byFile) {
  const lines = readFileSync(file, "utf8").split("\n");
  const perLine = new Map();
  for (const { line, col } of list) {
    if (!perLine.has(line)) perLine.set(line, []);
    perLine.get(line).push(col);
  }
  for (const [lineNo, cols] of perLine) {
    let l = lines[lineNo - 1];
    cols.sort((a, b) => b - a);
    for (const col of cols) {
      const i = col - 1;
      const tok = l.slice(i, i + 3);
      if (tok !== "any") {
        console.log("TOKEN?", file + ":" + lineNo + ":" + col, JSON.stringify(l.slice(Math.max(0, i - 12), i + 15)));
        continue;
      }
      l = l.slice(0, i) + "unknown" + l.slice(i + 3);
    }
    lines[lineNo - 1] = l;
  }
  writeFileSync(file, lines.join("\n"), "utf8");
}
console.log("done");
