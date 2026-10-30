/**
 * FormModal 专用 FormInstance hook（v1.24 P82 自 FormModal.tsx 拆出：
 * react-refresh 要求组件文件只导出组件；hook 与组件同文件会破坏 Fast Refresh 边界）
 */
import { Form } from "antd";
import type { FormInstance } from "antd";

/** FormModal 专用 FormInstance（与 useTableManager.editForm 同型，hook 须在组件体内调用） */
export function useFormModalForm(): FormInstance {
  return Form.useForm()[0];
}
