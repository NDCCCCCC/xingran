/**
 * API 工厂派生类型（Phase 94 D-02）
 *
 * 本文件为纯 type-only 模块：零 import、零运行时代码（对齐 base.ts 文件形态）。
 * 消费方:src/lib/apiFactory.ts 的 create/update 参数签名。
 */

/**
 * 服务端生成字段集合 — 客户端 create/update 误传即编译期报错（D-02）。
 *
 * 双命名并集依据：
 * - camelCase 四键:ops/knowledge/duty/rpa 域实体（id/createdAt/updatedAt/deletedAt）
 * - snake_case 三键:vdi 域实体实测使用 created_at/updated_at（VirtualMachine/VDIServer/VMAccount），
 *   camelCase-only 排除集对它们无效（Omit 对不存在的键无害,并集即安全）
 * - createdBy/updatedBy:审计人字段,后端从 JWT 落库,前端传值无效
 */
export type ServerGeneratedKeys =
  | "id"
  | "createdAt"
  | "updatedAt"
  | "deletedAt"
  | "created_at"
  | "updated_at"
  | "deleted_at"
  | "createdBy"
  | "updatedBy";

/**
 * 创建/更新载荷 — 排除服务端生成字段后的实体派生类型。
 *
 * 工厂 create/update 参数使用 Partial<CreatePayload<T>>:对象字面量误传
 * id/时间戳触发 excess property check 编译期报错;业务字段必选性原样保留,
 * 不排除任何业务字段（orgId/buildingId/status/memberIds 等）。
 */
export type CreatePayload<T> = Omit<T, ServerGeneratedKeys>;
