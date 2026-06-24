/**
 * MarkdownEditor - Lazy-loaded Markdown editor wrapper
 *
 * Wraps `@uiw/react-md-editor` (heavy markdown editor) with React.lazy + Suspense
 * so the editor library is only loaded when the system notice form is opened
 * with Markdown mode enabled.
 *
 * Per D-06 (Wave 2): @uiw/react-md-editor is heavy and should load on demand.
 * Per D-09 / D-17: Suspense fallback uses AntD Spin with a descriptive tip.
 *
 * Usage:
 *   import { MarkdownEditor } from '@/components/markdown/MarkdownEditor';
 *   <MarkdownEditor value={content} onChange={setContent} preview="live" height={400} />
 */

import { lazy, Suspense, type FC } from "react";
import { Spin } from "antd";
// 注：类型从 nohighlight 入口导入（与 lazy import 保持一致）。
// 主包 @uiw/react-md-editor 的 MDEditorProps 含 'full' preview 选项，
// 但 nohighlight 入口的 PreviewType 较窄（'edit' | 'live'），
// 混用会导致 TS2322 类型不兼容。
import type { MDEditorProps } from "@uiw/react-md-editor/nohighlight";

const MDEditor = lazy(() => import("@uiw/react-md-editor/nohighlight"));

const Loading: FC = () => (
  <div
    style={{
      display: "flex",
      justifyContent: "center",
      alignItems: "center",
      padding: 24,
      minHeight: 200,
    }}
  >
    <Spin>
      <div style={{ minHeight: 120 }} />
    </Spin>
    <div style={{ marginTop: 8, color: "rgba(0, 0, 0, 0.45)" }}>加载编辑器...</div>
  </div>
);

export const MarkdownEditor: FC<MDEditorProps> = (props) => (
  <Suspense fallback={<Loading />}>
    <MDEditor {...props} />
  </Suspense>
);

export default MarkdownEditor;