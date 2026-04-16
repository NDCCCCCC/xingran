# Milestones

## v1.1 信息点导入设备端口关联 (Shipped: 2026-04-16)

**Phases completed:** 1 phase, 1 plan, 2 tasks
**Git range:** dff1003 → 4dcc37e
**Files:** 9 changed (501 insertions, 31 deletions)

**Key accomplishments:**

- Added "所属设备" column to infoPoint Excel import via `deviceName` → `sys_network_device.device_name` → `device_id` Reference resolution
- Added "所属端口" column to infoPoint Excel import via `portName` → `sys_device_port_status.interface_name` → `port_id` Reference resolution
- Both columns optional — match failure leaves field empty, import never blocked
- Template downloads include device/port example values ("核心交换机A", "Gi0/1")

**Key decisions:**
- D-01: Device uses Reference to `sys_network_device.device_name` (exact match)
- D-02: Port uses Reference to `sys_device_port_status.interface_name` (exact match)
- D-03: Both columns optional, match failure leaves field empty
- D-04: Port uses global lookup, no DependsOn scoping to device

---
