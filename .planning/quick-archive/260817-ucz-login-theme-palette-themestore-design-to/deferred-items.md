# Deferred Items — quick-260817-ucz

## Out-of-scope discoveries (not fixed, per scope boundary)

1. **`internal/services/system/default_theme_service.go` not gofmt-clean (PRE-EXISTING)**
   - Discovered during Task 2 verification (`gofmt -l` flagged the file).
   - Verified pre-existing: the HEAD version (before this task's commits) is also flagged.
   - Affected regions unrelated to this task: `ThemeConfiguration` struct tag/comment alignment (lines ~13-17), `defaultThemeService` struct field alignment (~30-35), constructor (~37-42), and `validStyles` map key alignment (`"minimal": true` had no padding before this task).
   - Not fixed because: (a) scope boundary — reformatting touches lines outside this task's edits; (b) running `gofmt -w` would pad the new map entry to `"ink-amber":     true,` and break the plan's must_hives artifact check (`contains: "\"ink-amber\": true"`).
   - Suggested follow-up: one-off `gofmt -w internal/services/system/default_theme_service.go` in a standalone chore commit.

2. **Pre-existing frontend lint warnings (1033, 0 errors)**
   - `npm run lint` reports 1033 warnings across the codebase (e.g. `@typescript-eslint/no-unsafe-assignment` in `vite.config.ts`). All pre-existing, none introduced by this task (0 errors, exit 0).
