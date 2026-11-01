# Phase 129 真机 UAT 结果记录（v1.51 端口写全动作）

> 使用：每执行一条，追加一行。基线段先于一切写动作填写。
> 关联清单：`129-HUMAN-UAT.md`；执行工具：`scripts/integration/site-visit-v1.51.sh`

## 基线段（先填！恢复全靠它）

| 项 | 值 |
|---|---|
| 华为测试端口 | （接口名 / 初始 PVID / 初始 link-type / 初始 allowed 列表 / 初始 description / 初始 port-security 状态） |
| 锐捷测试端口 | （同上） |

## 执行记录（每条一行：动作 / 设备 / 实际输出摘要 / 结论 / 时间戳）

| # | 条目 | 设备 | dry-run 结论 | 实发结论 | 验证命令实际输出摘要 | 恢复动作已执行 | 结论(PASS/FAIL/DEFERRED) | 时间戳 |
|---|---|---|---|---|---|---|---|---|
| 1 | HW-00 基线 | S8700 | - | - | （抄录） | - | - | |
| 2 | RJ-00 基线 | RS8607E | - | - | （抄录） | - | - | |
| 3 | HW-01 shutdown | S8700 | | | | | | |
| 4 | HW-02 undo_shutdown | S8700 | | | | | | |
| 5 | HW-03 description | S8700 | | | | | | |
| 6 | HW-04 dot1x_enable | S8700 | | | | | | |
| 7 | HW-05 dot1x_disable | S8700 | | | | | | |
| 8 | HW-06 set_access_vlan | S8700 | | | | | | |
| 9 | HW-07 port_binding add IP+MAC | S8700 | | | | | | |
| 10 | HW-08 port_binding add IP | S8700 | | | | | | |
| 11 | HW-09 port_binding remove | S8700 | | | | | | |
| 12 | HW-10 trunk_set_native | S8700 | | | | | | |
| 13 | HW-11 trunk_allowed add | S8700 | | | | | | |
| 14 | HW-12 trunk_allowed remove | S8700 | | | | | | |
| 15 | HW-13 trunk_set_hybrid | S8700 | | | | | | |
| 16 | HW-14 portsec enable+max | S8700 | | | | | | |
| 17 | HW-15 sticky+static_bind | S8700 | | | | | | |
| 18 | RJ-01 shutdown | RS8607E | | | | | | |
| 19 | RJ-02 undo_shutdown | RS8607E | | | | | | |
| 20 | RJ-03 description | RS8607E | | | | | | |
| 21 | RJ-04 dot1x_enable | RS8607E | | | | | | |
| 22 | RJ-05 dot1x_disable | RS8607E | | | | | | |
| 23 | RJ-06 set_access_vlan | RS8607E | | | | | | |
| 24 | RJ-07 port_binding add IP+MAC | RS8607E | | | | | | |
| 25 | RJ-08 port_binding add IP | RS8607E | | | | | | |
| 26 | RJ-09 port_binding remove | RS8607E | | | | | | |
| 27 | RJ-10 trunk_set_native | RS8607E | | | | | | |
| 28 | RJ-11 trunk_allowed add | RS8607E | | | | | | |
| 29 | RJ-12 trunk_allowed remove | RS8607E | | | | | | |
| 30 | RJ-13 trunk_allowed replace | RS8607E | | | | | | |
| 31 | RJ-14 hybrid expected-reject | RS8607E | （400 断言） | - | - | - | | |
| 32 | RJ-15 portsec 全链 | RS8607E | | | | | | |

## 收尾核对（对照基线段）

- [ ] 华为测试端口已恢复初始状态（差异：无 / 见尾注）
- [ ] 锐捷测试端口已恢复初始状态（差异：无 / 见尾注）

## 尾注（遗留差异 / follow-up）

（无）
