// Phase 74 C2 第五批修复（用后删）
import { readFileSync, writeFileSync } from "node:fs";
function sub(p, a, b) {
  let t = readFileSync(p, "utf8");
  if (!t.includes(a)) {
    console.log("MISS", p, JSON.stringify(a.slice(0, 70)));
    return;
  }
  writeFileSync(p, t.split(a).join(b));
  console.log("OK", p);
}

// 1. server：filters 签名对齐 antd + FilterValue import
sub(
  "src/pages/monitor/server/index.tsx",
  'import type { SorterResult } from "antd/es/table/interface";',
  'import type { FilterValue, SorterResult } from "antd/es/table/interface";'
);
sub(
  "src/pages/monitor/server/index.tsx",
  "    _filters: Record<string, unknown>,",
  "    _filters: Record<string, FilterValue | null>,"
);

// 2. EditModal：后端 script JSON 是动态结构（与前端 Action 接口不同构），恢复 any + 登记豁免
sub(
  "src/pages/operations/rpa/tasks/modals/EditModal.tsx",
  "        const parsedActions: Action[] = (scriptData as Action[]).map((item: Action) => ({",
  "        // 后端 script 字段为动态 JSON（与前端 Action 接口不同构，含 attributes 等原始字段），\n        // 类型边界登记豁免（任务规范允许：动态结构不强行 unknown 化）\n        // eslint-disable-next-line @typescript-eslint/no-explicit-any -- 后端动态 JSON 结构边界\n        const parsedActions: Action[] = (scriptData as any[]).map((item: any) => ({"
);

// 3. default-theme：表单 string 断言到 ThemeConfiguration 字面量联合
sub(
  "src/pages/system/settings/default-theme.tsx",
  "      const config: ThemeConfiguration = {\n        mode: values.mode,\n        style: values.style,",
  "      const config: ThemeConfiguration = {\n        mode: values.mode as ThemeConfiguration[\"mode\"],\n        style: values.style as ThemeConfiguration[\"style\"],"
);
