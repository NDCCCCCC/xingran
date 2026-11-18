/**
 * 机构层级数据权限 —— 机构锚点（anchor）共享常量
 *
 * 契约来源：`.planning/notes/261118-anchor-scope-contract.md` §二 2.2 / §五 / §六
 *
 * ⚠️ 硬约束（违反即事故）：
 * 1. 锚点维度与「数据范围（dataScope）」是**正交的两个维度**，本文件的选项
 *    **绝不能**塞进 `DATA_SCOPE_OPTIONS`（角色页 constants.ts）——那会把机构模式
 *    数值混入 `MAX(data_scope)` 排序，导致 All(1) 用户被静默收窄（契约 C1）。
 * 2. 只实现 0 / 2 / 3 / 5 四个模式；1（相对层深）与 4（sys_manager_org）已弃用，
 *    后端收到即 400 拒绝，前端也不得提供、不得发出。
 */

import type { AnchorMode, OrgLevel } from "@/types/system";

/** 锚点模式选项（固定枚举 → 业务禁搜） */
export const ANCHOR_MODE_OPTIONS = [
  { label: "不参与机构授权", value: 0 },
  { label: "指定部门（含下级）", value: 2 },
  { label: "外部机构（最近）", value: 3 },
  { label: "指定机构层级（最近）", value: 5 },
] as const;

/** 已实现的锚点模式集合（0/2/3/5） */
export const ANCHOR_MODES: readonly AnchorMode[] = [0, 2, 3, 5];

/** 锚点模式中文名（含已弃用的 1/4 解释，便于把历史脏数据解释清楚） */
export const ANCHOR_MODE_LABELS: Record<number, string> = {
  0: "不参与机构授权",
  2: "指定部门（含下级）",
  3: "外部机构（最近）",
  5: "指定机构层级（最近）",
};

/** 机构层级选项（`sys_dept.org_level`） */
export const ORG_LEVEL_OPTIONS = [
  { label: "非机构", value: 0 },
  { label: "分公司", value: 1 },
  { label: "中支", value: 2 },
  { label: "支公司", value: 3 },
  { label: "网点", value: 4 },
] as const;

/** 机构层级中文名 */
export const ORG_LEVEL_LABELS: Record<number, string> = {
  0: "非机构",
  1: "分公司",
  2: "中支",
  3: "支公司",
  4: "网点",
};

/**
 * 锚点**层级参数**选项（仅 `anchorMode=5` 使用）
 *
 * 值必须是字符串 "1"~"4"（后端 `anchor_param` 为 string，`ValidateAnchor`
 * 要求 `"1"`~`"4"`），且 0=非机构不可选（契约 §二 2.2）。
 */
export const ANCHOR_LEVEL_OPTIONS = [
  { label: "分公司", value: "1" },
  { label: "中支", value: "2" },
  { label: "支公司", value: "3" },
  { label: "网点", value: "4" },
] as const;

/** 合法的锚点层级参数字符串集合 */
export const ANCHOR_LEVEL_PARAMS = ["1", "2", "3", "4"] as const;

/** 归一化任意输入为已实现的锚点模式（非法值一律落到 0=不参与） */
export function normalizeAnchorMode(value: unknown): AnchorMode {
  const n = typeof value === "number" ? value : Number(value);
  return n === 2 || n === 3 || n === 5 ? n : 0;
}

/** 归一化任意输入为机构层级（非法值一律落到 0=非机构） */
export function normalizeOrgLevel(value: unknown): OrgLevel {
  const n = typeof value === "number" ? value : Number(value);
  return n === 1 || n === 2 || n === 3 || n === 4 ? n : 0;
}

/** 锚点模式展示文案（用于列表列 / 预览摘要） */
export function anchorModeLabel(mode?: number | null): string {
  if (mode === undefined || mode === null) return ANCHOR_MODE_LABELS[0];
  return ANCHOR_MODE_LABELS[mode] ?? `未知模式(${mode})`;
}

/** 机构层级展示文案 */
export function orgLevelLabel(level?: number | null): string {
  if (level === undefined || level === null) return ORG_LEVEL_LABELS[0];
  return ORG_LEVEL_LABELS[level] ?? `未知层级(${level})`;
}

/** 锚点模式是否显示参数输入（mode=2 部门 / mode=5 层级） */
export function anchorModeNeedsParam(mode?: number | null): boolean {
  return mode === 2 || mode === 5;
}

/** 是否为合法 UUID（mode=2 的 dept_id 形态；与后端校验口径一致） */
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

/** 是否为合法锚点层级参数（"1"~"4"） */
export function isAnchorLevelParam(param?: string | null): boolean {
  return typeof param === "string" && (ANCHOR_LEVEL_PARAMS as readonly string[]).includes(param);
}

/** 是否为合法 UUID 形态的锚点参数 */
export function isAnchorDeptParam(param?: string | null): boolean {
  return typeof param === "string" && UUID_RE.test(param);
}

/**
 * 校验 (mode, param) 组合是否合法（前端侧与后端 `ValidateAnchor` 对齐的预检）。
 * mode=0/3 不携带参数；mode=2 必须 UUID；mode=5 必须 "1"~"4"。
 */
export function isValidAnchorConfig(mode: number | undefined | null, param?: string | null): boolean {
  const m = normalizeAnchorMode(mode);
  if (m === 0 || m === 3) return true;
  if (m === 2) return isAnchorDeptParam(param);
  return isAnchorLevelParam(param);
}

/**
 * 切换模式时的参数归位：形状不匹配的旧参数一律清空，避免"选了层级却留着部门 UUID"
 * 之类的静默脏值（契约 §五 防静默）。
 */
export function retargetAnchorParam(nextMode: AnchorMode, prevParam?: string | null): string {
  if (nextMode === 2) return isAnchorDeptParam(prevParam) ? (prevParam as string) : "";
  if (nextMode === 5) return isAnchorLevelParam(prevParam) ? (prevParam as string) : "";
  return "";
}

/**
 * 归一化后端返回/表单遗留的锚点参数：
 * - 非字符串（undefined/null/数字）→ 空串
 * - mode=2 只接受 UUID 形态，mode=5 只接受 "1"~"4"，其余模式恒为空串
 */
export function normalizeAnchorParam(mode: number | undefined | null, param: unknown): string {
  const asString =
    typeof param === "string" ? param : typeof param === "number" ? String(param) : "";
  return retargetAnchorParam(normalizeAnchorMode(mode), asString);
}
