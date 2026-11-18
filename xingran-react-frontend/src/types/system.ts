/**
 * 系统管理相关类型
 */

import type { Status, PageParams } from "./base";

// ==================== 用户相关 ====================

/**
 * 用户类型
 */
export interface User {
  id: string;
  username: string;
  nickname?: string;
  employeeNo?: string;
  email?: string;
  phone?: string;
  /** 手机号（后端 models.User.Mobile，AD 同步写入） */
  mobile?: string;
  /** 职位（后端 models.User.Title，AD 同步写入） */
  title?: string;
  avatar?: string;
  gender: 0 | 1 | 2;
  status: Status;
  deptId?: string;
  deptName?: string;
  /**
   * 岗位任职（多选，后端 sys_user_post 关联）。
   * 列表接口目前未必回填，编辑时可能为空 —— 表单以此字段/`postIds` 回显。
   */
  posts?: Post[];
  /** 岗位 ID 集合（创建/更新请求字段，也用于编辑回显兜底） */
  postIds?: string[];
  deptFullName?: string; // 完整部门路径（从二级开始）
  roles: string[];
  roleIds?: string[];
  permissions: string[];
  isAdmin?: boolean;
  dataScope?: string;
  loginIp?: string;
  loginTime?: string;
  createTime: string;
  updateTime: string;
}

/**
 * 用户列表查询参数
 */
export interface UserListParams extends PageParams {
  username?: string;
  nickname?: string;
  deptId?: string;
  status?: number;
  dateRange?: [string, string];
}

// ==================== 角色相关 ====================

/**
 * 数据范围
 */
export type DataScope = 1 | 2 | 3 | 4 | 5;

/**
 * 机构锚点模式（`sys_role.anchor_mode` / `sys_post.anchor_mode`）
 *
 * 只实现 4 个值：0=不参与、2=指定部门（含子树）、3=最近外部机构、5=最近机构层级。
 * 1（相对层深）与 4（sys_manager_org）已弃用，**后端收到即拒绝**，前端同样不得发出。
 *
 * ⚠️ 与 `DataScope` 是**正交**的两个维度：绝不参与 `MAX(data_scope)` 排序，
 * 也不得与数据范围合并成同一个下拉（契约 §五 / C1）。
 */
export type AnchorMode = 0 | 2 | 3 | 5;

/**
 * 机构层级（`sys_dept.org_level`）
 *
 * 0=非机构（普通部门/科室，默认），1=分公司，2=中支，3=支公司，4=网点（预留）。
 */
export type OrgLevel = 0 | 1 | 2 | 3 | 4;

/**
 * 角色类型
 */
export interface Role {
  id: string;
  roleName: string;
  roleKey: string;
  roleSort: number;
  dataScope: DataScope;
  /** 机构锚点模式（与 dataScope 正交）；缺省视为 0=不参与 */
  anchorMode?: AnchorMode;
  /** 锚点参数：mode=2 时为 dept_id(UUID)，mode=5 时为层级 "1"~"4"，其余为空 */
  anchorParam?: string;
  menuCheckStrictly: boolean;
  deptCheckStrictly: boolean;
  status: Status;
  menuIds: string[];
  deptIds?: string[];
  createTime: string;
  updateTime: string;
  remark?: string;
}

// ==================== 菜单相关 ====================

/**
 * 菜单类型：M=目录，C=菜单，F=按钮
 */
export type MenuType = "M" | "C" | "F";

/**
 * 菜单类型
 */
export interface Menu {
  id: string;
  menuName: string;
  parentId?: string | null;
  orderNum: number;
  path?: string | null;
  component?: string | null;
  menuType: MenuType;
  visible: Status;
  status: Status;
  perms?: string | null;
  icon?: string | null;
  remark?: string;
  meta?: import("./menu").RouteMeta | null;
  children?: Menu[];
  createTime: string;
  updateTime: string;
}

// ==================== 部门相关 ====================

/**
 * 部门类型
 */
export interface Department {
  id: string;
  deptName: string;
  deptCode: string;
  parentId?: string;
  ancestors: string;
  orderNum: number;
  leader?: string;
  leaderName?: string;
  leaderUsername?: string;
  phone?: string;
  email?: string;
  isExternalOrg?: Status;
  /** 机构层级（0=非机构 1=分公司 2=中支 3=支公司 4=网点）；缺省视为 0 */
  orgLevel?: OrgLevel;
  status: Status;
  children?: Department[];
  createdAt: string;
  updatedAt: string;
  remark?: string;
  accessible?: boolean;
}

// ==================== 岗位相关 ====================

/**
 * 岗位类型
 */
export interface Post {
  id: string;
  postCode: string;
  postName: string;
  postSort: number;
  status: Status;
  /** 机构锚点模式（与角色数据范围正交）；缺省视为 0=不参与 */
  anchorMode?: AnchorMode;
  /** 锚点参数：mode=2 时为 dept_id(UUID)，mode=5 时为层级 "1"~"4"，其余为空 */
  anchorParam?: string;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

// ==================== 机构锚点推导预览 ====================

/**
 * 预览中单个任职人的锚点解析结果。
 *
 * `resolved=true` 表示该任职人按其部门链命中了锚点，可获得锚点机构子树范围；
 * `resolved=false` 时 `reason` 说明未命中原因（这些人不会获得机构范围）。
 */
export interface AnchorPreviewItem {
  userId: string;
  username: string;
  nickname?: string;
  resolved: boolean;
  /** 命中时解析出的锚点部门 ID（未命中为空） */
  anchorDeptId?: string;
  /** 未命中原因（命中时可能为空） */
  reason?: string;
}

/**
 * 机构锚点推导预览结果（`POST /system/roles/:id/anchor-preview`、
 * `POST /system/posts/:id/anchor-preview`，契约 §六）。
 *
 * 后端可能以两种形态返回未命中项：完整 `items` 数组（含 resolved 标记），
 * 或只含未命中项的 `unresolved` 数组。前端一律优先用 `items` 自行过滤
 * `!resolved`；`unresolved` 仅作兜底。
 */
export interface AnchorPreviewResult {
  /** 当前角色/岗位生效的锚点模式 */
  mode: number;
  /** 当前锚点参数（mode=2 为 dept_id，mode=5 为层级） */
  param?: string;
  /** 已解析任职人数 */
  resolvedCount: number;
  /** 未解析任职人数 */
  unresolvedCount: number;
  /** 该角色/岗位全部任职人数 */
  totalCount: number;
  /** 逐个任职人的解析明细（优先数据源） */
  items?: AnchorPreviewItem[];
  /** 兜底形态：只含未命中项 */
  unresolved?: AnchorPreviewItem[];
  /** 可选告警（如岗位在目标机构内存在多名任职人） */
  warnings?: string[];
}

/** 编辑用户时"权限预览"中的单个锚点来源解析明细 */
export interface UserAnchorSource {
  source: "role" | "post";
  sourceName: string;
  mode: number;
  param?: string;
  modeLabel: string;
  resolved: boolean;
  anchorDeptId?: string;
  anchorDeptName?: string;
  reason?: string;
  deptCount?: number;
}

/** 编辑用户时"权限预览"的完整响应（菜单权限 + 数据范围画像） */
export interface UserPermissionPreview {
  /** 角色维度生效档位（MAX data_scope） */
  dataScope: number;
  dataScopeLabel: string;
  /** 是否全库（All 档优先，锚点不参与） */
  all: boolean;
  /** 机构锚点来源解析明细 */
  anchorSources: UserAnchorSource[];
  /** 最终可见部门 ID 集（All 档为空） */
  visibleDeptIds: string[];
  /** 可见部门名称（与 visibleDeptIds 顺序对应） */
  visibleDeptNames: string[];
  /** 解释性说明 */
  notes: string[];
  /** 选中角色汇总的菜单/按钮权限码 */
  permissions: string[];
  /** 菜单树勾选集（menu id），供 <Tree checkable> 渲染 */
  menuIds: string[];
  /** 部门树勾选集（可见部门 id），供 <Tree checkable> 渲染 */
  deptIds: string[];
  /** 是否全库可见（All 档）——部门树渲染为全选提示 */
  allDepts: boolean;
  /** 选中的角色名 */
  roleNames: string[];
  /** 选中的岗位名 */
  postNames: string[];
}

// ==================== 字典相关 ====================

/**
 * 字典类型
 */
export interface DictType {
  id: string;
  dictName: string;
  dictType: string;
  status: Status;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * 字典数据
 */
export interface DictData {
  id: string;
  dictSort: number;
  dictLabel: string;
  dictValue: string;
  dictType: string;
  cssClass?: string;
  listClass?: string;
  isDefault: boolean;
  status: Status;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

// ==================== 配置相关 ====================

/**
 * 配置类型：Y=是，N=否
 */
export type ConfigType = "Y" | "N";

/**
 * 参数配置
 */
export interface Config {
  id: string;
  configName: string;
  configKey: string;
  configValue: string;
  configType: ConfigType;
  isSystem: Status;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

// ==================== 通知公告 ====================

/**
 * 通知类型：1=公告，2=警告
 */
export type NoticeType = "1" | "2";

/**
 * 通知公告
 */
export interface Notice {
  id: string;
  noticeTitle: string;
  noticeType: NoticeType;
  noticeContent?: string;
  status: Status;
  remark?: string;
  createdAt: string;
  updatedAt: string;
}

// ==================== 日志相关 ====================

/**
 * 操作日志
 */
export interface OperLog {
  id: string;
  title?: string;
  businessType: number;
  method?: string;
  requestMethod?: string;
  operatorType: number;
  operatorName?: string;
  deptName?: string;
  operUrl?: string;
  operIp?: string;
  operLocation?: string;
  operParam?: string;
  jsonResult?: string;
  status: Status;
  errorMsg?: string;
  operTime: string;
  costTime: number;
}

/**
 * 登录日志
 */
export interface LoginLog {
  id: string;
  username?: string;
  ipaddr?: string;
  loginLocation?: string;
  browser?: string;
  os?: string;
  status: Status;
  msg?: string;
  loginTime: string;
}
