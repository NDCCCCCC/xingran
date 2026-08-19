/**
 * 通用状态共享常量（Phase 69 DICT-03）
 *
 * 前端 status 0/1 下拉的单一真相源：消除 user「启用/禁用」/ role「正常/停用」/
 * dict「启用/禁用」三份漂移拷贝。各页 constants 文件以别名 re-export 本模块，
 * 页面组件 import 路径与导出名不变。
 *
 * 对齐方式（前端镜像不引 Go 代码，注释即对齐凭证）：
 * 后端真相源为 internal/models/base.go 等常量定义，由
 * internal/models/status_constants_test.go AST 锁值测试守卫；
 * 本模块由 src/constants/status.test.ts 字面锁定 label 与 value。
 *
 * 安全决策（Phase 69 Q2 / T-69-13）：status 是代码分支语义，不进 sys_dict 字典——
 * 管理员可配的字典值若注入 status 通道，可配出代码未识别的值。
 */

/** 状态下拉选项（自带类型，不反向 import 页面类型，避免循环依赖） */
export interface StatusOption {
  label: string;
  value: number;
}

/** 状态标签配置（与 options 的 value 集合一一对应） */
export type StatusTagConfig = Record<number, { text: string; color: string }>;

// 对齐 models.UserStatusEnabled=0(启用) / UserStatusDisabled=1(禁用), internal/models/base.go
// 消费方: system/user、system/dict
export const ENABLE_DISABLE_OPTIONS: StatusOption[] = [
  { label: "启用", value: 0 },
  { label: "禁用", value: 1 },
];

// 对齐 models.UserStatus: 0=启用(绿), 1=禁用(红), internal/models/base.go
export const ENABLE_DISABLE_TAG_CONFIG: StatusTagConfig = {
  0: { text: "启用", color: "success" },
  1: { text: "禁用", color: "error" },
};

// 对齐 models.RoleStatus / DeptStatus / PostStatus / MenuStatus:
// 0=正常, 1=停用, internal/models/base.go（常量名 Enabled/Normal 混用，值语义一致）
// 消费方: system/role、system/dept、system/menu(status)、operations/floors
export const NORMAL_STOP_OPTIONS: StatusOption[] = [
  { label: "正常", value: 0 },
  { label: "停用", value: 1 },
];

// 对齐 models.RoleStatus / DeptStatus / PostStatus / MenuStatus: 0=正常(绿), 1=停用(红)
export const NORMAL_STOP_TAG_CONFIG: StatusTagConfig = {
  0: { text: "正常", color: "success" },
  1: { text: "停用", color: "error" },
};

// 对齐 models.WorkstationStatus: 0=空闲(可分配), 1=占用(已分配), 2=维护(维修中),
// internal/models/workstation.go —— 工位状态是三态业务簇（非通用启停 0/1），
// 严禁套用 NORMAL_STOP 两态组（会丢掉占用/维护语义）。
// 消费方: operations/workstations
export const WORKSTATION_STATUS_OPTIONS: StatusOption[] = [
  { label: "空闲", value: 0 },
  { label: "占用", value: 1 },
  { label: "维护", value: 2 },
];

// 对齐 models.WorkstationStatus 三态: 0=空闲(绿), 1=占用(红), 2=维护(橙)
export const WORKSTATION_STATUS_TAG_CONFIG: StatusTagConfig = {
  0: { text: "空闲", color: "success" },
  1: { text: "占用", color: "error" },
  2: { text: "维护", color: "warning" },
};
