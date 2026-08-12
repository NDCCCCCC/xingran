---
phase: 14-frontend-ux
plan: fix-03
type: execute
wave: 2
depends_on:
  - 14-fix-01
gap_closure: true
gap_refs:
  - B3
  - W1
  - W3
  - W4
files_modified:
  - xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx
  - xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx
  - xingran-react-frontend/src/pages/network/mac/history.tsx
autonomous: true
requirements:
  - UI-01
  - UI-04

must_haves:
  truths:
    - "ErrorAlertWithRetry 在 1007 错误下,整个组件实例生命周期内最多触发一次 logout()(ref 守护),React StrictMode 双调用场景下也只跳一次登录页"
    - "EmptyStateWithAction 在 actionLabel 提供但 actionPath 为空字符串或非字符串时不渲染 Link 按钮;type 守卫在 render 前完成"
    - "history.tsx 复制 MAC 后给用户可见反馈:成功 message.success 已复制 X,失败 message.error 复制失败"
    - "history.tsx 的 URL 参数注入 useEffect 移除 eslint-disable,改为依赖 searchParams 数组(或 useMemo 一次性计算 initial values)"
  artifacts:
    - path: "xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx"
      provides: "1007 logout ref 守护(cancelled 标记 + 跳过重复触发)"
      contains: "useRef"
    - path: "xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx"
      provides: "actionPath 类型守卫 + typeof check"
      contains: "typeof actionPath"
    - path: "xingran-react-frontend/src/pages/network/mac/history.tsx"
      provides: "copyMAC 反馈 + URL 参数 effect 重构"
      contains: "copyMAC\|message.success\|searchParams"
  key_links:
    - from: "xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx"
      to: "authStore.logout"
      via: "useRef 守护下的单次调用"
      pattern: "logout"
    - from: "xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx"
      to: "react-router-dom Link"
      via: "typeof 守卫后渲染"
      pattern: "Link"
    - from: "xingran-react-frontend/src/pages/network/mac/history.tsx"
      to: "URL params (searchParams)"
      via: "去掉 eslint-disable,补全依赖"
      pattern: "searchParams"
---

<objective>
修复 4 个分散在前端 React 组件中的小缺陷:
- **B3 (CR-03)**: ErrorAlertWithRetry 的 1007 logout 在 useEffect 重新触发时多次调用,产生 StrictMode 双调用 + token 清理竞态。
- **W1 (CR-04)**: EmptyStateWithAction 的 `Boolean(actionLabel && actionPath)` 在 actionPath 为空字符串或数字 0 时仍渲染 Link,加 `typeof === 'string' && length > 0` 守卫。
- **W3 (WR-07)**: history.tsx copyMAC 静默吞掉 clipboard promise 错误,无用户反馈。
- **W4 (WR-01)**: history.tsx URL 参数 useEffect 用 `eslint-disable-next-line` 隐藏依赖,运行期 URL 变化不重读。

Purpose: 这些是 Phase 14 各 plan 留下的小型 React 反模式。修复它们使 UI-01/UI-04 错误态与跨页跳转链路行为可预测;同时去掉 eslint-disable 以便后续重构不会再引入同类问题。
Output: 3 个文件修改 + 类型守卫 + ref 守护 + 用户反馈 + 依赖数组补全。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/14-frontend-ux/14-CONTEXT.md (D-18/D-19/D-20 错误态 + 空态 + 加载态规范;D-17 URL 参数契约)
@.planning/phases/14-frontend-ux/14-VERIFICATION.md (B3 + W1 + W3 + W4 段)
@.planning/phases/14-frontend-ux/14-REVIEW.md (CR-03 / CR-04 / WR-01 / WR-07 段)
@xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx (B3 目标)
@xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx (W1 目标)
@xingran-react-frontend/src/pages/network/mac/history.tsx (W3 + W4 目标 — copyMAC 行 409-411;URL effect 行 134-154)
@xingran-react-frontend/src/store/authStore.ts (logout 函数签名)
@xingran-react-frontend/src/pages/network/mac/trajectory.tsx (URL 参数预填模式参考 — 行 96-127)
</context>

<tasks>

<task type="auto">
  <name>Task 1: 修复 B3 + W1 + W3 + W4(共享组件 ref 守卫 + 类型守卫 + 用户反馈 + URL effect)</name>
  <files>xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx, xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx, xingran-react-frontend/src/pages/network/mac/history.tsx</files>
  <read_first>
    - xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx (B3: 行 74-87 — useEffect [code, logout])
    - xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx (W1: 行 32 + 42)
    - xingran-react-frontend/src/pages/network/mac/history.tsx (W3: copyMAC 行 409-411;W4: URL effect 行 134-154)
    - xingran-react-frontend/src/store/authStore.ts (logout 返回类型:Promise<void>)
    - xingran-react-frontend/src/pages/network/mac/trajectory.tsx (URL 参数模式参考)
  </read_first>
  <action>
    1. **B3 修复** (`ErrorAlertWithRetry.tsx`):
       - 顶部 import 加入 `import { useEffect, useRef, type FC } from 'react';`
       - 在组件体内 `const ranRef = useRef<number | null>(null);` + `const cancelledRef = useRef(false);`
       - 替换 useEffect(行 80-87)为:
         ```ts
         useEffect(() => {
           if (code !== 1007) return;
           if (ranRef.current === code) return; // 同一个 code 已经处理过,跳过
           ranRef.current = code;
           cancelledRef.current = false;
           logout()
             .catch(() => undefined)
             .finally(() => {
               if (!cancelledRef.current) {
                 window.location.href = '/login';
               }
             });
           return () => { cancelledRef.current = true; };
         }, [code, logout]);
         ```
       - 注释说明:`ranRef` 跨 render 持久化,即使父组件 state 变化导致重渲染也不会再触发 logout;`cancelledRef` 处理组件卸载时的 cleanup。
    2. **W1 修复** (`EmptyStateWithAction.tsx`):
       - 行 32 `const showAction = Boolean(actionLabel && actionPath);` 替换为:
         ```ts
         const showAction =
           Boolean(actionLabel) &&
           typeof actionPath === 'string' &&
           actionPath.length > 0;
         ```
       - 行 42 `to={actionPath as string}` 改为 `to={actionPath}`(因为上面已 narrow 到 string)。
       - 顶部 import `import type { FC, ReactNode } from 'react';` 保持不变。
    3. **W3 修复** (`history.tsx`):
       - 行 409-411 替换 `copyMAC` 为:
         ```ts
         const copyMAC = async (mac: string) => {
           try {
             if (!navigator.clipboard) {
               throw new Error('Clipboard API unavailable');
             }
             await navigator.clipboard.writeText(mac);
             message.success(`已复制 ${mac}`);
           } catch (err) {
             message.error(err instanceof Error ? `复制失败:${err.message}` : '复制失败,请手动复制');
           }
         };
         ```
       - 调用点(`<Button onClick={() => copyMAC(record.macAddress)}>`)由于 copyMAC 现在是 async,改为 `onClick={() => { void copyMAC(record.macAddress); }}` 保留 fire-and-forget 语义,但内部已有完整反馈。
    4. **W4 修复** (`history.tsx` URL effect 行 134-154):
       - 替换原 useEffect 块为:
         ```ts
         useEffect(() => {
           const deviceId = searchParams.get('deviceId');
           const portName = searchParams.get('portName');
           const startTime = searchParams.get('startTime');
           const endTime = searchParams.get('endTime');
           const mac = searchParams.get('mac');
           const initial: Record<string, unknown> = {};
           if (deviceId) initial.deviceId = deviceId;
           if (portName) initial.interfaceName = portName;
           if (mac) initial.mac = normalizeMACAddress(mac) || mac;
           if (Object.keys(initial).length > 0) form.setFieldsValue(initial);
           if (startTime && endTime) {
             setActivePreset('custom');
             setCustomRange([dayjs(startTime), dayjs(endTime)]);
             setTimeRange({ startTime, endTime });
           }
         }, [searchParams, form]);
         ```
       - 去掉 `// eslint-disable-next-line react-hooks/exhaustive-deps` 注释(已用 searchParams 作为依赖)。
       - searchParams 来自 react-router-dom 的 `useSearchParams`,返回的 `searchParams` 对象本身在 URL 变化时会换新引用(react-router-dom v7 行为),所以该依赖可正确触发。
       - 用 `Object.keys(initial).length > 0` 守卫避免无 URL 参数时调用 form.setFieldsValue({}) 触发 AntD warning。
    5. 运行 `npx tsc --noEmit -p .` 退出码 0。
  </action>
  <verify>
    <automated>cd xingran-react-frontend && npx tsc --noEmit -p .  # 退出码 0</automated>
    <automated>grep -n "useRef" xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx  # 至少 2 行(useRef + ranRef/cancelledRef)</automated>
    <automated>grep -n "typeof actionPath" xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx  # 至少 1 行</automated>
    <automated>grep -n "message.success.*已复制\|message.error.*复制失败" xingran-react-frontend/src/pages/network/mac/history.tsx  # 至少 2 行(W3 反馈)</automated>
    <automated>grep -c "eslint-disable-next-line react-hooks/exhaustive-deps" xingran-react-frontend/src/pages/network/mac/history.tsx  # 必须为 0(W4 移除)</automated>
    <automated>grep -n "searchParams, form\|searchParams,form" xingran-react-frontend/src/pages/network/mac/history.tsx  # 至少 1 行(W4 依赖数组)</automated>
  </verify>
  <acceptance_criteria>
    - `npx tsc --noEmit -p .` 退出码 0
    - `grep -c "useRef" xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx` >= 2
    - `grep -c "ranRef" xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx` >= 1
    - `grep -c "typeof actionPath" xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx` >= 1
    - `grep -c "message.success.*已复制" xingran-react-frontend/src/pages/network/mac/history.tsx` >= 1
    - `grep -c "message.error.*复制失败" xingran-react-frontend/src/pages/network/mac/history.tsx` >= 1
    - `grep -c "eslint-disable.*react-hooks/exhaustive-deps" xingran-react-frontend/src/pages/network/mac/history.tsx` == 0
    - `grep -c "searchParams, form\|searchParams,form" xingran-react-frontend/src/pages/network/mac/history.tsx` >= 1
    - `grep -c "as string" xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx` == 0(类型守卫后消除 cast)
    - copyMAC 内部 try/catch 包裹(grep "navigator.clipboard.writeText" 必须在 try 块内)
    - ErrorAlertWithRetry 的 cancelled 标记在 useEffect cleanup 中设置 true(grep "cancelledRef.current = true")
  </acceptance_criteria>
  <done>4 个 React 反模式全部修复:1007 logout 单次守护、EmptyState 类型守卫、copyMAC 用户反馈、URL effect 依赖数组补全(去除 eslint-disable);TypeScript 编译通过。</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| component → auth store | ErrorAlertWithRetry calls authStore.logout() — must be idempotent within instance |
| component → router Link | EmptyStateWithAction guards actionPath before render — prevents routing to invalid `to` |
| component → user clipboard | copyMAC wraps navigator.clipboard in try/catch — fail gracefully on insecure context |
| page mount → URL | history.tsx URL effect re-reads on searchParams change — drives re-navigation support |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-14B3-01 | Elevation of Privilege | ErrorAlertWithRetry 1007 race | mitigate | ranRef guard ensures logout() fires at most once per code value; cancelledRef guards unmount |
| T-14W1-01 | Tampering | EmptyStateWithAction invalid `to` | mitigate | typeof === 'string' && length > 0 narrows before Link render; removes `as string` cast that hid type errors |
| T-14W3-01 | Information Disclosure | copyMAC in insecure context | accept | navigator.clipboard throws; message.error surfaces it; user manually copies |
| T-14W4-01 | Repudiation | URL effect with eslint-disable | mitigate | Removed eslint-disable; deps array now includes searchParams so re-navigation triggers re-read |
| T-14B3-SC | Tampering | npm/pip/cargo installs | mitigate | No new dependencies — all fixes are React stdlib (useRef, useEffect) + AntD message utility |
</threat_model>

<verification>
- `cd D:/code/ClaudeCode/xingran-go-backend/xingran-react-frontend && npx tsc --noEmit -p .` exits 0
- `grep -c "useRef" xingran-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx` >= 2
- `grep -c "typeof actionPath" xingran-react-frontend/src/components/shared/EmptyStateWithAction.tsx` >= 1
- `grep -c "message.success.*已复制\|message.error.*复制失败" xingran-react-frontend/src/pages/network/mac/history.tsx` >= 2
- `grep -c "eslint-disable.*react-hooks/exhaustive-deps" xingran-react-frontend/src/pages/network/mac/history.tsx` == 0
- Manual smoke: load /network/mac/history?deviceId=<uuid>, page should populate device filter from URL
- Manual smoke: click "复制" icon on a MAC row → toast `已复制 XX:XX:XX:XX:XX:XX` appears
</verification>

<success_criteria>
- 1007 token-expired 错误场景下,logout() 只触发一次(StrictMode 下也只跳一次登录)
- EmptyStateWithAction 在 actionPath 为空字符串时不渲染 Link
- 复制 MAC 成功后 message.success toast 出现
- URL 参数变化时 useEffect 自动重读并填充表单
- TypeScript 0 退出码
- 不再有任何 eslint-disable-next-line react-hooks/exhaustive-deps 在 history.tsx
</success_criteria>

<output>
Create `.planning/phases/14-frontend-ux/14-fix-03-SUMMARY.md` when done
</output>