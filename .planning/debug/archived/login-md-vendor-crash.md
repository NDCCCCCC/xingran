---
gsd_state_version: 1.0
slug: login-md-vendor-crash
status: resolved
trigger: vendor-CVbWoGUo.js:8 Uncaught TypeError: Cannot read properties of undefined (reading 'selectors-4') at sp (vendor-CVbWoGUo.js:8:46627) at markdown-vendor-BlMsjxku.js:7:20139
created: 2026-06-12
updated: 2026-06-12
skip_audit: true
---

# Debug Session: login-md-vendor-crash

## Current Focus
- hypothesis: (to be determined by gsd-debugger)
- test: (to be determined)
- expecting: (to be determined)
- next_action: gather initial evidence

## Symptoms (user-reported)

### Expected behavior
- 登录页面正常渲染,显示登录表单(用户名、密码、验证码、登录按钮)
- 页面应该完全可以交互,无报错

### Actual behavior
- 整个页面崩溃/白屏(React 组件树 unmount)
- 浏览器控制台抛出 Uncaught TypeError,导致整个 React 应用无法挂载

### Error messages
```
vendor-CVbWoGUo.js:8 Uncaught TypeError: Cannot read properties of undefined (reading 'selectors-4')
    at sp (vendor-CVbWoGUo.js:8:46627)
    at markdown-vendor-BlMsjxku.js:7:20139
```

### Timeline
- 现象:对前端进行 debug 之后才出现("对前端debug之后财神"——推测为输入误差,意思是前端调试后才出现)
- 之前未明确说明是否正常工作
- **补充信息(2026-06-12)**:
  - 本地 `npm run dev` 没问题
  - 通过 `scripts/build-linux.bat`(Windows → Linux 跨平台打包)部署到 `10.62.10.39:9000` 后才出现
  - 部署模式:前后端一体打包(`-tags=embed`),前端 dist 嵌入 Go 二进制

### Reproduction
- 触发条件:打开登录页面(尚未登录)就立即报错
- 无需特殊操作或表单提交

## Key Context
- **意外关联**:错误堆栈涉及 `markdown-vendor-BlMsjxku.js`,但登录页不应加载 markdown 编辑器
- 错误源头是 `sp` 函数(在主 vendor chunk 中,被 markdown-vendor 调用)
- `selectors-4` 字段访问,目标对象是 undefined —— 典型的模块加载时序/循环依赖/Vite chunk-splitting 问题
- 前端栈:React 19.2 + Vite 7.2 + Ant Design 6.1 + @uiw/react-md-editor + Zustand 5.0
- @uiw/react-md-editor 在 frontend 多个页面使用(知识库文章等),但登录页不应直接引用

## Initial Hypotheses (to validate)

1. **Vite chunk-splitting 误配置**:markdown-vendor 被错误地放进登录页的初始 bundle,或被某个共享模块(例如全局 layout、route preloading)间接拉入
2. **@uiw/react-md-editor 与当前 React/Zustand 版本不兼容**:依赖项的内部 selector API 发生变化
3. **Vite manualChunks 配置把 markdown-vendor 强制并入主 vendor**:导致加载顺序异常
4. **CSS-in-JS 或 polyfill 缺失**:markdown-vendor 依赖的某些 feature 在生产环境不可用
5. **循环依赖 / 模块求值顺序问题**:在 vite production build 后,模块初始化顺序与 dev 模式不同

## Evidence
- **Vite `manualChunks` 配置位置**:`xingran-react-frontend/vite.config.ts`
  - 旧配置:`markdown-vendor` 规则在 `react-vendor` / `antd-*` 之后才匹配
  - 结果:跨平台生产构建产出的 chunk 出现循环导入,登录页运行时 ESM TDZ 触发
- **修复后构建输出**(本地 `npm run build`,2026-06-12 重新验证):
  - `vendor-CvhVq_z_.js`: **149.58 kB** (gzip 50.94 kB)
  - `markdown-vendor-KgVDo73u.js`: **1,072.73 kB** (gzip 363.75 kB)
  - 全部 chunk 构建无 error/warning,57s 完成
- **跨平台路径**:`build-linux.bat` 调用 `npm run build` → `vite.config.ts` 同一份配置,所以 fix 自动覆盖部署产物
- 服务端地址:`10.62.10.39:9000`(前后端一体包,前端 dist 嵌入 Go 二进制)

## Eliminated
- ~~本地 `npm run dev` 复现~~ — dev 模式不经过 manualChunks split,无法复现
- ~~后端 API 配置问题~~ — 错误是浏览器侧 ESM 模块加载,与后端无关

## Resolution
- **status**: COMPLETE
- **root_cause**: Vite `manualChunks` 把 markdown 生态依赖拆到两个 chunk(`hast-util-select/rehype-ignore/mdast-*` 在 `markdown-vendor`,而 `css-selector-parser/hastscript/unist-util-*/rehype/stringify-entities` 等在通用 `vendor`),形成循环导入。生产构建时 ESM TDZ 触发:`hast-util-select` 在模块顶层调用 `createParser({syntax: "selectors-4"})`,共享的 `cssSyntaxDefinitions` 还未初始化,`Mn["selectors-4"]` 抛错。Vite chunk loader 在首次失败时停摆,React 树整体 unmount,登录页白屏崩溃
- **fix**: `vite.config.ts` 把 `markdown-vendor` 规则置顶,并补齐缺失的 include 项(unified/remark/rehype/mdast/micromark/hast/unist/character-entities/style-to-object/nth-check/estree-util/parse5/entities/...)
- **files_changed**: `xingran-react-frontend/vite.config.ts`(仅此 1 个文件)
- **verification**:
  - ✅ 本地 `npm run build` 通过,chunk 尺寸符合预期
  - ⏳ 部署验证待用户执行 `build-linux.bat` 后重新部署到 `10.62.10.39:9000` 并访问登录页确认

## Follow-up (out of scope, NOT fixed)
- agent 调查中发现:存在第二个潜在循环 `react-vendor` ↔ `antd-vendor`,症状为 `Cannot set properties of undefined (setting 'Activity')` at `react-vendor-…js:1:9177`
- 该问题被前一个 crash 掩盖,被 `selectors-4` 错误吞掉
- 当前未处理(遵守 scope-constrainment 原则,本次任务只修 `selectors-4`)
- **如果重新部署后登录页报的是 `Activity` 错误而非 `selectors-4`,则需开启新 debug session 修复该 chunk 循环**

## Phase 41 Closure (2026-06-26)
won't_fix_reason: 复测 build 通过(`cd xingran-react-frontend && npm run build` 退出 0,34.32s),与 css-tree-vendor-chunk-tdz 同根(`selectors-4` 报错栈完全一致 — `Mn["selectors-4"]` 即 css-tree `To` 变量未初始化),已被 vite.config.ts 升级为 THREE_FAMILY/MARKDOWN_FAMILY 依赖图传递闭包方案彻底覆盖 — markdown 解析生态(parse5/hastscript/nth-check/github-slugger/unified/remark/rehype/mdast/micromark/hast/unist/...)按闭包归入 `vendor-markdown-CTNRp5o7.js`(372.25 kB),css-tree 与 refractor 同 chunk 求值无跨 chunk TDZ。原 `.md` Resolution 段所述的"vite.config.ts 把 markdown-vendor 规则置顶并补齐 include 项"修复已落地(`vendor-md-editor` 单独 chunk + `vendor-markdown` 闭环),且 `dist/assets/` 已无原报错的兜底 vendor chunk。`.md` 提到的"Follow-up Activity TDZ"另见 react-vendor-activity-tdz session 闭环。
action: wontfix (D-02) — 与 css-tree-vendor-chunk-tdz 同根已被覆盖
