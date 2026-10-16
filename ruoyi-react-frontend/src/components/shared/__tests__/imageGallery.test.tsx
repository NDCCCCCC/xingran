/**
 * Phase 88 Batch132 — components/shared/ImageGallery 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import ImageGallery, { type Photo } from "../ImageGallery";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const photos: Photo[] = [
  {
    id: "p1",
    roomId: "r1",
    fileId: "f1",
    fileUrl: "https://example.com/p1.jpg",
    sortOrder: 0,
    isPrimary: true,
    description: "P1",
    createdAt: "2026-01-01",
  },
  {
    id: "p2",
    roomId: "r1",
    fileId: "f2",
    fileUrl: "https://example.com/p2.jpg",
    sortOrder: 1,
    isPrimary: false,
    createdAt: "2026-01-01",
  },
];

describe("ImageGallery", () => {
  it("空 photos → 显示 Empty", () => {
    const { baseElement } = render(<ImageGallery photos={[]} />, { wrapper });
    expect(baseElement.textContent).toContain("暂无照片");
  });

  it("有 photos + loading → 渲染所有照片", () => {
    const { baseElement } = render(<ImageGallery photos={photos} />, { wrapper });
    expect(baseElement.textContent).toContain("P1");
    // p1 is primary so shows "主图" tag
    expect(baseElement.textContent).toContain("主图");
  });

  it("editable=true + onUpload → 显示上传按钮 + 计数", () => {
    const onUpload = vi.fn();
    const { baseElement, getByText } = render(
      <ImageGallery photos={photos} onUpload={onUpload} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("上传照片");
    expect(baseElement.textContent).toContain("共 2 张照片");
    fireEvent.click(getByText("上传照片"));
    expect(onUpload).toHaveBeenCalled();
  });

  it("editable=false → 不渲染设为主图/编辑/删除按钮", () => {
    const { baseElement, queryByText } = render(<ImageGallery photos={photos} editable={false} />, {
      wrapper,
    });
    expect(queryByText("设为主图")).toBeNull();
    expect(queryByText("编辑")).toBeNull();
    expect(queryByText("删除")).toBeNull();
  });

  it("onSetPrimary 提供 → 非主图显示设为主图按钮", () => {
    const onSetPrimary = vi.fn(() => Promise.resolve());
    const { baseElement } = render(<ImageGallery photos={photos} onSetPrimary={onSetPrimary} />, {
      wrapper,
    });
    const setPrimaryBtns = baseElement.querySelectorAll("button");
    // Find button containing 设为主图
    const found = Array.from(setPrimaryBtns).find((b) => b.textContent?.includes("设为主图"));
    expect(found).toBeTruthy();
  });

  it("点击 设为主图 → 调用 onSetPrimary", async () => {
    const onSetPrimary = vi.fn(() => Promise.resolve());
    const { baseElement } = render(<ImageGallery photos={photos} onSetPrimary={onSetPrimary} />, {
      wrapper,
    });
    const btns = Array.from(baseElement.querySelectorAll("button"));
    const btn = btns.find((b) => b.textContent?.includes("设为主图"));
    expect(btn).toBeTruthy();
    fireEvent.click(btn!);
    await waitFor(() => {
      expect(onSetPrimary).toHaveBeenCalledWith("p2");
    });
  });

  it("onSetPrimary 失败 → error message", async () => {
    const onSetPrimary = vi.fn(() => Promise.reject(new Error("net")));
    const { baseElement } = render(<ImageGallery photos={photos} onSetPrimary={onSetPrimary} />, {
      wrapper,
    });
    const btns = Array.from(baseElement.querySelectorAll("button"));
    const btn = btns.find((b) => b.textContent?.includes("设为主图"));
    fireEvent.click(btn!);
    await waitFor(() => {
      expect(baseElement.textContent).toMatch(/失败|错误/);
    });
  });

  it("onUpdateDescription 提供 → 编辑按钮打开 modal", async () => {
    const onUpdateDescription = vi.fn(() => Promise.resolve());
    const { baseElement, getAllByText } = render(
      <ImageGallery photos={photos} onUpdateDescription={onUpdateDescription} />,
      { wrapper }
    );
    const editBtns = getAllByText("编辑");
    fireEvent.click(editBtns[0]);
    await waitFor(() => {
      expect(baseElement.textContent).toContain("编辑照片描述");
    });
  });

  it("onDelete 提供 → Popconfirm 删除按钮", () => {
    const onDelete = vi.fn(() => Promise.resolve());
    const { baseElement, getAllByText } = render(
      <ImageGallery photos={photos} onDelete={onDelete} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("删除");
    expect(getAllByText("删除").length).toBeGreaterThan(0);
  });
});
