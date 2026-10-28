/* eslint-disable */
// ^ dev-only 工具脚本：scripts/*.mjs 被 eslint 全局忽略，本文件按任务规格命名为 .js，
//   故以文件内禁用保持 lint 预算（errors≤17）不被本文件击穿。
/**
 * v1.24 双框修复（VIS-01/02/03）CSS 契约静态检查 —— dev-only，node 运行、零依赖
 * ==============================================================================
 * 用法：node scripts/css-contract-check.js   （退出码 0 = 契约完好）
 *
 * 职责分工（STACK.md (e.1) 口径）：
 *   - 本脚本：静态结构契约（规则存在性 / 禁用形态 / 回归陷阱守护），CI 可跑；
 *   - scripts/css-contract-assertions.js：computed-style 动态断言（浏览器 Console
 *     粘贴执行）。jsdom 无法解析压缩版 antd.css 的 @layer/嵌套规则（实测数百条
 *     "Could not parse CSS stylesheet"），computed-style 断言必须用真浏览器。
 *
 * 守护项：
 *   C1  affix-wrapper 内层复位规则存在且不写 box-shadow（焦点环归 wrapper，VIS-02）
 *   C2  禁止 `:not(组合子)` 形态（CSS3 下整条规则静默丢弃）
 *   C3  回归陷阱：3 处局部补丁（login.css:351 / index.css dept-tree-search /
 *       user-form-input）不得删除
 *   C4  回归陷阱：`.ant-input:focus` 焦点环、wrapper 边框、wrapper 焦点环规则不得删除
 *   C5  VIS-03 status 边框色还原规则存在
 */
import { readFileSync, existsSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const feRoot = resolve(here, '..');
const indexCss = readFileSync(resolve(feRoot, 'src/index.css'), 'utf8');
const loginCssPath = resolve(feRoot, 'src/pages/login/login.css');
const loginCss = existsSync(loginCssPath) ? readFileSync(loginCssPath, 'utf8') : '';

let failures = 0;
const check = (name, ok, detail) => {
  if (!ok) failures++;
  console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}${detail ? '  → ' + detail : ''}`);
};

// C1 内层复位规则（VIS-01）：存在、含 6 项复位、不含 box-shadow
const resetRe = /\.ant-input-affix-wrapper\s*>\s*input\.ant-input,\s*\n\.ant-input-affix-wrapper\s*>\s*textarea\.ant-input\s*\{([\s\S]*?)\n\}/;
const resetMatch = indexCss.match(resetRe);
check('C1a 内层复位规则存在', !!resetMatch);
if (resetMatch) {
  const block = resetMatch[0];
  for (const prop of ['border: none !important', 'border-radius: 0 !important', 'background: transparent !important', 'color: inherit !important', 'padding: 0 !important']) {
    check(`C1  复位含 ${prop}`, block.includes(prop));
  }
  check('C1b 复位规则不写 box-shadow', !/box-shadow/.test(block), '焦点环归 wrapper（VIS-02）');
  check('C1c 复位覆盖 textarea 内层', /\.ant-input-affix-wrapper\s*>\s*textarea\.ant-input/.test(indexCss));
}

// C2 禁止 :not(组合子)
check('C2 无 :not(含空格组合子) 形态', !/:not\([^)]*[\s>+~][^)]*\)/.test(indexCss));

// C3 局部补丁守护
check('C3a login.css affix 内层补丁保留', /\.ant-input-affix-wrapper\s*>\s*input\.ant-input/.test(loginCss));
check('C3b dept-tree-search 补丁保留', /\.dept-tree-search \.ant-input-affix-wrapper \.ant-input/.test(indexCss));
check('C3c user-form-input 补丁保留', /\.user-form-input\.ant-input-affix-wrapper \.ant-input/.test(indexCss));

// C4 焦点/边框来源守护
check('C4a .ant-input:focus 焦点环保留', /\.ant-input:focus[\s\S]{0,300}box-shadow:\s*0 0 0 2px var\(--theme-primary-lighter\)/.test(indexCss));
check('C4b wrapper 边框规则保留', /\.ant-input-affix-wrapper\s*\{[\s\S]*?border:\s*1px solid var\(--input-border\)/.test(indexCss));
check('C4c wrapper 焦点环规则保留', /\.ant-input-affix-wrapper-focused\s*\{[\s\S]*?box-shadow/.test(indexCss));

// C5 VIS-03 status 边框色还原
check('C5 status-error 边框色还原存在', /\.ant-input-affix-wrapper\.ant-input-affix-wrapper-status-error/.test(indexCss));

console.log('----');
console.log(`结果: ${failures === 0 ? 'ALL PASS' : failures + ' 项 FAIL'}`);
process.exit(failures === 0 ? 0 : 1);
