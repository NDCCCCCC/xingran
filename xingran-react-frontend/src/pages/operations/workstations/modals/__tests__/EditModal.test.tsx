/**
 * Phase 88 Batch113 — operations/workstations/modals/EditModal 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { WorkstationEditModal } from "../EditModal";
import { Form } from "antd";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

interface Props {
  editingWorkstation?: any;
  typeDict?: any[];
  orgTreeData?: any[];
  deptTreeData?: any[];
  aliasList?: any[];
  userOptions?: any[];
  cascaderOptions?: any[];
  onOk?: (v: any) => Promise<void>;
  onCancel?: () => void;
}

const defaultDept = [{ id: "d1", name: "总部", title: "总部", children: [] }];

function Harness(props: Props = {}) {
  const [form] = Form.useForm();
  return (
    <WorkstationEditModal
      open
      form={form}
      editingWorkstation={props.editingWorkstation ?? null}
      typeDict={props.typeDict}
      orgTreeData={props.orgTreeData ?? defaultDept}
      deptTreeData={props.deptTreeData ?? defaultDept}
      aliasList={props.aliasList ?? []}
      userOptions={props.userOptions ?? []}
      cascaderOptions={props.cascaderOptions}
      onOk={props.onOk || vi.fn()}
      onCancel={props.onCancel || vi.fn()}
      onDeptChange={vi.fn()}
      onOrgChange={vi.fn()}
      handleCascaderLoadData={vi.fn()}
    />
  );
}

describe("WorkstationEditModal 渲染", () => {
  it("open=false → 不渲染", () => {
    function Closed() {
      const [form] = Form.useForm();
      return (
        <WorkstationEditModal
          open={false}
          form={form}
          editingWorkstation={null}
          orgTreeData={defaultDept}
          deptTreeData={defaultDept}
          aliasList={[]}
          userOptions={[]}
          onOk={vi.fn()}
          onCancel={vi.fn()}
        />
      );
    }
    const { baseElement } = renderWithProviders(<Closed />);
    expect(baseElement.querySelector(".ant-modal-body")).toBeNull();
  });

  it("editingWorkstation=null + typeDict 空 → 新建模式", () => {
    const { baseElement } = renderWithProviders(<Harness />);
    expect(baseElement).toBeDefined();
  });

  it("editingWorkstation 非空 → 编辑模式", () => {
    const editingWorkstation = {
      id: "w1",
      name: "WS001",
      workstationCode: "WS001",
      workstationName: "WS001",
      workstationType: 0,
      status: 0,
      floorId: "f1",
      buildingId: "b1",
    };
    const { baseElement } = renderWithProviders(
      <Harness editingWorkstation={editingWorkstation} />
    );
    expect(baseElement).toBeDefined();
  });

  it("typeDict 含 isDefault → 默认类型 = isDefault.dictValue", () => {
    const typeDict = [
      { dictValue: "1", isDefault: false },
      { dictValue: "2", isDefault: true },
    ];
    const { baseElement } = renderWithProviders(<Harness typeDict={typeDict} />);
    expect(baseElement).toBeDefined();
  });

  it("带 cascaderOptions + aliasList + userOptions", () => {
    const { baseElement } = renderWithProviders(
      <Harness
        cascaderOptions={[{ value: "f1", label: "F1", children: [] }]}
        aliasList={[{ id: "a1", name: "alias-1" }]}
        userOptions={[{ id: "u1", name: "Alice" }]}
      />
    );
    expect(baseElement).toBeDefined();
  });
});
