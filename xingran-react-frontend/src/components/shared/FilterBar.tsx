/**
 * FilterBar — 列表页筛选卡片统一骨架（响应式口径单一真相源）
 *
 * 取代各页面里手写的
 * ```tsx
 * <div style={{ display: "flex", justifyContent: "space-between",
 *               alignItems: "flex-start", flexWrap: "wrap", gap: 16 }}>…</div>
 * ```
 * 内联 flex 无法被媒体查询接管，窄宽下表单与操作按钮会互相挤压。
 *
 * 布局：左区 = 筛选表单（`form` + children 的 Form.Item schema），右区 = 操作按钮。
 * 宽屏保持原来的「左表单 + 右按钮」同排；≤1440px 由 `src/responsive.css`
 * 切成上下堆叠，并把 inline 表单改成等宽栅格（标签上置、控件铺满）。
 *
 * 用法：
 * ```tsx
 * <Card style={{ marginBottom: 16 }}>
 *   <FilterBar actions={<Space>…按钮…</Space>}>
 *     <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
 *       …
 *     </Form>
 *   </FilterBar>
 * </Card>
 * ```
 */
import type { FC, ReactNode } from "react";

export interface FilterBarProps {
  /** 筛选表单区（通常是一个 `layout="inline"` 的 Form） */
  children: ReactNode;
  /** 右侧操作按钮区（新增/导入/导出/批量操作等） */
  actions?: ReactNode;
  /** 额外类名，用于个别页面微调（一般不需要） */
  className?: string;
}

const FilterBar: FC<FilterBarProps> = ({ children, actions, className }) => (
  <div className={["xr-filter-bar", className].filter(Boolean).join(" ")}>
    <div className="xr-filter-bar__form">{children}</div>
    {actions ? <div className="xr-filter-bar__actions">{actions}</div> : null}
  </div>
);

export default FilterBar;
