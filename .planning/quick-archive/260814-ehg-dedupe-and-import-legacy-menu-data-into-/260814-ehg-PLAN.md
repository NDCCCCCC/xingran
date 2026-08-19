---
phase: quick-260814-ehg
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - scripts/tmp_menuimport/main.go
  - scripts/tmp_menuimport/parse.go
  - scripts/tmp_menuimport/dedupe.go
  - scripts/tmp_menuimport/import.go
  - xingran_menus_dedup.sql
  - .planning/quick/260814-ehg-dedupe-and-import-legacy-menu-data-into-/dedupe-report.md
autonomous: true
requirements:
  - DATA-MENU-DEDUPE
  - DATA-MENU-IMPORT-ADMIN

must_haves:
  truths:
    - "解析器全量读出 xingran_menus_clean.sql 的 386 条 sys_menu + 309 条 sys_role_menu + 5 条 sys_role（多行 VALUES、跨行、NULL、'' 转义全部正确处理）"
    - "去重后顶级 M 目录同名唯一（系统监控 ×6 → 1、网络设备 ×3+test ×1 → 1），且每个非顶级菜单的 parent_id 都在保留集内（无悬空）"
    - "被折叠目录的整个子树 parent_id 经 id 映射重定向到规范 id，层级链完整（F→C→M）"
    - "导入幂等（ON CONFLICT DO NOTHING），重复执行安全；不碰 sys_menu/sys_role_menu 以外的表，不导入旧 sys_role/sys_user_role"
    - "导入后 dev 库 admin 角色拥有保留集全部菜单，admin 登录可见所有模块菜单"
  artifacts:
    - path: "scripts/tmp_menuimport/parse.go"
      provides: "多行 VALUES 字符级解析器（引号状态机 + '' 转义 + 深度0 tuple 切分）"
      contains: "func parseInserts"
    - path: "scripts/tmp_menuimport/dedupe.go"
      provides: "去重引擎（R0-R5 规则）+ id 映射 + 不变量校验"
      contains: "idMap"
    - path: "xingran_menus_dedup.sql"
      provides: "去重后的 sys_menu 幂等 INSERT（parent_id 已重定向）"
      contains: "ON CONFLICT DO NOTHING"
    - path: ".planning/quick/260814-ehg-dedupe-and-import-legacy-menu-data-into-/dedupe-report.md"
      provides: "去重映射说明（保留/折叠 id 及理由）"
      contains: "折叠"
  key_links:
    - from: "scripts/tmp_menuimport/dedupe.go"
      to: "xingran_menus_dedup.sql"
      via: "保留集 + idMap 重定向 parent_id 后生成 INSERT"
      pattern: "ON CONFLICT DO NOTHING"
    - from: "scripts/tmp_menuimport/import.go"
      to: "dev 库 sys_role_menu"
      via: "保留集全量菜单 × dev admin role_id 幂等插入"
      pattern: "sys_role_menu"
---

<objective>
把生产库导出的菜单参考数据（`xingran_menus_clean.sql`，386 条 sys_menu + 309 条 sys_role_menu）去重后导入 dev 库，保持层级关系，并把保留集全量授予 dev 现有 admin 角色，让 admin 看到所有模块菜单。

核心难点是去重：旧库顶级目录存在重复（「系统监控」×6、「网络设备」×3、「网络设备管理test」×1、「运维管理」内部「值班池管理」×4 等），合并重复目录时必须把被折叠目录整棵子树的 `parent_id` 重定向到保留的规范 id，并同步修正 `sys_role_menu.menu_id`。

Purpose: dev 库当前只有 36 条菜单（仅运维管理一个目录），admin 看不到系统监控/网络设备/资产管理等模块。导入去重后的全量菜单恢复完整导航。
Output: `xingran_menus_dedup.sql`（干净幂等 SQL）+ `dedupe-report.md`（去重映射说明）+ dev 库导入与验证结果。临时 Go 脚本（`scripts/tmp_menuimport/`）用完即删。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@D:/code/ClaudeCode/guoguo/CLAUDE.md
@D:/code/ClaudeCode/guoguo/.planning/STATE.md
@D:/code/ClaudeCode/guoguo/internal/models/menu.go
@D:/code/ClaudeCode/guoguo/internal/models/role.go

<interfaces>
<!-- 输入文件事实（planner 已核实，executor 无需重新调查） -->

xingran_menus_clean.sql（项目根，712 行）INSERT 列清单：
- sys_menu（386 条，19 列）：("id","created_at","updated_at","deleted_at","created_by","updated_by","version","menu_name","parent_id","order_num","path","component","menu_type","visible","status","perms","icon","remark","meta")
- sys_role（5 条，15 列）：("id","created_at","updated_at","deleted_at","created_by","updated_by","version","role_name","role_key","role_sort","data_scope","menu_check_strictly","dept_check_strictly","status","remark")
- sys_role_menu（309 条，4 列）：("id","role_id","menu_id","created_at")

INSERT 为多行 VALUES 格式：
  INSERT INTO "sys_menu" (...) VALUES
  \t('uuid', '2026-06-25 ...', ..., NULL),
  \t('uuid', ...)
  ON CONFLICT DO NOTHING;

解析陷阱（必须字符级状态机，禁止用行正则切 tuple）：
- remark/meta 字符串内含括号与逗号（如 'Phase 39 工位部门映射管理权限 (D-04: 不自动授权任何角色)'），按 `),` 简单切分会错切
- SQL 字符串内单引号转义为 ''（两个单引号）
- NULL 是无引号字面量；'' 是空字符串（≠ NULL）
- deleted_at 大多 NULL，但部分行为时间戳（软删行，如 网络设备空壳 f5c087d6/95f849d7）——解析必须保留 deleted_at 字段供去重规则使用

dev 库连接（.env，事务 pooler，必须 sslmode=require）：
- DB_HOST=aws-0-ap-southeast-1.pooler.supabase.com, DB_PORT=5432
- DB_USER=postgres.ovgfhrphadkvdkareigj, DB_NAME=postgres
- DB_PASSWORD 从环境变量读取（os.Getenv），禁止硬编码
- driver: github.com/lib/pq（go.mod 已有 v1.10.9）；建议 connect_timeout=15

dev 库现状（已核实）：
- sys_menu 仅 36 条 = 1 M（运维管理）+ 7 C + 28 F（ops:* query/add/edit/delete）
- 现有角色：admin（超级管理员）、user（普通用户）；admin 已分配全部 36 条
- 临时脚本放 scripts/tmp_menuimport/（独立目录独立 package main，避免根目录 temp_*.go 的 main redeclared 问题）；用完整个目录删除
</interfaces>
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: 字符级 SQL 解析器（全量 386+309+5 行）</name>
  <files>scripts/tmp_menuimport/parse.go, scripts/tmp_menuimport/main.go</files>
  <behavior>
    - 解析 xingran_menus_clean.sql 后菜单数必须恰为 386、sys_role_menu 恰为 309、sys_role 恰为 5，否则 exit 1
    - 含括号/逗号的 remark（如 '(D-04: ...)'）被正确保留在字段内，不会错切 tuple
    - NULL → nil，'' → 空字符串指针，'O''Brien' 风格 '' 转义 → 单引号
    - parent_id 为 NULL（顶级目录）与为 uuid 两种情况都正确
  </behavior>
  <action>创建 scripts/tmp_menuimport/（package main，独立目录避免 main redeclared）。parse.go 实现 parseInserts(sql string) map[string][][]*string：按 INSERT 块定位（以 `INSERT INTO "表名"` 开头到 `ON CONFLICT` 为止），随后用字符级状态机扫描——维护 inQuote 布尔（' 进入，' 退出，遇到 '' 视为转义跳过）与括号深度；仅在深度 0 且非引号内的 `),(` 边界切 tuple，在深度 1 且非引号内的逗号切字段；字段去首尾空白后，NULL→nil，其余去掉外层单引号并把 '' 还原为 '。main.go 提供 -mode=parse：读文件、调用 parseInserts、断言 sys_menu=386 / sys_role_menu=309 / sys_role=5，打印每表行数与抽样（前 3 个顶级 M 目录名）。把行解析为结构体 MenuRow{ID, DeletedAt *string, MenuName, ParentID *string, OrderNum, Path, Component, MenuType, Perms, Icon, Remark ...}（按上方 19 列顺序取字段，deleted_at 是索引 3、menu_name 7、parent_id 8、order_num 9、path 10、component 11、menu_type 12、perms 15）。RoleMenuRow{ID, RoleID, MenuID}。</action>
  <verify>
    <automated>cd D:/code/ClaudeCode/guoguo && go build ./scripts/tmp_menuimport/ && go run ./scripts/tmp_menuimport -mode=parse 输出含 "sys_menu=386 sys_role_menu=309 sys_role=5" 且 exit 0</automated>
  </verify>
  <done>解析器全量读出 386+309+5 行，计数断言通过，go build 无错误</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: 去重引擎（R0-R5 规则）+ 生成干净 SQL + 映射报告</name>
  <files>scripts/tmp_menuimport/dedupe.go, scripts/tmp_menuimport/main.go, xingran_menus_dedup.sql, .planning/quick/260814-ehg-dedupe-and-import-legacy-menu-data-into-/dedupe-report.md</files>
  <behavior>
    - 去重后顶级 M 目录分组键唯一：系统监控 ×6 → 1、网络设备 ×3 → 1（test 版 5cd243d3 经 R1 归并）
    - 保留集中每个非顶级行的 parent_id（经 idMap 闭包重定向后）都在保留集 id 集合内，否则 exit 1
    - 同 (parent_id, menu_name, menu_type, perms) 的重复行只保留 1 行
    - 生成的 SQL 幂等（每条 INSERT 带 ON CONFLICT DO NOTHING），可重复执行
  </behavior>
  <action>在 dedupe.go 实现去重引擎，规则必须机械可解释（写进代码注释与报告）：
R0 软删过滤：deleted_at != nil 的行默认不进保留集；例外——若某软删 M/C 是保留行的祖先且其分组内无存活等价目录，则复活该行（生成 SQL 时 deleted_at 写 NULL）并在报告标注。
R1 顶级目录合并：对 parent_id==nil 的 M 按分组键分组，分组键 = TrimSuffix(menu_name, "test")（使「网络设备管理test」与「网络设备」同组）。每组选规范 id，优先级依次：① deleted_at IS NULL ② 子树节点数最多 ③ path 非空 ④ created_at 最早。其余折叠进 idMap。
R2 重定向：idMap 求传递闭包（折叠 id 的父若也被折叠继续追），所有保留行的 parent_id 经闭包映射。
R3 同级内容去重（fixpoint）：按 (映射后parent_id, menu_name, menu_type, COALESCE(perms,'')) 分组保留 1 行（优先级同 R1），折叠其余进 idMap；重复 R2-R3 直到不动点。
R4 sys_role_menu：解析后 menu_id 经 idMap 映射并去重 (role_id, menu_id)，仅用于报告（哪 309 条映射后剩多少、引用折叠 id 的有多少），不生成导入 SQL——旧 role id 在 dev 不存在。
R5 sys_role 不导入（dev 复用现有 admin/user）。
不变量校验（代码内断言，违反即 exit 1）：① 保留集顶级分组键无重复 ② 每个非顶级保留行 parent_id ∈ 保留集 ③ 保留集内无 (parent_id,menu_name,menu_type,perms) 重复 ④ idMap 闭包无环。
main.go 增加 -mode=gen：执行去重，生成 xingran_menus_dedup.sql（sys_menu INSERT，列顺序与输入一致，字符串字段单引号转义 ''，nil→NULL，每条 INSERT ON CONFLICT DO NOTHING，按 order_num 排序便于阅读），并写 dedupe-report.md：每个折叠组的【规范 id（保留理由）← 折叠 id 列表】、复活行清单、R3 去重统计、最终保留行数。perms 字符串原样保留不改（xxx:list vs xxx:query 版本对齐不在本次范围）。</action>
  <verify>
    <automated>cd D:/code/ClaudeCode/guoguo && go run ./scripts/tmp_menuimport -mode=gen 输出 "INVARIANTS PASS" 且 exit 0，且 test -f xingran_menus_dedup.sql && grep -c "ON CONFLICT DO NOTHING" xingran_menus_dedup.sql 大于 0</automated>
  </verify>
  <done>xingran_menus_dedup.sql 生成且全部不变量断言通过；dedupe-report.md 含每个折叠组的保留/折叠 id 及理由；顶级目录同名唯一</done>
</task>

<task type="auto">
  <name>Task 3: 导入 dev 库 + admin 全量授权 + 复核 + 清理临时脚本</name>
  <files>scripts/tmp_menuimport/import.go, scripts/tmp_menuimport/main.go</files>
  <action>import.go 实现 -mode=import，分四步，全部用 database/sql + lib/pq，连接串从环境变量组装（host=%s port=%s user=%s password=%s dbname=%s sslmode=require connect_timeout=15，password 必须 os.Getenv("DB_PASSWORD")，缺失则 exit 1）。运行前先 `set -a; source .env; set +a`（Git Bash）加载环境变量。
步骤 1 预检：SELECT count(*) FROM sys_menu WHERE deleted_at IS NULL（预期 36）；SELECT id FROM sys_role WHERE role_key='admin'（取 adminRoleID，查不到则 exit 1）。
步骤 2 导入：读 xingran_menus_dedup.sql 整块执行（单事务，失败回滚）。执行前打印"将 upsert X 条菜单"。
步骤 3 授权：对保留集每个菜单 id 执行 INSERT INTO sys_role_menu (id, role_id, menu_id, created_at) VALUES (gen_random_uuid(), $1, $2, now()) ON CONFLICT DO NOTHING——先查 information_schema 确认 sys_role_menu 的 id 列默认值/唯一约束，若唯一约束在 (role_id,menu_id) 则 ON CONFLICT (role_id,menu_id) DO NOTHING，否则 ON CONFLICT DO NOTHING 并先 SELECT 跳过已存在对（保证幂等）。不导 sys_user_role。打印"为 admin 补 Y 条角色菜单"。
步骤 4 复核（全部打印，任一失败 exit 1）：① sys_menu 总数（应 = 36 + 新增数，且与保留集 ∪ 既有 36 的并集一致——注意保留集中可能含与既有 36 条同 id 的行，ON CONFLICT 跳过属预期）② 顶级 M 目录按 menu_name 分组无 count>1 ③ 无悬空 parent_id：SELECT count(*) FROM sys_menu m WHERE m.parent_id IS NOT NULL AND m.deleted_at IS NULL AND NOT EXISTS (SELECT 1 FROM sys_menu p WHERE p.id = m.parent_id) 必须 = 0 ④ admin 角色菜单总数 ⑤ 打印最终目录树（递归：顶级 M 按 order_num，每级缩进打印 menu_name(menu_type)，限两层 + 每层子节点计数）。
步骤 5 清理：复核 PASS 后删除整个 scripts/tmp_menuimport/ 目录（rm -rf），运行 go build ./... 确认无残留编译错误。若导入或复核失败，保留脚本目录供调试，在 SUMMARY 记录原因。</action>
  <verify>
    <automated>cd D:/code/ClaudeCode/guoguo && set -a && source .env && set +a && go run ./scripts/tmp_menuimport -mode=import 输出含 "VERIFY PASS" 且 exit 0；清理后 go build ./... exit 0</automated>
  </verify>
  <done>dev 库 sys_menu 含去重后全量菜单；顶级目录无重名；无悬空 parent_id；admin 角色拥有保留集全部菜单；scripts/tmp_menuimport/ 已删除且 go build ./... 通过</done>
</task>

</tasks>

<verification>
- `go run ./scripts/tmp_menuimport -mode=parse`：386/309/5 计数断言通过
- `go run ./scripts/tmp_menuimport -mode=gen`：不变量全过，生成 xingran_menus_dedup.sql + dedupe-report.md
- `go run ./scripts/tmp_menuimport -mode=import`：预检 → 事务导入 → admin 授权 → 五项复核全 PASS
- 清理后 `go build ./...` exit 0（无 temp 脚本残留）
- 重复执行 -mode=import 幂等（ON CONFLICT DO NOTHING，第二次运行新增数为 0）
</verification>

<success_criteria>
- dev 库 sys_menu 从 36 条扩到去重后全量（预期 300+ 条，准确数以 dedupe-report 为准）
- 顶级 M 目录每个模块唯一（系统监控/网络设备各 1 份，无 test 残留目录）
- 层级完整：无悬空 parent_id，F→C→M 链不断
- admin 角色菜单数 = 既有 36 ∪ 保留集（全量）
- 产出物齐备：xingran_menus_dedup.sql + dedupe-report.md；临时脚本已删除
</success_criteria>

<output>
创建 `.planning/quick/260814-ehg-dedupe-and-import-legacy-menu-data-into-/260814-ehg-SUMMARY.md`，必须包含：去重规则落地情况（R0-R5 每组保留/折叠 id 及理由，引用 dedupe-report.md）、导入前后计数、五项复核结果、最终目录树概览、遗留事项（perms 版本差异 xxx:list vs xxx:query 未对齐——仅记录不处理）。
</output>
