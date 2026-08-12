/**
 * AssetRow - 资产列表行操作列
 *
 * 提取自 `src/pages/operations/assets/index.tsx` 操作列的 render 函数，
 * 用 React.memo 包装以避免父组件重渲染时无谓地重建按钮子树。
 *
 * 父组件应当：
 * - 通过 useCallback 稳定 onEdit / onDelete 引用
 * - 通过 useMemo 稳定 props 对象（可选，但推荐）
 */

import { memo } from "react";
import { Button, Space } from "antd";
import type { Asset } from "@/types/operations";

export interface AssetRowProps {
  record: Asset;
  onEdit?: (record: Asset) => void;
  onDelete?: (id: string) => void;
}

function AssetRowImpl({ record, onEdit, onDelete }: AssetRowProps) {
  return (
    <Space size="small">
      <Button
        type="link"
        size="small"
        onClick={() => onEdit?.(record)}
      >
        编辑
      </Button>
      <Button
        type="link"
        size="small"
        danger
        onClick={() => onDelete?.(record.id)}
      >
        删除
      </Button>
    </Space>
  );
}

export const AssetRow = memo(AssetRowImpl);
AssetRow.displayName = "AssetRow";

export default AssetRow;
