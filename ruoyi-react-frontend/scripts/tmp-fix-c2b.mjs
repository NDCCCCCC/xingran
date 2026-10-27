// Phase 74 C2 第二批修复（用后删）
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

// 1. server imports
sub(
  "src/pages/monitor/server/index.tsx",
  'import type { FC } from "react";',
  'import type { FC } from "react";\nimport type { TablePaginationConfig } from "antd";\nimport type { SorterResult } from "antd/es/table/interface";'
);

// 3. FilterValue 签名（articles/templates 的 request 回调 filters 参数）
const fvImport = 'import type { FilterValue } from "antd/es/table/interface";';
sub(
  "src/pages/knowledge/articles/index.tsx",
  "_filters: Record<string, unknown>,",
  "_filters: Record<string, FilterValue | null>,"
);
let t = readFileSync("src/pages/knowledge/articles/index.tsx", "utf8");
if (!t.includes("FilterValue")) {
  // 在首个 antd import 后插入
  t = t.replace(/(import \{[\s\S]*?\} from "antd";\n)/, `$1${fvImport}\n`);
  writeFileSync("src/pages/knowledge/articles/index.tsx", t);
  console.log("OK articles import");
}
sub(
  "src/pages/network/templates/index.tsx",
  "_filters: Record<string, unknown>,",
  "_filters: Record<string, FilterValue | null>,"
);
t = readFileSync("src/pages/network/templates/index.tsx", "utf8");
if (!t.includes("FilterValue")) {
  t = t.replace(/(import \{[\s\S]*?\} from "antd";\n)/, `$1${fvImport}\n`);
  writeFileSync("src/pages/network/templates/index.tsx", t);
  console.log("OK templates import");
}

// 6. iconUtils FC<any> 恢复 + 登记式 disable（antd AntdIconProps 特例，注释已说明）
sub(
  "src/utils/iconUtils.tsx",
  "// not assignable to ComponentType<unknown> but is to FC<unknown>\nexport const STATIC_ICON_REGISTRY: Record<string, FC<unknown>> = {",
  "// not assignable to ComponentType<unknown> but is to FC<any>（antd AntdIconProps 泛型面限制，\n// unknown 会破坏全部 antd 图标赋值——此处 any 为既有类型边界契约，登记保留）\n// eslint-disable-next-line @typescript-eslint/no-explicit-any -- antd 图标组件类型边界（见上注释）\nexport const STATIC_ICON_REGISTRY: Record<string, FC<any>> = {"
);
