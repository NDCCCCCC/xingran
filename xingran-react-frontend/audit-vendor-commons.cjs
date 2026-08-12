const fs = require('fs');
const path = require('path');
const distDir = path.join(__dirname, 'dist', 'assets');
const file = path.join(distDir, 'vendor-commons-CJE7cZRG.js');
const data = fs.readFileSync(file, 'utf8');

const apis = [
  'createContext', 'createElement', 'cloneElement', 'isValidElement',
  'useState', 'useEffect', 'useLayoutEffect', 'useReducer', 'useRef',
  'useMemo', 'useCallback', 'useContext', 'useImperativeHandle', 'useDebugValue',
  'forwardRef', 'memo', 'lazy', 'Suspense', 'Fragment', 'Component',
  'createRef', 'Children'
];

const counts = {};
for (const api of apis) {
  // Match X.api where X is any short identifier
  const re = new RegExp('\\b[a-zA-Z_$][a-zA-Z0-9_$]{0,4}\\.\\b' + api + '\\b', 'g');
  let c = 0;
  while (re.exec(data)) c++;
  if (c > 0) counts[api] = c;
}

console.log('React API call sites in vendor-commons-CJE7cZRG.js:');
const sorted = Object.entries(counts).sort((a, b) => b[1] - a[1]);
for (const [k, v] of sorted) console.log('  ' + k + ': ' + v);
console.log('Total React-namespace accesses:', sorted.reduce((s, [_, v]) => s + v, 0));

// Also: which distinct binding identifiers appear before each API?
console.log('\nDistinct binding identifiers accessing React APIs in vendor-commons:');
const bindingSet = new Set();
for (const api of apis) {
  const re = new RegExp('\\b([a-zA-Z_$][a-zA-Z0-9_$]{0,4})\\.\\b' + api + '\\b', 'g');
  let m;
  while ((m = re.exec(data)) !== null) {
    bindingSet.add(m[1] + '.' + api);
  }
}
console.log('  total distinct binding.api pairs:', bindingSet.size);

// Top-level statements: find patterns of `var X=...useXXX` etc.
// Look for top-level use of createContext: `const X = Y.createContext(...)`
const topCreate = data.match(/^[A-Za-z_$]{0,4}=[A-Za-z_$]{0,4}\.createContext\(/gm);
console.log('\nTop-level createContext assignments:', topCreate ? topCreate.length : 0);
if (topCreate) console.log('  examples:', topCreate.slice(0, 5));

// Top-level useLayoutEffect assignments
const topLayout = data.match(/^[A-Za-z_$]{0,4}=[A-Za-z_$]{0,4}\.useLayoutEffect\(/gm);
console.log('Top-level useLayoutEffect assignments:', topLayout ? topLayout.length : 0);
if (topLayout) console.log('  examples:', topLayout.slice(0, 5));