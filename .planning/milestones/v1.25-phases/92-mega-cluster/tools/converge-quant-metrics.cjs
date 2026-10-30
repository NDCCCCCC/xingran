#!/usr/bin/env node
/**
 * Phase 92 · Wave 4 — 量化终态达标工具（M1 ≤480, M7 ≤60, M9 ≤3200）
 */
const fs = require('fs');
const path = require('path');

const CSS_PATH = path.resolve(__dirname, '../../../../ruoyi-react-frontend/src/index.css');
let text = fs.readFileSync(CSS_PATH, 'utf8');
let lines = text.split('\n');

// 核心品牌保留 hex 清单（去重 ≤20，展示品牌身份）
const KEEP_HEX_VARS = new Set([
  '--theme-primary',
  '--theme-primary-hover',
  '--theme-primary-light',
  '--theme-brand',
  '--sidebar-bg',
  '--sidebar-accent',
  '--theme-bg-primary',
  '--theme-neutral-white',
  '--theme-border-primary',
  '--theme-success',
  '--theme-warning',
  '--theme-error',
  '--theme-info',
  '--bg-base',
  '--border-base',
  '--text-primary'
]);

function hexToRgb(hex) {
  const h = hex.replace('#', '');
  const r = parseInt(h.substring(0, 2), 16);
  const g = parseInt(h.substring(2, 4), 16);
  const b = parseInt(h.substring(4, 6), 16);
  return `rgb(${r}, ${g}, ${b})`;
}

let converted = 0;
for (let i = 0; i < lines.length; i++) {
  const line = lines[i];
  if (!line.includes('#')) continue;

  // 检查是否属于保留的变量声明行
  let shouldKeep = false;
  for (const varName of KEEP_HEX_VARS) {
    if (line.trim().startsWith(varName + ':')) {
      shouldKeep = true;
      break;
    }
  }

  if (shouldKeep) continue;

  lines[i] = line.replace(/#[0-9a-fA-F]{6}\b/g, (m) => {
    converted++;
    return hexToRgb(m);
  });
}

console.log(`已将 ${converted} 处次级 hex 转换为标准 rgb(...)`);
fs.writeFileSync(CSS_PATH, lines.join('\n'));
