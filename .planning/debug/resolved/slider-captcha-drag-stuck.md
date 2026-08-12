---
slug: slider-captcha-drag-stuck
status: resolved
trigger: 滑动验证码滑动时不流畅,偶尔会卡住——鼠标在动但滑块不跟着动
created: 2026-06-21
updated: 2026-06-21 (iter2: 滑不到最右 + 坐标尺度错位)
session_type: investigation
goal: find_and_fix
---

# Debug Session: 滑动验证码拖动中途卡住

## Symptoms

### Expected Behavior
- 在登录页拖动滑块(`.slider-button`)时,鼠标移动滑块应实时跟手,松手后提交验证
- 拖动过程应全程流畅,无中断

### Actual Behavior(用户确认)
- **拖动中途突然定住**:滑块拖到一半时,鼠标继续移动但滑块不再跟手
- **松开鼠标/停顿后才恢复**:符合"事件流被抢占,松手(dragend)后才归还"的特征
- **偶尔出现**(竞态特征,非每次必现)
- 浏览器:**Chrome**

### Error Messages
- 无控制台报错(纯交互时序问题)
- 已知日志:`SliderCaptcha.tsx` 在验证流程有 `console.log` 输出,但卡住发生在验证(mouseup)之前

### Timeline
- **2026-06-21**:用户报告该问题
- 历史是否一直存在:未知(怀疑是既有缺陷,非回归)

### Reproduction(待 gsd-debugger 实测确认)
1. 前置:`sys_config` 中验证码类型需为 `slider`(参考 memory `captcha-enabled-invalid-value-trap`,非 `slider` 时不显示滑块组件)
2. 导航到登录页,触发安全验证 Modal(`CaptchaModal`)
3. 按住滑块快速拖动,拖动轨迹**经过上方的背景图区域**(`.slider-bg` `<img>`)
4. 观察:中途滑块是否定住,松手是否恢复

## Current Focus

- **hypothesis**(确认):**主根因 = 假设 1**——`handleMouseDown` 无 `e.preventDefault()` + `slider-bg` 缺 `draggable={false}`,Chrome 在光标经过背景图时启动原生图片拖拽,document 的 `mousemove` 被替换为 `dragstart`/`drag`,滑块定住,松手(`dragend`)后事件流恢复。
- **dynamic_verification**:chrome-devtools MCP 不在本会话可用;根据 orchestrator 指示("静态证据已足够强,不必 block 在动态复现"),跳过动态验证,直接应用修复。静态证据链完整且互锁(handleMouseDown 无 preventDefault + slider-bg 无 draggable=false + .slider-bg 交互式可点击 + 用户三特征完全吻合)。
- **reasoning_checkpoint**:
  - hypothesis: handleMouseDown 缺 preventDefault + slider-bg 无 draggable=false,导致 Chrome 原生图片拖拽抢占 document mousemove,滑块定住
  - confirming_evidence: L58-61 handleMouseDown 无 event 参数无 preventDefault;L135-143 slider-bg 缺 draggable=false;L150 slider-piece 有 draggable=false 形成对照;CSS slider-bg 交互式可点击无 pointer-events:none;用户三特征(中途定住/松手恢复/偶尔)与原生 dragstart→dragend 语义精确吻合
  - falsification_test: 若加 preventDefault + draggable=false 后拖动不再卡住,则根因确认;若仍卡住,则需重新评估假设 3(pointer capture)
  - fix_rationale: 方案 A(preventDefault+draggable=false)直接消除原生拖拽触发条件,方案 B(Pointer Events + setPointerCapture + rAF)从根本重构事件流免疫所有抢占场景。A+B 组合同时根治假设 1+2+3
  - blind_spots: 未动态复现;未测试触屏设备;maxX=240 硬编码假设轨道宽度恒定(迁移时已保留)
- **next_action**: 应用方案 A+B 到 SliderCaptcha.tsx → npm run build → npm run lint → 报告

## 根因假设(按置信度排序)

### 假设 1(主根因,高置信度):原生拖拽抢占 mousemove 流
**机制**:
- `SliderCaptcha.tsx:58-61` `handleMouseDown` **没有接收 event 参数,也没有 `e.preventDefault()`**:
  ```js
  const handleMouseDown = useCallback(() => {  // ← 无 event,无 preventDefault
    if (verified) return;
    setDragging(true);
  }, [verified]);
  ```
- `SliderCaptcha.tsx:135-143` 背景图 `slider-bg` **未设 `draggable={false}`**(对比 L150 的 `slider-piece` 有设)
- 组合后果:用户按住滑块拖动时,光标一旦移动经过背景图 `<img>`,Chrome 判定为原生图片拖拽 → 触发 `dragstart` → **document 的 `mousemove` 停止派发** → 滑块定住 → 松手触发 `dragend` → 恢复
- "偶尔"=只有拖动轨迹恰好经过背景图时才触发

### 假设 2(次要,中置信度):高频 setState 无 rAF 节流导致渲染堆积
**机制**:`handleMouseMove`(L100-110) 每次 mousemove 都 `getBoundingClientRect()`(强制 reflow) + `setCurrentX(x)`(触发重渲染),无 `requestAnimationFrame` 节流。快速拖动时渲染队列堆积,视觉滞后。
**为何次要**:用户主诉是"定住不跟手"而非"延迟/抖动",且假设 1 更符合"松手才恢复"。

### 假设 3(次要,中置信度):无 pointer capture,鼠标移出窗口时事件丢失
**机制**:全局监听用 `document.addEventListener("mousemove", ...)`,未用 Pointer Events + `setPointerCapture`。鼠标快速移出浏览器窗口或越过 iframe 时,document mousemove 停止。
**为何次要**:也符合"松手才恢复"(鼠标回到窗口),但不如假设 1 精确(假设1解释了"经过背景图"这个具体诱因)。

### 假设 4(低置信度):useEffect 依赖链导致拖拽中监听器重绑
**机制**:`useEffect`(L113-128) 依赖 `handleMouseUp`,其闭包捕获 `onVerified`/`captchaData`。若父组件 `CaptchaModal` 在拖拽中重渲染,监听器被 cleanup→re-add,瞬间丢失事件。
**为何低置信**:拖拽中 CaptchaModal 的 state(`captchaValue/captchaId/sliderVerified`)不变,不会重渲染。但 `onSuccess` 来自登录页,若登录页重渲染透传新引用则可能。属脆弱设计,值得加固。

## Evidence

- timestamp: 2026-06-21
  source: code_analysis
  finding: |
    `SliderCaptcha.tsx:58-61` `handleMouseDown` 既未接收 event 也未调用 preventDefault:
    ```js
    const handleMouseDown = useCallback(() => {
      if (verified) return;
      setDragging(true);
    }, [verified]);
    ```
    这是拖拽交互的标准反模式——必须在 mousedown 调用 preventDefault 阻止原生选择/拖拽默认行为。

- timestamp: 2026-06-21
  source: code_analysis
  finding: |
    `SliderCaptcha.tsx:135-143` 背景图未设 draggable=false:
    ```jsx
    <img src={captchaData.sliderImg} alt="滑动验证码" className="slider-bg"
         onClick={loadCaptcha} title="点击刷新验证码" />
    ```
    而拼图块 L150 有 `draggable={false}`。背景图缺失是触发原生图片拖拽的直接条件。

- timestamp: 2026-06-21
  source: code_analysis
  finding: |
    `SliderCaptcha.tsx:100-110` `handleMouseMove` 每次都 getBoundingClientRect + setCurrentX,无 rAF 节流:
    ```js
    const handleMouseMove = useCallback((e: MouseEvent) => {
      const rect = sliderRef.current.getBoundingClientRect();  // 强制 reflow
      let x = e.clientX - rect.left;
      x = Math.max(0, Math.min(x, 240));
      dragStateRef.current.currentX = x;
      setCurrentX(x);  // 高频重渲染
    }, []);
    ```

- timestamp: 2026-06-21
  source: code_analysis
  finding: |
    `SliderCaptcha.tsx:113-128` 用 document mousemove/mouseup 监听,未用 Pointer Events + setPointerCapture:
    ```js
    useEffect(() => {
      if (!dragging) return;
      document.addEventListener("mousemove", mouseMoveListener);
      document.addEventListener("mouseup", mouseUpListener);
      return () => { document.removeEventListener("mousemove", mouseMoveListener);
                     document.removeEventListener("mouseup", mouseUpListener); };
    }, [dragging, handleMouseMove, handleMouseUp]);
    ```

- timestamp: 2026-06-21
  source: user_symptom
  finding: |
    用户描述三特征高度吻合假设 1:
    1. "拖动中途突然定住" = mousemove 被 dragstart 替换
    2. "松开鼠标才恢复" = dragend 后事件流归还
    3. "偶尔出现" = 仅轨迹经过背景图时触发

## Eliminated

(暂无,待 gsd-debugger 动态验证后填充)

## 修复方案(待用户确认后实施)

### 方案 A:最小修复(解决主根因假设 1)— 必做
1. `handleMouseDown` 改签名为 `(e: React.MouseEvent<HTMLDivElement>)`,首行 `e.preventDefault()`
2. `slider-bg` `<img>` 加 `draggable={false}`

### 方案 B:根治(现代最佳实践)— 强烈推荐
3. 改用 Pointer Events:`onPointerDown` + `e.currentTarget.setPointerCapture(e.pointerId)` + `onPointerMove` + `onPointerUp`
   - pointer capture 保证拖动期间持续接收事件,免疫鼠标移出窗口/越过 iframe/原生行为抢占(同时根治假设 1 和假设 3)
4. `pointermove` 用 `requestAnimationFrame` 节流 setState,避免假设 2 的渲染堆积

### 方案 C:可选加固
5. 拖动期间直接操作 DOM `transform`(via ref),跳过 React 渲染,达到 60fps 丝滑
6. `handleMouseUp` 用 ref 包装,从 useEffect 依赖移除,消除假设 4 的重绑风险

**推荐组合:A + B**(preventDefault/draggable 保兼容兜底,Pointer Events 从根本解决事件流,rAF 解决流畅度)。
注意 `maxX = 240` 是写死的轨道宽度,迁移到 Pointer Events 时需保留该 clamp 逻辑。

## Verification(修复后)
1. `npm run build` 通过(无 TS 错误)
2. `npm run lint` 通过
3. chrome-devtools 实测:快速拖动 + 轨迹经过背景图,滑块全程跟手不卡
4. 注入诊断器确认拖动中无 `dragstart` 触发,`mousemove`(或 pointermove)持续派发
5. 正常验证流程通过(松手后能成功验证)

## Resolution

### Root Cause
**假设 1 确认为主根因**(静态证据互锁 + 用户三特征精确吻合;动态验证因 chrome-devtools MCP 不可用而跳过,经 orchestrator 授权)。
`SliderCaptcha.tsx` 的 `handleMouseDown` 缺 `e.preventDefault()`,背景图 `slider-bg` 缺 `draggable={false}`。Chrome 在拖动滑块时光标经过背景图触发原生图片拖拽,document 的 `mousemove` 被替换为 `dragstart`/`drag`,滑块定住;松手触发 `dragend`,事件流恢复。次要贡献假设 2(无 rAF 节流致渲染堆积)与假设 3(无 pointer capture 致鼠标移出窗口时事件丢失)在同一修复中被一并根治。

### Fix Applied
**方案 A + 方案 B 合并实施**:
1. **方案 A(兼容兜底)**: slider-bg `<img>` 加 `draggable={false}`(消除原生图片拖拽触发条件)
2. **方案 B(根治)**:
   - 迁移到 Pointer Events:`onPointerDown` / `onPointerMove` / `onPointerUp` / `onPointerCancel`
   - `e.currentTarget.setPointerCapture(e.pointerId)` 锁定指针,拖动期间持续接收事件,免疫鼠标移出窗口/越过 iframe/原生行为抢占
   - `e.preventDefault()` 在 pointerdown 阻止原生拖拽默认行为
   - `requestAnimationFrame` 节流 `setCurrentX`,避免高频 getBoundingClientRect + setState 渲染堆积
   - pointerup 时 flush 最后一帧位置(canelAnimationFrame + setState pendingX)避免视觉跳变
3. **CSS 加固**: `.slider-button` 加 `touch-action: none`(触屏设备等价防护,阻止浏览器手势抢占 pointermove)
4. **保留原 verify-on-mouseup 流程**: `handleMouseUp`(现已重命名为内部逻辑)仍由 pointer up 触发,`dragStateRef.currentX` + `onVerified(token, captchaId)` 调用链不变
5. **保留 maxX = 240 clamp 逻辑**(提取为模块常量 `MAX_X`)
6. **兜底 onMouseDown preventDefault**: 老旧浏览器无 Pointer Events 时,至少阻止原生拖拽默认行为

### Files Changed
- `xingran-react-frontend/src/components/captcha/SliderCaptcha.tsx`(主要改动)
- `xingran-react-frontend/src/components/captcha/SliderCaptcha.css`(touch-action: none 加固)

### Verification

**自动化验证**(已执行):
- `npm run build`: 通过(无 TS 错误,1m39s 完成全量产物)
- `npx eslint src/components/captcha/SliderCaptcha.tsx`: 2 errors + 5 warnings,经 git stash 对照确认**全部为预先存在问题**(`App.useApp()` 的 message 假阳性 + 原作者遗留的 console.log),本次改动**零新增 lint 问题**;codebase 全量 lint 基线仍为 3790 problems 不变
- 无 TS 类型错误(PointerEvent 类型从 `react` 导入为 `PointerEvent as ReactPointerEvent`)

**动态/人工验证**(待用户确认):
- 未动态复现(chrome-devtools MCP 不可用)
- 待人工确认:登录页快速拖动滑块 + 轨迹经过背景图,滑块全程跟手不卡;松手验证流程正常通过

### Next Action
- 返回 `## DEBUG COMPLETE` 报告
- 等待用户人工验证反馈
- 未提交代码(遵循用户"提交前需确认"约定)

---

## 迭代 2: 滑块滑不到最右 + 坐标尺度错位(2026-06-21)

### 用户反馈
迭代 1 修复后,卡顿问题大幅改善。但出现新问题:**滑块滑不到最右**,若缺口在最右边可能导致无法验证。

### 根因分析(两个叠加 bug)

**B1(主诉,UI clamp 硬编码)**:
- 迭代 1 保留的 `MAX_X = 240`(源自原代码 `const maxX = 240`)作为 UI clamp 限制了滑块 `left` 最大值
- 但 `.slider-canvas-wrapper` / `.slider-track` 是 `width: 100%`(antd Modal 500px 内 ≈452px),实际可拖动范围应是 ~402px
- 240 << 402,滑块拖到 240px 就被 clamp,视觉上停在容器中部偏左,到不了最右
- 注:此 bug 修复前就存在,只是迭代 1 前卡顿严重、滑块根本拖不远才没暴露

**B2(更深的隐患,显示坐标 ≠ 原始坐标)**:
- 后端 `pkg/captcha/slider.go` 在**固定 400×200 原始坐标系**生成缺口:
  - `Width=400, Height=200, PieceWidth=80`(默认难度)
  - `xPos ∈ [PieceWidth+20, Width-PieceWidth*2]` = `[100, 240]`(默认);困难难度 PieceWidth=70 → 最大 260
  - 验证容差仅 8 像素(`internal/core/captcha.go:428`,且难度对应的 generator tolerance 未传到 VerifySliderCaptcha)
- 前端容器响应式缩放显示(≈452px ≠ 400px),原代码直接把**显示坐标** `e.clientX - rect.left` 作为 `xPos` 提交,与后端**原始坐标**系统性错位
- 算例:缩放比 400/452≈0.885,缺口原始 240 → 显示 294,用户拖到 294 视觉对齐,但提交 294(显示)而后端期望 240(原始),差 54 ≫ 8 容差 → 验证失败

### 修复方案(迭代 2)

1. **B1**: 移除硬编码 `MAX_X=240`,改为动态 `maxDisplayX = rect.width - SLIDER_BUTTON_WIDTH(50)`,滑块可拖到容器最右
2. **B2**: 提交前做坐标缩放转换 `xPos = round(displayX * naturalWidth / clientWidth)`,用 `<img>.naturalWidth`(=400,后端原始宽度)自适应还原缩放比——不硬编码 400,后端将来改尺寸也兼容
3. 新增 `bgImgRef` 引用背景图以读取 `naturalWidth`;`naturalWidth=0`(图片未 decode)时退回显示坐标(极少见竞态)

### 迭代 2 改动文件
- `xingran-react-frontend/src/components/captcha/SliderCaptcha.tsx`:5 处编辑
  - 移除 `MAX_X=240` 常量,新增 `SLIDER_BUTTON_WIDTH=50` + `bgImgRef`
  - `handlePointerMove`:动态 clamp `rect.width - SLIDER_BUTTON_WIDTH`
  - `handleMouseUp`:`displayX → xPos` 缩放转换后提交
  - 背景 `<img>` 加 `ref={bgImgRef}`

### 迭代 2 验证(已执行)
- `npm run type-check`:通过(无 TS 错误)
- `npm run build`:通过(35.82s,无错误)

### 迭代 2 待人工验证
- 滑块能拖到最右(不再卡在 240px)
- 拖到缺口位置松手,验证应正常通过(坐标缩放转换生效)
- 缺口在最右边时也能完成验证

### 未改后端(范围克制)
- 后端 generator 已在 400×200 固定坐标系正确工作,容差 8px 合理
- 未改后端 API(如返回原始尺寸)——前端用 `naturalWidth` 自治适配,避免契约改动蔓延
- 已知小遗留:困难难度下 generator tolerance=8 未传入 `VerifySliderCaptcha`(后者硬编码 8),两者恰好一致,暂不影响
