---
phase: 14-frontend-ux
plan: fix-02
type: execute
wave: 2
depends_on:
  - 14-fix-01
gap_closure: true
gap_refs:
  - B2
files_modified:
  - xingran-react-frontend/src/lib/api/networkApi.ts
  - xingran-react-frontend/src/pages/network/mac/history.tsx
autonomous: true
requirements:
  - UI-02

must_haves:
  truths:
    - "点击 '导出当前查询' / '导出全量' 按钮后,浏览器下载一个真实的 .xlsx 文件(可在 Excel/WPS 打开看到 MAC 历史数据表),而非含 JSON envelope 的损坏文件"
    - "导出失败时(后端 500 / 网络断开 / 后端返回 JSON 错误体),前端抛出可读错误信息,UI 用 message.error 提示"
  artifacts:
    - path: "xingran-react-frontend/src/lib/api/networkApi.ts"
      provides: "exportMACHistory 重写为 fetch + Blob 直传"
      contains: "exportMACHistory"
    - path: "xingran-react-frontend/src/pages/network/mac/history.tsx"
      provides: "handleExport 调用 exportMACHistory 拿到 Blob 后用 URL.createObjectURL + a.download(D-15 锁定)"
      contains: "handleExport"
  key_links:
    - from: "xingran-react-frontend/src/pages/network/mac/history.tsx"
      to: "xingran-react-frontend/src/lib/api/networkApi.ts"
      via: "exportMACHistory 拉取 xlsx Blob"
      pattern: "exportMACHistory"
    - from: "xingran-react-frontend/src/lib/api/networkApi.ts"
      to: "/network/history/list"
      via: "fetch 直传,绕过 src/lib/api.ts 的 response interceptor"
      pattern: "fetch.*history/list"
    - from: "xingran-react-frontend/src/pages/network/mac/history.tsx"
      to: "URL.createObjectURL + a.download"
      via: "D-15 锁定下载模式"
      pattern: "URL.createObjectURL"
---

<objective>
修复 `exportMACHistory` 返回 BaseResponse envelope 而非 Blob 的 CR-01 缺陷,让 UI-02 的 Excel 导出功能从"按钮存在但下载损坏文件"变为"真实 .xlsx 下载"。

Purpose: Phase 33 M2 已确立正确模式 — Excel 下载走 `fetch()` + `getAccessToken()` + `response.blob()`,绕过 `src/lib/api.ts` 的响应拦截器(后者会解包 `{code, message, data}` envelope)。当前 `networkApi.ts` 错误地用 `api.default.get + responseType: 'blob'`,但响应拦截器仍会把后端的 BaseResponse 当 JSON 解包,导致 Blob 实际是 JSON 对象。该 plan 把 `exportMACHistory` 改为 Phase 33 M2 模式,与 `lib/opsApi.ts:265-290 excelApi.export` 对齐。
Output: 1 个重写的 `exportMACHistory` 函数 + 1 处 history.tsx 调用微调(对齐新返回值)。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/14-frontend-ux/14-CONTEXT.md (D-15 锁定下载模式 = URL.createObjectURL + a.download)
@.planning/phases/14-frontend-ux/14-VERIFICATION.md (CR-01 段 + B2 段)
@.planning/phases/14-frontend-ux/14-REVIEW.md (CR-01 + CR-02 段)
@.planning/phases/33-vercel-react-best-practices-20260613-26/CONTEXT.md (M2 ExcelImport axios blob 修正经验)
@xingran-react-frontend/src/lib/opsApi.ts (excelApi.export 行 265-290 — Phase 33 M2 正确模板)
@xingran-react-frontend/src/utils/authHelpers.ts (getAccessToken — 用于 fetch Authorization header)
@xingran-react-frontend/src/pages/network/mac/history.tsx (handleExport 行 380-405 — 当前调用方)
@xingran-react-frontend/src/lib/api/networkApi.ts (当前 exportMACHistory 行 130-141 — 错误实现)
@.planning/phases/14-frontend-ux/14-fix-01-PLAN.md (上游:后端 GET /history/list 已就位)
</context>

<tasks>

<task type="auto">
  <name>Task 1: 重写 exportMACHistory 走 fetch + Blob 流程</name>
  <files>xingran-react-frontend/src/lib/api/networkApi.ts</files>
  <read_first>
    - xingran-react-frontend/src/lib/api/networkApi.ts (当前 exportMACHistory 行 130-141;MACHistoryQueryParams 类型行 45-56)
    - xingran-react-frontend/src/lib/opsApi.ts (excelApi.export 行 265-290 — Phase 33 M2 模板;extractFilenameFromResponse 行 240-252)
    - xingran-react-frontend/src/utils/authHelpers.ts (getAccessToken 函数签名)
    - xingran-react-frontend/src/pages/network/mac/history.tsx (handleExport 行 380-405 — 确认新返回值类型)
  </read_first>
  <action>
    1. 删除 `exportMACHistory` 当前实现(行 130-141)。
    2. 在 `networkApi.ts` 顶部 import 区追加:`import { getAccessToken } from '@/utils/authHelpers';`(已存在 `@/utils/authHelpers` 路径别名,参考 `lib/opsApi.ts:217`)。
    3. 重写为以下签名(完整复刻 opsApi.ts 的 excelApi.export 模式):
       ```ts
       export const exportMACHistory = async (
         params: MACHistoryQueryParams,
         exportScope: 'current' | 'all' = 'current',
       ): Promise<{ blob: Blob; filename: string }> => {
         const token = await getAccessToken();
         const search = new URLSearchParams();
         Object.entries({ ...params, format: 'xlsx', exportScope }).forEach(([k, v]) => {
           if (v !== undefined && v !== null && v !== '') search.set(k, String(v));
         });
         const response = await fetch(`/api/v1/network/history/list?${search.toString()}`, {
           method: 'GET',
           headers: { Authorization: `Bearer ${token}` },
         });
         if (!response.ok) {
           throw new Error(`导出失败:HTTP ${response.status}`);
         }
         const blob = await response.blob();
         // CR-01 错误反序列化:若 blob 实际是 JSON 错误体,后端在异常时仍可能返回 application/json
         if (blob.size < 1024 && blob.type.includes('json')) {
           const text = await blob.text();
           try {
             const errBody = JSON.parse(text) as { message?: string; msg?: string };
             throw new Error(errBody.message || errBody.msg || '导出失败');
           } catch (e) {
             if (e instanceof Error && e.message !== '导出失败') throw e;
             // parse 失败说明不是 JSON,降级为通用错误
             throw new Error(`导出失败:${response.status}`);
           }
         }
         const contentDisposition = response.headers.get('content-disposition');
         let filename = `mac_history_${exportScope}_${Date.now()}.xlsx`;
         if (contentDisposition) {
           const match = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
           if (match && match[1]) filename = decodeURIComponent(match[1].replace(/['"]/g, ''));
         }
         return { blob, filename };
       };
       ```
    4. 因为返回值类型从 `Promise<Blob>` 变为 `Promise<{ blob: Blob; filename: string }>`,**调用方 `history.tsx:377-393` 需要相应调整**(在此 plan 内一并修复,因为同 plan 修复可避免类型不匹配):
       - 替换:`const blob = await exportMACHistory(...)` → `const { blob, filename } = await exportMACHistory(...)`
       - 替换:`const filename = \`mac_history_${exportScope}_${ts}.xlsx\`;` 行删除(改用后端 Content-Disposition 提供的真实 filename)。
       - `URL.createObjectURL(blob)` + `a.download = filename` + `a.click()` + `URL.revokeObjectURL(url)` 保持不变(D-15)。
       - 保留现有 try/catch + message.success/error + finally setExporting(false) 逻辑。
    5. 修改完毕后确认 `npx tsc --noEmit -p .` 退出码 0(history.tsx 的 destructuring 调整必须同步)。
  </action>
  <verify>
    <automated>cd xingran-react-frontend && npx tsc --noEmit -p .  # 退出码 0</automated>
    <automated>grep -n "fetch.*history/list" xingran-react-frontend/src/lib/api/networkApi.ts  # 至少 1 行</automated>
    <automated>grep -n "URLSearchParams" xingran-react-frontend/src/lib/api/networkApi.ts  # 至少 1 行</automated>
    <automated>grep -n "content-disposition\|Content-Disposition" xingran-react-frontend/src/lib/api/networkApi.ts  # 至少 1 行</automated>
    <automated>grep -n "api.default.get" xingran-react-frontend/src/lib/api/networkApi.ts  # 必须为 0 行(完全删除旧实现)</automated>
    <automated>grep -n "exportMACHistory\|URL.createObjectURL" xingran-react-frontend/src/pages/network/mac/history.tsx  # 至少 2 行</automated>
  </verify>
  <acceptance_criteria>
    - `npx tsc --noEmit -p .` 退出码 0
    - `grep -c "api.default.get" xingran-react-frontend/src/lib/api/networkApi.ts` == 0(旧实现完全删除)
    - `grep -c "fetch.*history/list" xingran-react-frontend/src/lib/api/networkApi.ts` >= 1
    - `grep -c "getAccessToken" xingran-react-frontend/src/lib/api/networkApi.ts` >= 1
    - `grep -c "blob.size < 1024" xingran-react-frontend/src/lib/api/networkApi.ts` >= 1(错误反序列化分支)
    - `grep -c "URL.createObjectURL" xingran-react-frontend/src/pages/network/mac/history.tsx` >= 1(D-15 模式保留)
    - `exportMACHistory` 返回类型变为 `Promise<{ blob: Blob; filename: string }>`
    - history.tsx 调用方解构更新为 `{ blob, filename }`
    - filename 优先取 Content-Disposition header,缺失时回退 `mac_history_<scope>_<ts>.xlsx`
    - Authorization 头通过 `getAccessToken()` 获取(走 TokenManager,与 opsApi 一致)
    - 不再 import `api`(原 `import { post } from '../api'` 保留;只删掉 dynamic import '../api')
  </acceptance_criteria>
  <done>exportMACHistory 重写为 fetch + Blob 模式,与 opsApi.excelApi.export 对齐;history.tsx 调用方解构更新;错误反序列化分支就位;TypeScript 编译通过 0 退出码。</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| browser → backend | GET /network/history/list?format=xlsx crosses here; Authorization header required |
| browser → filesystem | Blob URL revoked after download (no leak) |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-14B2-01 | Information Disclosure | token in fetch | accept | TokenManager-managed access token; same risk as all other authenticated requests |
| T-14B2-02 | Tampering | Content-Disposition filename | mitigate | Filename parsed with regex match + decodeURIComponent; bounded by backend response header (not user-controlled path) |
| T-14B2-03 | Repudiation | export failures | mitigate | response.ok check + message.error via caller; blob.size < 1024 JSON deserialize fallback |
| T-14B2-04 | DoS | large xlsx in memory | accept | 30-day cap enforced backend-side (14-fix-01); browser blob storage bounded |
| T-14B2-05 | Elevation of Privilege | missing auth header | mitigate | getAccessToken() throws if unauthenticated; fetch requires manual header (unlike api wrapper that injects automatically); explicit Bearer header |
| T-14B2-SC | Tampering | npm/pip/cargo installs | mitigate | No new dependencies — uses native fetch + getAccessToken (already in project) |
</threat_model>

<verification>
- `cd D:/code/ClaudeCode/xingran-go-backend/xingran-react-frontend && npx tsc --noEmit -p .` exits 0
- `grep -c "api.default.get" xingran-react-frontend/src/lib/api/networkApi.ts` == 0
- `grep -c "fetch.*history/list" xingran-react-frontend/src/lib/api/networkApi.ts` >= 1
- `grep -c "getAccessToken" xingran-react-frontend/src/lib/api/networkApi.ts` >= 1
- `grep -c "URL.createObjectURL" xingran-react-frontend/src/pages/network/mac/history.tsx` >= 1
- End-to-end smoke (out-of-band): start backend, login, navigate to /network/mac/history, click "导出当前查询" → browser downloads `mac_history_current_<ts>.xlsx` that opens in Excel
</verification>

<success_criteria>
- exportMACHistory 返回真实 Blob(可被 URL.createObjectURL 转 blob: URL)
- 后端 Content-Disposition header 决定 filename,缺失时回退 ts 后缀
- 错误反序列化分支(size < 1024 + JSON)就位
- TypeScript 编译 0 退出码
- 浏览器下载的文件在 Excel/WPS 中可正常打开看到表头与数据
- 工具栏两个按钮 "导出当前查询" / "导出全量" 均走通
</success_criteria>

<output>
Create `.planning/phases/14-frontend-ux/14-fix-02-SUMMARY.md` when done
</output>