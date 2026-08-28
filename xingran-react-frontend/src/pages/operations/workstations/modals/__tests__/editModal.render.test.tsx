import { describe, it, expect, vi } from "vitest";
import { Form } from "antd";
import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import { WorkstationEditModal } from "../EditModal";

function Harness({ open }: { open: boolean }) {
  const [form] = Form.useForm();
  return (
    <WorkstationEditModal
      open={open}
      form={form}
      editingWorkstation={null}
      orgTreeData={[]}
      deptTreeData={[]}
      userOptions={[]}
      onSubmit={vi.fn()}
      onCancel={vi.fn()}
    />
  );
}

describe("WorkstationEditModal 渲染", () => {
  it("closed renders without error", async () => {
    const { rendered } = renderPageWithEndpoints(<Harness open={false} />, {});
    await vi.waitFor(() => expect(rendered.container.firstChild).not.toBeNull(), { timeout: 8000 });
  });
});
