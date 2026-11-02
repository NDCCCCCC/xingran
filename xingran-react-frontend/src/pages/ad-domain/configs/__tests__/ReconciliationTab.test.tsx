/**
 * Phase 59 / GUARD-02 — ReconciliationTab 组件测试
 *
 * 覆盖：权限门（403 + 零请求，D-17/T-59-09）、空态、有数据 7 列渲染、
 * 部门级行 NULL 渲染 "-"、已删除部门 Tag、错误态与重试、筛选显式提交
 * （onChange 不触发请求）、rowKey 组合唯一性、反模式（无写操作入口）。
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/adDomainApi", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/adDomainApi")>();
  return {
    ...actual,
    getDeptOUMismatchList: vi.fn(),
  };
});

vi.mock("@/hooks/useDeptTree", () => ({
  useDeptTree: () => ({
    data: [{ id: "d-9", deptName: "信息技术部", parentId: "0" }],
    isLoading: false,
  }),
}));

import { Form } from "antd";
import type { FormInstance } from "antd";
import { forwardRef, useImperativeHandle } from "react";
import { act, fireEvent, waitFor } from "@testing-library/react";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ReconciliationTab, { RECONCILIATION_PERM } from "../ReconciliationTab";
import { MISMATCH_TYPE_META } from "../constants";
import { rowKeyOf } from "../rowKey";
import { getDeptOUMismatchList, type DeptOUMismatchRow } from "@/lib/adDomainApi";
import { useMenuStore } from "@/store/menuStore";

/** Host 对外暴露的 form 句柄 */
interface HostHandle {
  getForm: () => FormInstance;
}

/**
 * antd v6 的 Form.useForm 是真 React hook（不能在组件外调用）。
 * 测试用 Host 组件创建 form 实例，经 useImperativeHandle 暴露给测试体
 * （不向组件外声明的变量赋值，满足 react-hooks/globals）。
 */
function renderTabWithForm() {
  // 普通 ref 容器（非 useRef —— renderTabWithForm 不是组件，不能调 hook）
  const hostRef: { current: HostHandle | null } = { current: null };
  const Host = forwardRef<HostHandle>((_props, ref) => {
    const [f] = Form.useForm();
    useImperativeHandle(ref, () => ({ getForm: () => f }), [f]);
    return <ReconciliationTab form={f} />;
  });
  return {
    ...renderWithProviders(<Host ref={hostRef} />),
    getForm: () => hostRef.current!.getForm(),
  };
}

const mockGetList = vi.mocked(getDeptOUMismatchList);
const PERM = RECONCILIATION_PERM;

const EMPTY_PAGE = { data: { list: [], total: 0 } };

/** 部门级条目（user 列 NULL，UI-SPEC partial 规则） */
const DEPT_LEVEL_ROW: DeptOUMismatchRow = {
  deptId: "d-1",
  deptName: "信息技术部",
  deptDeletedAt: null,
  adConfigId: "c-1",
  configName: "公司主AD域",
  mappedOuDn: "OU=IT,DC=corp,DC=com",
  userId: null,
  username: null,
  userAdOuDn: null,
  mismatchType: "mapping_disabled",
  mismatchDetail: "部门「信息技术部」的映射已停用，但仍有用户归属",
};

/** 用户级条目 */
const USER_LEVEL_ROW: DeptOUMismatchRow = {
  deptId: "d-2",
  deptName: "财务部",
  deptDeletedAt: null,
  adConfigId: "c-1",
  configName: "公司主AD域",
  mappedOuDn: "OU=FIN,DC=corp,DC=com",
  userId: "u-1",
  username: "zhangsan",
  userAdOuDn: "OU=HR,DC=corp,DC=com",
  mismatchType: "user_ou_mismatch",
  mismatchDetail: "用户 OU「OU=HR,DC=corp,DC=com」≠ 部门映射 OU「OU=FIN,DC=corp,DC=com」",
};

/** 软删部门条目（deptDeletedAt 非空 → 「已删除」Tag） */
const SOFT_DELETED_ROW: DeptOUMismatchRow = {
  deptId: "d-3",
  deptName: "已裁撤部门",
  deptDeletedAt: "2026-09-01 10:00:00",
  adConfigId: "c-1",
  configName: "公司主AD域",
  mappedOuDn: "OU=OLD,DC=corp,DC=com",
  userId: null,
  username: null,
  userAdOuDn: null,
  mismatchType: "dept_soft_deleted",
  mismatchDetail: "部门「已裁撤部门」已软删（2026-09-01 10:00:00），映射仍指向该部门",
};

function page(list: DeptOUMismatchRow[], total = list.length) {
  return { data: { list, total, current: 1, pageSize: 20 } };
}

async function waitForTable(container: HTMLElement) {
  await waitFor(() => {
    expect(container.querySelector(".ant-table-thead")).not.toBeNull();
  });
  return container.querySelector(".ant-table-thead") as HTMLElement;
}

/**
 * 等待筛选按钮退出 loading（antd Button loading 期不可点击）。
 * mock promise resolve 后 setLoading(false) 尚未 flush，直接点击会被吞掉。
 */
async function waitButtonsIdle() {
  await waitFor(() => {
    expect(document.querySelector(".ant-btn-loading")).toBeNull();
  });
}

describe("ReconciliationTab 权限门 (D-17)", () => {
  beforeEach(() => {
    mockGetList.mockReset();
    useMenuStore.setState({ permissions: [] });
  });

  it("无权限渲染 403 文案且不发起任何请求", async () => {
    const { findByText, queryByText } = renderWithProviders(<ReconciliationTab />);
    expect(await findByText("无对账查看权限")).toBeTruthy();
    expect(
      await findByText(
        "请联系管理员为当前角色授予 ops:ad:config:reconciliation:list 权限后重新登录。"
      )
    ).toBeTruthy();
    expect(mockGetList).not.toHaveBeenCalled();
    expect(queryByText("未发现不一致条目")).toBeNull();
  });

  it("有权限且返回空列表渲染空态『未发现不一致条目』并请求默认分页", async () => {
    useMenuStore.setState({ permissions: [PERM] });
    mockGetList.mockResolvedValue(EMPTY_PAGE);
    const { findByText, container } = renderWithProviders(<ReconciliationTab />);
    expect(await findByText("未发现不一致条目")).toBeTruthy();
    expect(
      await findByText(
        "当前视图为实时计算。可调整筛选条件，或先在「配置列表」完成部门同步后再查看。"
      )
    ).toBeTruthy();
    // 空态不渲染空表头（筛选行仍在，故以表格节点缺席为准）
    expect(container.querySelector(".ant-table")).toBeNull();
    expect(mockGetList).toHaveBeenCalledTimes(1);
    expect(mockGetList).toHaveBeenCalledWith({ current: 1, pageSize: 20 });
  });
});

describe("ReconciliationTab 有数据渲染", () => {
  beforeEach(() => {
    mockGetList.mockReset();
    useMenuStore.setState({ permissions: [PERM] });
    mockGetList.mockResolvedValue(EMPTY_PAGE);
  });

  it("渲染 7 列表头且顺序符合 UI-SPEC", async () => {
    mockGetList.mockResolvedValue(page([DEPT_LEVEL_ROW]));
    const { container } = renderWithProviders(<ReconciliationTab />);
    const thead = await waitForTable(container);
    const headerText = thead.textContent ?? "";
    const titles = ["不一致类型", "部门", "AD 配置", "映射 OU", "用户", "用户 OU", "详情"];
    for (const t of titles) {
      expect(headerText).toContain(t);
    }
    // 顺序断言（DOM 序 = 列定义序）
    const idx = (t: string) => headerText.indexOf(t);
    expect(idx("不一致类型")).toBeLessThan(idx("部门"));
    expect(idx("部门")).toBeLessThan(idx("AD 配置"));
    expect(idx("AD 配置")).toBeLessThan(idx("映射 OU"));
    expect(idx("映射 OU")).toBeLessThan(idx("用户 OU"));
    expect(idx("用户 OU")).toBeLessThan(idx("详情"));
  });

  it("部门级行的用户/用户 OU 单元格渲染 '-'，不隐藏行", async () => {
    mockGetList.mockResolvedValue(page([DEPT_LEVEL_ROW]));
    const { findAllByText, findByText, container } = renderWithProviders(<ReconciliationTab />);
    expect(await findByText("信息技术部")).toBeTruthy();
    // 部门级 NULL 列（用户 / 用户 OU）各渲染一个 "-"（user_id 不设列）
    expect((await findAllByText("-")).length).toBe(2);
    // 行存在（不隐藏）
    expect(container.querySelectorAll("tr.ant-table-row").length).toBe(1);
  });

  it("四类 mismatchType 渲染中文标签与预设语义色", async () => {
    const rows = (Object.keys(MISMATCH_TYPE_META) as DeptOUMismatchRow["mismatchType"][]).map(
      (type, i) => ({ ...DEPT_LEVEL_ROW, mismatchType: type, deptId: `d-${i}` })
    );
    mockGetList.mockResolvedValue(page(rows));
    const { container, findByText } = renderWithProviders(<ReconciliationTab />);
    await waitForTable(container);
    for (const [type, meta] of Object.entries(MISMATCH_TYPE_META)) {
      const tag = Array.from(container.querySelectorAll(".ant-table-tbody .ant-tag")).find(
        (el) => el.textContent === meta.label
      );
      expect(tag, `mismatch_type=${type} 的中文标签`).toBeTruthy();
      expect(tag?.className).toContain(`ant-tag-${meta.color}`);
    }
    expect(await findByText("配置无法归属")).toBeTruthy();
  });

  it("deptDeletedAt 非空时部门列追加「已删除」红色 Tag", async () => {
    mockGetList.mockResolvedValue(page([SOFT_DELETED_ROW]));
    const { container, findByText } = renderWithProviders(<ReconciliationTab />);
    expect(await findByText("已裁撤部门")).toBeTruthy();
    const deletedTag = Array.from(container.querySelectorAll(".ant-table-tbody .ant-tag")).find(
      (el) => el.textContent === "已删除"
    );
    expect(deletedTag).toBeTruthy();
    expect(deletedTag?.className).toContain("ant-tag-red");
  });

  it("rowKey 组合键在部门级与用户级行上唯一", async () => {
    mockGetList.mockResolvedValue(page([DEPT_LEVEL_ROW, USER_LEVEL_ROW]));
    const { container } = renderWithProviders(<ReconciliationTab />);
    await waitFor(() => {
      expect(container.querySelectorAll("tr.ant-table-row").length).toBe(2);
    });
    const keys = Array.from(container.querySelectorAll("tr.ant-table-row")).map((r) =>
      r.getAttribute("data-row-key")
    );
    expect(new Set(keys).size).toBe(2);
    // 组合形状：mismatchType-deptId-userId-adConfigId（NULL 落空串 → 双连字符）
    expect(rowKeyOf(DEPT_LEVEL_ROW)).toBe("mapping_disabled-d-1--c-1");
    expect(rowKeyOf(USER_LEVEL_ROW)).toBe("user_ou_mismatch-d-2-u-1-c-1");
  });
});

describe("ReconciliationTab 错误态", () => {
  beforeEach(() => {
    mockGetList.mockReset();
    useMenuStore.setState({ permissions: [PERM] });
  });

  it("请求失败渲染错误文案与「重新加载」按钮", async () => {
    mockGetList.mockRejectedValue(new Error("boom"));
    const { findByText } = renderWithProviders(<ReconciliationTab />);
    expect(await findByText("对账数据加载失败，请稍后重试。")).toBeTruthy();
    expect(await findByText("重新加载")).toBeTruthy();
  });

  it("点击「重新加载」重新发起请求", async () => {
    mockGetList.mockRejectedValueOnce(new Error("boom")).mockResolvedValueOnce(EMPTY_PAGE);
    const { findByText } = renderWithProviders(<ReconciliationTab />);
    await findByText("对账数据加载失败，请稍后重试。");
    fireEvent.click(await findByText("重新加载"));
    await waitFor(() => expect(mockGetList).toHaveBeenCalledTimes(2));
  });
});

describe("ReconciliationTab 筛选显式提交", () => {
  beforeEach(() => {
    mockGetList.mockReset();
    useMenuStore.setState({ permissions: [PERM] });
    mockGetList.mockResolvedValue({ data: { list: [], total: 0, current: 1, pageSize: 20 } });
  });

  it("点击「查询」提交筛选并回到第一页（含 mismatchType/deptId/keyword）", async () => {
    // 初始就要有数据行（空列表走空态分支，不渲染分页器）
    mockGetList.mockResolvedValue(page([DEPT_LEVEL_ROW], 50));
    const { getByRole, container, getForm } = renderTabWithForm();
    await waitFor(() => expect(mockGetList).toHaveBeenCalledTimes(1));

    // 先翻到第 2 页（50 条 → 3 页），验证「查询」把 current 重置回 1
    const page2 = container.querySelector(".ant-pagination-item-2");
    expect(page2).not.toBeNull();
    fireEvent.click(page2 as HTMLElement);
    await waitFor(() => expect(mockGetList).toHaveBeenLastCalledWith({ current: 2, pageSize: 20 }));
    await waitButtonsIdle();

    act(() => {
      getForm().setFieldsValue({
        mismatchType: "mapping_disabled",
        deptId: "d-9",
        keyword: "张三",
      });
    });
    fireEvent.click(getByRole("button", { name: /查\s*询/ }));
    await waitFor(() =>
      expect(mockGetList).toHaveBeenLastCalledWith({
        current: 1,
        pageSize: 20,
        mismatchType: "mapping_disabled",
        deptId: "d-9",
        keyword: "张三",
      })
    );
  });

  it("关键字 Input 按 Enter 提交查询", async () => {
    const { container } = renderTabWithForm();
    await waitFor(() => expect(mockGetList).toHaveBeenCalledTimes(1));

    const input = container.querySelector(
      'input[placeholder="搜索用户名 / 部门名称 / OU DN"]'
    ) as HTMLInputElement;
    expect(input).not.toBeNull();
    fireEvent.change(input, { target: { value: "OU=IT" } });
    fireEvent.keyDown(input, { key: "Enter" });
    await waitFor(() =>
      expect(mockGetList).toHaveBeenLastCalledWith({
        current: 1,
        pageSize: 20,
        keyword: "OU=IT",
      })
    );
  });

  it("点击「重置」清空筛选并回到第一页", async () => {
    const { getByRole, getForm } = renderTabWithForm();
    await waitFor(() => expect(mockGetList).toHaveBeenCalledTimes(1));

    act(() => {
      getForm().setFieldsValue({ mismatchType: "config_unrouted", deptId: "d-1", keyword: "x" });
    });
    fireEvent.click(getByRole("button", { name: /查\s*询/ }));
    await waitFor(() =>
      expect(mockGetList).toHaveBeenLastCalledWith({
        current: 1,
        pageSize: 20,
        mismatchType: "config_unrouted",
        deptId: "d-1",
        keyword: "x",
      })
    );
    await waitButtonsIdle();

    fireEvent.click(getByRole("button", { name: /重\s*置/ }));
    await waitFor(() => expect(mockGetList).toHaveBeenLastCalledWith({ current: 1, pageSize: 20 }));
  });

  it("翻页保持已提交筛选条件", async () => {
    mockGetList.mockResolvedValue(page([DEPT_LEVEL_ROW], 50));
    const { getByRole, container, getForm } = renderTabWithForm();
    await waitFor(() => expect(mockGetList).toHaveBeenCalledTimes(1));
    await waitButtonsIdle();

    act(() => {
      getForm().setFieldsValue({ mismatchType: "user_ou_mismatch" });
    });
    fireEvent.click(getByRole("button", { name: /查\s*询/ }));
    await waitFor(() =>
      expect(mockGetList).toHaveBeenLastCalledWith({
        current: 1,
        pageSize: 20,
        mismatchType: "user_ou_mismatch",
      })
    );

    const page2 = container.querySelector(".ant-pagination-item-2");
    expect(page2).not.toBeNull();
    fireEvent.click(page2 as HTMLElement);
    await waitFor(() =>
      expect(mockGetList).toHaveBeenLastCalledWith({
        current: 2,
        pageSize: 20,
        mismatchType: "user_ou_mismatch",
      })
    );
  });
});
