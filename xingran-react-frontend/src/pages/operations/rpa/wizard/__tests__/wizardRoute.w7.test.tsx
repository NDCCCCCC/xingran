/**
 * W7 / UI-FIX-01：向导页「返回任务列表」落地路由断言（用户视角）
 *
 * 这是缺陷 1 的**用户视角**回归：
 *   用户在「新建自动化任务」页点「返回任务列表」→ 修前跳到 dashboard
 *   （因为 /operations/rpa/tasks 不匹配任何路由 → SPA catch-all），修后必须落到
 *   真实路由 /vdi/rpa（RPA 管理页，任务管理/执行记录/Worker 监控三个 Tab 的宿主）。
 *
 * 断言方式是「渲染 → 找到按钮 → 点它 → 断言 router 落地路径」，
 * 而不是断言代码里写了什么字符串（那只能证明改了字面量，证明不了用户会去哪）。
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter, Routes, Route, useLocation } from "react-router-dom";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import TaskWizardPage from "../index";

/** 落地探针：把当前位置渲染出来，供断言读取（不改被测组件）。 */
function LocationProbe() {
  const loc = useLocation();
  return <div data-testid="location-probe">{loc.pathname}</div>;
}

function renderWizardAt(initialPath: string) {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <AntdApp>
        <Routes>
          <Route path="/vdi/rpa/wizard" element={<TaskWizardPage />} />
          <Route path="/vdi/rpa" element={<div data-testid="task-list-page">任务列表</div>} />
          {/* 修前那个错误路径：若真的落到这里，说明回归又出现了 */}
          <Route
            path="/operations/rpa/tasks"
            element={<div data-testid="wrong-path">错误路径</div>}
          />
          <Route path="/dashboard" element={<div data-testid="dashboard">仪表盘</div>} />
        </Routes>
        <LocationProbe />
      </AntdApp>
    </MemoryRouter>
  );
}

describe("向导页「返回任务列表」落地路由", () => {
  it("点「返回任务列表」→ 落到 /vdi/rpa（不是 dashboard，也不是 obsolete 路径）", async () => {
    renderWizardAt("/vdi/rpa/wizard");

    const backBtn = await screen.findByRole("button", { name: /返回任务列表/ });
    fireEvent.click(backBtn);

    // 用户看到的是「任务列表」页（RPA 管理宿主页），不是仪表盘
    expect(await screen.findByTestId("task-list-page")).toBeInTheDocument();
    expect(screen.queryByTestId("dashboard")).toBeNull();
    expect(screen.queryByTestId("wrong-path")).toBeNull();
    expect(screen.getByTestId("location-probe")).toHaveTextContent("/vdi/rpa");
  });

  it("向导内的「取消」也落到 /vdi/rpa（同一个常量，不出现第二种口径）", async () => {
    renderWizardAt("/vdi/rpa/wizard");

    // TaskWizard 的取消按钮文案是「取消」；antd 会在两字间插空格，用正则规避
    const cancelBtn = await screen.findByRole("button", { name: /取\s*消/ });
    fireEvent.click(cancelBtn);

    expect(await screen.findByTestId("task-list-page")).toBeInTheDocument();
    expect(screen.queryByTestId("dashboard")).toBeNull();
    expect(screen.getByTestId("location-probe")).toHaveTextContent("/vdi/rpa");
  });
});
