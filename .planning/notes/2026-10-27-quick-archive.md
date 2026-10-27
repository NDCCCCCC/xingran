# HYG-01：`.planning/quick/` 94 个历史任务目录归属判定与归档（2026-10-27）

## 统计

**94 = v1.1×1 + v1.5×1 + v1.7×1 + v1.8×3 + v1.11×14 + v1.12×27 + v1.13×7 + v1.14×1 + v1.15×6 + v1.16×8 + v1.17×9 + v1.18×1 + v1.20×2 + v1.21×11 + obsolete×2**

- 归档位置：`.planning/milestones/<版本>-phases/quick/<原目录名>/`（如该版本无 `-phases` 目录则新建；v1.4~v1.20 的既有 phase 归档平铺在 `phases-archive-pre-v1.21/`，为不改动既有结构，本任务按任务规范统一新建 `<版本>-phases/quick/`）
- 无法归属：2 个 → `.planning/milestones/obsolete-quick/`（原因见该目录 README.md）
- 判定依据：任务完成日期落在该里程碑 SHIPPED 日之前（窗口法）+ 主题相关。AD 系列在 05-27~06-05 间落在 v1.11（AD 组自动同步，shipped 05-26）之后，按主题延续归 v1.11；260712/260713 落在 v1.20（07-10）之后紧邻，按主题归 v1.20；261012~261015 楼层工位统计类与 AD 无关但窗口落在 v1.21（10-22），按窗口归 v1.21

## 归属清单

| 目录 | 归属 | 判定依据 |
|---|---|---|
| 20260416-backend-optimize | v1.1 | 日期 2026-04-16 = v1.0/v1.1 shipped 当日 |
| default-theme-config | v1.5 | 日期 2026-06-15 = v1.5 shipped 当日（默认主题前端功能） |
| 260520-jrp-request-encryption-config-toggle | v1.7 | 日期 05-20 + 主题「请求加密开关参数」= v1.7 前后端加密配置同步 |
| 260521-build-embedded-linux | v1.8 | 日期 05-21 窗口（嵌入式构建杂务） |
| 260521-ksr-1-exe-cmd-exe-main-exe-ruoyi-backend-exe | v1.8 | 日期 05-21 窗口（清理构建产物杂务） |
| 260521-m45-clean-empty-folders | v1.8 | 日期 05-21 窗口（清理空目录杂务） |
| 260525-enhance-ou-management | v1.11 | 日期 05-25 + OU 管理主题（AD 扩展） |
| 260525-ldap-factory-refactor | v1.11 | 日期 05-25 + LDAP 重构主题（AD 扩展；任务自身标记 OBSOLETE=已关闭，仍按主题归档） |
| 20260525-vdi-menu-registration | v1.12 | VDI 菜单注册 → 深信服桌面云主题 |
| 20260526-fix-update-request-id-validation | v1.11 | 日期 05-26 窗口 |
| 260526-nij | v1.11 | 日期 05-26 窗口（用户管理部门路径显示修复） |
| 260526-nwn | v1.11 | 日期 05-26 窗口（用户编辑角色显示修复） |
| 260526-pxd | v1.11 | 日期 05-26 窗口（用户表格部门路径显示） |
| 260526-0o7-vdi-config-to-params | v1.12 | VDI 主题 |
| 260526-d69-vdi-sync-hybrid | v1.12 | VDI 混合同步主题 |
| 260526-dbu-vdi-sync-enhancement | v1.12 | VDI 同步完善主题 |
| 20260527-ad-user-sync-dept-refactor | v1.11 | AD 用户同步主题延续（v1.11 shipped 05-26） |
| 20260527-fix-addn-field-name | v1.11 | AD 主题延续（ADDN 字段修复） |
| 20260527-ou-user-sync | v1.11 | AD/OU 用户同步主题延续 |
| 20260527-remove-automap-add-ou-groups | v1.11 | AD/OU 用户组管理主题延续 |
| 20260527-vdi-last-sync-time-fix | v1.12 | VDI 主题 |
| 20260528-ou-group-mapping | v1.11 | OU-组映射主题延续 |
| 20260529-vdi-create-vm | v1.12 | VDI 创建虚拟机主题 |
| 260527-commit-fixes | v1.12 | 日期 05-27 窗口（分批提交杂务） |
| 260527-compile-test | v1.12 | 日期 05-27 窗口（编译测试杂务） |
| 260527-crypto-type-audit | v1.12 | 日期 05-27 窗口（加密类型审计） |
| 260527-crypto-type-fix | v1.12 | 日期 05-27 窗口（加密类型修复） |
| 260527-enable-three-sync-tasks | v1.12 | 日期 05-27 窗口（启用同步调度任务） |
| 260527-gdc-ad-git | v1.12 | 日期 05-27 窗口（AD 相关 git 杂务） |
| 260527-gra-user-adoudn | v1.12 | 日期 05-27 窗口（user adOU DN 字段） |
| 260528-vxy-vdi-id-vdi-id-cpu | v1.12 | VDI 主题 |
| 260529-d95-vdi | v1.12 | VDI 主题 |
| 260529-ivp-vdi-vdi-test-standalone-go-host-id-run-p | v1.12 | VDI 主题 |
| 260529-j0l-vdi | v1.12 | VDI 主题 |
| 260529-j3y-vdi-host-id-run-position-id-vtp-id-vdi-t | v1.12 | VDI 主题 |
| 260529-vdi-form | v1.12 | VDI 创建表单 UI 主题 |
| 20260602-remove-vdi-config-ip | v1.12 | 日期 06-02 = v1.12 shipped 当日，VDI 主题 |
| 20260602-vdi-merge-simplified-create | v1.12 | 日期 06-02，VDI 主题 |
| 20260602-vdi-vm-list-fix | v1.12 | 日期 06-02，VDI 虚拟机列表主题 |
| 20260602-vmp-api-integration | v1.12 | 日期 06-02，VMP 虚拟机 API 集成主题 |
| 260602-jk9-vm-list-controls | v1.12 | 日期 06-02 窗口末，虚拟机列表控制 |
| 260602-k1b-vdi | v1.12 | 日期 06-02，VDI 主题 |
| 260605-irs-plan-22b-02-callbackend-401-403-handler | v1.12 | 主题「22B 回调后端 401/403」= 深信服 22B |
| 260604-cqo-nickname-username | v1.13 | 日期 06-04 窗口（资产/用户字段） |
| 260604-unify-cache-key-naming | v1.13 | 日期 06-04 窗口（缓存键统一，技术杂务） |
| 260605-h3p-ad-group-auto-sync | v1.11 | 主题 = v1.11 同名「AD 组自动同步」（日期 06-05 延续） |
| 260605-hf7-fix-ou-mapping-query | v1.11 | OU 映射查询修复（AD 主题延续） |
| 260605-iq7-ad-ad-1-ad-2-3-ad-4 | v1.11 | AD 组成员同步数据源分析（AD 主题延续） |
| 260608-asset-excel-columns | v1.13 | 日期 06-08 = v1.13 资产管理 shipped 当日 |
| 260608-ljx-migration-146-add-asset-menu-permissions | v1.13 | 日期 06-08 + 资产菜单权限主题 |
| 260608-monitor-dashboard-stats-layout | v1.13 | 日期 06-08 窗口（监控仪表盘布局） |
| 260609-009-asset-type-filter | v1.13 | 资产类型过滤主题（06-09 延续） |
| 260609-asset-list-columns-add | v1.13 | 日期 06-09，资产列表字段主题 |
| 20260609-ad-user-rowselection-fix | v1.14 | 日期 06-09 窗口（域控用户多选框修复） |
| 260610-device-source-refactor | v1.15 | 日期 06-10 窗口（设备数据源重构） |
| 260610-workstation-device-expand | v1.15 | 日期 06-10 + 工位设备主题 = v1.15 工位设备关联 |
| 260612-d2x-ip | v1.15 | 日期 06-12 窗口 |
| 260613-vp5-workstation-expand-fix | v1.15 | 工位设备展开修复主题 |
| 260614-dpz-workstation-main-device-merge | v1.15 | 工位主设备合并主题 |
| 260614-rrk-id | v1.15 | 日期 06-14 窗口（工位编辑字段显示） |
| 20260613-pre-existing-ts-errors | v1.16 | 主题「清理 TypeScript 错误」= 技术债清理（06-26 前） |
| 20260623-batch-sync-optimize | v1.16 | 日期 06-23 窗口（批量同步优化） |
| 260617-ch1-add-table-sorting | v1.16 | 日期 06-17 窗口（表格排序） |
| 260622-lv9-xlsx-email | v1.16 | 日期 06-22 窗口（xlsx 邮件） |
| 260622-u8n-ad-manager-leader-ad | v1.16 | 日期 06-22 窗口（AD manager 字段） |
| 260626-dai-workstation-export-fields | v1.16 | 日期 06-26 = v1.16 shipped 当日（工位导出字段） |
| 260626-n71-workstation-status-occupied-from-user | v1.16 | 日期 06-26（工位占用状态） |
| 260626-workstation-primary-device-serial | v1.16 | 日期 06-26（工位主设备序列号） |
| 20260630-mac-page-dept-tree | v1.17 | 日期 06-30 窗口（v1.16 后 v1.17 前） |
| 20260630-merge-mac-trajectory-to-list-drawer | v1.17 | 日期 06-30 窗口（MAC 轨迹合并） |
| 20260630-skip-setup-impl | v1.17 | 日期 06-30 窗口 |
| 20260630-startup-flow-classify | v1.17 | 日期 06-30 窗口（启动流程分类） |
| 260701-iface-sort | v1.17 | 日期 07-01 窗口（接口排序） |
| 260702-148-fix-dashboard-exception-rules-link | v1.17 | 日期 07-01 + 对账看板主题 = v1.17 资产对账 |
| 260703-asset-list-fields-remediation | v1.17 | 日期 07-03 = v1.17 shipped 当日 |
| 260703-dkc-ruoyi-react-frontend-src-pages-operation | v1.17 | 日期 07-03 窗口（前端 operation 页面） |
| 260703-eaj-b-schema-embed-ruoyi-react-frontend-src- | v1.17 | 日期 07-03 窗口（schema 嵌入） |
| 260704-ne5-database-go-migration-schema-seed-snapsh | v1.18 | 日期 07-04 = v1.18 shipped 当日（migration/snapshot 杂务） |
| 260712-vpj-mac-mac | v1.20 | 日期 07-12 紧邻 v1.20（07-10）+ 端口 MAC 展示主题延续 |
| 260713-df0-username | v1.20 | 日期 07-13 窗口兜底（工位导入部门/用户字段扩展） |
| 261012-l7k-floor-workstation-stats | v1.21 | 日期 10-12 落在 v1.21 窗口（10-22 前，楼层工位统计） |
| 261013-jf1-building-floor-count | v1.21 | 日期 10-13 窗口（楼宇楼层数统计） |
| 261013-lqb-ops-buildings-total-floors | v1.21 | 日期 10-13 窗口（total_floors 写路径重算） |
| 261014-ctp-create-update-count-total-floors | v1.21 | 日期 10-14 窗口（261013-lqb 收尾） |
| 261015-ops-fk-drift-guard | v1.21 | 日期 10-15 窗口（外键漂移守卫） |
| 261016-ad-login-history | v1.21 | 日期 10-16 + AD 登录历史主题（Phase 59 窗口期） |
| 261019-mmc-extractrdnfromdn-rfc4514-modifydn-ad-pha | v1.21 | 日期 10-19 + ExtractRDNFromDN/ModifyDN = v1.21 Phase 57/58 AD 主题 |
| 261020-d0d-embed-xingran-api-xingran-nginx-conf | v1.21 | 日期 10-20 窗口（xingran nginx 部署杂务） |
| 261021-iqq-drain-interval-hot-reload | v1.21 | 日期 10-21 + drain interval 热加载 = v1.21 Phase 58 ADPushDrainer 主题 |
| 20261022-ad-sync-fixes | v1.21 | 日期 10-22 = v1.21 shipped 当日 + AD 同步修复主题 |
| 20261023-ad-domain-dept-sync-fixes | v1.21 | 日期 10-23 紧邻 + 域控部门同步主题延续 |

## obsolete（2 个，详见 obsolete-quick/README.md）

| 目录 | 原因 |
|---|---|
| 20250610-fix-workstation-filter-loss | 归属不明：目录日期 2025-06-10 早于项目起点（2026-04-16），无法按日期+主题定位 |
| 20250622-filter-computer-accounts | 归属不明：目录日期 2025-06-22 早于项目起点，无法定位 |
