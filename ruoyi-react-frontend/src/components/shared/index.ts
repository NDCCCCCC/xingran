/**
 * 共享组件统一导出
 */

export { default as ActionButtons } from "./ActionButtons";
export type { ActionButton } from "./ActionButtons";

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
