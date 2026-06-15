---
slug: reset-theme-hardcoded-colors
status: fixing
trigger: "默认主题修改保存成功了，但是在用户设置界面重置主题时还是恢复到了硬编码的颜色和布局"
created: 2026-06-15
updated: 2026-06-15
---

## Reasoning Checkpoint

```yaml
reasoning_checkpoint:
  hypothesis: "handleReset() in pages/settings/index.tsx falls back to hardcoded hex literals (#4F46E5, #1E293B) when preferences.theme.customColors is empty, instead of fetching the admin-configured default theme via getDefaultThemeConfig()."
  confirming_evidence:
    - "Lines 98 and 103 of pages/settings/index.tsx contain hardcoded '#4F46E5' and '#1E293B' in the reset path."
    - "lib/defaultThemeApi.ts exposes getDefaultThemeConfig() hitting /system/settings/config/theme/default — but the reset path never calls it."
    - "Backend default_theme_service.go.GetDefaultThemeConfig returns the admin-configured sys.theme.default config from sys_config — but no frontend reset code consumes it."
    - "themeStore has no resetTheme action; reset logic lives in the settings page handler."
  falsification_test: "If the reset path actually called getDefaultThemeConfig() and applied its result, the hardcoded literals would not be the fallback — they would be the result of the network call or its error path. Observation: lines 95-104 contain NO call to the API or to the theme store, only setState to literal hex strings."
  fix_rationale: "Replace the hardcoded fallback branch in handleReset with a call to getDefaultThemeConfig() so reset restores admin-configured defaults. Scope: only modify handleReset, not themeStore or layout system, per scope constraint."
  blind_spots:
    - "Are there other UI reset paths in the app? (ThemeSwitcher has no reset; only the settings page has a reset button.)"
    - "What if the backend returns no customColors? The frontend should gracefully fall back to current prefs, not the hardcoded values."
    - "If admin never set a default, GetDefaultThemeConfig returns { mode: 'light', style: 'minimal' } with no customColors — that is the correct admin-default fallback."
```

# Debug Session: reset-theme-hardcoded-colors

## Trigger (verbatim)

> 默认主题修改保存成功了，但是在用户设置界面重置主题时还是恢复到了硬编码的颜色和布局

## Symptoms (from trigger)

- **Component**: User Settings → Theme → "重置主题" (reset theme) action
- **Working baseline**: Default theme config page (settings → default-theme) saves successfully to backend (commit `9f2f02a` introduced this).
- **Bug**: When user clicks reset in user settings UI, the UI reverts to **hardcoded colors/layout** instead of the admin-configured default theme.
- **Expected**: Reset should restore the system default theme (which admin configured via default-theme page).
- **Actual**: Reset uses literal hardcoded fallback values (somewhere in `themeStore` or layout reset handler).

## Hypotheses to investigate

1. Reset handler uses local fallback constants instead of fetching `defaultThemeConfig` from backend
2. `themeStore.reset()` ignores the admin-saved default theme override
3. Backend `getDefaultThemeConfig` is called but response is not applied
4. Multiple reset paths exist; only some are broken

## Related context

- Just-fixed sibling bug: `default-theme-missing-fields` (load fields from current settings → now populates primaryColor/sidebarColor)
- Files to inspect:
  - `ruoyi-react-frontend/src/store/themeStore.ts` (resetTheme action)
  - `ruoyi-react-frontend/src/pages/settings/index.tsx` (reset button handler)
  - `ruoyi-react-frontend/src/services/configService.ts` (getDefaultThemeConfig API)
  - `ruoyi-react-frontend/src/design-system/themes/index.ts` (theme registry / fallback constants)
  - `ruoyi-react-frontend/src/design-system/components/ThemeSwitcher.tsx`
  - `internal/services/system/default_theme_service.go` (backend default)

## Current Focus

**Hypothesis**: `handleReset()` in `pages/settings/index.tsx` (lines 90-105) resets the color pickers to HARDCODED values `#4F46E5` and `#1E293B` instead of restoring the admin-configured default theme. The form's `form.resetFields()` only restores to `initialValues={preferences}` (current user prefs) — not the system default. The "重置" button in the user settings page does NOT use the backend's `getDefaultThemeConfig` endpoint at all.

**Test**: Trace `handleReset` line-by-line; confirm hardcoded literals `#4F46E5` and `#1E293B`.

**Expecting**: Confirmed. Reset has no path to fetch `getDefaultThemeConfig`.

**Next action**: Verify root cause with reasoning checkpoint, then apply minimal scoped fix to the reset path.

## Evidence

- timestamp: 2026-06-15
  - checked: `pages/settings/index.tsx` line 90-105 (`handleReset`)
  - found: Hardcoded fallback values `'#4F46E5'` and `'#1E293B'` on lines 98 and 103.
  - implication: Reset DOES NOT consult the admin-configured default theme.
- timestamp: 2026-06-15
  - checked: `pages/settings/index.tsx` `initialValues={preferences}` (line 155)
  - found: Form is initialized from CURRENT user preferences, not system default.
  - implication: `form.resetFields()` only restores to current user prefs.
- timestamp: 2026-06-15
  - checked: `themeStore.ts`
  - found: No `resetTheme` action exists. Has `resetPreview` (cancels preview) and `syncFromSettings` (applies prefs from settings store). No code path calls `getDefaultThemeConfig`.
  - implication: themeStore is not the broken layer; the bug is in the settings page handler.
- timestamp: 2026-06-15
  - checked: `lib/defaultThemeApi.ts`
  - found: `getDefaultThemeConfig()` calls `GET /system/settings/config/theme/default` returning `ThemeConfiguration { mode, style, customColors }`.
  - implication: The backend admin-default API is available but never used by the user settings reset.
- timestamp: 2026-06-15
  - checked: `internal/services/system/default_theme_service.go`
  - found: `GetDefaultThemeConfig` returns admin-configured `Mode`, `Style`, and `CustomColors` from `sys_config` table.
  - implication: Backend correctly serves admin default; frontend never calls it on reset.

## Eliminated

- hypothesis: "themeStore.resetTheme ignores the admin default"
  - evidence: themeStore has no `resetTheme` action; the reset is in the settings page handler.
  - timestamp: 2026-06-15