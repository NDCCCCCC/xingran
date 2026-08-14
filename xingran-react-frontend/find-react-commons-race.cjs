const fs = require("fs");
const path = require("path");
const distDir = path.join(__dirname, "dist", "assets");

function getImports(file) {
  const data = fs.readFileSync(path.join(distDir, file), "utf8");
  const reDyn = /import\(\s*["']([^"']+)["']\s*\)/g;
  const reStatic = /import\s*\{[^}]+\}\s*from\s*["']([^"']+)["']/g;
  const targets = [];
  let m;
  while ((m = reDyn.exec(data)) !== null) targets.push(m[1].replace(/^\.\//, ""));
  while ((m = reStatic.exec(data)) !== null) targets.push(m[1].replace(/^\.\//, ""));
  return Array.from(new Set(targets));
}

function dfs(file, depth = 0, path = []) {
  const imports = getImports(file);
  const newPath = [...path, file];
  // Check for the suspect pair
  const hasReact = imports.some((i) => i.includes("vendor-react"));
  const hasCommons = imports.some((i) => i.includes("vendor-commons"));
  if (hasReact && hasCommons) {
    console.log("!!! " + file + " directly imports BOTH vendor-react AND vendor-commons");
    console.log(
      "    react:",
      imports.filter((i) => i.includes("vendor-react"))
    );
    console.log(
      "    commons:",
      imports.filter((i) => i.includes("vendor-commons"))
    );
  }
  for (const i of imports) {
    if (i.includes("vendor-") || i.includes("index-")) {
      dfs(i, depth + 1, newPath);
    }
  }
}

console.log("=== Searching for files that import BOTH vendor-react AND vendor-commons ===");
dfs("index-B0GEnYSh.js");
