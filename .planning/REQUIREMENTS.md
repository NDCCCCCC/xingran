---
type: requirements
project: RuoYi-Next
milestone: v1.16
status: planning
last_updated: 2026-06-25
---

# Requirements: v1.16 技术债清理 (Tech-Debt Cleanup)

**Project:** RuoYi-Next 网络运维管理系统
**Milestone:** v1.16 技术债清理 (Tech-Debt Cleanup)
**Status:** Planning — 处理 v1.15 累积的旧 debug 会话与 audit 数据质量缺口
**Last Updated:** 2026-06-25
**Previous:** v1.15 完结于 Phase 39（workstation dept-location alias）

---

## Milestone v1.16 Requirements

### TECH — 技术债清理

- [ ] **TECH-01**: 修复 16 个 `root_cause_found` debug 会话(根因已找到,实施修复 + 提交 commit)。涉及 AD 域控 6 个、VDI 2 个、infopoint/captcha/permission/api-key 等 8 个。
- [ ] **TECH-02**: 验证 5 个 `awaiting_human_verify` debug 会话(workstation-device-build-errors、workstation-expand-device-load-empty、reset-theme-hardcoded-colors、user-filter-lost-after-edit、vite-hmr-token-blank-page)。需在 dev 环境复现原 bug → 确认修复有效 → 标记 resolved。
- [ ] **TECH-03**: 处理 1 个 `root-cause-found` 会话(apikey-route-path-duplication,连字符变体状态值,与 TECH-01 一并处理)。
- [ ] **TECH-04**: 整理 6 个 audit 数据质量问题:
  - 为 `login-400-bad-request` / `ops-asset-constraint-uni-ops-asset-devicesn-not-exist` 补标准 YAML frontmatter(status + slug + updated + session_type + related)
  - `knowledge-base` 不是 debug 会话(是 GSD 知识库文件),明确移出 audit 范围(添加 `skip_audit: true` 注释或改名)
  - `info-point-type-null-import` / `sys-mac-filter-rules-relation-does-not-exist` 实际已是 resolved/Resolved,统一 frontmatter 格式让 audit 能正确识别
- [ ] **TECH-05**: 全部清理后,审计指标达到:`audit-open` 中 debug_sessions < 5(从当前 57 大幅下降);所有 `.planning/debug/*.md` 文件均有标准 YAML frontmatter。

### Future / Out of Scope

| 项 | 原因 |
|----|------|
| 新功能 (AI features, mobile app) | v1.16 是清理 milestone,不做功能开发 |
| VDI 22C / 22D | 依赖 v1.11 VDI 生产稳定性观察数据 |
| Phase 22 VDI 剩余扩展 | 同上 |
| DICT-29 (sys_dict 重构) | 纯样式,无功能增益,推 v1.18 |

---

## Traceability

| REQ-ID | Description | Phase | Status |
|--------|-------------|-------|--------|
| TECH-01..03 | 22 个 deferred debug 会话处理 | 40 (tech-debt-cleanup) | Planned |
| TECH-04 | 6 个 audit 数据质量问题 | 40 (tech-debt-cleanup) | Planned |
| TECH-05 | 清理完成的审计指标验收 | 40 (tech-debt-cleanup) | Planned |

---

*This file replaces the stale v1.16 requirements (which contained OPERLOG-01..06 from Phase 35, completed in v1.15).*
