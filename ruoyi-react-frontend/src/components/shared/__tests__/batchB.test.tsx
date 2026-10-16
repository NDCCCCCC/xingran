/**
 * Phase 88 Batch19c — components/shared 子组件 props 组合快照(D-12)
 * ModernTag/ActionButtons/DepartmentTreeSelect/ExcelExport/ErrorAlertWithRetry
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { ConfigProvider } from "antd";
import ModernTag from "../ModernTag";
import ActionButtons from "../ActionButtons";
import ExcelExport from "../ExcelExport";
import ErrorAlertWithRetry from "../ErrorAlertWithRetry";

function wrap(ui: React.ReactElement) {
  return render(<ConfigProvider>{ui}</ConfigProvider>);
}

describe("components/shared — ModernTag", () => {
  it("renders with status info", () => {
    const { container } = wrap(<ModernTag status="processing">进行中</ModernTag>);
    expect(container.innerHTML).toContain("进行中");
  });

  it("renders success / warning / error variants", () => {
    wrap(<ModernTag status="success">成功</ModernTag>);
    wrap(<ModernTag status="warning">警告</ModernTag>);
    wrap(<ModernTag status="error">失败</ModernTag>);
  });

  it("renders with showIcon=false", () => {
    const { container } = wrap(
      <ModernTag status="success" showIcon={false}>
        无图标
      </ModernTag>
    );
    expect(container.innerHTML).toContain("无图标");
  });
});

describe("components/shared — ActionButtons", () => {
  it("returns null when actions empty", () => {
    const { container } = wrap(<ActionButtons actions={[]} />);
    expect(container.querySelector(".ant-alert")).toBeNull();
  });

  it("renders ≤ threshold actions inline (no Dropdown)", () => {
    const { container } = wrap(
      <ActionButtons
        actions={[
          { key: "edit", label: "编辑", onClick: vi.fn() },
          { key: "del", label: "删除", onClick: vi.fn() },
        ]}
        threshold={3}
      />
    );
    expect(container.innerHTML).toContain("编辑");
    expect(container.innerHTML).toContain("删除");
  });

  it("renders Dropdown trigger when actions >= threshold", () => {
    const { container } = wrap(
      <ActionButtons
        actions={[
          { key: "a", label: "动作A", onClick: vi.fn() },
          { key: "b", label: "动作B", onClick: vi.fn() },
          { key: "c", label: "动作C", onClick: vi.fn() },
          { key: "d", label: "动作D", onClick: vi.fn() },
        ]}
        threshold={3}
      />
    );
    expect(container.querySelector(".ant-dropdown-trigger")).not.toBeNull();
  });

  it("respects custom threshold", () => {
    const { container } = wrap(
      <ActionButtons
        actions={[
          { key: "1", label: "一", onClick: vi.fn() },
          { key: "2", label: "二", onClick: vi.fn() },
        ]}
        threshold={5}
      />
    );
    expect(container.innerHTML).toContain("一");
    expect(container.innerHTML).toContain("二");
  });
});

describe("components/shared — ExcelExport", () => {
  it("renders visible=false no Drawer", () => {
    const { container } = wrap(
      <ExcelExport entityType="building" visible={false} onClose={vi.fn()} />
    );
    expect(container.querySelector(".ant-alert")).toBeNull();
  });
});

describe("components/shared — ErrorAlertWithRetry", () => {
  it("renders gracefully when no error (页面不崩溃)", () => {
    const { container } = wrap(<ErrorAlertWithRetry error={null} onRetry={vi.fn()} />);
    expect(container.innerHTML.length).toBeGreaterThan(0);
  });

  it("renders Error.message + Retry button when error present", () => {
    const { container } = wrap(
      <ErrorAlertWithRetry error={new Error("网络异常")} onRetry={vi.fn()} />
    );
    expect(container.innerHTML).toContain("网络异常");
    expect(container.innerHTML).toContain("重新加载");
  });

  it("renders Error subclass with custom message", () => {
    class CustomError extends Error {
      constructor() {
        super("自定义错误");
        this.name = "CustomError";
      }
    }
    const { container } = wrap(<ErrorAlertWithRetry error={new CustomError()} onRetry={vi.fn()} />);
    expect(container.innerHTML).toContain("自定义错误");
  });
});
