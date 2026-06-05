---
phase: 25
slug: vm-data-scope-permissions
status: verified
threats_open: 0
asvs_level: 1
created: "2026-06-05T08:25:00Z"
---

# Phase 25 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Database → Application | SQL injection via permission strings | Permission identifiers (vdi:vm:*) |
| Admin → Database | Direct SQL modification bypassing migration | Role-menu associations |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-25-01 | Tampering | Migration script | mitigate | 权限字符串为硬编码常量，非用户输入 | closed |
| T-25-02 | Spoofing | Role permission assignment | accept | 迁移在可信环境中执行，风险低 | closed |
| T-25-03 | Information Disclosure | sys_menu table | accept | 菜单权限非敏感数据，风险低 | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-25-01 | T-25-02 | 数据库迁移在可信环境执行，无需额外缓解 | GSD Security | 2026-06-05 |
| AR-25-02 | T-25-03 | 菜单权限数据为公开配置，非敏感信息 | GSD Security | 2026-06-05 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-06-05 | 3 | 3 | 0 | gsd-secure-phase |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-06-05
