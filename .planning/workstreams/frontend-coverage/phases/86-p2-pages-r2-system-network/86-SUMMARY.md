---
phase: 86-p2-pages-r2-system-network
plan: 00
wave: all
status: complete
completed: 2026-08-27
---

## Phase 86 Complete: P2 页面层 R2 — system + network

**pages/system 2203 stmts 2.72%→6.95%(floor 2.2→6.4) / pages/network 1962 stmts 3.11%→7.65%(floor 2.6→7.1)**

### 4 Waves / 66 tests
| Wave | 范围 | tests | floor |
|------|------|-------|-------|
| 1 | system: dept utils + role utils + notice hooks | 17 | sys→3.7 |
| 2 | system: menu/user utils + dict constants + apikeys/config/post | 22 | sys→6.4 |
| 3 | network: discoveries parseIPRanges + exec/tmpl constants | 14 | net→4.2 |
| 4 | network: backups computeDiff/groupByDevice + devices utils + command constants + credentials | 13 | net→7.1 |

### 高价值纯函数覆盖
- dept: flattenTreeToList / transformToParentTreeOptions / renderTreeData
- menu: flattenTree / buildParentOptions / calculateStatistics
- user: formatGender / formatStatus
- discoveries: parseIPRanges(dash/CIDR24/CIDR16/CIDR8/单 IP/多行/空)
- backups: computeDiff / groupBackupsByDevice
- devices: getOptionLabel / getStatusColor

### Verification
- **1128/1128 tests PASS / 128 files**(1067 存量 + 61 新增)
- gate **0 FAIL**,GLOBAL **21.60%**
- ratchet 4 行(86-W1..W4)

### Gotcha 沉淀
- ESLint no-restricted-syntax 禁止硬编码内网 IP——测试向量 IP 需文件头 eslint-disable 注释
- noticeApi 返回 {data: {...}} 嵌套,mock 需双层包装