/**
 * Phase 88 Batch21 — system/notice 子组件 props 渲染
 * ChannelSelector/TargetSelector/RecurrenceConfig/StatisticsDrawer
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { ConfigProvider, Form } from "antd";
import { ChannelSelector } from "../ChannelSelector";
import { TargetSelector } from "../TargetSelector";
import { RecurrenceConfig } from "../RecurrenceConfig";
import { StatisticsDrawer } from "../StatisticsDrawer";

function wrap(ui: React.ReactElement) {
  return render(<ConfigProvider>{ui}</ConfigProvider>);
}

describe("notice/components — ChannelSelector", () => {
  it("renders with selected channels", () => {
    const { baseElement } = wrap(
      <ChannelSelector
        selectedChannels={["email"]}
        selectedAPIConfigId={undefined}
        apiConfigs={[]}
        loadingAPIConfigs={false}
        customEmails=""
        customWeComUsers=""
        onChannelsChange={vi.fn()}
        onAPIConfigChange={vi.fn()}
        onCustomEmailsChange={vi.fn()}
        onCustomWeComUsersChange={vi.fn()}
      />
    );
    expect(baseElement.innerHTML.length).toBeGreaterThan(100);
  });

  it("renders all channels selected", () => {
    const { baseElement } = wrap(
      <ChannelSelector
        selectedChannels={["email", "wecom", "api", "site"]}
        selectedAPIConfigId="cfg1"
        apiConfigs={[{ id: "cfg1", name: "Webhook" } as any]}
        loadingAPIConfigs={false}
        customEmails="a@test.com,b@test.com"
        customWeComUsers="user1|user2"
        onChannelsChange={vi.fn()}
        onAPIConfigChange={vi.fn()}
        onCustomEmailsChange={vi.fn()}
        onCustomWeComUsersChange={vi.fn()}
      />
    );
    expect(baseElement.innerHTML.length).toBeGreaterThan(200);
  });
});

describe("notice/components — TargetSelector", () => {
  it("renders with target depts/roles/users", () => {
    const { baseElement } = wrap(
      <TargetSelector
        targetType={1}
        targetDepts={[]}
        targetRoles={[]}
        targetUsers={[]}
        deptTree={[{ key: "d1", title: "信息部" }]}
        roles={[{ key: "r1", title: "管理员" }]}
        users={[{ key: "u1", title: "张三" }]}
        loadingDepts={false}
        loadingRoles={false}
        loadingUsers={false}
        onDeptChange={vi.fn()}
        onRoleChange={vi.fn()}
        onUserChange={vi.fn()}
      />
    );
    expect(baseElement.innerHTML.length).toBeGreaterThan(100);
  });
});

describe("notice/components — RecurrenceConfig", () => {
  it("renders inside Form (Cron + EndDate fields)", () => {
    const { baseElement } = render(
      <ConfigProvider>
        <Form>
          <RecurrenceConfig />
        </Form>
      </ConfigProvider>
    );
    expect(baseElement.innerHTML).toContain("Cron 表达式");
    expect(baseElement.innerHTML).toContain("结束时间");
  });
});

describe("notice/components — StatisticsDrawer", () => {
  it("renders closed without statistics", () => {
    const { baseElement } = wrap(
      <StatisticsDrawer visible={false} onClose={vi.fn()} loading={false} statistics={null} />
    );
    expect(baseElement).not.toBeNull();
  });

  it("renders open with statistics", () => {
    const { baseElement } = wrap(
      <StatisticsDrawer
        visible
        onClose={vi.fn()}
        loading={false}
        statistics={
          {
            total: 100,
            readCount: 80,
            unreadCount: 20,
            readRate: 0.8,
          } as any
        }
      />
    );
    expect(baseElement.innerHTML.length).toBeGreaterThan(100);
  });
});
