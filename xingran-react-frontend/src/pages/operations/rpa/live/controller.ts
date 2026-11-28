/**
 * W7 / UI-FIX-02：接管者判定（纯函数，唯一口径）
 *
 * 为什么单独抽出来（缺陷实证）：
 *   生产 10.62.10.39 真实浏览器实测 —— 点「接管」成功后，页面角标显示
 *   「已接管 · ninedrunk」，但遮罩（`live-view-mask`）仍在，鼠标注入发不出去。
 *
 *   根因是**两边取值口径不一致**：
 *     · 后端 `operatorName(c)` = `claims.nickname`（SM2 token 里压根没有
 *       nickname 字段 → 恒回落 username）→ takeoverBy 永远是 username；
 *     · 前端 `myName` = 登录响应 `user.nickname`（真实中文昵称）。
 *   ninedrunk 的 nickname 恰好等于 username，所以这个账号「碰巧能用」；
 *   把 store 里的 nickname 改成「超级管理员」再刷新页面，遮罩立刻不消失 ——
 *   我们就是这么复现的。
 *
 *   所以判据不能是展示名。后端现在同时下发 `takeoverById`（用户 ID），
 *   前端只认 userID；展示名比对仅作为「历史会话没有 userID」时的兜底。
 *
 * 返回 true 的语义 = 「我是当前接管者」= 遮罩隐藏 + 允许注入输入。
 */
export interface ControllerInput {
  /** 当前登录用户的 ID（authStore.user.id） */
  myUserId: string;
  /** 当前登录用户的展示名（nickname || username），仅兜底用 */
  myName: string;
  /** 后端下发的接管者用户 ID（LiveContext.takeoverById） */
  takeoverById?: string;
  /** 后端下发的接管者展示名（LiveContext.takeoverBy），仅兜底用 */
  takeoverBy?: string;
}

/**
 * 判定当前用户是否为接管者。
 *
 * 优先级：
 *   1. 有 takeoverById → 只认 userID（两边都不为空且相等）。
 *   2. 没有 takeoverById（历史会话）→ 回落展示名比对，且必须非空。
 *   3. 都没有 → false（未接管）。
 */
export function resolveIsController(input: ControllerInput): boolean {
  const takeoverById = (input.takeoverById ?? "").trim();
  const takeoverBy = (input.takeoverBy ?? "").trim();
  const myUserId = (input.myUserId ?? "").trim();
  const myName = (input.myName ?? "").trim();

  if (takeoverById !== "") {
    return myUserId !== "" && takeoverById === myUserId;
  }
  if (takeoverBy !== "") {
    return myName !== "" && takeoverBy === myName;
  }
  return false;
}

/**
 * HUD 文案用的接管者展示名。
 *
 * 后端 takeoverBy 为空但 takeoverById 非空时（理论上不会，但防御），
 * 显示为「他人」而不是空字符串 —— 避免角标渲染成「已接管 ·」这种半截文本。
 */
export function controllerDisplayName(takeoverBy?: string, takeoverById?: string): string {
  const name = (takeoverBy ?? "").trim();
  if (name !== "") return name;
  return (takeoverById ?? "").trim() !== "" ? "其他用户" : "";
}

/**
 * 是否处于「被接管」状态（HUD 角标用）：任意一份接管者信息非空即为真。
 * 与 resolveIsController 严格区分 —— 一个是「有人在接管」，一个是「是我」。
 */
export function isTakenOver(takeoverBy?: string, takeoverById?: string): boolean {
  return (takeoverBy ?? "").trim() !== "" || (takeoverById ?? "").trim() !== "";
}
