/* eslint-disable */
// ^ 浏览器 Console 粘贴片段（dev-only）：含裸 document/getComputedStyle，
//   scripts/*.js 不在 eslint 忽略清单，文件内禁用以免击穿 lint 预算。
/**
 * v1.24 双框修复（VIS-01/02/03）computed-style 断言 —— 浏览器 Console 粘贴片段
 * =============================================================================
 * 用法（dev-only）：
 *   1. npm run dev 启动前端，登录进入任一含 antd Input（allowClear）的页面；
 *   2. 打开 DevTools Console，整段粘贴本文件除首尾注释外的代码并回车；
 *   3. 阅读输出的 JSON 报告：
 *      - doubleCount 必须 = 0（v1.24 修复前基线 = 3：浅色 Input 内层 / TextArea 内层 /
 *        暗色 Input 内层各 1，实测于 2026-10-28 静态 harness）
 *      - inner.shadow 必须 'none'，wrapperFocused.shadow 必须 ≠ 'none'
 *      - wrapperError.borderTopColor 必须 rgb(186, 54, 48)（品牌红 --ant-color-error #BA3630，
 *        由 AntdThemeBridge 显式设 colorError；暗色实测为 rgb(161, 49, 44)，仅作记录——
 *        本脚本在 light fixture 上断言）
 *
 * 口径：STACK.md (e.1)。本片段会在页面里动态注入 5 个 fixture（不触碰真实表单），
 * 执行完自动清理。暗色断言会把 fixture 挂到 data-color-mode="dark" 容器下。
 *
 * 验证记录（2026-10-28，file:// 静态 harness + antd 6.6.4 dist css + src/index.css）：
 *   修复前基线：doubleCount=3；内层 btw=1px/#dbd7ce 白底；wrapperError=#dbd7ce（F1 实证）
 *   修复后：doubleCount=0；内层 btw=0px/transparent/圆角 0/padding 0；
 *           wrapperFocused ring=#e9efeb 2px；wrapperError=rgb(186,54,48)（品牌红 #BA3630）
 */
(function () {
  // fixture 宿主必须挂进 antd cssVar scope 容器（App/ConfigProvider 树内），
  // 否则 var(--ant-color-error) 不可见 → VIS-03 还原块落到 fallback #ff4d4f，
  // 与真实 App 错误态（品牌红 #BA3630）不符。真实页面里 html/body 上无该变量。
  const cssVarScope = document.querySelector('.ant-app, [class*="css-var-"]');
  const host = document.createElement('div');
  (cssVarScope || document.body).appendChild(host);
  const light = document.createElement('div');
  light.setAttribute('data-color-mode', 'light');
  const dark = document.createElement('div');
  dark.setAttribute('data-color-mode', 'dark');
  host.append(light, dark);

  function el(tag, cls, parent) {
    const n = document.createElement(tag);
    n.className = cls;
    parent.appendChild(n);
    return n;
  }
  const w1 = el('span', 'ant-input-affix-wrapper ant-input-outlined', light);
  const in1 = el('input', 'ant-input ant-input-outlined', w1);
  el('span', 'ant-input-suffix', w1);
  const w2 = el('span', 'ant-input-affix-wrapper ant-input-outlined', light);
  const ta = el('textarea', 'ant-input ant-input-outlined', w2);
  const w3 = el('span', 'ant-input-affix-wrapper ant-input-outlined', dark);
  const in3 = el('input', 'ant-input ant-input-outlined', w3);
  const w4 = el('span', 'ant-input-affix-wrapper ant-input-outlined ant-input-affix-wrapper-focused', light);
  const in4 = el('input', 'ant-input ant-input-outlined', w4);
  const w5 = el('span', 'ant-input-affix-wrapper ant-input-outlined ant-input-status-error', light);

  const cs = (n) => getComputedStyle(n);
  const report = {
    doubleCount: [in1, ta, in3].filter((n) => cs(n).borderTopWidth !== '0px').length,
    inner: {
      borderTopWidth: cs(in1).borderTopWidth,
      background: cs(in1).backgroundColor,
      radius: cs(in1).borderTopLeftRadius,
      padding: cs(in1).paddingTop,
      shadow: cs(in1).boxShadow,
    },
    innerTextarea: { borderTopWidth: cs(ta).borderTopWidth, shadow: cs(ta).boxShadow },
    innerDark: { borderTopWidth: cs(in3).borderTopWidth, background: cs(in3).backgroundColor },
    wrapperFocused: { shadow: cs(w4).boxShadow },
    wrapperError: { borderTopColor: cs(w5).borderTopColor },
    passed:
      [in1, ta, in3].every((n) => cs(n).borderTopWidth === '0px') &&
      cs(in1).boxShadow === 'none' &&
      cs(in4).boxShadow === 'none' &&
      cs(w4).boxShadow !== 'none' &&
      /186,\s*54,\s*48/.test(cs(w5).borderTopColor),
  };
  host.remove();
  console.log('[css-contract-assertions]', JSON.stringify(report, null, 1));
  return report;
})();
