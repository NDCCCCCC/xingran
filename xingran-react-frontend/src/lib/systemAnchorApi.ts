/**
 * 机构层级数据权限 —— 角色/岗位「机构锚点」推导预览 API
 *
 * 契约来源：`.planning/notes/261118-anchor-scope-contract.md` §六
 *
 * - `POST /system/roles/:id/anchor-preview`（需 system:role:view）
 * - `POST /system/posts/:id/anchor-preview`（需 system:post:view）
 *
 * 统一走封装好的 `post()`（@/lib/api），不在页面里裸调 axios。
 * 后端预览接口可能尚未上线：本模块**只做请求与解包**，不吞异常 ——
 * 由调用方 catch 后降级提示（页面不得因预览失败而崩溃）。
 */

import { post } from "@/lib/api";
import type { AnchorPreviewResult, UserPermissionPreview } from "@/types/system";

/**
 * 归一化预览响应，兼容后端两种未命中项形态：
 * 1. 完整 `items`（含 resolved 标记）→ 按 `!resolved` 过滤出未命中项；
 * 2. 只有 `unresolved` 数组（仅未命中项）→ 直接采用。
 *
 * 计数一律以 items 实际过滤结果为准做交叉校验：后端若整体只给
 * `unresolved` 而无 `items`，则 resolvedCount 用 totalCount - 未命中数兜底。
 */
export function normalizeAnchorPreview(raw: unknown): AnchorPreviewResult {
  const data = (raw ?? {}) as Partial<AnchorPreviewResult> & {
    items?: AnchorPreviewResult["items"];
    unresolved?: AnchorPreviewResult["unresolved"];
  };

  const items = Array.isArray(data.items) ? data.items : undefined;
  const unresolvedFromItems = items ? items.filter((it) => !it.resolved) : undefined;
  const unresolvedFallback = Array.isArray(data.unresolved) ? data.unresolved : undefined;
  const unresolved = unresolvedFromItems ?? unresolvedFallback ?? [];

  const totalCount =
    typeof data.totalCount === "number" ? data.totalCount : (items?.length ?? unresolved.length);
  const unresolvedCount =
    typeof data.unresolvedCount === "number" ? data.unresolvedCount : unresolved.length;
  const resolvedCount =
    typeof data.resolvedCount === "number"
      ? data.resolvedCount
      : Math.max(0, totalCount - unresolvedCount);

  return {
    mode: typeof data.mode === "number" ? data.mode : 0,
    param: data.param ?? "",
    resolvedCount,
    unresolvedCount,
    totalCount,
    items,
    unresolved,
    warnings: Array.isArray(data.warnings) ? data.warnings : [],
  };
}

/**
 * 预览角色的机构锚点生效范围。
 *
 * @param roleId 角色 ID
 * @returns 归一化后的预览结果；接口不存在/失败时**抛错**，由调用方降级
 */
export async function previewRoleAnchor(roleId: string): Promise<AnchorPreviewResult> {
  if (!roleId || roleId === "undefined" || roleId === "null") {
    throw new Error("无效的角色 ID，无法预览机构授权范围");
  }
  const result = await post<AnchorPreviewResult>(`/system/roles/${roleId}/anchor-preview`, {});
  return normalizeAnchorPreview(result.data);
}

/**
 * 预览用户表单当前值对应的**完整权限画像**（菜单权限 + 数据范围 + 机构锚点明细）。
 *
 * 与角色/岗位 anchor-preview 的关键区别：本接口接受**尚未保存**的表单值，
 * 因此传的是 deptId/roleIds/postIds 而非用户 ID。
 *
 * 失败时抛错，由调用方 catch 降级（页面不得崩）。
 */
export async function previewUserPermission(params: {
  deptId?: string;
  roleIds?: string[];
  postIds?: string[];
}): Promise<UserPermissionPreview> {
  const result = await post<UserPermissionPreview>("/system/users/permission-preview", {
    deptId: params.deptId ?? "",
    roleIds: params.roleIds ?? [],
    postIds: params.postIds ?? [],
  });
  if (!result.data) {
    throw new Error("权限预览响应为空");
  }
  // 后端在无角色/无岗位时，menuIds/roleNames/postNames 等字段为 JSON null
  // （user_permission_preview.go 仅初始化部分字段），面板直接 .length 会
  // TypeError 白屏。这里统一归一化为数组，与 normalizeAnchorPreview 同法。
  const d = result.data;
  const arr = <T>(v: T[] | null | undefined): T[] => (Array.isArray(v) ? v : []);
  return {
    ...d,
    permissions: arr(d.permissions),
    menuIds: arr(d.menuIds),
    deptIds: arr(d.deptIds),
    visibleDeptIds: arr(d.visibleDeptIds),
    visibleDeptNames: arr(d.visibleDeptNames),
    anchorSources: arr(d.anchorSources),
    notes: arr(d.notes),
    roleNames: arr(d.roleNames),
    postNames: arr(d.postNames),
  };
}

/**
 * 预览岗位的机构锚点生效范围。
 *
 * @param postId 岗位 ID
 * @returns 归一化后的预览结果；接口不存在/失败时**抛错**，由调用方降级
 */
export async function previewPostAnchor(postId: string): Promise<AnchorPreviewResult> {
  if (!postId || postId === "undefined" || postId === "null") {
    throw new Error("无效的岗位 ID，无法预览机构授权范围");
  }
  const result = await post<AnchorPreviewResult>(`/system/posts/${postId}/anchor-preview`, {});
  return normalizeAnchorPreview(result.data);
}
