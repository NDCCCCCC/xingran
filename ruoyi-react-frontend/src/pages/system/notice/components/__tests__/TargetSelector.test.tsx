/**
 * Phase 88 Batch143 — pages/system/notice/components/TargetSelector 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { TargetSelector } from "../TargetSelector";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("TargetSelector", () => {
  it("targetType=0 → 显示'所有用户'", () => {
    const { baseElement } = render(
      <TargetSelector
        targetType={0}
        targetDepts={[]}
        targetRoles={[]}
        targetUsers={[]}
        deptTree={[]}
        roles={[]}
        users={[]}
        loadingDepts={false}
        loadingRoles={false}
        loadingUsers={false}
        onDeptChange={vi.fn()}
        onRoleChange={vi.fn()}
        onUserChange={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("所有用户");
  });

  it("targetType=1 → 渲染 Tree + 显示已选部门数", () => {
    const { baseElement } = render(
      <TargetSelector
        targetType={1}
        targetDepts={["d1", "d2"]}
        targetRoles={[]}
        targetUsers={[]}
        deptTree={[{ id: "d1", title: "Dept1", children: [] } as any]}
        roles={[]}
        users={[]}
        loadingDepts={false}
        loadingRoles={false}
        loadingUsers={false}
        onDeptChange={vi.fn()}
        onRoleChange={vi.fn()}
        onUserChange={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("已选择 2 个部门");
  });

  it("targetType=2 → 渲染 Checkbox.Group + 显示已选角色数", () => {
    const { baseElement } = render(
      <TargetSelector
        targetType={2}
        targetDepts={[]}
        targetRoles={["r1"]}
        targetUsers={[]}
        deptTree={[]}
        roles={
          [
            { id: "r1", roleName: "管理员", roleKey: "admin" },
            { id: "r2", roleName: "用户", roleKey: "user" },
          ] as any
        }
        users={[]}
        loadingDepts={false}
        loadingRoles={false}
        loadingUsers={false}
        onDeptChange={vi.fn()}
        onRoleChange={vi.fn()}
        onUserChange={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("管理员");
    expect(baseElement.textContent).toContain("已选择 1 个角色");
  });

  it("targetType=2 → 点击 checkbox → onRoleChange", () => {
    const onRoleChange = vi.fn();
    const { baseElement } = render(
      <TargetSelector
        targetType={2}
        targetDepts={[]}
        targetRoles={[]}
        targetUsers={[]}
        deptTree={[]}
        roles={[{ id: "r1", roleName: "管理员", roleKey: "admin" }] as any}
        users={[]}
        loadingDepts={false}
        loadingRoles={false}
        loadingUsers={false}
        onDeptChange={vi.fn()}
        onRoleChange={onRoleChange}
        onUserChange={vi.fn()}
      />,
      { wrapper }
    );
    const checkboxes = baseElement.querySelectorAll("input[type='checkbox']");
    fireEvent.click(checkboxes[0]);
    expect(onRoleChange).toHaveBeenCalled();
  });

  it("targetType=3 → 渲染 Select + 显示已选用户数", () => {
    const { baseElement } = render(
      <TargetSelector
        targetType={3}
        targetDepts={[]}
        targetRoles={[]}
        targetUsers={["u1", "u2", "u3"]}
        deptTree={[]}
        roles={[]}
        users={
          [
            { id: "u1", username: "alice", nickname: "Alice" },
            { id: "u2", username: "bob", nickname: "Bob" },
          ] as any
        }
        loadingDepts={false}
        loadingRoles={false}
        loadingUsers={false}
        onDeptChange={vi.fn()}
        onRoleChange={vi.fn()}
        onUserChange={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("已选择 3 个用户");
    // Filter out users with null id
    expect(baseElement.textContent).toContain("Alice");
  });

  it("targetType=1 + loadingDepts=true → Spin 渲染", () => {
    const { baseElement } = render(
      <TargetSelector
        targetType={1}
        targetDepts={[]}
        targetRoles={[]}
        targetUsers={[]}
        deptTree={[]}
        roles={[]}
        users={[]}
        loadingDepts
        loadingRoles={false}
        loadingUsers={false}
        onDeptChange={vi.fn()}
        onRoleChange={vi.fn()}
        onUserChange={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.querySelector(".ant-spin")).toBeTruthy();
  });

  it("未匹配 targetType → 返回 null", () => {
    const { baseElement } = render(
      <TargetSelector
        targetType={99}
        targetDepts={[]}
        targetRoles={[]}
        targetUsers={[]}
        deptTree={[]}
        roles={[]}
        users={[]}
        loadingDepts={false}
        loadingRoles={false}
        loadingUsers={false}
        onDeptChange={vi.fn()}
        onRoleChange={vi.fn()}
        onUserChange={vi.fn()}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toBe("");
  });
});
