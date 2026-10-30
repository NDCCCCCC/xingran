/**
 * 共享组件统一导出
 */

export { default as ActionButtons } from "./ActionButtons";
export type { ActionButton } from "./ActionButtons";

// v1.24 PERM-02 —— 操作级权限门控（唯一判定入口，三态：ready/loading/error）
export { useActionPermission } from "./useActionPermission";
export type {
  ActionPermissionDecision,
  ActionPermissionResult,
  PermissionGateState,
  UseActionPermissionOptions,
} from "./useActionPermission";

export { default as ExcelImport } from "./ExcelImport";
export type { ExcelImportProps } from "./ExcelImport";

export { default as ExcelImportLazy } from "./ExcelImportLazy";
export type { ExcelImportLazyProps } from "./ExcelImportLazy";

export { default as ExcelExport } from "./ExcelExport";
export type { ExcelExportProps } from "./ExcelExport";

export { default as FileUpload } from "./FileUpload";
export type { FileUploadProps } from "./FileUpload";

export { default as ImageGallery } from "./ImageGallery";
export type { ImageGalleryProps } from "./ImageGallery";

export { default as GlobalSearch } from "./GlobalSearch";

export {
  DepartmentTreeSelect,
  DepartmentTreeSelectWithTop,
  type DepartmentTreeSelectProps,
  type DepartmentTreeSelectWithTopProps,
  type Department,
  type TreeNode,
} from "./DepartmentTreeSelect";

export { default as FloorPlanEditor } from "./FloorPlanEditor";
export type {
  FloorPlanEditorProps,
  WorkstationNode,
  ViewState,
  DragState,
  ContextMenuState,
} from "./FloorPlanEditor.types";

export { default as BatchExportModal } from "./BatchExportModal";
export type { BatchExportModalProps, EntityType } from "./BatchExportModal";

// Phase 14 - 三态(空/加载/错误)共享组件
export { default as EmptyStateWithAction } from "./EmptyStateWithAction";
export type { EmptyStateWithActionProps } from "./EmptyStateWithAction";

export { default as ErrorAlertWithRetry } from "./ErrorAlertWithRetry";
export type { ErrorAlertWithRetryProps, ApiErrorShape } from "./ErrorAlertWithRetry";

// Phase 65 — 危险删除唯一入口（声明式 + 命令式）
// 显式扩展名：Windows 大小写不敏感解析下 "./ConfirmDelete" 会命中 confirmDelete.ts
export { ConfirmDelete } from "./ConfirmDelete.tsx";
export type { ConfirmDeleteProps } from "./ConfirmDelete.tsx";
export { confirmDelete } from "./confirmDelete";
export type { ConfirmDeleteOptions } from "./confirmDelete";

// Phase 66：PageTable（新列表页唯一合法列表数据层入口）
export { default as PageTable } from "./PageTable";
export type { PageTableProps, PageTableRequest, PageTableRenderContext } from "./PageTable";

// Phase 66：SearchForm（schema 驱动搜索区）
export { SearchForm } from "./SearchForm";
export type { SearchFormProps, SearchField, SearchFieldOption } from "./SearchForm";

// Phase 66：FormModal（编辑弹窗封装：脏检查 + 统一提交/错误分流）
// useFormModalForm 自 v1.24 P82 拆至独立 hook 文件（react-refresh 边界）
export { FormModal } from "./FormModal";
export type { FormModalProps } from "./FormModal";
export { useFormModalForm } from "./hooks/useFormModalForm";

// Phase 68 DEBT-02a：状态展示统一 helper（唯一落点裁定 shared/，勿移 utils/）
export { StatusTag, statusLabel, statusColor, UNKNOWN_STATUS_LABEL } from "./statusHelpers";
export type { StatusDomain, StatusColor } from "./statusHelpers";
