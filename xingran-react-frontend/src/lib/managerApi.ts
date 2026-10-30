/**
 * 经理同步 —— 机构（sys_manager_org）与规则（sys_manager_rule）API 封装
 *
 * 后端契约：
 * - POST /system/manager-orgs/list         分页列表 入参 {current,pageSize,orgName?}
 * - POST /system/manager-orgs               新增   入参 {orgName,deptId,parentOrgId?,orgType,enabled}
 * - POST /system/manager-orgs/:id/update    修改
 * - POST /system/manager-orgs/:id/delete    删除
 * - POST /system/manager-rules/list         分页列表 入参 {current,pageSize,ruleName?,subjectType?,enabled?}
 * - POST /system/manager-rules              新增
 * - POST /system/manager-rules/:id/update   修改
 * - POST /system/manager-rules/:id/delete   删除
 *
 * 统一走封装好的 post()（@/lib/api），不在页面里裸调 axios。
 */

import { post } from "@/lib/api";
import type { PageResponse } from "@/types/base";

/**
 * 机构类型（对应系统三层架构：分公司 → 中支（含分公司本部）→ 支公司）
 */
export const MANAGER_ORG_TYPE = {
  /** 分公司 */
  BRANCH_COMPANY: 0,
  /** 中支（含分公司本部） */
  MIDDLE_BRANCH: 1,
  /** 支公司 */
  SUB_BRANCH: 2,
} as const;

/** 机构类型中文名 */
export const MANAGER_ORG_TYPE_LABELS: Record<number, string> = {
  [MANAGER_ORG_TYPE.BRANCH_COMPANY]: "分公司",
  [MANAGER_ORG_TYPE.MIDDLE_BRANCH]: "中支",
  [MANAGER_ORG_TYPE.SUB_BRANCH]: "支公司",
};

/**
 * 机构
 */
export interface ManagerOrg {
  id: string;
  /** 机构名称 */
  orgName: string;
  /** 对应的部门节点（UUID，后端唯一约束） */
  deptId: string;
  /** 后端返回的可读部门名 */
  deptName?: string;
  /** 上级机构（UUID，可空） */
  parentOrgId?: string;
  /** 后端返回的可读上级机构名 */
  parentOrgName?: string;
  /** 机构层级：0=分公司 1=中支 2=支公司 */
  orgType: number;
  /** 是否启用 */
  enabled: boolean;
  createdAt?: string;
  updatedAt?: string;
}

/** 机构列表查询参数 */
export interface ManagerOrgQuery {
  current?: number;
  pageSize?: number;
  orgName?: string;
}

/** 机构新增/修改请求体 */
export interface ManagerOrgPayload {
  orgName: string;
  deptId: string;
  parentOrgId?: string;
  orgType: number;
  enabled: boolean;
}

/** 适用主体 */
export type ManagerSubjectType = "employee" | "dept_leader" | "org_head" | "post_holder";

/** 经理来源 */
export type ManagerType = "dept_leader" | "post_holder" | "org_leader" | "org_head";

/** 作用范围：0=全部 1=部门子树 2=机构子树 */
export const MANAGER_SCOPE_TYPE = {
  ALL: 0,
  DEPT_SUBTREE: 1,
  ORG_SUBTREE: 2,
} as const;

/** 作用范围中文名 */
export const MANAGER_SCOPE_LABELS: Record<number, string> = {
  [MANAGER_SCOPE_TYPE.ALL]: "全部",
  [MANAGER_SCOPE_TYPE.DEPT_SUBTREE]: "部门子树",
  [MANAGER_SCOPE_TYPE.ORG_SUBTREE]: "机构子树",
};

/** 适用主体中文名 */
export const MANAGER_SUBJECT_LABELS: Record<ManagerSubjectType, string> = {
  employee: "普通员工",
  dept_leader: "部门负责人",
  org_head: "机构一把手",
  post_holder: "指定岗位任职人",
};

/** 经理来源中文名 */
export const MANAGER_TYPE_LABELS: Record<ManagerType, string> = {
  dept_leader: "部门负责人",
  post_holder: "岗位任职人",
  org_leader: "机构负责人",
  org_head: "机构一把手",
};

/**
 * 经理同步规则
 */
export interface ManagerRule {
  id: string;
  /** 规则名（必填） */
  ruleName: string;
  /** 优先级，小者优先 */
  priority: number;
  /** 是否启用 */
  enabled: boolean;
  /** 适用主体 */
  subjectType: ManagerSubjectType;
  /** 主体岗位（UUID），subjectType=post_holder 时必填 */
  subjectPostId?: string;
  /** 主体所属机构（UUID），可空 */
  subjectOrgId?: string;
  /** 经理来源 */
  managerType: ManagerType;
  /** 经理目标岗位（UUID），managerType=post_holder|org_head 时必填 */
  managerPostId?: string;
  /** 目标机构：空=用户自身机构，"parent"=上级机构，UUID=指定机构 */
  managerOrgId?: string;
  /** 作用范围：0=全部 1=部门子树 2=机构子树 */
  scopeType: number;
  /** 范围根 ID（UUID），scopeType!=0 时必填 */
  scopeId?: string;
  // ---- 后端列表附带的可读名称（可能为空）----
  subjectPostName?: string;
  subjectOrgName?: string;
  managerPostName?: string;
  managerOrgName?: string;
  scopeDeptName?: string;
  scopeOrgName?: string;
  createdAt?: string;
  updatedAt?: string;
}

/** 规则列表查询参数 */
export interface ManagerRuleQuery {
  current?: number;
  pageSize?: number;
  ruleName?: string;
  subjectType?: string;
  enabled?: boolean;
}

/** 规则新增/修改请求体 */
export interface ManagerRulePayload {
  ruleName: string;
  priority: number;
  enabled: boolean;
  subjectType: ManagerSubjectType;
  subjectPostId?: string;
  subjectOrgId?: string;
  managerType: ManagerType;
  managerPostId?: string;
  managerOrgId?: string;
  scopeType: number;
  scopeId?: string;
}

/** 目标机构的特殊值：上级机构 */
export const MANAGER_ORG_TARGET_PARENT = "parent";

/**
 * 判断 enabled 字段是否处于「启用」状态。
 *
 * 后端若以 0/1 表达（项目通用约定 0=启用、1=停用），此处按该约定归一化；
 * 若以 boolean 表达则直接取值。
 */
export function isManagerEnabled(value: boolean | number | undefined | null): boolean {
  if (value === undefined || value === null) return false;
  if (typeof value === "number") return value === 0;
  return value === true;
}

// ==================== 机构 API ====================

/** 机构分页列表 */
export async function getManagerOrgList(
  params: ManagerOrgQuery
): Promise<PageResponse<ManagerOrg>> {
  const result = await post<PageResponse<ManagerOrg>>("/system/manager-orgs/list", params);
  return (
    result.data ?? {
      list: [],
      total: 0,
      current: params.current ?? 1,
      pageSize: params.pageSize ?? 10,
    }
  );
}

/** 新增机构 */
export async function createManagerOrg(payload: ManagerOrgPayload): Promise<void> {
  await post("/system/manager-orgs", payload);
}

/** 修改机构 */
export async function updateManagerOrg(id: string, payload: ManagerOrgPayload): Promise<void> {
  await post(`/system/manager-orgs/${id}/update`, payload);
}

/** 删除机构 */
export async function deleteManagerOrg(id: string): Promise<void> {
  await post(`/system/manager-orgs/${id}/delete`, {});
}

/** 从部门架构自动生成机构映射的结果 */
export interface GenerateOrgsResult {
  totalCandidates: number;
  created: number;
  skippedExisting: number;
  orphanCount: number;
  warnings?: string[];
}

/** 从部门架构自动生成机构映射（幂等；已存在的不覆盖手工修改） */
export async function generateManagerOrgsFromDept(): Promise<GenerateOrgsResult> {
  const result = await post<GenerateOrgsResult>("/system/manager-orgs/generate-from-dept", {});
  return result.data ?? { totalCandidates: 0, created: 0, skippedExisting: 0, orphanCount: 0 };
}

// ==================== 规则 API ====================

/** 规则分页列表 */
export async function getManagerRuleList(
  params: ManagerRuleQuery
): Promise<PageResponse<ManagerRule>> {
  const result = await post<PageResponse<ManagerRule>>("/system/manager-rules/list", params);
  return (
    result.data ?? {
      list: [],
      total: 0,
      current: params.current ?? 1,
      pageSize: params.pageSize ?? 10,
    }
  );
}

/** 新增规则 */
export async function createManagerRule(payload: ManagerRulePayload): Promise<void> {
  await post("/system/manager-rules", payload);
}

/** 修改规则 */
export async function updateManagerRule(id: string, payload: ManagerRulePayload): Promise<void> {
  await post(`/system/manager-rules/${id}/update`, payload);
}

/** 删除规则 */
export async function deleteManagerRule(id: string): Promise<void> {
  await post(`/system/manager-rules/${id}/delete`, {});
}

// ==================== 经理预览 API ====================

/** 经理预览项（用户列表「经理」列展示用，纯计算不落库） */
export interface ManagerPreviewItem {
  userId: string;
  managerId?: string;
  managerName?: string;
  managerUsername?: string;
  ruleName?: string;
  reason?: string;
  warnings?: string[];
}

/** 批量预览用户的经理（按当前页 userIds 查询；无启用规则返回空数组） */
export async function previewManagers(userIds: string[]): Promise<ManagerPreviewItem[]> {
  if (userIds.length === 0) return [];
  try {
    const result = await post<ManagerPreviewItem[]>("/system/manager-rules/manager-preview", {
      userIds,
    });
    return result.data ?? [];
  } catch {
    // 预览是增强展示，失败静默降级为不显示经理列内容
    return [];
  }
}
