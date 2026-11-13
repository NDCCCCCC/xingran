/**
 * 冒烟：全库备份 RestoreModal 的 sqlite 前置提示 + 二次确认文案。
 *
 * 与 A5（quick 261112 恢复一致性整改）前端改动对齐：A5 已在
 * sqliteNotice.render.test.tsx 覆盖「sqlite 前置提示/离线命令入口/postgres 不显示」，
 * 本文件不重复造轮子，只补 A5 未覆盖的「二次确认」链路——选择 .zip 备份文件后
 * confirmDelete 弹窗里的真实文案（覆盖现有全部数据 / 期间不可写入 / 恢复后需
 * 重新登录 / 自动恢复前备份 / 开始恢复按钮 / 标题「确认导入恢复」）。
 *
 * 同时再断言一次 A5 的 sqlite 前置提示与常驻「高风险操作」Alert，形成同一组件
 * 内「前置提示 → 二次确认」的完整冒烟链。零新依赖。
 *
 * 注意：antd Modal 通过 portal 挂到 document.body，file input 与确认弹窗都要从
 * document 查，而非 render 返回的 container。页面 Modal 标题与 confirmDelete
 * 标题同为「确认导入恢复」，故用「开始恢复」按钮出现与否来区分是否弹出二次确认。
 */
import { describe, it, expect, vi, afterEach } from "vitest";
import { screen, fireEvent, waitFor, cleanup } from "@testing-library/react";
import { Modal } from "antd";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});
vi.mock("@/lib/download", () => ({
  downloadFile: vi.fn(async () => ""),
}));

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import type { Capabilities } from "@/types/backup";
import { RESTORE_CONFIRM_TEXT, SQLITE_OFFLINE_NOTICE, SQLITE_OFFLINE_COMMAND } from "../types";
import RestoreModal from "../components/RestoreModal";

const sqliteCapabilities: Capabilities = {
  dbType: "sqlite",
  pgDumpAvailable: true,
  pgRestoreAvailable: true,
  pgDumpPath: "",
  pgRestorePath: "",
  hint: "SQLite 模式：整库恢复需停服后离线执行。",
};

function renderModal(capabilities: Capabilities) {
  return renderWithProviders(
    <RestoreModal
      open
      taskRunning={false}
      capabilities={capabilities}
      onClose={() => {}}
      onStartTask={() => {}}
    />
  );
}

/** antd Modal portal 到 body，从 document 取 Upload.Dragger 的 file input */
function getFileInput(): HTMLInputElement | null {
  return document.querySelector('input[type="file"]') as HTMLInputElement | null;
}

afterEach(() => {
  // Modal.confirm 是命令式弹窗、挂在 body 且不在 React 树内，cleanup() 卸载不了它；
  // 不 destroyAll 会让上一个用例的「开始恢复」确认框残留到下一个用例。
  Modal.destroyAll();
  cleanup();
  vi.clearAllMocks();
});

describe("RestoreModal — sqlite 前置提示（A5 对齐）", () => {
  it("dbType=sqlite 时常驻离线说明 + 离线命令入口 + 高风险常驻提示", () => {
    renderModal(sqliteCapabilities);
    expect(screen.getByText("SQLite 整库恢复需停服离线执行")).toBeTruthy();
    expect(screen.getByText(SQLITE_OFFLINE_NOTICE)).toBeTruthy();
    expect(screen.getByText(SQLITE_OFFLINE_COMMAND)).toBeTruthy();
    // 常驻「高风险操作」Alert（RESTORE_CONFIRM_TEXT 同源文案，打开弹窗即见）
    expect(screen.getByText(/恢复将覆盖现有全部数据/)).toBeTruthy();
    expect(screen.getByText(/恢复后需重新登录/)).toBeTruthy();
  });
});

describe("RestoreModal — 选择 .zip 后的二次确认文案", () => {
  it("confirmDelete 弹窗展示 RESTORE_CONFIRM_TEXT 全文、标题与「开始恢复」按钮", async () => {
    renderModal(sqliteCapabilities);
    const input = getFileInput();
    expect(input).toBeTruthy();
    const file = new File(["zip-bytes"], "full-backup.zip", {
      type: "application/zip",
    });

    // Upload.Dragger 的 customRequest → handleCustomRequest → confirmDelete
    fireEvent.change(input as HTMLInputElement, { target: { files: [file] } });

    // 「开始恢复」是 confirmDelete 的 actionLabel（页面 Modal 自身没有该按钮）
    // 该弹窗由 Upload.Dragger 的 customRequest 异步触发，CI 分片高负载下
    // 默认 waitFor 超时（1s）偶发不足 → 放宽到 5s，避免 flaky 假失败。
    await waitFor(
      () => {
        expect(screen.getByRole("button", { name: "开始恢复" })).toBeTruthy();
      },
      { timeout: 5000 }
    );
    // 标题「确认导入恢复」出现（页面 Modal + 确认弹窗，用 getAllByText 容纳多命中）
    expect(screen.getAllByText("确认导入恢复").length).toBeGreaterThan(0);
    // contentText 里的 {target} 已被替换为「备份文件（full-backup.zip）」
    expect(screen.getByText(/确认使用 备份文件（full-backup.zip） 进行恢复吗/)).toBeTruthy();
    // 逐条锁定破坏性操作的告知语义（RESTORE_CONFIRM_TEXT 的关键片段）
    expect(screen.getAllByText(/恢复将覆盖现有全部数据/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/期间不可写入/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/恢复后需重新登录/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/操作前系统会自动生成一份恢复前备份/).length).toBeGreaterThan(0);
    // 文案常量本身与组件渲染一致
    expect(RESTORE_CONFIRM_TEXT).toContain("恢复将覆盖现有全部数据");
  });
});
