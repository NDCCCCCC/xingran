const fs = require("fs");
const path = require("path");
const distDir = path.join(__dirname, "dist", "assets");

// Build transitive import graph
function getImports(file) {
  const data = fs.readFileSync(path.join(distDir, file), "utf8");
  const re = /import\(\s*["']([^"']+)["']\s*\)/g;
  const reStatic = /import\s*\{[^}]+\}\s*from\s*["']([^"']+)["']/g;
  const targets = new Set();
  let m;
  while ((m = re.exec(data)) !== null) targets.add(m[1].replace(/^\.\//, ""));
  while ((m = reStatic.exec(data)) !== null) targets.add(m[1].replace(/^\.\//, ""));
  return Array.from(targets);
}

function dfs(start, visited = new Set(), depth = 0, parent = null) {
  if (visited.has(start)) return;
  visited.add(start);
  const imports = getImports(start);
  console.log("  ".repeat(depth) + start + " (from " + parent + ")");
  for (const i of imports) {
    if (i.includes("vendor-") || i.includes("index-")) {
      dfs(i, visited, depth + 1, start);
    }
  }
}

console.log("=== vendor-utils import graph ===");
dfs("vendor-utils-BBYKONI5.js");
