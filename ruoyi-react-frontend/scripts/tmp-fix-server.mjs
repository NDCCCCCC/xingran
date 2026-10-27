// server handleTableChange 多列排序签名修复（用后删）
import { readFileSync, writeFileSync } from "node:fs";
const p = "src/pages/monitor/server/index.tsx";
const lines = readFileSync(p, "utf8").split("\n");
// 找到 "sorter: SorterResult<ServerInfo>" 行（idx），重写从该行到函数结尾的块
const start = lines.findIndex((l) => l.includes("sorter: SorterResult<ServerInfo>"));
if (start < 0) {
  console.log("MISS");
  process.exit(1);
}
const body = [
  "    sorter: SorterResult<ServerInfo> | SorterResult<ServerInfo>[]",
  "  ) => {",
  "    const s = Array.isArray(sorter) ? sorter[0] : sorter;",
  "    if (s && s.field) {",
  '      setOrderByColumn(String(s.field));',
  '      setIsAsc(s.order === "ascend");',
  "    }",
  "    setCurrent(pagination.current ?? 1);",
  "    setPageSize(pagination.pageSize ?? 10);",
  '    const sortField = s && s.field ? String(s.field) : undefined;',
  '    const sortAsc = s && s.field ? s.order === "ascend" : undefined;',
  "    fetchServers(sortField, sortAsc);",
  "  };",
];
// 原块：start 行到 "  };" 行（含）
let end = start;
while (end < lines.length && lines[end].trim() !== "};") end++;
lines.splice(start, end - start + 1, ...body);
writeFileSync(p, lines.join("\n"));
console.log("OK");
