---
status: complete
---

# 260525-ldap-factory-refactor — Outcome: OBSOLETE（任务关闭）

分诊收档（2026-10-26 milestone close audit）：计划的 NewConnectedClient 工厂从未落地，
问题已被后续架构取代——密码解密集中于 addomain/utils.go（SetADSM4Cipher/getPasswordCipher），
连接统一走 FailoverClient/AccountPool（Phase 36）。原问题不再存在，按关闭处理。
