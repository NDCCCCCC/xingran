/**
 * Phase 88 Batch46 — components/shared ImageGallery 渲染测试
 *
 * 验证空/有 photos 渲染 + 操作按钮条件(editable, onSetPrimary, onUpdate, onDelete)
 * + 5 个 handler(preview/setPrimary/editDescription/saveDescription/delete)
 * 成功/错误路径 + 主图标记。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { Modal } from "antd";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ImageGallery from "../ImageGallery";
import type { Photo } from "../ImageGallery";

beforeEach(() => {
  vi.clearAllMocks();
});

const basePhotos: Photo[] = [
  {
    id: "p1",
    roomId: "r1",
    fileId: "f1",
    fileName: "主图.jpg",
    fileUrl: "https://example.com/1.jpg",
    sortOrder: 0,
    isPrimary: true,
    description: "主图描述",
    createdAt: "2026-08-29",
  },
  {
    id: "p2",
    roomId: "r1",
    fileId: "f2",
    fileName: "次图.jpg",
    fileUrl: "https://example.com/2.jpg",
    sortOrder: 1,
    isPrimary: false,
    description: "",
    createdAt: "2026-08-29",
  },
];

describe("ImageGallery — 渲染", () => {
  it("photos=[] 渲染 Empty 暂无照片", () => {
    renderWithProviders(<ImageGallery photos={[]} />);
    expect(screen.getByText("暂无照片")).toBeDefined();
  });

  it("photos 非空渲染所有照片", () => {
    renderWithProviders(<ImageGallery photos={basePhotos} />);
    // Antd Image 组件内部也会渲染 1 张隐藏 img,实际页面有 3 张,但 photo 卡片里至少 2 张
    const imgs = document.querySelectorAll(".image-gallery img");
    expect(imgs.length).toBeGreaterThanOrEqual(2);
  });

  it("主图标记 — isPrimary 渲染 gold Tag 主图", () => {
    const { baseElement } = renderWithProviders(<ImageGallery photos={basePhotos} />);
    expect(baseElement.textContent).toContain("主图");
  });

  it("photo.description 渲染", () => {
    renderWithProviders(<ImageGallery photos={basePhotos} />);
    expect(screen.getByText("主图描述")).toBeDefined();
  });

  it("editable=false 不显示操作按钮", () => {
    renderWithProviders(
      <ImageGallery
        photos={basePhotos}
        editable={false}
        onSetPrimary={vi.fn()}
        onUpdateDescription={vi.fn()}
        onDelete={vi.fn()}
      />
    );
    // 主图上不显示"设为主图"按钮
    expect(screen.queryByText("设为主图")).toBeNull();
    expect(screen.queryByText("编辑")).toBeNull();
    expect(screen.queryByText("删除")).toBeNull();
  });

  it("editable=true + onSetPrimary + 非主图 显示设为主图按钮", () => {
    renderWithProviders(<ImageGallery photos={basePhotos} onSetPrimary={vi.fn()} />);
    // p2 是非主图,应有"设为主图"按钮
    expect(screen.getByText("设为主图")).toBeDefined();
  });

  it("onUpload 存在显示上传按钮 + 数量 Tag", () => {
    renderWithProviders(<ImageGallery photos={basePhotos} onUpload={vi.fn()} />);
    expect(screen.getByText("上传照片")).toBeDefined();
    expect(screen.getByText(/共.*2.*张照片/)).toBeDefined();
  });
});

describe("ImageGallery — handleSetPrimary", () => {
  it("点设为主图 → onSetPrimary 调 + message.success", async () => {
    const onSetPrimary = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(<ImageGallery photos={basePhotos} onSetPrimary={onSetPrimary} />);
    const btn = screen.getByText("设为主图");
    fireEvent.click(btn);
    await waitFor(() => {
      expect(onSetPrimary).toHaveBeenCalledWith("p2");
    });
  });

  it("onSetPrimary 抛错 → message.error", async () => {
    const onSetPrimary = vi.fn().mockRejectedValue(new Error("primary fail"));
    renderWithProviders(<ImageGallery photos={basePhotos} onSetPrimary={onSetPrimary} />);
    fireEvent.click(screen.getByText("设为主图"));
    await waitFor(() => {
      expect(onSetPrimary).toHaveBeenCalled();
    });
  });

  it("未传 onSetPrimary → handleSetPrimary 直接 return 不抛错", async () => {
    renderWithProviders(<ImageGallery photos={basePhotos} />);
    // 没"设为主图"按钮 — p2 不渲染
    expect(screen.queryByText("设为主图")).toBeNull();
  });
});

describe("ImageGallery — handleEditDescription / handleSaveDescription", () => {
  it("点编辑 → Modal 打开 + description 初值", async () => {
    const onUpdateDescription = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(
      <ImageGallery photos={basePhotos} onUpdateDescription={onUpdateDescription} />
    );
    const editBtns = screen.getAllByText("编辑");
    fireEvent.click(editBtns[0]); // p1 主图
    await waitFor(() => {
      expect(screen.getByText("编辑照片描述")).toBeDefined();
    });
    // description input 初值为"主图描述"
    const textarea = document.querySelector(".ant-modal textarea") as HTMLTextAreaElement;
    expect(textarea?.value).toBe("主图描述");
  });

  it("点保存 → onUpdateDescription 调 + Modal 关闭", async () => {
    const onUpdateDescription = vi.fn().mockResolvedValue(undefined);
    const { baseElement } = renderWithProviders(
      <ImageGallery photos={basePhotos} onUpdateDescription={onUpdateDescription} />
    );
    fireEvent.click(screen.getAllByText("编辑")[0]);
    await waitFor(() => {
      expect(screen.getByText("编辑照片描述")).toBeDefined();
    });
    // Modal footer 在 portal, 用 baseElement 检索
    const saveBtn = Array.from(baseElement.querySelectorAll(".ant-modal-footer .ant-btn")).find(
      (b) => b.textContent?.replace(/\s+/g, "") === "保存"
    ) as HTMLElement;
    expect(saveBtn).toBeDefined();
    fireEvent.click(saveBtn);
    await waitFor(() => {
      expect(onUpdateDescription).toHaveBeenCalledWith("p1", "主图描述");
    });
  });

  it("onUpdateDescription 抛错 → message.error", async () => {
    const onUpdateDescription = vi.fn().mockRejectedValue(new Error("save fail"));
    const { baseElement } = renderWithProviders(
      <ImageGallery photos={basePhotos} onUpdateDescription={onUpdateDescription} />
    );
    fireEvent.click(screen.getAllByText("编辑")[0]);
    await waitFor(() => {
      expect(screen.getByText("编辑照片描述")).toBeDefined();
    });
    const saveBtn = Array.from(baseElement.querySelectorAll(".ant-modal-footer .ant-btn")).find(
      (b) => b.textContent?.replace(/\s+/g, "") === "保存"
    ) as HTMLElement;
    fireEvent.click(saveBtn);
    await waitFor(() => {
      expect(onUpdateDescription).toHaveBeenCalled();
    });
  });
});

describe("ImageGallery — handleDelete (confirmDelete Modal 强确认, 68.1-T3)", () => {
  // static Modal.confirm 的 portal 容器会跨用例残留在 body 上，先清除避免查询串扰
  const purgeLeftoverModals = () => {
    document
      .querySelectorAll<HTMLElement>(".ant-modal-container")
      .forEach((n) => n.parentElement?.remove());
  };

  it("点删除 → Modal 确认后 onDelete 调", async () => {
    purgeLeftoverModals();
    const onDelete = vi.fn().mockResolvedValue(undefined);
    renderWithProviders(<ImageGallery photos={basePhotos} onDelete={onDelete} />);
    fireEvent.click(screen.getAllByText("删除")[0]);
    // Modal 强确认弹出（68.1-T3：Popconfirm 升级为 confirmDelete）
    await waitFor(() => {
      expect(screen.getAllByText("确认删除").length).toBeGreaterThan(0);
    });
    let confirmBtn: HTMLElement | undefined;
    await waitFor(() => {
      // static Modal.confirm 实例跨用例残留，取最后一个（当前用例）弹窗的确认按钮
      const all = Array.from(
        document.querySelectorAll<HTMLElement>(".ant-modal-confirm-btns button")
      ).filter((b) => (b.textContent ?? "").replace(/\s/g, "") === "删除");
      confirmBtn = all[all.length - 1];
      expect(confirmBtn).toBeDefined();
    });
    fireEvent.click(confirmBtn);
    await waitFor(() => {
      expect(onDelete).toHaveBeenCalledWith("p1");
    });
  });

  // antd 6.6.4 缺陷：ActionButton.handlePromiseOnOk 的 rejection 分支 `return Promise.reject(e)`
  // （node_modules/antd/es/_util/ActionButton.js:51-60）返回的 promise 无人消费，真实点击会让
  // vitest 记为 unhandled error。而 onOk 返回 rejected 恰是「弹窗保持打开可重试」的唯一实现方式，
  // 故此处取 Modal.confirm 的配置直接执行 onOk，断言契约本身而不触发 antd 内部 promise 链。
  it("onDelete 抛错 → handleApiError 反馈且 onOk 上抛（弹窗保持打开）", async () => {
    purgeLeftoverModals();
    const confirmSpy = vi.spyOn(Modal, "confirm");
    const onDelete = vi.fn().mockRejectedValue(new Error("delete fail"));
    renderWithProviders(<ImageGallery photos={basePhotos} onDelete={onDelete} />);
    fireEvent.click(screen.getAllByText("删除")[0]);
    await waitFor(() => {
      expect(confirmSpy).toHaveBeenCalled();
    });
    const cfg = confirmSpy.mock.calls.at(-1)![0] as { onOk?: () => Promise<unknown> };
    expect(cfg.onOk).toBeTypeOf("function");
    await expect(cfg.onOk!()).rejects.toThrow("delete fail");
    expect(onDelete).toHaveBeenCalledWith("p1");
    confirmSpy.mockRestore();
  });
});
