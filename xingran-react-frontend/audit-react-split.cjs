const fs = require("fs");
const path = require("path");
const distDir = path.join(__dirname, "dist", "assets");
const reactChunk = fs.readFileSync(path.join(distDir, "vendor-react-AuwjJ4Ro.js"), "utf8");

// Check if vendor-react itself imports from anywhere else (it should only export react APIs)
const importRe = /import\s*\{([^}]+)\}\s*from\s*["']([^"']+)["']/g;
const imports = [];
let m;
while ((m = importRe.exec(reactChunk)) !== null) {
  imports.push({ names: m[1].trim().slice(0, 100), from: m[2] });
}
console.log("=== vendor-react imports ===");
for (const i of imports) console.log("  from " + i.from + " → " + i.names);
console.log("Total:", imports.length);

const commonsChunk = fs.readFileSync(path.join(distDir, "vendor-commons-CJE7cZRG.js"), "utf8");
const commonsImports = [];
const re2 = /import\s*\{([^}]+)\}\s*from\s*["']([^"']+)["']/g;
while ((m = re2.exec(commonsChunk)) !== null) {
  commonsImports.push({ names: m[1].trim().slice(0, 80), from: m[2] });
}
console.log("\n=== vendor-commons imports (where does it get React?) ===");
for (const i of commonsImports.slice(0, 5)) console.log("  from " + i.from + " → " + i.names);

// How many libs in vendor-commons import React?
const reactImportRe =
  /import\s*\{[^}]*\b[a-zA-Z]+\b[^}]*\}\s*from\s*["'][^"']*react(?:-dom)?(?:\/[^"']*)?["']/g;
let reactImportCount = 0;
while (reactImportRe.exec(commonsChunk)) reactImportCount++;
console.log("\n=== react package import statements in vendor-commons ===");
console.log(
  "  " + reactImportCount + ' import statements directly from "react"/"react-dom" packages'
);
