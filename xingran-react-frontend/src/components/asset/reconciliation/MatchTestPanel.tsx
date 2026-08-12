/**
 * MatchTestPanel 命中测试面板 (Phase 44 R3 / Plan 44-01 Task 6c)
 *
 * 内容:
 *   - 输入区:IP(Input,必填) + UserID(Select,可选) + DeptID(Select,可选) + "测试"按钮
 *   - 结果区顶部 Card:合并卡片(mergedActions Tag union + finalSeverity Tag + isSilence Badge
 *     + needsUserDept Alert)
 *   - 结果区下方 Table:命中规则列表(name/ip_range/actions/severity_override/scope/expires_at/reason)
 *
 * 用 useQuery + queryKeys.reconciliation.matchTest 入参缓存,enabled=!!testInput.ip
 * 仅在用户点"测试"后才查询。
 *
 * CLAUDE.md 强约束:
 *   - useQuery 依赖稳定(testInput 用 useState 持有原始字符串,queryKey 入参 useMemo)
 *   - 所有 API 调用用 reconciliationApi.exceptionRule.test (不用 raw axios)
 *
 * D-R3-A2-03 / A3-03 锁定:
 *   - 合并卡片形态展示(actions 并集 + finalSeverity + isSilence + needsUserDept)
 *   - 未指定 user/dept 时 needsUserDept=true,提示"需指定 user/dept 才能评估 dept/user scope 规则"
 */

import { useMemo, useState } from "react";
import {
  Form,
  Input,
  Button,
  Card,
  Table,
  Tag,
  Badge,
  Alert,
  Space,
  Empty,
  App,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { ExperimentOutlined } from "@ant-design/icons";
import { useQuery } from "@tanstack/react-query";
import { reconciliationApi } from "@/lib/assetApi";
import { queryKeys } from "@/lib/queryKeys";

interface MatchTestPanelProps {
  /** 嵌入 Drawer 时不显示外层 Card padding */
  embedded?: boolean;
}

interface MatchedRule {
  id: string;
  name: string;
  ipRange: string;
  exceptionActions?: string[];
  severityOverride?: string | null;
  scopeType: string;
  scopeId?: string | null;
  expiresAt?: string | null;
  reason: string;
}

interface TestInput {
  ip: string;
  userId?: string;
  deptId?: string;
}

const MatchTestPanel: React.FC<MatchTestPanelProps> = ({ embedded }) => {
  const { message } = App.useApp();
  const [form] = Form.useForm<TestInput>();
  const [testInput, setTestInput] = useState<TestInput>({ ip: "" });

  // queryKey 入参对象 useMemo 稳定(CLAUDE.md useEffect 强约束)
  const stableQueryKey = useMemo(
    () => testInput,
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [JSON.stringify(testInput)]
  );

  // useQuery enabled 仅在 ip 非空时触发
  const { data, isFetching, refetch } = useQuery({
    queryKey: queryKeys.reconciliation.matchTest(stableQueryKey),
    queryFn: () => reconciliationApi.exceptionRule.test(testInput),
    enabled: !!testInput.ip,
    staleTime: 30 * 1000,
  });

  const handleTest = async () => {
    try {
      const values = await form.validateFields();
      setTestInput({
        ip: values.ip.trim(),
        userId: values.userId?.trim() || undefined,
        deptId: values.deptId?.trim() || undefined,
      });
      // 触发 refetch(若 queryKey 相同,enabled 已设为 ip 非空时自动查)
      await refetch();
    } catch (err) {
      if ((err as { errorFields?: unknown }).errorFields) return;
      message.error((err as Error)?.message ?? "测试失败");
    }
  };

  const matchedRules = (data?.matchedRules ?? []) as MatchedRule[];
  const mergedActions = data?.mergedActions ?? [];
  const finalSeverity = data?.finalSeverity ?? "";
  const isSilence = data?.isSilence ?? false;
  const needsUserDept = data?.needsUserDept ?? false;

  const ruleColumns: ColumnsType<MatchedRule> = [
    {
      title: "规则名称",
      dataIndex: "name",
      key: "name",
      width: 180,
      ellipsis: true,
    },
    {
      title: "IP段",
      dataIndex: "ipRange",
      key: "ipRange",
      width: 150,
      render: (v: string) => <code>{v}</code>,
    },
    {
      title: "动作",
      dataIndex: "exceptionActions",
      key: "exceptionActions",
      width: 200,
      render: (actions?: string[]) => (
        <Space size={4} wrap>
          {(actions ?? []).map((a) => (
            <Tag key={a} color={actionColor(a)}>{a}</Tag>
          ))}
        </Space>
      ),
    },
    {
      title: "严重度覆盖",
      dataIndex: "severityOverride",
      key: "severityOverride",
      width: 100,
      render: (v?: string | null) => (v ? <Tag color="blue">{v}</Tag> : "-"),
    },
    {
      title: "范围",
      key: "scope",
      width: 140,
      render: (_: unknown, r: MatchedRule) => (
        <Tag color={r.scopeType === "global" ? "green" : "orange"}>{r.scopeType}</Tag>
      ),
    },
    {
      title: "过期",
      dataIndex: "expiresAt",
      key: "expiresAt",
      width: 150,
      render: (v?: string | null) => (v ? new Date(v).toLocaleString("zh-CN") : "永久"),
    },
    {
      title: "原因",
      dataIndex: "reason",
      key: "reason",
      ellipsis: true,
    },
  ];

  return (
    <div style={embedded ? { padding: 0 } : { padding: 16 }}>
      {/* 输入区 */}
      <Card title="输入" size="small" style={{ marginBottom: 12 }}>
        <Form form={form} layout="inline">
          <Form.Item
            name="ip"
            label="IP"
            rules={[{ required: true, message: "请输入 IP" }]}
          >
            <Input placeholder="192.168.0.10" style={{ width: 200 }} />
          </Form.Item>
          <Form.Item name="userId" label="User ID">
            <Input placeholder="可选 UUID" style={{ width: 220 }} />
          </Form.Item>
          <Form.Item name="deptId" label="Dept ID">
            <Input placeholder="可选 UUID" style={{ width: 220 }} />
          </Form.Item>
          <Form.Item>
            <Button
              type="primary"
              icon={<ExperimentOutlined />}
              loading={isFetching}
              onClick={handleTest}
            >
              测试
            </Button>
          </Form.Item>
        </Form>
      </Card>

      {/* 合并卡片(结果顶部) */}
      {testInput.ip && data && (
        <Card
          title={
            <Space>
              <span>合并结果</span>
              {isSilence && <Badge status="error" text="已静默" />}
            </Space>
          }
          size="small"
          style={{ marginBottom: 12 }}
        >
          {needsUserDept && (
            <Alert
              type="info"
              showIcon
              message="需指定 user/dept 才能评估 dept/user scope 规则"
              description="当前未传 userID/deptID,仅 global 规则参与合并。dept/user scope 规则需双条件(IP CIDR + scopeID 匹配)才生效。"
              style={{ marginBottom: 12 }}
            />
          )}
          <Space size={12} wrap>
            <div>
              <span style={{ color: "#999", marginRight: 8 }}>合并动作:</span>
              {mergedActions.length === 0 ? (
                <span style={{ color: "#999" }}>(无)</span>
              ) : (
                mergedActions.map((a) => (
                  <Tag key={a} color={actionColor(a)}>{a}</Tag>
                ))
              )}
            </div>
            <div>
              <span style={{ color: "#999", marginRight: 8 }}>最终严重度:</span>
              <Tag color={severityColor(finalSeverity)}>{finalSeverity || "-"}</Tag>
            </div>
          </Space>
        </Card>
      )}

      {/* 命中规则列表 */}
      {testInput.ip && data && (
        <Card title={`命中规则 (${matchedRules.length})`} size="small">
          {matchedRules.length === 0 ? (
            <Empty description="未命中任何 active 规则" />
          ) : (
            <Table
              rowKey="id"
              size="small"
              columns={ruleColumns}
              dataSource={matchedRules}
              pagination={false}
              scroll={{ x: 1100 }}
            />
          )}
        </Card>
      )}

      {/* 初始空态 */}
      {!testInput.ip && (
        <Empty description="输入 IP 后点击测试" />
      )}
    </div>
  );
};

function actionColor(action: string): string {
  switch (action) {
    case "silence": return "red";
    case "no_alert": return "orange";
    case "no_notice": return "gold";
    case "no_workorder": return "purple";
    case "skip_severity": return "blue";
    default: return "default";
  }
}

function severityColor(sev: string): string {
  switch (sev) {
    case "critical": return "red";
    case "high": return "volcano";
    case "medium": return "orange";
    case "low": return "green";
    default: return "default";
  }
}

export default MatchTestPanel;
