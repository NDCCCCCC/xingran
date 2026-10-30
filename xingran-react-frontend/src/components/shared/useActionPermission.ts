/**
 * 操作级权限门控 helper（v1.24 PERM-02）。
 *
 * 三态语义（不可简化）：
 *
 * | state    | 条件                       | 渲染行为                          |
 * |----------|----------------------------|-----------------------------------|
 * | `loading`| 权限集尚未加载完成          | 渲染但 **disabled** + 提示        |
 * | `error`  | 权限接口失败                | 渲染但 **disabled** + 提示        |
 * | `ready`  | 权限集已加载                | 含权限码 → 可用；不含 → **不渲染** |
 *
 * ⚠️ **禁止**用 `permissions.length === 0` 判「未就绪」：
 * 空权限集（用户确实没有权限）与「还没加载完」在数据上不可区分，
 * 那样判会让权限接口失败时管理员也看不到按钮，而且没有任何提示。
 * 「是否就绪」必须来自显式状态位（menuStore 的 `permissionsLoaded` / `permissionsError`）。
 *
 * 权限源：`useMenuStore((s) => s.permissions)` + 同 store 的显式就绪位。
 */

import { useCallback, useMemo } from "react";
import { useMenuStore } from "@/store/menuStore";

/** 门控三态 */
export type PermissionGateState = "ready" | "loading" | "error";

/** 单个操作入口的门控决策 */
export interface ActionPermissionDecision {
  /** 是否渲染该操作入口（ready 且无权限 → false） */
  visible: boolean;
  /** 是否禁用（未就绪 → true） */
  disabled: boolean;
  /** 未就绪时的提示文案（用于 title / aria-label） */
  hint?: string;
}

/** 门控结果 */
export interface ActionPermissionResult {
  /** 三态：权限集是否已就绪 */
  state: PermissionGateState;
  /** 就绪且含该权限码 → true；未就绪 → false；未传权限码 → true（不做判定） */
  allowed: (perm?: string) => boolean;
  /** 未就绪时的提示文案（用于 disabled + title/aria）；已就绪时为 undefined */
  notReadyHint?: string;
  /**
   * 逐权限码的门控决策（渲染 / 禁用 / 提示）。
   *
   * 为什么在 allowed 之外再给 decide：`allowed` 只能表达「能不能」，
   * 而三态门控还需要区分「不渲染」（ready 且无权限）与
   * 「渲染但禁用并提示」（loading / error）——后者必须保留按钮可见性，
   * 否则用户看不到任何操作入口也无从得知原因。
   */
  decide: (perm?: string) => ActionPermissionDecision;
}

/** 门控提示文案（可由调用方覆盖） */
export interface UseActionPermissionOptions {
  /** 未就绪（加载中）提示 */
  loadingHint?: string;
  /** 权限接口失败提示 */
  errorHint?: string;
}

const DEFAULT_LOADING_HINT = "权限信息加载中，操作暂不可用";
const DEFAULT_ERROR_HINT = "权限信息加载失败，操作暂不可用";

/**
 * useActionPermission —— 操作级权限门控（唯一判定入口，勿各页自造 hasPermission）。
 *
 * @example
 *   const gate = useActionPermission();
 *   const edit = gate.decide(PERMISSION_CODES.systemUser.edit);
 *   // edit.visible === false  → 不渲染
 *   // edit.disabled === true  → 渲染但禁用（配合 edit.hint 做 title/aria）
 */
export function useActionPermission(options: UseActionPermissionOptions = {}): ActionPermissionResult {
  const permissions = useMenuStore((s) => s.permissions);
  const permissionsLoaded = useMenuStore((s) => s.permissionsLoaded);
  const permissionsError = useMenuStore((s) => s.permissionsError);

  const { loadingHint = DEFAULT_LOADING_HINT, errorHint = DEFAULT_ERROR_HINT } = options;

  const state: PermissionGateState = permissionsError
    ? "error"
    : permissionsLoaded
      ? "ready"
      : "loading";

  const notReadyHint = state === "error" ? errorHint : state === "loading" ? loadingHint : undefined;

  const allowed = useCallback(
    (perm?: string) => {
      // 未传权限码 → 不做权限判定（保持既有调用方行为）
      if (!perm) return true;
      if (state !== "ready") return false;
      return permissions.includes(perm);
    },
    [permissions, state]
  );

  const decide = useCallback(
    (perm?: string): ActionPermissionDecision => {
      if (!perm) return { visible: true, disabled: false };
      // 未就绪 → 渲染但禁用 + 提示（不能隐藏：用户需要看到入口 + 原因）
      if (state !== "ready") return { visible: true, disabled: true, hint: notReadyHint };
      // 已就绪 → 有权限可用；无权限不渲染
      return permissions.includes(perm) ? { visible: true, disabled: false } : { visible: false, disabled: false };
    },
    [permissions, state, notReadyHint]
  );

  return useMemo(
    () => ({ state, allowed, notReadyHint, decide }),
    [state, allowed, notReadyHint, decide]
  );
}

export default useActionPermission;
