# Phase 59 — UI Review

**Audited:** 2026-10-20
**Baseline:** 59-UI-SPEC.md (approved 2026-10-19, 7/7 checker pass)
**Screenshots:** NOT captured — dev server probe found only localhost:8080 (Go backend, not the Vite frontend on :4000); all 3 playwright captures failed. Code-only audit; visual findings are code-derived and say so.
**Interaction captures:** off (workflow.ui_interaction_capture is false)

---

## Pillar Scores

| Pillar | Score | Key Finding |
|--------|-------|-------------|
| 1. Copywriting | 3/4 | Every contract string is verbatim; error headline leaks raw technical text (`查询失败:Network Error`) above the contract copy |
| 2. Visuals | 3/4 | Focal table + contract-exact 7 columns; a failed pagination wipes already-loaded rows instead of alerting above the table |
| 3. Color | 4/4 | 0 hex values, 0 accent misuse; Tag presets exactly match the SPEC mapping (orange/red/gold/blue) |
| 4. Typography | 3/4 | Heading 600 override mechanism verified sound in code, but Label-role text (Tab 标签/筛选项标题) renders antd-default 400 vs declared 600 |
| 5. Spacing | 4/4 | marginBottom 16, empty-state 32px, no arbitrary values; SPEC-internal contradiction noted (sm=8px vs `Space size="middle"`=16px) |
| 6. Experience Design | 3/4 | Full state coverage (loading/error/empty/retry/persisted tab); unauthorized user's tab click still fires a `getDeptTree()` request, breaching the "不发任何请求" gate |

**Overall: 19/24**

---

## Top 3 Priority Fixes

1. **[WARNING] Dept-tree request fires for unauthorized users** — `ReconciliationTab.tsx:80` calls `useDeptTree()` unconditionally before the `canView` early return (line 232). React Query mounts and fetches `getDeptTree()` the moment an unauthorized user clicks the「对账」Tab, violating D-17/UI-SPEC「无权限时…**不发任何请求**」(the 38 automated cases only prove `getDeptOUMismatchList` count = 0). Impact: contract breach + pointless API load for every 403 viewer. Fix: add an `enabled` option to `useDeptTree(options)` and pass `enabled: canView`, or move the `useDeptTree()` call into a gated child component.
2. **[WARNING] Raw technical error surfaces as the alert headline** — `ReconciliationTab.tsx:280-285` passes `description` but not a controlled `message`, so `ErrorAlertWithRetry.tsx:113` renders `查询失败:${messageText}` — for axios failures that is English ("Network Error" / "Request failed with status code 502") while the contract copy「对账数据加载失败，请稍后重试。」is demoted to the secondary line. Fix: add an optional `message` prop to `ErrorAlertWithRetry` and pass「对账数据加载失败」(keep the contract sentence as description), or fall back to a neutral Chinese headline instead of the raw error text.
3. **[WARNING] Label-role typography unmet + heading override unverified in a real browser** — UI-SPEC Typography declares Label (表头、Tab 标签、筛选项标题) = 14px/600. antd 6 defaults give table headers 600 (correct) but Tab labels and `<Form.Item label>` render 400; the phase added no styling for them. The one enforced item — empty-state title 16px/600 via `ReconciliationTab.css:16` `!important` override — is mechanism-sound (selector `.ant-empty-description > div > div:first-child` matches antd 6.6.4's Empty DOM at `node_modules/antd/es/empty/index.js:85`, and author `!important` beats non-important inline style) but was never observed rendered (jsdom + no screenshots; D7 is Manual-Only). Fix: either set weight 600 on the Tab's tab-label/filter-label chrome or amend the SPEC to accept antd 400 defaults; run one real-browser check of the empty-state weight at UAT.

---

## Detailed Findings

### Pillar 1: Copywriting (3/4)

Contract strings verified verbatim in `ReconciliationTab.tsx` / `index.tsx`:

| Contract item | Expected | Actual | Verdict |
|---|---|---|---|
| Primary/secondary CTA | 查询 / 重置 | lines 272 / 275 | PASS |
| Tab labels | 配置列表 / 对账 | `index.tsx:343,525` | PASS |
| Empty heading/body | 未发现不一致条目 / 当前视图为实时计算。… | lines 289-290 | PASS |
| Error copy + button | 对账数据加载失败，请稍后重试。 / 重新加载 | line 283 + shared `ErrorAlertWithRetry.tsx:129` | PASS (as description) |
| 403 title/subTitle | 无对账查看权限 / 请联系管理员…重新登录。 | lines 236-237 | PASS |
| Placeholders | 全部类型 / 全部部门 / 搜索用户名 / 部门名称 / OU DN | lines 248 / 257 / 265 | PASS |
| `showTotal` | 共 N 条 | line 306 | PASS |
| Tag labels | 四类中文标签 + 已删除 | `MISMATCH_TYPE_META:40-45`, line 165 | PASS |
| `mismatchDetail` 原样渲染、不二次拼装 | — | line 220 renders value directly | PASS |

- **WARNING:** error-state headline is uncontracted. `ErrorAlertWithRetry` default branch (`ErrorAlertWithRetry.tsx:113`) puts `查询失败:${messageText}` in the Alert `message` slot; `messageText` is the raw axios/backend message (can be English). The SPEC's error copy only governs the description slot.
- Minor (pre-existing, out of phase contract but in a modified file): `index.tsx:321-322` delete confirm uses generic 确定/取消.
- Keyword Input intentionally has no `label` — matches the SPEC table literally (only 类型/部门 declare labels), though it is visually asymmetric next to two labeled siblings. Not scored as a defect.

### Pillar 2: Visuals (3/4)

- Focal point is the 7-column result table; filter row is clearly secondary — matches the checker's declared hierarchy.
- No icon-only buttons anywhere in the new Tab; 查询/重置/重新加载 all carry text. No aria-label debt.
- Column widths and order exactly match SPEC (180/180/150/260/160/260/320, `scroll.x=1400`, `ReconciliationTab.tsx:139-229`); ellipsis+Tooltip on the three long-text columns; NULL cells render `-`.
- **WARNING:** state transition erases data. When a later page/filter request fails, `error ? <ErrorAlertWithRetry/> : …` (line 280) replaces the Table entirely — rows the user was just reading disappear, leaving only filter + alert. antd-idiomatic behavior is Alert above the surviving table. Not a SPEC violation (SPEC is silent) but a real UX regression on flaky networks.
- Optional refresh entry (SPEC「[刷新]（可选…）」) not implemented — declared optional, recorded for UAT awareness.
- `已删除` red Tag renders directly against the dept name with only antd Tag's `margin-inline-end` — no leading gap; cosmetic.

### Pillar 3: Color (4/4)

- `grep -nE '#[0-9A-Fa-f]{3,8}|rgb\('` over `ReconciliationTab.tsx` + `ReconciliationTab.css` → **0 hits** (SPEC hard constraint "Tab 内禁 hex" met; the CSS file deliberately writes only `font-weight`).
- Tag semantics exact vs SPEC mapping: `user_ou_mismatch`=orange, `dept_soft_deleted`=red, `mapping_disabled`=gold, `config_unrouted`=blue (`MISMATCH_TYPE_META`, lines 40-45); soft-deleted dept gets red 已删除 (line 165). Preset names only — dark-mode safe.
- Accent discipline: `type="primary"` only on 查询 (line 271); Tabs active underline via theme `colorPrimary`; no accent on body text, headers, or row hover. No `text-primary`/`bg-primary` class usage.
- No hardcoded colors introduced in `index.tsx` either; the pre-existing blue/green/success Tags are untouched prior code.

### Pillar 4: Typography (3/4)

- No Tailwind `text-*`/`font-*` classes and no inline font sizes in the new Tab; type comes from antd theme = Body 14/400 ✓, Table headers 14/600 ✓ (both match SPEC roles).
- **WARNING (label role):** SPEC declares Tab 标签 and 筛选项标题 at 14px/600; antd 6 renders both at 400 and the phase adds nothing to correct it. Only the heading role was enforced.
- Empty-state title: shared component hardcodes inline `fontSize:16, fontWeight:500` (`EmptyStateWithAction.tsx:48`); `ReconciliationTab.css:16-18` overrides weight to 600 via tab-scoped `!important` (author-important beats non-important inline). Selector path verified against antd 6.6.4 source (`es/empty/index.js:85` applies `ant-empty-description`). Rendering itself could not be observed (no browser) — the plan's own Manual-Only item, still open.
- The 500 in the shared component remains in the DOM but is neutralized within this Tab only — no other surface affected.

### Pillar 5: Spacing (4/4)

- `marginBottom: 16` on the filter form (line 244) = SPEC md; empty-state vertical whitespace `padding: 32px 0` (shared `EmptyStateWithAction.tsx:55`) = SPEC xl; `Space size="middle"` between controls = the layout contract's explicit directive.
- Zero arbitrary bracket values (`[...]px`/`[...]rem`) in phase files; the only inline styles are two control widths (220/240/260) and `whiteSpace: nowrap` on Tags (SPEC overflow rule) — all permitted.
- Observation, not a defect: the SPEC contradicts itself — its spacing table says 筛选控件之间 = sm 8px while its layout contract mandates `<Space size="middle">` (16px). The implementation follows the explicit component directive; the token table row should be amended.
- Configs-tab children keep the pre-existing `marginBottom: 16` toolbar rhythm — consistent.

### Pillar 6: Experience Design (3/4)

State coverage (all code-verified, tests claimed 38 green in 59-04-SUMMARY):

- Loading: Table `loading` mask; both 查询/重置 enter `loading` during requests (lines 271-276), preventing duplicate submits ✓
- Error: `ErrorAlertWithRetry` with `onRetry={loadList}` (line 280-285); shared component additionally handles code 1007 (auto-logout + redirect) and 500 ✓; no stacked `message` toasts ✓
- Empty: `EmptyStateWithAction`, not an empty table header (lines 286-292) ✓
- Destructive actions: none (read-only; tests assert no 修复/删除/导出 buttons) ✓
- Filter contract: explicit submit only — `grep -cE "onChange=.*loadList|onChange=.*setCurrent\(1\)"` = 0; Enter submits (line 268); reset returns to page 1; filters survive pagination (submitted values in `loadList` deps, line 113) ✓
- Tab: `activeTab` persisted via `usePersistedStateController` (session storage default confirmed at `usePersistedState.ts:105`), antd default keeps both panes mounted after first visit, so the reconciliation tab mounts/fetches only on first activation ✓
- **WARNING:** the zero-request permission gate is only half-true. `useDeptTree()` (line 80) runs before the `canView` return and fetches the dept tree for unauthorized viewers. Fix as in Top-3 item 1.
- Minor: `loadList` has no request sequencing/AbortController — a slow earlier response could overwrite a newer one after rapid pagination. Low probability (buttons gate re-entry; only page-size changes race), but cheap to harden.
- Minor: after a failed first load the retry path clears `error` and shows the loading table correctly — verified sound (lines 96-119).

---

## Files Audited

- `ruoyi-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.tsx` (new, full read)
- `ruoyi-react-frontend/src/pages/ad-domain/configs/ReconciliationTab.css` (new, full read)
- `ruoyi-react-frontend/src/pages/ad-domain/configs/index.tsx` (modified, full read — Tabs integration)
- `ruoyi-react-frontend/src/lib/adDomainApi.ts` (modified — lines 225-243, 785-839: `getDeptOUMismatchList`, `DeptOUMismatchRow`, `withDefaultPagination`)
- `ruoyi-react-frontend/src/components/shared/EmptyStateWithAction.tsx` (dependency, full read)
- `ruoyi-react-frontend/src/components/shared/ErrorAlertWithRetry.tsx` (dependency, full read)
- `ruoyi-react-frontend/src/components/shared/DepartmentTreeSelect.tsx` (dependency, full read)
- `ruoyi-react-frontend/src/hooks/useDeptTree.ts` (dependency, full read)
- `ruoyi-react-frontend/src/hooks/usePersistedState.ts` (grep — session default)
- `node_modules/antd/es/empty/index.js` (grep — `ant-empty-description` class binding, antd 6.6.4)

Registry audit: skipped — `components.json` absent (NO_SHADCN); UI-SPEC Registry Safety table declares no third-party blocks.

Backend plans 01-03 produced no UI surface (guard is server-side; view/endpoint are data layer) and were audited only via their SUMMARY/PLAN contracts where they constrain the frontend.
