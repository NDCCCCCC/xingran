---
slug: vm-list-empty-vdi-server
status: resolved
trigger: 虚拟机列表是空的!请检查原因，该列表应该会连接vdi服务器获取虚拟机信息才对！
created: 2026-05-26T00:00:00Z
updated: 2026-05-26T00:00:01Z
resolution_type: requirement_change
---

## 解决方案

根本原因：VDI服务器未在配置中启用

用户决定：将VDI配置迁移到参数管理页面（数据库动态配置），而非修改配置文件

需要新阶段：VDI配置迁移到参数管理

---

## 症状

### 预期行为
根据用户的数据范围权限决定显示所有虚拟机还是仅显示本部门/本人的虚拟机

### 实际行为
显示"暂无数据"或空状态提示

### 错误信息
没有错误消息

### 时间线
一直都不工作（这是新开发的功能，从未成功获取到数据）

---

## 根本原因

VDI 服务器未启用导致虚拟机列表为空

### 证据

- timestamp: 2026-05-26T00:00:00Z
  source: code_inspection
  finding: |
    检查 `configs/config.yaml` 第 171-186 行，发现 VDI 服务器配置：
    ```yaml
    vdi:
      servers:
        - name: "生产环境"
          endpoint: "https://vdi-prod.example.com"
          username: "admin"
          password: "${VDI_PASSWORD_PROD}"
          tenant_id: 0
          enabled: false
        - name: "测试环境"
          endpoint: "https://vdi-test.example.com"
          username: "admin"
          password: "${VDI_PASSWORD_TEST}"
          tenant_id: 1
          enabled: false
    ```

- timestamp: 2026-05-26T00:00:00Z
  source: code_inspection
  finding: |
    检查 `internal/api/v1/vdi/vm_router.go`：路由在没有启用服务器时跳过注册

- timestamp: 2026-05-26T00:00:00Z
  source: code_inspection
  finding: |
    检查 `internal/services/vdi/vm_service_impl.go`：ListVMs 仅查询本地数据库

---

## 后续行动

使用 `/gsd-plan-phase` 或 `/gsd-quick` 创建新阶段：
- 从 `configs/config.yaml` 删除VDI配置
- 将VDI配置迁移到参数管理页面（`sys_config` 表）
- 修改代码从数据库读取VDI配置
- 更新VDI客户端初始化逻辑
