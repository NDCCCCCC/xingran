/**
 * W7 / UI-FIX-02：实时画面页「接管 → 遮罩消失」用户视角集成测
 *
 * 这是缺陷 2 的**用户视角**回归：不 mock 内部函数，直接渲染整页
 * `RpaLiveViewPage` + mock 后端 HTTP，然后按用户的动作链走一遍：
 *   进入页面 → 点「接管」→ 确认弹窗「接管」→ 断言遮罩消失、角标出现、
 *   释放按钮出现、鼠标事件开始下发 input。
 *
 * 为什么必须有这一层（而不只是 controller.ts 的纯函数测试）：
 *   controller 单测能证明「算法对」，但证明不了「页面真的把 takeoverById
 *   接进去了」—— 历史上正是「后端字段没人消费」这类断链导致用户看不出问题。
 *
 * 覆盖两组人格：
 *   A. 昵称与账号名一致的普通账号（ninedrunk 型）
 *   B. 昵称与账号名不一致的账号（超级管理员/管理员型）—— 缺陷原形
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, act, cleanup } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import RpaLiveViewPage from "../index";
import { createApiMock, resetApiMocks, setGenericFallback } from "@/test/utils/createApiMock";
import { useAuthStore } from "@/store/authStore";
import { useMenuStore } from "@/store/menuStore";
import { useNoticeStore } from "@/store/noticeStore";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

const EXEC_ID = "exec-w7-1";

/** 让权限门控进入 ready 且持有读权限（否则页面不渲染任何入口）。 */
function grantPermission() {
  useMenuStore.setState({
    permissions: [PERMISSION_CODES.rpaExecution.list],
    permissionsLoaded: true,
    permissionsError: null,
  } as never);
}

/** 设置当前登录用户（store 里 user 形状与真实登录响应一致）。 */
function loginAs(user: { id: string; username: string; nickname?: string }) {
  useAuthStore.setState({ user, isAuthenticated: true } as never);
}

/** 注册本轮要用到的全部后端端点并给出默认响应。 */
function mockBackend(opts: { takeoverById: string; takeoverBy: string }) {
  // 兜底：任何未注册端点统一返回安全空响应，避免 mock 返回 undefined 让
  // 页面的 `.catch()` 抛 TypeError（那不是被测逻辑的问题）。
  setGenericFallback({ code: 0, data: {} });

  // 可观看执行列表：两个状态各一次
  for (const path of ["/rpa/executions/list"]) {
    const h = createApiMock(path);
    h.endpoint.mockResolvedValue({
      code: 0,
      data: { list: [{ id: EXEC_ID, taskName: "门店巡检", status: "running" }], total: 1 },
    });
  }

  // 进入即读 state（HUD 初始值）
  const stateApi = createApiMock(`/rpa/executions/${EXEC_ID}/liveview/state`);
  stateApi.endpoint.mockResolvedValue({
    code: 0,
    data: {
      context: { executionId: EXEC_ID, step: 0, totalSteps: 3, mode: "copilot" },
      watchers: 1,
    },
  });

  // watch 成功
  const watchApi = createApiMock(`/rpa/executions/${EXEC_ID}/liveview/watch`);
  watchApi.endpoint.mockResolvedValue({ code: 0, data: { executionId: EXEC_ID, mode: "copilot" } });

  // 接管成功（后端在成功后会把权威值写进 hub，随后 state 读回）
  const takeoverApi = createApiMock(`/rpa/executions/${EXEC_ID}/liveview/takeover`);
  takeoverApi.endpoint.mockImplementation(async () => {
    // 接管后 state 端点返回新的权威 context
    stateApi.endpoint.mockResolvedValue({
      code: 0,
      data: {
        context: {
          executionId: EXEC_ID,
          step: 0,
          totalSteps: 3,
          mode: "copilot",
          takeoverBy: opts.takeoverBy,
          takeoverById: opts.takeoverById,
        },
        watchers: 1,
      },
    });
    return { code: 0, data: { ok: true } };
  });

  // 输入下发（只记录调用次数，用于证明「遮罩消失后鼠标事件能出去」）
  const inputApi = createApiMock(`/rpa/executions/${EXEC_ID}/liveview/input`);
  inputApi.endpoint.mockResolvedValue({ code: 0, data: { ok: true } });

  // stop / keepalive / release：页面卸载与心跳会调用，未注册会让 mock 返回
  // undefined（调用方 .catch 抛 TypeError），必须在每条用例里都备好。
  const stopApi = createApiMock(`/rpa/executions/${EXEC_ID}/liveview/stop`);
  stopApi.endpoint.mockResolvedValue({ code: 0, data: { ok: true } });
  const keepaliveApi = createApiMock(`/rpa/executions/${EXEC_ID}/liveview/keepalive`);
  keepaliveApi.endpoint.mockResolvedValue({ code: 0, data: { ok: true } });
  const releaseApi = createApiMock(`/rpa/executions/${EXEC_ID}/liveview/release`);
  releaseApi.endpoint.mockResolvedValue({ code: 0, data: { ok: true } });

  return { stateApi, watchApi, takeoverApi, inputApi, stopApi, keepaliveApi, releaseApi };
}

/** jsdom 下 canvas 无 2d ctx；getBoundingClientRect 恒 0 → 补一个真实矩形。 */
function stubCanvasLayout() {
  vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockReturnValue(null);
  vi.spyOn(HTMLCanvasElement.prototype, "getBoundingClientRect").mockReturnValue({
    left: 0,
    top: 0,
    width: 400,
    height: 225,
    right: 400,
    bottom: 225,
    x: 0,
    y: 0,
    toJSON: () => ({}),
  } as DOMRect);
}

/** 点「接管」按钮 + 确认弹窗；返回后等页面重新渲染。 */
async function clickTakeoverAndConfirm() {
  // 页面顶部卡片 extra 里才是真正的「接管」入口；用容器定位避免与
  // 弹窗内同名按钮 / 其它文案撞名。
  await waitFor(() => {
    expect(document.querySelector(".ant-card-extra")).not.toBeNull();
  });
  const cardExtra = document.querySelector(".ant-card-extra") as HTMLElement;
  const takeoverBtn = Array.from(cardExtra.querySelectorAll("button")).find((b) =>
    /接\s*管/.test((b.textContent ?? "").trim())
  );
  expect(takeoverBtn).toBeTruthy();
  fireEvent.click(takeoverBtn as HTMLButtonElement);

  // antd Modal.confirm 的确定按钮（弹窗内的 primary 才是「接管」）
  await waitFor(() => {
    const confirm = document.querySelector(".ant-modal-confirm .ant-btn-primary");
    expect(confirm).not.toBeNull();
  });
  const confirmBtn = document.querySelector(".ant-modal-confirm .ant-btn-primary") as HTMLElement;
  fireEvent.click(confirmBtn);
}

async function renderPage() {
  render(
    <MemoryRouter initialEntries={[`/vdi/rpa/live?executionId=${EXEC_ID}`]}>
      <AntdApp>
        <RpaLiveViewPage />
      </AntdApp>
    </MemoryRouter>
  );
}

beforeEach(() => {
  resetApiMocks();
  grantPermission();
  stubCanvasLayout();
});

afterEach(() => {
  // 顺序敏感：必须先卸载（页面卸载会调 stopExecution，需要 mock 仍可用），
  // 再清 registry —— 否则卸载请求落到已 reset 的通用 spy，返回 undefined，
  // 页面里的 `.catch()` 直接 TypeError（不是被测逻辑的问题）。
  cleanup();
  vi.restoreAllMocks();
  resetApiMocks();
});

describe("缺陷 2 用户视角：接管后遮罩必须消失", () => {
  it("人格 A（昵称==账号名）：接管 → 遮罩消失 + 已接管角标 + 出现释放按钮", async () => {
    loginAs({ id: "uid-a", username: "ninedrunk", nickname: "ninedrunk" });
    mockBackend({ takeoverById: "uid-a", takeoverBy: "ninedrunk" });

    await renderPage();

    // 进入页面即有遮罩（未接管）
    expect(await screen.findByTestId("live-view-mask")).toBeInTheDocument();

    await clickTakeoverAndConfirm();

    await waitFor(() => {
      expect(screen.queryByTestId("live-view-mask")).toBeNull();
    });
    // 角标文案（卡片标题里的 Tag 与 HUD 角标都会出现）
    expect(screen.getAllByText(/已接管 · ninedrunk/).length).toBeGreaterThan(0);
    // 接管者才有「释放接管」
    expect(await screen.findByRole("button", { name: /释放接管/ })).toBeInTheDocument();
  });

  it("人格 B（昵称≠账号名，缺陷原形）：接管 → 遮罩同样消失", async () => {
    // 这是生产复现的那一组：前端 store nickname=超级管理员、username=ninedrunk
    loginAs({ id: "uid-b", username: "ninedrunk", nickname: "超级管理员" });
    // 后端权威：userID=uid-b，展示名=username（SM2 token 无 nickname 时的形态）
    mockBackend({ takeoverById: "uid-b", takeoverBy: "ninedrunk" });

    await renderPage();

    expect(await screen.findByTestId("live-view-mask")).toBeInTheDocument();

    await clickTakeoverAndConfirm();

    await waitFor(() => {
      expect(screen.queryByTestId("live-view-mask")).toBeNull();
    });
    // 角标展示的是后端的展示名（ninedrunk），但判定用的是 userID —— 两者可以不同
    expect(screen.getAllByText(/已接管 · ninedrunk/).length).toBeGreaterThan(0);
    expect(await screen.findByRole("button", { name: /释放接管/ })).toBeInTheDocument();
  });

  it("他人接管：遮罩保持，且没有释放按钮（说明判定不是恒真）", async () => {
    loginAs({ id: "uid-me", username: "ninedrunk", nickname: "超级管理员" });
    // 接管者是别人（userID 不同），但展示名恰好与我的 username 相同 ——
    // 这是旧的字符串判据会误判成「我」的陷阱。
    mockBackend({ takeoverById: "uid-other", takeoverBy: "ninedrunk" });

    await renderPage();
    expect(await screen.findByTestId("live-view-mask")).toBeInTheDocument();

    await clickTakeoverAndConfirm();

    await waitFor(() => {
      expect(screen.getAllByText(/已接管 · ninedrunk/).length).toBeGreaterThan(0);
    });
    // 仍然是别人在接管 → 遮罩保留、无释放按钮、仍有「接管」按钮（可抢占）
    expect(screen.getByTestId("live-view-mask")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /释放接管/ })).toBeNull();
  });
});

describe("缺陷 2 补充：接管后鼠标事件真的下发 input（用户视角终判）", () => {
  it("人格 B（昵称≠账号名）接管后：收到帧 → 画布上移动鼠标 → /liveview/input 被调用", async () => {
    loginAs({ id: "uid-b2", username: "ninedrunk", nickname: "超级管理员" });
    const { inputApi } = mockBackend({ takeoverById: "uid-b2", takeoverBy: "ninedrunk" });

    await renderPage();
    await clickTakeoverAndConfirm();
    await waitFor(() => {
      expect(screen.queryByTestId("live-view-mask")).toBeNull();
    });

    // 通过真实的 WS 分流路径喂一帧（不 mock 内部函数）——
    // 页面在挂载时已 installLiveFrameTap，handleWsMessage 会把帧派发给页面。
    //
    // ⚠ context 必须带 takeoverById：后端（W7 / UI-FIX-02 修复后）在每一帧
    // 都用**后端权威 context** 回写（IngestFrame 里用 s.context 而非 Worker
    // 原样透传），所以真实帧一定带这个字段。这里照实模拟，否则前端会把
    // 「没有接管信息的 context」当成「无人接管」而退回遮罩。
    const frameContent = JSON.stringify({
      frame: {
        executionId: EXEC_ID,
        seq: 1,
        timestamp: Date.now(),
        data: "AAAA",
        byteSize: 100,
        width: 1280,
        height: 720,
        deviceWidth: 1280,
        deviceHeight: 720,
      },
      context: {
        executionId: EXEC_ID,
        step: 1,
        totalSteps: 3,
        takeoverBy: "ninedrunk",
        takeoverById: "uid-b2",
      },
    });
    await act(async () => {
      useNoticeStore.getState().handleWsMessage({
        type: "rpa_liveview_frame",
        content: frameContent,
        data: null,
        timestamp: 0,
      } as never);
    });
    // 帧必须真的进了页面 state，否则 deviceRef 仍是 0（坐标换算返回 null）。
    await waitFor(() => {
      expect(screen.getByTestId("rpa-live-canvas")).toBeInTheDocument();
    });

    // 帧到后 deviceRef 有尺寸，坐标换算才有结果 → input 才会下发。
    // ⚠ 必须用 fireEvent.mouseMove 的 bubbles=true（默认）且带 clientX/Y；
    // canvas 的 mousemove 由 LiveCanvas 的 onMouseMove 接收。
    const canvas = screen.getByTestId("rpa-live-canvas");
    // 50ms 节流：第一次移动是 leading，应立即发出（不用等窗口）。
    await act(async () => {
      fireEvent.mouseMove(canvas, { clientX: 200, clientY: 100, bubbles: true });
    });
    // 等待一次微任务/宏任务，让 sendLiveInput 的 promise 链真正跑到端点
    await waitFor(() => {
      expect(inputApi.endpoint).toHaveBeenCalled();
    });
    // 断言 payload 带正确 executionId 与坐标（不是空对象）。
    // 端点 spy 的入参是 (url, body)，故取第 2 个参数。
    const firstCall = inputApi.endpoint.mock.calls[0] as unknown[];
    expect(firstCall[0]).toBe(`/rpa/executions/${EXEC_ID}/liveview/input`);
    const payload = firstCall[1] as { executionId?: string; kind?: string; x?: number; y?: number };
    expect(payload.executionId).toBe(EXEC_ID);
    expect(payload.kind).toBe("mouse_move");
    // 坐标换算：clientX=200 → (200-0)/400 * 1280 = 640；clientY=100 → 100/225*720 = 320
    expect(Math.round(payload.x ?? 0)).toBe(640);
    expect(Math.round(payload.y ?? 0)).toBe(320);
  });

  it("反向：非接管者即使收到帧也不下发任何 input（遮罩拦在源头）", async () => {
    loginAs({ id: "uid-c", username: "ninedrunk", nickname: "超级管理员" });
    const { inputApi } = mockBackend({ takeoverById: "uid-other", takeoverBy: "ninedrunk" });

    await renderPage();
    await clickTakeoverAndConfirm();
    await waitFor(() => {
      expect(screen.getByTestId("live-view-mask")).toBeInTheDocument();
    });

    const canvas = screen.getByTestId("rpa-live-canvas");
    await act(async () => {
      fireEvent.mouseMove(canvas, { clientX: 200, clientY: 100 });
      fireEvent.mouseDown(canvas, { clientX: 200, clientY: 100 });
    });

    expect(inputApi.endpoint).not.toHaveBeenCalled();
  });
});
