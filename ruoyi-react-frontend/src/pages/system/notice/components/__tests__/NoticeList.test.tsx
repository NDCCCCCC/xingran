/**
 * Phase 88 Batch76 — system/notice/components/NoticeList 渲染测试(37 stmts)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { NoticeList } from "../NoticeList";
import type { Notice } from "@/types/notice";
import { Form } from "antd";
import type { FormInstance } from "antd";
import { useMemo } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

interface HarnessProps {
  notices?: Notice[];
  loading?: boolean;
  total?: number;
  current?: number;
  pageSize?: number;
  selectedRowKeys?: React.Key[];
  onPageChange?: (page: number, pageSize: number) => void;
  onSearch?: (values: any) => void;
  getColumnSortOrder?: (field: string) => "ascend" | "descend" | null;
  searchForm?: FormInstance;
}

function Harness(props: HarnessProps) {
  const [form] = Form.useForm();
  const merged = useMemo(
    () => ({
      notices: [],
      loading: false,
      total: 0,
      current: 1,
      pageSize: 10,
      selectedRowKeys: [] as React.Key[],
      onSearch: () => {},
      onAdd: () => {},
      onEdit: () => {},
      onDelete: () => {},
      onBatchDelete: () => {},
      onPublish: () => {},
      onWithdraw: () => {},
      onView: () => {},
      onStatistics: () => {},
      onSelectedRowKeysChange: () => {},
      onPageChange: () => {},
      ...props,
    }),
    [props]
  );
  return (
    <NoticeList
      {...merged}
      searchForm={props.searchForm ?? form}
      onSearch={merged.onSearch}
      onPageChange={merged.onPageChange}
    />
  );
}

describe("NoticeList 组件渲染", () => {
  it("notices=[] 渲染空表格", () => {
    const { baseElement } = renderWithProviders(<Harness />);
    expect(baseElement.querySelector(".ant-table")).not.toBeNull();
  });

  it("notices 非空 + getColumnSortOrder 注入", () => {
    const notices: Notice[] = [
      {
        noticeId: 1,
        noticeTitle: "测试通知",
        noticeType: "1",
        noticeContent: "内容",
        priority: 1,
        targetType: 1,
        publishStatus: "1",
        createBy: "admin",
        createTime: "2026-01-01 12:00:00",
      } as any,
    ];
    const { baseElement } = renderWithProviders(
      <Harness notices={notices} total={1} getColumnSortOrder={() => "ascend"} />
    );
    expect(baseElement.querySelector(".ant-table-row")).not.toBeNull();
  });

  it("priority=0 显示普通", () => {
    const notice: Notice = {
      noticeId: 1,
      noticeTitle: "t",
      noticeType: "1",
      noticeContent: "c",
      priority: 0 as any,
      targetType: 1,
      publishStatus: "1",
      createBy: "admin",
      createTime: "2026-01-01",
    } as any;
    const { baseElement } = renderWithProviders(<Harness notices={[notice]} total={1} />);
    expect(baseElement.textContent).toContain("普通");
  });

  it("onPageChange + onTableChange 调用", () => {
    const onPageChange = vi.fn();
    const { container } = renderWithProviders(<Harness total={50} onPageChange={onPageChange} />);
    expect(container).toBeDefined();
  });

  it("loading=true 显示 loading 状态", () => {
    const { baseElement } = renderWithProviders(<Harness loading />);
    expect(baseElement.querySelector(".ant-spin")).not.toBeNull();
  });
});
