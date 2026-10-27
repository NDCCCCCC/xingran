// Phase 74 C2 第三批修复（用后删）
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

// iconUtils：FC<unknown> → FC<any> + disable 登记
sub(
  "src/utils/iconUtils.tsx",
  "// not assignable to ComponentType<unknown> but is to FC<any>\nexport const STATIC_ICON_REGISTRY: Record<string, FC<unknown>> = {",
  "// not assignable to ComponentType<unknown> but is to FC<any>（antd 图标类型边界契约，\n// unknown 会破坏全部 antd 图标赋值——登记保留）\n// eslint-disable-next-line @typescript-eslint/no-explicit-any -- antd 图标组件类型边界（见上注释）\nexport const STATIC_ICON_REGISTRY: Record<string, FC<any>> = {"
);

// ExcelImport：customRequest 断言恢复 antd UploadProps 类型
sub(
  "src/components/shared/ExcelImport.tsx",
  "customRequest={customRequest as unknown}",
  "customRequest={customRequest as UploadProps[\"customRequest\"]}"
);

// EditModal：scriptData 明确为 Action[]（源本就是 editingTask.script as Action[]）
sub(
  "src/pages/operations/rpa/tasks/modals/EditModal.tsx",
  "const parsedActions: Action[] = (scriptData as unknown[]).map((item: unknown) => ({",
  "const parsedActions: Action[] = (scriptData as Action[]).map((item: Action) => ({"
);

// default-theme：values 为 antd 表单值形态 + toHex 参数 Color 兼容形态
sub(
  "src/pages/system/settings/default-theme.tsx",
  "const handleSave = async (values: unknown) => {",
  "const handleSave = async (values: { mode?: string; style?: string; primaryColor?: string | { toHexString?: () => string }; sidebarColor?: string | { toHexString?: () => string } }) => {"
);
sub(
  "src/pages/system/settings/default-theme.tsx",
  "const toHex = (v: unknown): string | undefined => {",
  "const toHex = (v?: string | { toHexString?: () => string }): string | undefined => {"
);
sub(
  "src/pages/system/settings/default-theme.tsx",
  "if (typeof v?.toHexString === \"function\") return v.toHexString();",
  "if (typeof v === \"object\" && v && typeof v.toHexString === \"function\") return v.toHexString();"
);
