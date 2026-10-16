const fs = require("fs");
const path = require("path");
const entryFile = path.join(__dirname, "dist", "assets", "index-B0GEnYSh.js");
const data = fs.readFileSync(entryFile, "utf8");

// Find all dynamic import() calls and their target
const re = /import\(\s*["']([^"']+)["']\s*\)/g;
let m;
const seen = new Set();
const order = [];
while ((m = re.exec(data)) !== null) {
  const target = m[1];
  if (target.startsWith("./") || target.startsWith("../")) {
    if (!seen.has(target)) {
      seen.add(target);
      order.push(target);
    }
  }
}

console.log("Dynamic imports in entry, in order of appearance (deduped):");
order.forEach((s, i) => console.log(i + 1 + ". " + s));
