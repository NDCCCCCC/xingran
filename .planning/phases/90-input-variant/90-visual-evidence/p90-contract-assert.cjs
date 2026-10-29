// Phase 90 W2/W3 —— 权威 computed-style 断言执行器
//
// 口径来源：`ruoyi-react-frontend/scripts/css-contract-assertions.js`（v1.24 VIS-01/02/03 的
// 权威断言片段）。本脚本**逐字复用**它的 fixture 构造与判定式，只把
// 「Console 粘贴 → 人工读 JSON」改为「CDP 求值 → 写盘 → 机器判定」。
//
// ⚠️ 该断言片段的 fixture **自带 light / dark 两个子容器**（`data-color-mode` 属性），
//    ��此它**一次运行即可同时覆盖明暗两侧** —— 这是它比 p90-assert-probe.cjs 更权威的原因
//    （后者依赖页面自身的模式，登录态不可得时无法覆盖暗色）。
//
// 用法：node p90-contract-assert.cjs <输出文件名>
const fs = require("fs");
const path = require("path");
const HERE = __dirname;
const FE_ROOT = path.resolve(HERE, "../../../../ruoyi-react-frontend");
const { openCdp, sleep } = require("./p90-session.cjs");

// 权威断言片段（去掉 IIFE 包裹与 console.log，改为返回 report；判定式逐字保留）
// ⚠️ 剥离方式：按**首个** `(function () {` 与**末尾** `})();` 定位，
//    而不是用 replace(/^.../) —— 该文件的注释块里有括号，正则不可靠。
function extractContractBody(src) {
  const start = src.indexOf("(function () {");
  const end = src.lastIndexOf("})();");
  if (start < 0 || end < 0) throw new Error("无法定位权威断言片段的 IIFE 边界");
  return src.slice(start + "(function () {".length, end);
}
const CONTRACT_BODY = extractContractBody(
  fs
    .readFileSync(path.join(FE_ROOT, "scripts/css-contract-assertions.js"), "utf8")
    .replace(/\/\* eslint-disable \*\//, "")
    .replace(/console\.log\('\[css-contract-assertions\]', JSON\.stringify\(report, null, 1\)\);/, ""),
);

(async () => {
  const outFile = process.argv[2] || "probe-contract.json";
  const { ev, nav, close } = await openCdp();
  try {
    // 断言片段需要页面里存在 antd 的 cssVar scope（登录页即有）
    await nav("/login", 5000);
    const scopeOk = await ev(
      `!!document.querySelector('.ant-app, [class*="css-var-"]')`,
    );
    if (!scopeOk) throw new Error("无 antd cssVar scope —— 断言片段的 fixture 会落到 fallback 值");

    const report = await ev(`(function(){ ${CONTRACT_BODY} })()`);

    // 明暗各自实测（断言片段自带双容器 → 从 fixture 补采两侧的 border 色）
    const sides = await ev(`(function(){
      var cssVarScope = document.querySelector('.ant-app, [class*="css-var-"]');
      var host = document.createElement('div');
      (cssVarScope || document.body).appendChild(host);
      function mk(mode, cls){
        var d = document.createElement('div');
        d.setAttribute('data-color-mode', mode);
        host.appendChild(d);
        var n = document.createElement('span');
        n.className = cls;
        d.appendChild(n);
        return n;
      }
      var wErrL = mk('light','ant-input-affix-wrapper ant-input-outlined ant-input-status-error');
      var wErrD = mk('dark','ant-input-affix-wrapper ant-input-outlined ant-input-status-error');
      var inpL = mk('light','ant-input ant-input-outlined');
      var inpD = mk('dark','ant-input ant-input-outlined');
      var numErrL = document.createElement('div');
      numErrL.className = 'ant-input-number ant-input-number-outlined ant-input-number-status-error';
      numErrL.innerHTML = '<div class="ant-input-number-input-wrap"><input class="ant-input-number-input" value="1"></div>';
      var numL = document.createElement('div'); numL.setAttribute('data-color-mode','light'); numL.appendChild(numErrL);
      host.appendChild(numL);
      var numErrD = document.createElement('div');
      numErrD.className = 'ant-input-number ant-input-number-outlined ant-input-number-status-error';
      numErrD.innerHTML = '<div class="ant-input-number-input-wrap"><input class="ant-input-number-input" value="1"></div>';
      var numD = document.createElement('div'); numD.setAttribute('data-color-mode','dark'); numD.appendChild(numErrD);
      host.appendChild(numD);
      var cs = n => getComputedStyle(n);
      var out = {
        lightInputErrorBorder: cs(wErrL).borderTopColor,
        darkInputErrorBorder: cs(wErrD).borderTopColor,
        lightInputBorder: cs(inpL).borderTopColor,
        darkInputBorder: cs(inpD).borderTopColor,
        lightInputNumberErrorBorder: cs(numErrL).borderTopColor,
        darkInputNumberErrorBorder: cs(numErrD).borderTopColor,
        lightInputRadius: cs(inpL).borderTopLeftRadius,
        darkInputRadius: cs(inpD).borderTopLeftRadius,
        lightInputFontSize: cs(inpL).fontSize,
        darkInputFontSize: cs(inpD).fontSize
      };
      host.remove();
      return JSON.stringify(out);
    })()`);

    const payload = {
      capturedAt: new Date().toISOString(),
      contract: report,
      sides: JSON.parse(sides),
    };
    fs.writeFileSync(path.join(HERE, outFile), JSON.stringify(payload, null, 1));
    console.log(JSON.stringify(payload, null, 1));
    console.log(`\n→ ${outFile}`);
    console.log(`passed = ${report.passed}`);
  } finally {
    close();
  }
  process.exit(0);
})().catch((e) => {
  console.error("FATAL", e);
  process.exit(1);
});
