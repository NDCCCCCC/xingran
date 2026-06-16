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
import type { MDEditorProps } from "@uiw/react-md-editor";

const MDEditor = lazy(() => import("@uiw/react-md-editor"));

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