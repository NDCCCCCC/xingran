// Phase 74 C2 第四批修复（用后删）
import { readFileSync, writeFileSync } from "node:fs";
function sub(p, a, b) {
  let t = readFileSync(p, "utf8");
  if (!t.includes(a)) {
    console.log("MISS", p, JSON.stringify(a.slice(0, 70)));
    return;
  }
  writeFileSync(p, t.split(a).join(b));
  console.log("OK", p);
}

// ExcelImport：补 UploadProps 类型导入
sub(
  "src/components/shared/ExcelImport.tsx",
  'import type { UploadFile } from "antd";',
  'import type { UploadFile, UploadProps } from "antd";'
);

// FilterValue 导入
sub(
  "src/pages/knowledge/articles/index.tsx",
  'import { Table, Button, Form, Card, Row, Col, Statistic, Space, App } from "antd";',
  'import { Table, Button, Form, Card, Row, Col, Statistic, Space, App } from "antd";\nimport type { FilterValue } from "antd/es/table/interface";'
);
sub(
  "src/pages/network/templates/index.tsx",
  'import { Table, Button, Space, Form, Card, Row, Col, Statistic, App } from "antd";',
  'import { Table, Button, Space, Form, Card, Row, Col, Statistic, App } from "antd";\nimport type { FilterValue } from "antd/es/table/interface";'
);

// server/index：SorterResult 泛型对齐 ServerInfo 表格 + Key→string 转换 + 可选分页兜底
sub(
  "src/pages/monitor/server/index.tsx",
  `    sorter: SorterResult<Record<string, unknown>>
  ) => {
    if (sorter && sorter.field) {
      setOrderByColumn(sorter.field);
      setIsAsc(sorter.order === "ascend");
    }
    setCurrent(pagination.current);
    setPageSize(pagination.pageSize);
    const sortField = sorter && sorter.field ? sorter.field : undefined;
    const sortAsc = sorter && sorter.field ? sorter.order === "ascend" : undefined;
    fetchServers(sortField, sortAsc);
  };`,
  `    sorter: SorterResult<ServerInfo>
  ) => {
    if (sorter && sorter.field) {
      setOrderByColumn(String(sorter.field));
      setIsAsc(sorter.order === "ascend");
    }
    setCurrent(pagination.current ?? 1);
    setPageSize(pagination.pageSize ?? 10);
    const sortField = sorter && sorter.field ? String(sorter.field) : undefined;
    const sortAsc = sorter && sorter.field ? sorter.order === "ascend" : undefined;
    fetchServers(sortField, sortAsc);
  };`
);
