---
slug: vdi-vm-creation-validation-error
status: resolved
trigger: 无法创建虚拟机，运行位置已经选择，但还是提示请选择运行位置。cpu颗数等所有滑动选择条不以实际大小作为位置标准，每个选项间的距离平均分配，cpu颗数可选1，2，4，内存可选8，16，32，cpu核数可选4，8，16，磁盘可选，80-500
created: 2026-05-29T10:30:00+08:00
updated: 2026-05-29T10:45:00+08:00
---

# VDI 虚拟机创建验证错误

## 症状

### 预期行为
- 选择运行位置后应该能创建
- 滑块应该有平均分布的刻度

### 实际行为
- 提示"请选择运行位置"（即使已选择）
- 滑块刻度分布不均匀

### 时间线
- 一直存在（从第一次实现功能时就存在）

### 重现步骤
1. 打开创建虚拟机表单
2. 选择所有必填字段
3. 点击确定创建
4. 查看滑块刻度

## Current Focus

hypothesis: 两处bug - (1) 自动选择运行位置时设为undefined导致required验证失败, (2) 滑块marks设置与用户期望不符
next_action: 修复完成
test: TypeScript编译通过，lint无错误
expecting: 表单能正常提交，滑块刻度均匀
reasoning_checkpoint: resolved

## Evidence

- timestamp: 2026-05-29T10:45:00
  type: root_cause
  detail: |
    Bug 1: 第325行和第383行，自动选择运行位置时，当 position.id === position.father_id 时，
    run_position_id 被设为 undefined。但表单字段有 required: true 验证规则，
    导致 validateFields() 抛出"请选择运行位置"错误。
    
    Bug 2: 四个 Slider 组件的 marks 配置与用户需要的离散选项不匹配：
    - CPU颗数：marks={1,4,8,16}，用户需要 {1,2,4}，step应为null（仅允许mark值）
    - 内存：range 512-65536 step=512，用户需要离散值 {8GB,16GB,32GB}
    - CPU核数：marks={1,8,16,32}，用户需要 {4,8,16}
    - 磁盘：min=20，用户需要 min=80

## Eliminated

## Files Referenced

- xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx

## Resolution

root_cause: |
  1. run_position_id 自动选择逻辑在 id===father_id 时设为 undefined，导致 required 验证失败
  2. Slider 组件 marks/step 配置不符合用户需求的离散选项和均匀分布

fix: |
  1. 将自动选择逻辑改为始终设置 run_position_id: firstPosition.id（在 handleCreate 中已有 id===father_id 时清空的逻辑）
  2. 重配置四个 Slider：
     - CPU颗数：min=1 max=4 step=null marks={1,2,4}，initialValue=1
     - 内存：min=8192 max=32768 step=null marks={8GB,16GB,32GB}，initialValue=8192
     - CPU核数：min=4 max=16 step=null marks={4,8,16}，initialValue=4
     - 磁盘：min=80 max=500 step=10 marks={80GB,200GB,350GB,500GB}，initialValue=80

verification: TypeScript编译通过，eslint无错误
files_changed:
  - xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx

## TDD Checkpoint

tdd_mode: false
