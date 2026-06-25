---
slug: vm-sync-ip-address-issue
name: vm-sync-ip-address-issue
status: resolved
trigger: 虚拟机信息定时同步任务，使用的api接口是可以获取ip地址的，但是现在只有两台获取静态地址，其他的都是-，请检查原因
created: 2026-06-02
updated: 2026-06-02
---

## Symptoms

**Expected Behavior:**
所有虚拟机都应显示IP地址

**Actual Behavior:**
只有配置了静态IP的2台虚拟机显示IP地址，其余虚拟机的IP字段显示"-"

**Error Messages:**
定时任务正常执行，没有错误日志

**Timeline:**
从功能上线开始就一直如此（不是最近才出现的问题）

**Reproduction:**
通过定时任务同步虚拟机信息后，查看虚拟机列表，发现大部分虚拟机的IP地址为"-"

**Key Pattern:**
能显示IP的2台虚拟机都有静态IP配置，其他虚拟机可能使用动态IP

## Current Focus

**Hypothesis:**
待生成 - 需要调查虚拟机同步代码，确定IP地址获取逻辑

**Test:**
待设计

**Expecting:**
待明确

**Next Action:**
gather initial evidence - 查看虚拟机同步相关代码，特别是IP地址字段的处理逻辑

**Reasoning Checkpoint:**
待填充

## Evidence

- timestamp: 2026-06-02T13:21:00
  source: code_analysis
  finding: |
    VDI API返回两个IP相关字段：
    - `IP`: 当前IP地址（running VM才有）
    - `AssignIP`: 分配的IP地址（DHCP或静态配置）

    当前代码在 `vm_service_impl.go:251` 和 `vm_service_impl.go:288` 只保存 `resource.IP` 到 `IPAddress` 字段

    问题根因：
    对于DHCP虚拟机，如果VM未运行或未连接，`resource.IP` 为空或"-"，但 `resource.AssignIP` 包含配置的IP地址
    静态IP的VM因为配置了固定IP，所以能正常显示

**Root Cause Identified:**
VDI API返回 `IP`（当前实际IP）和 `AssignIP`（分配的IP）两个字段。当前同步逻辑只保存 `IP` 字段，导致DHCP虚拟机在未运行时无法显示配置的IP地址。静态IP虚拟机因为有固定配置所以能正常显示。

## Eliminated

- VDI API调用失败 - 排除，定时任务正常执行
- 数据库字段问题 - 排除，字段定义正确
- 权限问题 - 排除，静态IP VM能正常显示

## Specialist Hints

- **frontend**: React组件显示逻辑
- **backend**: Go后端服务层修改

## Resolution

**Root Cause:**
虚拟机同步代码 `vm_service_impl.go` 在保存IP地址时，只使用了VDI API返回的 `IP` 字段（当前实际IP），而没有使用 `AssignIP` 字段（分配的IP地址）。对于DHCP虚拟机，当VM未运行或未获得DHCP租约时，`IP` 字段为空或"-"，导致前端显示"-"。

**Fix:**
修改 `vm_service_impl.go` 的 `saveOrUpdateVM` 方法（line 236-342），在保存IP地址时优先使用 `AssignIP`，如果为空则回退到 `IP` 字段：

```go
// 优先使用AssignIP（分配的IP），如果为空则使用IP（当前实际IP）
ipAddress := resource.AssignIP
if ipAddress == "" || ipAddress == "-" {
    ipAddress = resource.IP
}
```

然后在创建和更新记录时使用 `ipAddress` 而不是 `resource.IP`

**Verification:**
1. 重新运行虚拟机同步任务
2. 检查数据库中虚拟机的 `ip_address` 字段是否正确填充
3. 前端刷新虚拟机列表，确认所有VM都显示IP地址
4. 特别验证DHCP虚拟机在关机状态下也能显示配置的IP

**Files Changed:**
- `internal/services/vdi/vm_service_impl.go` - 修改IP地址获取逻辑
