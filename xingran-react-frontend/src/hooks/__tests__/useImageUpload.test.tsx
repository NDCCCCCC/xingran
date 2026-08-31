/**
 * Phase 88 Batch377 — hooks/useImageUpload 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

import { useImageUpload } from "../useImageUpload";

describe("hooks/useImageUpload", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("初始 state", () => {
    const { result } = renderHook(() => useImageUpload(), { wrapper });
    expect(result.current.uploading).toBe(false);
    expect(result.current.fileList).toEqual([]);
    expect(result.current.imageId).toBeUndefined();
    expect(result.current.imageUrl).toBeUndefined();
  });

  it("handleUploadChange 设置 fileList", () => {
    const { result } = renderHook(() => useImageUpload(), { wrapper });
    const file: any = { uid: "1", name: "test.png", status: "uploading" };
    act(() => result.current.handleUploadChange([file]));
    expect(result.current.fileList.length).toBe(1);
  });

  it("handleUploadChange 清空 → imageId/imageUrl 复位", () => {
    const { result } = renderHook(() => useImageUpload(), { wrapper });
    act(() => result.current.handleUploadChange([{ uid: "1", name: "x" } as any]));
    act(() => result.current.handleUploadChange([]));
    expect(result.current.imageId).toBeUndefined();
    expect(result.current.imageUrl).toBeUndefined();
  });

  it("handleUploadSuccess 设置 imageId/imageUrl + 调 onSuccess", () => {
    const onSuccess = vi.fn();
    const { result } = renderHook(() => useImageUpload({ onSuccess }), { wrapper });
    const file: any = { uid: "1", name: "test.png" };
    act(() => {
      result.current.handleUploadSuccess(file, {
        id: "f1",
        storagePath: "x.png",
      } as any);
    });
    expect(result.current.imageId).toBe("f1");
    expect(result.current.imageUrl).toContain("x.png");
    expect(onSuccess).toHaveBeenCalledWith("f1", expect.stringContaining("x.png"));
  });

  it("handleUploadError 调 onError", () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useImageUpload({ onError }), { wrapper });
    act(() => {
      result.current.handleUploadError({ uid: "1" } as any, new Error("upload failed"));
    });
    expect(onError).toHaveBeenCalledWith(expect.any(Error));
  });

  it("handleUploadError 非 Error → 包装为 Error", () => {
    const onError = vi.fn();
    const { result } = renderHook(() => useImageUpload({ onError }), { wrapper });
    act(() => {
      result.current.handleUploadError({ uid: "1" } as any, "string error");
    });
    expect(onError).toHaveBeenCalledWith(expect.any(Error));
  });

  it("resetUpload 清空所有状态", () => {
    const { result } = renderHook(() => useImageUpload(), { wrapper });
    act(() => result.current.handleUploadChange([{ uid: "1", name: "x" } as any]));
    act(() => result.current.resetUpload());
    expect(result.current.fileList).toEqual([]);
    expect(result.current.uploading).toBe(false);
  });

  it("setInitialValue 设置 fileId + fileUrl", () => {
    const { result } = renderHook(() => useImageUpload(), { wrapper });
    act(() => result.current.setInitialValue("init-id", "/uploads/init.png"));
    expect(result.current.imageId).toBe("init-id");
    expect(result.current.imageUrl).toBe("/uploads/init.png");
    expect(result.current.fileList.length).toBe(1);
  });

  it("setInitialValue 仅 fileId", () => {
    const { result } = renderHook(() => useImageUpload(), { wrapper });
    act(() => result.current.setInitialValue("init-id"));
    expect(result.current.imageId).toBe("init-id");
    expect(result.current.fileList).toEqual([]);
  });

  it("setInitialValue 空 fileId → 不操作", () => {
    const { result } = renderHook(() => useImageUpload(), { wrapper });
    act(() => result.current.setInitialValue(""));
    expect(result.current.imageId).toBeUndefined();
  });
});
