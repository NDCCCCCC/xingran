# 系统管理域前端组件审查报告（system / profile / login / settings / my-notices）

> 审查对象：\`ruoyi-react-frontend/src/pages/system/\`（114 文件，重点 75 个非测试 ts/tsx、16283 行）+ \`pages/profile\` \`pages/login\` \`pages/settings\` \`pages/my-notices\`（15 文件）+ 本域实际使用的共享组件与 hooks。
> 审查方式：只读。逐文件读源码 + Node 解析 JSX opening tag/属性做量化（明细 \`.planning/notes/frontend-audit/_perfile.json\`），再回到源码逐一核对行号。
> 维度框架：功能完整性 / 数据展示完整性 / 交互反馈 / 表单校验 / 状态与分页 / 错误与空态 / 样式与主题一致性 / 可访问性与国际化 / 性能与副作用 / 代码复用与可维护性。第 3 章每条问题均以 \`[维度名]\` 开头。
> 结论先行：本域**不缺能力，缺统一出口**。\`usePagination\`、\`useTableManager\`、\`useServerSort\`、\`ActionButtons\`、\`BaseEditModal\`、\`EmptyStateWithAction\`、\`ErrorAlertWithRetry\` 都已存在，但只有 \`post\` 一页算是完整样板；其余 13 个列表页各自手写搜索区、分页、弹窗、确认框、错误处理，导致用户一眼可见的缺口（**分页看不到总条数、输入框没有一键清除、保存无反馈/可重复提交**）。

## 1. 审查范围与方法

| 范围 | 文件数 | 说明 |
| --- | --- | --- |
| \`pages/system/{user,role,menu,dept,dict,config,post,notice,apikeys,manager-orgs,manager-rules,captcha-background}\` | 约 100 | 逐个读主页面 + hooks/columns/constants/modals/components |
| \`pages/system/settings\`（email/api/captcha/default-theme） | 5 | 设置类表单页，重点看弹窗、分页参数、状态语义 |
| \`pages/profile\` \`pages/login\` \`pages/my-notices\` \`pages/settings-page\` | 15 | 个人中心 / 登录 / 我的通知 / 已废弃设置壳 |
| 共享层（只读参考） | 10 | \`hooks/usePagination.ts\`、\`hooks/useTableManager.ts\`、\`hooks/useServerSort.ts\`、\`components/shared/*\`、\`components/modal/BaseEditModal.tsx\`、\`utils/errorHandler.ts\` |

方法：

1. 对 75 个非测试文件解析 JSX 开标签及属性，统计 \`allowClear / onPressEnter / showTotal / confirmLoading / destroyOnHidden / pagination / scroll / rowKey / fixed / sorter\` 等 20+ 属性，落盘 \`_perfile.json\`（含 flagIndex 反向索引）。
2. 正则全片扫描三类硬编码：hex 颜色、状态 \`0/1\` 裸三元、手写日期格式，逐条回源码核实行号与语义（结果见 §3.10 汇总）。
3. 对候选问题逐条回到源码确认行号与上下文，剔除误报（如 \`onSearch\` 出现在真正带 \`showSearch\` 的角色多选上）。
4. 与共享层实现对照，判断"是缺组件"还是"有组件没用"，再评估抽取必要性（§4）。

## 2. 问题总览

### 2.1 数量统计（本域实测，非全项目）

| 指标 | 本域实测 | 说明 |
| --- | --- | --- |
| 列表/表格页 | 14 | user、role、menu、dept、dict(×2 表)、config、post、notice、apikeys、apikeys 日志、manager-orgs、manager-rules、captcha-background、api-config、email-config、my-notices（含同页多表） |
| \`<Table>\` | 17 | 全部有 \`pagination\` 属性与 \`rowKey\`；2 个树表 \`pagination={false}\` |
| 分页显示"共 N 条"的表 | **5/17** | 全部是手写 \`showTotal\`：apikeys、LogsModal、manager-orgs、manager-rules、NoticeList |
| 分页只显示"共 N 页"的表 | 11/17 | 走 \`usePagination\` 的 \`paginationProps\`，**无一处输出总条数** |
| 横向滚动 \`scroll.x\` | **5/17** | apikeys:697(1400)、LogsModal:438(1000)、NoticeList:370(1600)、api-config:376(1200)、email-config:393(1200)；其余 12 表无横向滚动 |
| \`fixed:"right"\` 操作列 | 10 | 其中 6 处落在无 \`scroll.x\` 的表上（fixed 不生效/错位）：user:272、config:291、dict:129/231、captcha-background:96 |
| 服务端排序接入 | 8 页 / 11 表 | user、role、post、config、dict×2、notice、apikeys；未接入 9 表（dept、menu、manager-orgs、manager-rules、captcha-background、api-config、email-config、my-notices、LogsModal） |
| 本地排序（与服务端混用） | 1 | \`captcha-background/columns.tsx:77\` \`sorter: (a,b)=>a.useCount-b.useCount\`，只排当前页 |
| 搜索输入框（本域） | 32 | 有 \`allowClear\` 13；有 \`onPressEnter\` **仅 4**（apikeys:635、manager-orgs:290、manager-rules:339、settings/captcha-background:338） |
| 表单输入框（弹窗内） | 48 | 有 \`allowClear\` 仅 3（几乎全部缺"一键清除"） |
| \`<Select>\` | 69 | \`allowClear\` 24；\`onSearch={()=>{}}\` 空实现且未开 \`showSearch\` 有 24+ 处（死代码） |
| \`<Modal>\`（非 BaseEditModal） | 22 | \`destroyOnHidden\` 7、\`confirmLoading\` 7、无防连点 15 |
| 危险操作确认 | 16 处 \`Modal.confirm\` + 3 处 \`Popconfirm\` | 但 6 个页面的批量删除**完全没有确认** |
| 表格文本省略 | 19 处 / 8 文件 | 无统一列配置，长文本列换行或撑破布局 |
| 校验规则块 | 84 | 其中 **15 处无 \`message\`**（用户看到 antd 默认英文提示或无提示） |
| 静默失败（catch 仅 console.error） | 11 处 / 8 文件 | 含 menu 保存、dict 两个列表、dept 删除、role 三个加载 |
| \`message.*\` / \`App.useApp\` | 116 处 / 20 文件、43 处 / 20 文件 | 另有 \`getAppMessage()\` 第三套出口 |
| \`EmptyStateWithAction\` / \`ErrorAlertWithRetry\` / \`Skeleton\` | 0 / 0 / 0 | 组件存在但本域零使用 |
| \`BaseEditModal\` | 1/14 页 | 仅 post 使用 |
| \`useColumnConfig\` / \`useTableQuery\` / \`ColumnConfigModal\` | 0 / 0 / 0 | 已有能力未接入 |
| 硬编码 hex 颜色 | **62 处** | 双写兜底 \`var(--theme-x, #hex)\` 约 57 处 + 裸 hex 5 处（§3.10） |
| 状态 \`0/1\` 裸三元 | **20 处** | 且存在 1 处语义反转（captcha 1=启用） |
| 手写日期格式 | **3 处** | notice×2（含手拼 UTC 偏移）、apikeys×1 |
| 可访问性标注 | aria 7 处 / icon-only 按钮 51 处 | 51 个纯图标按钮中仅 1 个有 \`aria-label\`（user:737） |
| 国际化 | 0 | 无 \`useTranslation\` / \`FormattedMessage\`，文案全部硬编码中文 |

### 2.2 严重度分布（第 3 章共 68 条，均带 `[维度名]` 标签）

- **P0（用户直接可见的功能缺失/数据风险）6 条**：分页永不显示总条数（11 表）、空列表显示"共 1 页"、7 处批量删除无二次确认、13 个列表页搜索框无回车搜索、搜索/表单输入框普遍无一键清除、菜单保存失败完全静默。
- **P1（体验/一致性/可维护性）29 条**：树表无总数、12 表无 scroll 与 6 处失效 fixed 列、服务端排序覆盖不足且混入本地排序、弹窗无 confirmLoading/防连点、Switch 无 loading/回滚、状态标签三套、captcha 状态语义反转、分页参数 page/current 混用与双 onChange、错误处理三套并存、空态/错误态未接入、静默 case、校验规则无 message、hex/品牌色散落、按钮顺序与文案不统一。
- **P2（代码卫生/体验细节）33 条**：行内样式与尺寸硬编码、icon 按钮无 aria-label、无 i18n、无防抖/折叠搜索、树表全展开性能、\`any\`、死文件 \`settings-page\`、\`onSearch\` 死代码、统计卡与 processTreeData 复制、手写日期序列化。

### 2.3 Top5 最严重问题

1. **分页永远看不到"共 N 条"**：11 个列表页统一用 \`usePagination\`（\`src/hooks/usePagination.ts:122\`），\`showTotal\` 被实现成 \`共 ${Math.max(1, Math.ceil(t / pageSize))} 页\`——用户诉求的"总数量"在组件层就被丢掉；而 5 个手写 \`showTotal\` 的页面反而显示"共 N 条"。同一系统两套分页语义，且默认那套是错的（详见 §3.2/§3.5）。
2. **表格横向展示不完整**：17 个表只有 5 个配了 \`scroll.x\`，列多的 user/role/config/dict/captcha-background 等页在窄屏下挤压换行；同时 6 处操作列声明了 \`fixed:"right"\` 却没有 \`scroll.x\`（antd 下不生效），"操作列固定"只写在代码里，没体现在界面上（§3.2）。
3. **批量删除无确认**：\`user\`、\`config\`、\`post\`、\`dict\`（类型+数据）、\`role\`、\`notice\` 共 7 处 \`batch-delete\` 直接发请求（如 \`pages/system/user/index.tsx:544-558\`、\`config/index.tsx:189-203\`、\`post/index.tsx:156-169\`、\`dict/hooks/useDictActions.ts:114-133\` 与 \`180-199\`、\`role/hooks/useRoleActions.ts:132-149\`、\`notice/hooks/useNoticeData.ts:99-109\`）；而单条删除反而几乎都确认，风险倒挂。
4. **菜单保存失败完全静默**：\`pages/system/menu/hooks/useMenuActions.tsx:315-317\` catch 里只有 \`console.error("保存菜单失败:", error)\`，校验失败与接口失败都不关弹窗、不提示，用户表现为"点保存没反应"。
5. **输入控件默认能力缺失**：全片 32 个搜索框仅 4 个支持回车、13 个有清除按钮；48 个弹窗表单输入框只有 3 个有 \`allowClear\`；69 个 Select 里 24+ 处写了永不触发的 \`onSearch={() => {}}\`。用户"输完发现没清除键、回车没反应、下拉不能搜"的直观感受即来源于此（§3.1/§3.7）。

## 3. 十维度问题清单

格式：\`[维度名] 文件:行号 —— 现象 —— 级别 —— 修法\`。所有行号均为实读核对；同类问题合并成一条并列出代表位置，避免刷条数。

### 3.1 功能完整性

\`\`\`
[功能完整性] pages/system/user/index.tsx:715、role/index.tsx:279/:282、menu/index.tsx:280 等 13 个列表页搜索框 —— 输入后回车无反应，必须点击"搜索"按钮；全片仅 apikeys:635、manager-orgs:290、manager-rules:339、settings/captcha-background:338 支持 onPressEnter —— P0 —— 搜索区统一绑定 onPressEnter（SearchToolbar/FormInput 默认），textarea 除外
[功能完整性] pages/system/user/index.tsx:715、role/index.tsx:279/:282、menu/index.tsx:280、apikeys/index.tsx:745/:753/:779/:784、config/index.tsx:498/:506/:517、profile/index.tsx:334/:342/:350 —— 搜索框与弹窗表单输入框无 allowClear，用户无法一键清空；本域 32 个搜索框仅 13 个有、48 个表单输入框仅 3 个有 —— P0 —— FormInput 默认 allowClear=true，搜索框强制 allowClear
[功能完整性] pages/system/user/index.tsx:544-558、config/index.tsx:189-203、post/index.tsx:156-169、dict/hooks/useDictActions.ts:114-133 与 :180-199、role/hooks/useRoleActions.ts:132-149、notice/hooks/useNoticeData.ts:99-109 —— 7 处批量删除点击即发请求，无二次确认、无选中数提示 —— P0 —— 统一 confirmBatchDelete（Modal.confirm + "将删除 N 项" + okType=danger）
[功能完整性] pages/system/dept/index.tsx:287、menu/index.tsx:352 —— 两个树表 pagination={false}，既没有分页也没有"共 N 项"总量展示，节点多时用户无法判断规模 —— P1 —— Table title 插槽或页头输出"共 N 项"（递归计数），或按 antd 树表分页
[功能完整性] pages/system/menu/hooks/useMenuActions.tsx:112-185、:188-279 —— "级联删除"确认框用内嵌函数组件 + checkboxRef hack 自造，约 200 行只为两个确认框，且第二个确认框行为与第一个不一致 —— P1 —— 抽 ConfirmDeleteModal（支持 extra 复选项），确认框全域收敛为 1 处实现
[功能完整性] pages/system/apikeys/index.tsx:784、role/index.tsx:379-384、settings/captcha-background.tsx:593-594 —— 时间/数字用 <Input type="datetime-local"> / <Input type="number"> 实现，无选择器、无校验、提交 string —— P1 —— 时间用 DatePicker+dayjs，数字用 InputNumber min=0
[功能完整性] 全片 useColumnConfig / ColumnConfigModal / useTableQuery 使用 0 次 —— 14 个列表页都无法让用户选列、调密度、持久化视图偏好 —— P2 —— 至少给 user/role/notice/apikeys 四个宽表接入 useColumnConfig
\`\`\`

### 3.2 数据展示完整性

\`\`\`
[数据展示完整性] src/hooks/usePagination.ts:122 —— paginationProps.showTotal 只输出"共 ${Math.max(1, Math.ceil(t / pageSize))} 页"，11 个列表页因此永远看不到总条数（用户核心诉求） —— P0 —— 改为"共 ${t} 条 / 第 ${current} 页"；total 透出给页面
[数据展示完整性] src/hooks/usePagination.ts:122 —— total=0 时 Math.max(1,…) 使空列表显示"共 1 页"（空数据说谎） —— P0 —— t===0 时输出"暂无数据"或不渲染 showTotal
[数据展示完整性] pages/system/apikeys/index.tsx:688-695、apikeys/LogsModal.tsx:429-436、manager-orgs/index.tsx:323-328、manager-rules/index.tsx:388-393、notice/components/NoticeList.tsx:371-381 —— 5 份手写 pagination + showTotal，文案"共 N 条"与默认"共 N 页"双语义，维护时漏改一处即不一致 —— P1 —— 全部改走 paginationProps（含总条数），删除 5 份内联对象
[数据展示完整性] pages/system/user/index.tsx:785、role/index.tsx:329、config/index.tsx:467、dict/index.tsx:655/:829、captcha-background/index.tsx:193 等 12 个表 —— 无 scroll.x，列宽合计超过容器时挤压换行；其中 user:272、config:291、dict:129/:231、captcha-background/columns.tsx:96 声明了 fixed:"right" 但因缺 scroll.x 而不生效 —— P1 —— DataTable 默认 scroll={{ x: "max-content" }}，操作列固定需与 scroll 配对；列少时再显式关掉
[数据展示完整性] 全片仅 19 处 ellipsis（user:119/:140/:149/:169/:183、config:241/:275、dict:113/:215、apikeys:504、LogsModal:180、captcha-background/columns:43/:91、notice/NoticeList:87/:135/:159、post:217、api-config:231），其余长文本列换行撑高行高 —— P2 —— 列定义统一 ellipsis + Tooltip，或 DataTable 统一注入默认 ellipsis
[数据展示完整性] pages/system/user/utils.tsx:83-84（STATUS_TAG_CONFIG 正常/停用）、role/utils.tsx:30-36（ModernTag 正常/停用）、notice/NoticeList.tsx:189（正常/关闭）、captcha-background/constants.tsx:57（启用/禁用）、settings/api-config.tsx:252-253 与 email-config.tsx:268-269（xr-tag span） —— 同一"0/1"语义有 3 套组件 5 种文案，颜色与尺寸也不一致 —— P1 —— 抽 <StatusTag domain="user|role|captcha|notice"/>，文案/颜色集中配置
[数据展示完整性] 17 个 <Table> 全部有 rowKey（正例），但取值两套：15 个 "id"，dept/index.tsx:285 用 "key"（dept 树数据 key 派生），menu/index.tsx:350 用 "id"（menu/utils.tsx:72 树节点同时带 id/key） —— P2 —— 统一以记录主键 "id" 为 rowKey；树数据在 selector 层保留 id 字段
\`\`\`

### 3.3 交互反馈

\`\`\`
[交互反馈] pages/system/menu/hooks/useMenuActions.tsx:315-317 —— 保存菜单 catch 仅 console.error，校验失败/接口失败都不提示、不关弹窗，用户看到"点保存没反应" —— P0 —— isFormValidationError 提前 return + handleApiError(error,"保存菜单")
[交互反馈] pages/system/user/index.tsx:814-820、role/index.tsx:344-352、menu/index.tsx:365-376、config/index.tsx:485-491、dict/index.tsx:849-892/:895-950、notice/components/NoticeForm.tsx:122、profile/index.tsx:321-327/:372-378 —— 15 处 Modal 无 confirmLoading、onOk 直接调异步函数、无 submitting 守卫 → 双击重复提交，网络慢时用户不知道是否成功 —— P1 —— BaseEditModal 全量替换，onOk 返回 Promise 或传 confirmLoading={submitting}
[交互反馈] pages/system/user/index.tsx:560-572、role/columns/roleColumns.tsx:77-80 —— 状态 Switch 点击即调接口，无 loading、无失败回滚；role 还有"停用/启用"按钮无 loading —— P1 —— 请求中 setLoading 禁用控件，catch 回滚 checked，重要状态变更加确认
[交互反馈] pages/system/{user,role,menu,...} 20 文件 —— 提示出口三套并存：App.useApp().message（43 处）、utils/errorHandler.ts 的 getAppMessage()、裸 message.*（116 处）；统一性差且静态 message 在 antd 6 下拿不到上下文主题 —— P2 —— 统一 useAppMessage()（内部 App.useApp），禁止裸 message.*
[交互反馈] pages/system/user/index.tsx:564、role/hooks/useRoleActions.ts:159、apikeys/index.tsx:326、dict/hooks/useDictActions.ts、post/index.tsx:122-141 —— 成功提示文案三种风格（handleSuccess(x)、message.success(status===0?"启用成功":"停用成功")、直接中文拼接） —— P2 —— 抽 useCrudFeedback：success("已删除"/"已启用") 统一句式与时长
[交互反馈] pages/system/role/index.tsx:349、user/index.tsx:819、dict/index.tsx:853-857 —— 弹窗关闭有的靠 afterOpenChange 回填、有的手动 resetFields、有的两者都做，二次打开可能残留上次值或闪旧值 —— P2 —— BaseEditModal destroyOnHidden + afterOpenChange 统一回填时序
[交互反馈] 全片 Skeleton 使用 0，首屏 await 期间只靠 Table loading 蒙层，统计卡/详情页空白 —— P2 —— 统计卡与详情页加 Skeleton，抽 PageLoading 统一页面级加载
[交互反馈] pages/system/notice/index.tsx:458-475、my-notices/index.tsx:271-279 —— 列表刷新失败后仍显示上一次数据且无"数据可能过期"提示 —— P2 —— 刷新失败保留旧数据 + message.warning，或显示上次更新时间
\`\`\`

### 3.4 表单校验

\`\`\`
[表单校验] pages/system/config/index.tsx:519/:525、dict/index.tsx:879/:937、settings/api-config.tsx、settings/captcha-background.tsx:562/:571/:587/:593、captcha-background/modals/EditModal.tsx、manager-orgs/index.tsx —— 84 个 rules 块中 15 处只有 { required: true } 无 message，用户看到 antd 默认英文提示或定位不到字段 —— P1 —— rules 统一带中文 message（FormInput 内置 required 文案），并在 lint 层加约束
[表单校验] pages/system/user/index.tsx:851-863 —— 邮箱/手机号/工号仅 required，无 pattern 校验，可写入任意字符串；而 profile/index.tsx:380-417 改密有完整强校验（正例） —— P1 —— 抽 emailRule/phoneRule（与后端一致），App 内复用
[表单校验] pages/system/dept/index.tsx:70-73 —— handleSearch 用 searchForm.validateFields()，搜索条件被当表单校验，条件非法时整个搜索被阻断 —— P2 —— getFieldsValue()
[表单校验] pages/system/role/index.tsx:379-384、settings/captcha-background.tsx:593-594 —— 显示顺序/排序用 Input type="number"，提交为 string，后端需容错；无 min/max/精度约束 —— P1 —— InputNumber min=0 precision=0
[表单校验] pages/system/user/index.tsx:860-863、apikeys/index.tsx:745-784、settings/email-config.tsx:440-476 —— 长文本输入无 maxLength/showCount，备注类字段可直接灌满数据库列 —— P2 —— remark/description 统一 maxLength+showCount，与后端列宽对齐
\`\`\`

### 3.5 状态与分页

\`\`\`
[状态与分页] pages/system/settings/captcha-background.tsx:381-382、captcha-background/constants.tsx:41-42 —— 启用=1 / 禁用=0，与全局约定（0=正常/启用、1=停用/禁用）完全相反；:442-449、columns.tsx:104/:57 全部按反转语义渲染；源码注释也承认"1=启用 反转语义"（settings/captcha-background.tsx:434） —— P1 —— 前后端统一为 0=启用，或至少在 constants 里用命名常量 STATUS_ENABLED 封装，禁止裸 1/0
[状态与分页] captcha-background/columns.tsx:104、captcha-background/constants.tsx:57、dept/constants.tsx:20、menu/index.tsx:110、notice/NoticeList.tsx:189、role/columns/roleColumns.tsx:77-79、role/hooks/useRoleActions.ts:159、role/utils.tsx:32-33、settings/api-config.tsx:252-253、settings/captcha-background.tsx:442-443/:449、settings/email-config.tsx:268-269、user/index.tsx:564、profile/index.tsx:206 —— 状态 0/1 裸三元共 20 处，语义散落（正常/停用、启用/禁用、正常/关闭、true/false） —— P1 —— 常量模块 + <StatusTag> 收口，禁止在渲染层直接比较 0/1（ESLint no-magic-numbers 局部规则）
[状态与分页] pages/system/settings/api-config.tsx:90-91、email-config.tsx:92-93 —— 请求分页参数名用 page / pageSize，与全项目 current/pageSize 约定不一致，后端被迫双兼容 —— P1 —— 前端统一 current
[状态与分页] pages/system/settings/api-config.tsx:371-383、settings/email-config.tsx:388-400、my-notices/index.tsx:271-282 —— spread paginationProps 后又手写 onChange 调 setCurrent/setPageSize，而 paginationProps 自身已带 onChange（usePagination.ts:123）→ 一次翻页可能触发两次请求 —— P1 —— 只保留 paginationProps.onChange
[状态与分页] pages/system/dict/index.tsx:655-666、:829-840、:631-650 —— 类型表与数据表共用同一 paginationProps 和 selectedRowKeys，切换 Tab 后页码/选中残留，可能在"数据"Tab 上删掉"类型"Tab 选的记录 —— P1 —— 两个 Tab 各自 usePagination + useTableManager 实例，切换时 reset
[状态与分页] pages/system/apikeys/index.tsx:688-695、LogsModal:429-436、manager-orgs:323-328、manager-rules:388-393、NoticeList:371-381 —— 5 份手写分页（含 showTotal）语义与默认不一致，size 档位也各写一遍 —— P1 —— 收敛到 paginationProps
[状态与分页] src/hooks/usePagination.ts（current/pageSize 经 usePersistedStateController 按 location.pathname 持久化到 sessionStorage） —— 刷新/回退恢复旧页码，但筛选条件不持久化，页码可能超出结果集而显示空表 —— P2 —— 恢复页码时校验 <= 总页数，或筛选变更时 reset 到第 1 页
[状态与分页] pages/system/notice/index.tsx:487-488、notice/components/NoticeList.tsx:371 —— 父页从 paginationProps 取值再传子组件，子组件又自己拼一份 pagination，分页状态双来源 —— P2 —— 分页状态单一来源（父持有、子只渲染）
[状态与分页] pages/system/notice/index.tsx:69/:458-475、config/index.tsx:55-111、dict/index.tsx:350-366、user/index.tsx:90-264、role/index.tsx、post/index.tsx:190-224 —— 服务端排序只覆盖 8 页 11 表，dept/menu/manager-orgs/manager-rules/api-config/email-config/my-notices/LogsModal 无排序；captcha-background/columns.tsx:77 用本地 sorter 只排当前页，与其他页"点击排序 → 请求带 orderByColumn/isAsc"的行为不一致 —— P1 —— 所有分页表统一 useServerSort（本地排序仅用于当前页全量数据场景，需显式注明）
\`\`\`

### 3.6 错误与空态

\`\`\`
[错误与空态] pages/system/menu/hooks/useMenuActions.tsx:315-317、dict/hooks/useDictData.ts:77/:107、dept/index.tsx:124、role/hooks/useRoleData.ts:84/:105/:126、role/hooks/useRoleActions.ts:147 —— 11 处 catch 只 console.error，用户侧完全静默（列表空白、删除无反应） —— P1 —— handleApiError(error,"动作名") + 列表失败置 error state
[错误与空态] pages/system/menu/hooks/useMenuActions.tsx:86-101、:271-273 —— 手写三层 "response" in error / "data" in / "message" in 拼错误文案，与 utils/errorHandler.ts:67-86 extractErrorMessage 完全重复 —— P1 —— 直接 handleApiError
[错误与空态] pages/system/dict/hooks/useDictActions.ts:88-90、:154-156 —— 手写 "errorFields" in error 判表单错误，与 @/types/common 的 isFormValidationError 重复 —— P1 —— import isFormValidationError
[错误与空态] src/hooks/useTableManager.ts:242-271 —— 加载失败 catch 里清空 data/total，页面变为"空列表"且无错误提示，用户误以为没有数据 —— P1 —— 保留 error state，渲染 ErrorAlertWithRetry(error,onRetry=loadData)
[错误与空态] components/shared/EmptyStateWithAction.tsx、components/shared/ErrorAlertWithRetry.tsx —— 组件已存在，本域使用 0 次；14 个列表页各写各的空态或直接用 antd 默认 —— P1 —— DataTable 内建空态（区分"无数据"与"无匹配结果"）与错误态
[错误与空态] pages/system/settings/api-config.tsx:385-390、email-config.tsx:402-406 —— 仅这 2 处自定义 emptyText；其余 15 个 Table 用默认"暂无数据"，没有"去新增/清除筛选"引导 —— P2 —— 空态组件化：有筛选条件→"无匹配结果，清除筛选"；无筛选→"暂无数据，去新增"
[错误与空态] pages/system/notice/detail.tsx、my-notices/detail.tsx、settings/captcha-background.tsx:Spin —— 3 处页面级加载用裸 Spin，无统一加载容器/骨架/错误重试 —— P2 —— 抽 PageLoading（Skeleton + ErrorAlertWithRetry）
\`\`\`

### 3.7 样式与主题一致性

\`\`\`
[样式与主题一致性] 本域 62 处 hex 颜色，其中约 57 处写成 var(--theme-x, #hex) 双份兜底（token 与字面量并存，主题切换时字面量成为隐患）—— 代表行号：apikeys/index.tsx:507/:540/:559、apikeys/LogsModal.tsx:256-259/:284-287/:303/:331/:362/:371/:382/:409、config/index.tsx:314/:349/:359/:429、dept/index.tsx:210/:220、dept/modals/EditModal.tsx:119、dict/index.tsx:526/:536/:613/:693/:703/:789、menu/hooks/useMenuActions.tsx:172/:244、menu/index.tsx:130/:240、captcha-background/index.tsx:92/:101 —— P1 —— 颜色全部收进 design-system token（CSS 变量），组件里只允许 var(--theme-x)，删掉字面量兜底
[样式与主题一致性] 5 处裸 hex 无 token：apikeys/index.tsx:546 "#ba3630"（过期红）、apikeys/index.tsx:716 "#f5f5f5"（提示块背景，暗色主题下刺眼）、manager-orgs/index.tsx:212 "#faad14"、manager-rules/index.tsx:285/:298 "#888"、apikeys/LogsModal.tsx:372 trailColor="#e9efeb" —— P1 —— 分别映射到 --theme-error / --theme-fill-secondary / --theme-warning / --theme-text-tertiary
[样式与主题一致性] pages/system/user/utils.tsx、role/utils.tsx（ModernTag）、settings/api-config.tsx:252（xr-tag span）、captcha-background/constants.tsx:57（Tag） —— 同一状态语义 3 套视觉实现，圆角/字号/色彩体系均不同 —— P2 —— StatusTag 统一实现（内部可用 Tag/ModernTag，对外一个 API）
[样式与主题一致性] pages/system/user/index.tsx:733-737（重置→搜索→刷新）、role/index.tsx:300-306、menu/index.tsx:278-307、dept/index.tsx:255-265（搜索→重置→刷新） —— 搜索区按钮顺序与主次不统一，user 页把"搜索"放中间、右边才是刷新 —— P1 —— SearchToolbar 固定顺序：搜索(primary)→重置→刷新，其余按钮进 extra
[样式与主题一致性] pages/system/dict/index.tsx:596-605 —— 按钮文案"刷新"实际执行 resetFields()+重新加载，等同"重置"；该页没有真正的刷新 —— P1 —— 文案改"重置"，另提供"刷新"（仅重载不清条件）
[样式与主题一致性] 全片搜索框/按钮/Modal 宽度硬编码（150/160/180/200/120 分散，Modal width 520/600/800 混用） —— P2 —— 搜索项宽度档位（s/m/l）+ Modal 尺寸档位统一
\`\`\`

### 3.8 可访问性与国际化

\`\`\`
[可访问性与国际化] 51 处 icon-only <Button icon={...}/> 中仅 1 处带无障碍名称（user/index.tsx:737 aria-label="刷新"），其余如 apikeys/index.tsx:611、config/index.tsx:435/:438、captcha-background/index.tsx:147/:177 等只有图标，屏幕阅读器读作"按钮" —— P2 —— ActionButtons/图标按钮统一 aria-label + Tooltip
[可访问性与国际化] 全片 aria 属性仅 7 处（user:737、login:166/:168/:173、settings/index.tsx:98/:105/:122），列表、筛选、分页、弹窗均无 aria-label/aria-describedby/role 补充 —— P2 —— DataTable/SearchToolbar/确认框统一补 aria 属性
[可访问性与国际化] 139 处 placeholder 但无对应 label/Form.Item extra 说明，搜索框和表单字段仅靠 placeholder 表达含义（placeholder 失焦即消失，可访问性差） —— P2 —— 搜索项加 Form.Item label 或 aria-label
[可访问性与国际化] 全片 useTranslation/FormattedMessage 使用 0，中文文案硬编码在组件与 hook 内（如 usePagination 的"共 N 页"、各列 title、"启用/停用"），无法国际化也无法集中校对 —— P2 —— 至少把文案常量抽到 i18n 字典（zh-CN.json），为后续多语言留接口
[可访问性与国际化] autoComplete 仅登录页 3 处（login/index.tsx:265/:274/:285），其他用户名/邮箱/密码类输入未标注，浏览器密码管理器无法正确填充 —— P2 —— 登录/改密/个人资料表单补 autoComplete
\`\`\`

### 3.9 性能与副作用

\`\`\`
[性能与副作用] pages/system/menu/index.tsx:347-361 —— 树表 defaultExpandAllRows + 动态数据：首渲染全展开 O(n)，数据刷新后展开态不再重算，大菜单卡顿且与数据不同步 —— P1 —— 与 dept 一致改受控 expandedRowKeys，默认只展开根节点
[性能与副作用] pages/system/dept/index.tsx:55-68 —— 数据变化 useEffect 里全量重算根节点并 setState，每次刷新重置用户展开态 —— P2 —— 仅在首次加载或根节点 key 集合变化时更新
[性能与副作用] 全片搜索无防抖（debounce/useDebounce 使用 0） —— 关键词类查询每敲一个字符都要点按钮，若改成实时搜索会直接打满接口 —— P2 —— 抽 useDebouncedSearch（300ms），输入框回车/按钮仍立即触发
[性能与副作用] pages/system/user/index.tsx:455-461/:480-481、dict/hooks/useDictData.ts 等 —— 多处 eslint-disable exhaustive-deps 掩盖不稳定依赖，存在多拉/漏拉风险 —— P2 —— 回调 useCallback + 原始值依赖，去掉 disable
[性能与副作用] pages/system/notice/index.tsx:458-475 —— Table onChange 内直接拼参数 fetch（绕过 state），与 useTableManager.handleTableChange 逻辑重复，排序/分页同时变时会发两次请求 —— P2 —— 统一 useTableManager.handleTableChange（含排序+分页合并一次请求）
[性能与副作用] 17 个 Table 无 size 档位统一、无虚拟滚动，分页大小档位 sizeOptions 使用 0；dict 数据表在 1 万+ 字典项时全量渲染 —— P2 —— paginationProps 提供统一 pageSizeOptions（10/20/50/100），超大表评估虚拟滚动
\`\`\`

### 3.10 代码复用与可维护性

\`\`\`
[代码复用与可维护性] pages/system/role/utils.tsx:13-27、role/hooks/useRoleData.ts:103-111、:122-130 —— processTreeData 同一递归实现 3 份（menu/dept 另有各自树处理） —— P2 —— 保留 utils 一份，hook 内复用
[代码复用与可维护性] pages/system/user/index.tsx:665-700、role/index.tsx:234-264、post/index.tsx:270-300、dict/index.tsx:508-540 与 :675-707、config/index.tsx:330-360 —— 列表页统计卡 6 处复制（Row/Col/Card/Statistic 结构 + 颜色常量） —— P2 —— 抽 <StatCards items={[{title,value,color,icon}]} />
[代码复用与可维护性] 24+ 处 onSearch={() => {}}（user:718-730、role:285-289、menu:298、config:401/:520/:531、apikeys:645/:656、dict:573-586、post:453-457、profile:354、NoticeForm:159/:177/:192/:287、api-config:342/:349/:431/:487/:553、email-config:366/:497、default-theme:177/:185、captcha-background:119/:128/:137、settings/captcha-background:515/:528/:535/:563/:572/:579/:588） —— Select 未开 showSearch，onSearch 永不触发，纯死代码且误导后续维护者 —— P2 —— 删除；需要远程搜索的 Select 改为 showSearch+filterOption（或直接 FormSelect 默认开）
[代码复用与可维护性] pages/system/settings/api-config.tsx:252、email-config.tsx:268、captcha-background.tsx:442 —— 状态展示用裸 <span className="xr-tag">，与共享 Tag/ModernTag 并存 —— P2 —— 统一 StatusTag
[代码复用与可维护性] pages/system/apikeys/index.tsx:430（sorter: any）、settings/default-theme.tsx、captcha-background —— "any" 3 文件，列 render 与自定义字段绕开类型 —— P2 —— ColumnsType<T> + 泛型 DataTable
[代码复用与可维护性] pages/system/settings-page/index.tsx（55 行） —— 旧 Tabs 设置壳仍留在仓库但路由已改指向 settings/index.tsx（SettingsShell），死文件误导 —— P2 —— 删除
[代码复用与可维护性] pages/system/role/utils.tsx:54-56 formatLocalTime 仅对 formatDateTime 再做一层包装 —— P2 —— 直接使用统一日期工具，删除无意义包装
[代码复用与可维护性] 分页/空态/错误提示/确认框/弹窗五类横切逻辑在 14 页各自实现（详见 §4 表），改一处需改 14 处 —— P1 —— 按 §4 的 3 个必抽组件 + 已有组件推广收敛
\`\`\`

**硬编码统计汇总（三项必统计）**

| 类别 | 数量 | 分布与代表行号 | 结论 |
| --- | --- | --- | --- |
| hex 颜色 | **62 处** | \`var(--theme-x, #hex)\` 双写约 57 处：apikeys/index.tsx:507/540/559、apikeys/LogsModal.tsx:256-259/284-287/303/331/362/371/382/409、config/index.tsx:314/349/359/429、dept/index.tsx:210/220、dept/modals/EditModal.tsx:119、dict/index.tsx:526/536/613/693/703/789、menu/hooks/useMenuActions.tsx:172/244、menu/index.tsx:130/240、captcha-background/index.tsx:92/101；**裸 hex 5 处**：apikeys/index.tsx:546 \`#ba3630\`、apikeys/index.tsx:716 \`#f5f5f5\`、manager-orgs/index.tsx:212 \`#faad14\`、manager-rules/index.tsx:285/298 \`#888\`、apikeys/LogsModal.tsx:372 \`#e9efeb\` | 全部应映射 design-system token，禁止字面量兜底 |
| 状态 0/1 裸三元 | **20 处** | captcha-background/columns.tsx:104、captcha-background/constants.tsx:57、dept/constants.tsx:20、menu/index.tsx:110、notice/NoticeList.tsx:189、role/columns/roleColumns.tsx:77-79、role/hooks/useRoleActions.ts:159、role/utils.tsx:32-33、settings/api-config.tsx:252-253、settings/captcha-background.tsx:442-443/449、settings/email-config.tsx:268-269、user/index.tsx:564、profile/index.tsx:206 | 其中 captcha 系列语义反转（1=启用），须优先修 |
| 手写日期格式 | **3 处** | notice/index.tsx:250-257（手拼时区偏移 \`offsetSign/offsetHours/offsetMins\` + \`format("YYYY-MM-DDTHH:mm:ss")\`）、notice/index.tsx:280（\`recurrenceConfig.endDate\` 同样手写格式）、apikeys/index.tsx:361（\`values.expiresAt.toISOString()\`） | 抽 \`toBackendDateTime()\` / \`toBackendISOString()\` 到 utils/datetime；dayjs 目前只有 2 文件在用 |

## 4. 公共组件抽取候选与必要性评估

估算方法：单页当前重复代码行数（搜索区 \`Form\` 约 30-40 行、分页对象 8-10 行、确认框 12-20 行、弹窗 6-10 行、空态 8 行）× 涉及页面数；"必抽"= 改 1 处可一次性修复全域同类问题。

| 候选组件/能力 | 本模块复用次数 | 涉及文件数 | 可减代码量估算 | 结论 |
| --- | --- | --- | --- | --- |
| \`usePagination\` 增强：showTotal 输出总条数 + \`totalText\` + 空数据文案 | 11 页 | 1（\`src/hooks/usePagination.ts\`） | 改 1 行修复 11 页；若各页自写 showTotal，11×8 行 ≈ 90 行重复 | **必抽**（最高 ROI，P0） |
| \`<DataTable>\`（rowKey/loading/pagination/**scroll.x 默认**/空态/错误态/选中态/size） | 17 表 / 14 页 | 14 | 每页省 15-25 行 × 14 ≈ 300 行；同时修 12 表无 scroll、6 处 fixed 失效 | **必抽** |
| \`SearchToolbar\`（inline Form + 搜索/重置/刷新固定顺序 + 回车搜索 + 折叠更多 + extra 按钮槽） | 13 页 | 13 | 每页省 25-35 行 × 13 ≈ 350 行 | **必抽** |
| \`<FormInput>\`/\`<FormSelect>\`（默认 allowClear；搜索场景默认回车；Select 默认 showSearch+filterOption；rules 默认中文 message） | 80 Input + 69 Select | 15 | 属性级每处 1-2 行 × 149 ≈ 200 行，收益在"默认即正确" | **必抽**（可配合 ESLint 规则强制） |
| \`ConfirmDeleteModal\` / \`useConfirmAction\`（含批量 + extra 复选项如级联删除） | 16 处 Modal.confirm + 3 Popconfirm + 7 处缺失确认 | 12 | 每处省 10-18 行 × 19 ≈ 250 行；menu 两处 hack 直接省 ~180 行 | **必抽** |
| \`BaseEditModal\` 全量推广（内置 confirmLoading、destroyOnHidden、防连点） | 1/22 Modal | 12 | 每页省 6-10 行 × 12 ≈ 90 行；消除 15 处"可重复提交" | 应统一 |
| \`<StatusTag domain>\`（0/1 语义 + 文案 + 颜色统一，内置 captcha 反转映射） | 3 套实现 / 20+ 使用点 | 10 | 每点省 5-10 行 × 20 ≈ 150 行；同时消除 20 处裸三元 | **必抽** |
| \`useServerSort\` 全量接入 + 移除本地 sorter | 8 页已接入 / 9 表未接入 | 17 | 每表 3-5 行 sortOrder 绑定 | 应统一 |
| \`<StatCards>\`（统计卡组） | 6 页 | 6 | 每页省 25-30 行 × 6 ≈ 170 行 | 应统一 |
| \`useListResource\`（加载三态 + filterEmpty + handleApiError + 重试） | 7 页 useTableManager + 8 页裸 useState | 15 | 每页省 20-30 行 × 15 ≈ 400 行 | 应统一（与 \`useTableManager\` 合并而非新增） |
| 空态/错误态/页面加载（\`EmptyStateWithAction\`+\`ErrorAlertWithRetry\`+\`PageLoading\`） | 0 使用 | 14 | 每页省 8-12 行 × 14 ≈ 140 行 | 应统一 |
| \`toBackendDateTime\` / 日期工具、\`processTreeData\`、树表受控展开 | 3 处日期 + 3 份树函数 + 2 树表 | 6 | 省 60 行；修 menu 全展开性能 | 应统一 |
| \`useFormValidationMessage\`（统一 rules message） | 15 处缺 message | 8 | 收益是文案一致，代码量中性 | 建议并入 FormInput 默认值，不单独抽 |

结论：**必须新增的只有 3 个出口**——\`usePagination\` 修 showTotal、\`SearchToolbar\`、\`ConfirmDeleteModal\`；\`DataTable\`/\`FormInput\`/\`StatusTag\` 属于"必要统一"（可先做 DataTable + FormInput 两个默认值层）；其余是"已有组件没接"（BaseEditModal/EmptyState/ErrorAlert/useTableManager/useServerSort/useColumnConfig），不需要再造。

## 5. 最佳实践方案

### 5.1 分页：改 1 行，全域生效（P0）

\`\`\`ts
// src/hooks/usePagination.ts
const paginationProps: TablePaginationConfig = useMemo(
  () => ({
    current,
    pageSize,
    total,
    showSizeChanger: true,
    showQuickJumper: true,
    pageSizeOptions: ["10", "20", "50", "100"],
    showTotal: (t) =>
      t > 0
        ? \`共 ${t} 条 / 第 ${current} 页，共 ${Math.max(1, Math.ceil(t / pageSize))} 页\`
        : "暂无数据",
    onChange: handlePageChange,
  }),
  [current, pageSize, total, handlePageChange]
);
\`\`\`

约定：

- 树表（dept/menu）用 \`pagination={false}\` 时，必须在 \`Table\` 的 \`title\` 插槽或页头渲染"共 N 项"（递归计数）。
- 页面**不得**再在 \`paginationProps\` 之上手写 \`onChange\`（当前 api-config/email-config/my-notices 三处双 onChange，可能双请求）。
- 分页参数统一 \`current/pageSize\`，禁止 \`page\`。

### 5.2 DataTable：表格默认即完整（P0/P1）

\`\`\`tsx
// src/components/shared/DataTable.tsx
interface DataTableProps<T> extends TableProps<T> {
  error?: unknown;            // 传入则渲染错误态 + 重试
  onRetry?: () => void;
  emptyType?: "no-data" | "no-match";
}
export function DataTable<T extends object>({ error, onRetry, emptyType, scroll, ...rest }: DataTableProps<T>) {
  if (error) return <ErrorAlertWithRetry error={error} onRetry={onRetry} />;
  return (
    <Table<T>
      size="middle"
      rowKey="id"
      scroll={scroll ?? { x: "max-content" }}   // 默认开启横向滚动，fixed 列才生效
      locale={{ emptyText: emptyType === "no-match" ? <EmptyNoMatch /> : <EmptyStateWithAction /> }}
      {...rest}
    />
  );
}
\`\`\`

配套约定：\`rowKey\` 统一主键 \`id\`；\`fixed:"right"\` 必须与 \`scroll.x\` 同时出现；长文本列统一 \`ellipsis\`；排序一律 \`useServerSort\`（\`getColumnSortOrder\` + \`resolveSorter\`），本地 \`sorter\` 仅用于当前页全量数据并注明原因。

### 5.3 SearchToolbar：搜索区唯一实现（P0/P1）

\`\`\`tsx
// src/components/shared/SearchToolbar.tsx
interface SearchToolbarProps {
  form: FormInstance;
  onSearch: () => void;
  onReset?: () => void;
  onRefresh?: () => void;
  children: ReactNode;                 // Form.Item 集合（内部统一 <FormInput>/<FormSelect>）
  collapsible?: boolean;               // 条件多时折叠
  extra?: ReactNode;                   // 右侧操作按钮（新增/导入/导出）
}
export const SearchToolbar: FC<SearchToolbarProps> = ({ form, onSearch, onReset, onRefresh, children, extra }) => (
  <Card style={{ marginBottom: 16 }}>
    <div style={{ display: "flex", justifyContent: "space-between", gap: 16, flexWrap: "wrap" }}>
      <Form form={form} layout="inline" style={{ flex: 1, minWidth: 0 }} onKeyDown={(e) => {
        // 任意输入框回车即搜索；textarea 不拦截
        if (e.key === "Enter" && !(e.target as HTMLElement).matches("textarea")) { e.preventDefault(); onSearch(); }
      }}>
        {children}
        <Form.Item>
          <Space>
            <Button type="primary" icon={<SearchOutlined />} onClick={onSearch}>搜索</Button>
            {onReset && <Button onClick={onReset}>重置</Button>}
            {onRefresh && <Button icon={<ReloadOutlined />} aria-label="刷新" onClick={onRefresh}>刷新</Button>}
          </Space>
        </Form.Item>
      </Form>
      {extra && <Space>{extra}</Space>}
    </div>
  </Card>
);
\`\`\`

配套约定：\`<FormInput>\` 默认 \`allowClear\`；\`<FormSelect>\` 默认 \`allowClear showSearch optionFilterProp="label"\`；\`onSearch={() => {}}\` 一律删除（需要远程搜索的显式写 \`showSearch + filterOption={false} + onSearch=debounce(...)\`）。

### 5.4 StatusTag 与硬编码收敛（P1）

\`\`\`tsx
// src/components/shared/StatusTag.tsx
type StatusDomain = "common" | "captcha" | "notice";
const STATUS_MAP: Record<StatusDomain, Record<0 | 1, { text: string; color: string }>> = {
  common:  { 0: { text: "正常", color: "success" }, 1: { text: "停用", color: "error" } },
  captcha: { 1: { text: "启用", color: "success" }, 0: { text: "禁用", color: "error" } }, // 兼容既有反转语义
  notice:  { 0: { text: "正常", color: "success" }, 1: { text: "关闭", color: "default" } },
};
export const StatusTag: FC<{ status: 0 | 1; domain?: StatusDomain }> = ({ status, domain = "common" }) => {
  const item = STATUS_MAP[domain][status];
  return <Tag color={item.color}>{item.text}</Tag>;
};
\`\`\`

- 颜色只允许 \`var(--theme-*)\`，禁止 \`var(--theme-x, #hex)\` 双写与裸 hex；server 端返回的枚举值必须映射为命名常量（禁止渲染层比较 0/1）。
- 日期统一 \`formatDateTime\`（展示）/ \`toBackendDateTime\`（提交，含时区）：禁止在页面里手拼 offset 或直接 \`toISOString()\`。

### 5.5 统一错误出口（P1）

\`\`\`ts
// 页面/ Hook 内统一写法
try { await post(...); message.success("保存成功"); }
catch (e) {
  if (isFormValidationError(e)) return;         // 校验失败交给 Form 展示
  handleApiError(e, "保存菜单");                 // 统一文案："保存菜单失败: xxx"
}
\`\`\`

禁止 \`catch { console.error(...) }\` 单出口；列表加载失败额外置 \`error\` state，渲染 \`<ErrorAlertWithRetry error={error} onRetry={load} />\`；提示统一走 \`useAppMessage()\`（内部 \`App.useApp()\`），禁止裸 \`message.*\` 与 \`getAppMessage()\` 混用。

## 6. 落地路线图

### P0（1-2 天，专项修复，可独立发版）

1. \`usePagination.showTotal\` 改为总条数 + 空数据文案；树表补"共 N 项"。
2. \`ConfirmDeleteModal\`（或临时 \`confirmBatchDelete\`）+ 接入 7 处批量删除（user/config/post/dict×2/role/notice）。
3. menu 保存 catch 补 \`isFormValidationError\` + \`handleApiError\`。
4. 搜索框批量补 \`allowClear\`（user/role/menu 等 19 处搜索框）与回车搜索；删除 24+ 处 \`onSearch={() => {}}\`。
5. 弹窗补 \`confirmLoading\`（至少 user/role/menu/config/dict/notice 6 处）+ submitting 防连点。

**验收口径**：17 个 Table 中所有分页表（15）均输出"共 N 条/项"（\`grep -c "showTotal"\` + 手工 14 页回归）；7 处 batch-delete 均有确认弹窗（grep \`batch-delete\` 上下文断言）；空列表不出现"共 1 页"；菜单保存接口 500 时页面出现 message.error；\`npm run lint && npm run type-check\` 通过。

### P1（1 周，组件统一）

1. 落地 \`SearchToolbar\` + \`FormInput/FormSelect\` 默认值，迁移 13 个列表页。
2. 落地 \`DataTable\`：默认 \`scroll.x\`、空态/错误态、pagination 透传；修 6 处 fixed 失效。
3. \`BaseEditModal\` 全量替换 15 个裸 Modal，统一 \`destroyOnHidden/confirmLoading\`。
4. \`<StatusTag>\` 收口 20 处裸三元与 3 套标签；captcha 状态语义反转（1=启用）在 constants 层显式封装并安排后端对齐。
5. 服务端排序补齐 9 表 + 移除 captcha-background 本地 sorter；分页参数 page→current；去掉 3 处双 onChange。
6. 错误/空态接入 \`EmptyStateWithAction\`/\`ErrorAlertWithRetry\`；11 处静默 catch 全部补提示。
7. menu 树表改受控展开（默认只展开根）。

**验收口径**：13 个列表页搜索区 JSX 行数下降 ≥50%；\`grep -r "onSearch={() => {}}"\` 为 0；\`grep -rc "var(--theme-[a-z]*, #"\` 下降 80%；17 表全部有 scroll.x 或显式 \`scroll={false}\` 并注明；Service 排序字段与后端 orderByColumn 白名单对齐；新增组件有单测（SearchToolbar/DataTable/ConfirmDeleteModal/StatusTag）。

### P2（2 周，体验与卫生）

1. \`StatCards\`、\`processTreeData\`、日期工具去重；\`any\` 清零；\`Input type="number"/"datetime-local"\` → \`InputNumber/DatePicker\`。
2. 搜索防抖 300ms、条件折叠、页面级 Skeleton、\`PageLoading\`。
3. hex 颜色与宽度收敛到 design token；按钮/尺寸档位统一；icon 按钮补 \`aria-label\`。
4. 删除死文件 \`pages/system/settings-page/index.tsx\`；移除 \`eslint-disable exhaustive-deps\`，修真实依赖；文案抽到 zh-CN 字典。

**验收口径**：\`grep -r "#[0-9a-fA-F]\{3,6\}"\` 在本域为 0（CSS 变量文件除外）；状态比较不再出现裸 \`0/1\`；51 个 icon 按钮全部有 aria-label；统计卡/树工具函数/日期序列化各只剩 1 处实现。
