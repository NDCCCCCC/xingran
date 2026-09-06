// Package constants 集中定义 XingRan-Next 后端跨包共享的常量:
// Redis 缓存键格式、分页参数、缓存 TTL / 设备 / JWT 时间常量、
// 网络命令与 AD 同步超时、端口、URL 协议、并发上限、UUID 校验正则等。
//
// 本包是叶子包,仅依赖标准库(time、regexp),不导入任何内部业务包,
// 因此可被任意层(api / services / core / utils)安全引用而不会引入循环依赖。
//
// 使用约定:
//   - 新增 key 或数值常量时,优先在此定义,而非在各包内联字面量。
//   - 引用方应使用本包导出的常量,避免重复定义导致的取值分叉
//     (参考 stat-cards-from-list-length-capped-at-100 教训)。
//
// 本包是项目唯一的常量权威:原 internal/constants 已于 V130R-09
// 整体合并入本包(D-03-10, Phase 89/90 leaf-const 权威的既定归宿)。
package constants
