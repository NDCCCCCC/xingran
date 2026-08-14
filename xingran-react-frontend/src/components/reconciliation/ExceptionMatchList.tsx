/**
 * ExceptionMatchList — Phase 45 R4 / D-A3-03
 *
 * 抽屉"例外规则"Tab 内容:命中该资产 IP 段的当前生效例外规则列表。
 *
 * 锁定决策 (UI-SPEC):
 *   - antd List size="small"
 *   - 每项 3 行:规则名 + scope Tag / IP range / actions + reason + 有效期
 *   - actionTagColor helper 复用 R3 exception-rules/index.tsx 行为
 *   - 空态:"当前没有例外规则覆盖该资产所在 IP 段。" + "去创建例外规则" 按钮
 *   - 加载态:Spin
 *
 * 数据源:sys_reconciliation_exception WHERE is_active=0 AND deleted_at IS NULL
 *   AND ipRange CIDR 包含该资产 IP。Plan 02 接入实际查询;Plan 01 固定组件契约。
 *
 * Phase 45 R4 / Plan 02: 抽屉 "申请例外" 跳转携带 assetIp + conflictType 预填
 */
import React, { useCallback } from "react";
import { List, Tag, Empty, Spin, Button, Space } from "antd";
import dayjs from "dayjs";

export interface ExceptionRuleItem {
  id: string;
  name: string;
  scopeType: "global" | "building" | "floor" | "dept" | "user";
  ipRange: string;
  exceptionActions: string[];
  reason?: string;
  expiresAt?: string | null;
}

export interface ExceptionMatchListProps {
  rules: ExceptionRuleItem[];
  loading: boolean;
  /** 当前选中资产的 IP(预填到新建规则页) */
  assetIp?: string;
  /** 当前选中资产的冲突类型(预填到新建规则页) */
  conflictType?: string;
  /** 兜底回调(由 ReconciliationDrawer 注入,优先级低于本组件内 navigate) */
  onCreateRule?: () => void;
}

// actionTagColor 给 exception action 上色(运维识别)
// 复用 R3 exception-rules/index.tsx:478 行为
function actionTagColor(action: string): string {
  switch (action) {
    case "silence":
      return "red";
    case "no_alert":
      return "orange";
    case "no_notice":
      return "gold";
    case "no_workorder":
      return "purple";
    case "skip_severity":
      return "blue";
    default:
      return "default";
  }
}

const SCOPE_COLOR: Record<string, string> = {
  global: "green",
  building: "orange",
  floor: "blue",
  dept: "cyan",
  user: "purple",
};

const ListInner: React.FC<ExceptionMatchListProps> = ({
  rules,
  loading,
  assetIp,
  conflictType,
  onCreateRule,
}) => {
  // Plan 02 锁定(SC9): 申请例外按钮携带 assetIp + conflictType query 参数
  // (ReconciliationDrawer 顶部"申请例外"按钮携带 workstationId + assetIp + conflictType 三个参数,
  //  本组件的"去创建例外规则"按钮只携带 assetIp + conflictType 两个参数 — 因为本组件在例外规则 Tab,
  //  上下文是单条资产,工位 ID 已隐含在 ReconciliationDrawer 的 state 中可由父级注入)
  const handleCreateRule = useCallback(() => {
    if (onCreateRule) {
      // 父级已注入 navigate(携带 workstationId) → 优先用父级回调
      onCreateRule();
      return;
    }
    const url = `/asset/reconciliation/exception-rules/new?assetIp=${encodeURIComponent(assetIp ?? "")}&conflictType=${encodeURIComponent(conflictType ?? "")}`;
    window.open(url, "_blank");
  }, [onCreateRule, assetIp, conflictType]);

  if (loading) {
    return <Spin />;
  }

  if (!rules || rules.length === 0) {
    return (
      <Empty description="当前没有例外规则覆盖该资产所在 IP 段。">
        <Button type="primary" onClick={handleCreateRule}>
          去创建例外规则
        </Button>
      </Empty>
    );
  }

  return (
    <List
      size="small"
      dataSource={rules}
      renderItem={(rule) => (
        <List.Item>
          <div style={{ width: "100%" }}>
            <div>
              <strong>{rule.name}</strong>{" "}
              <Tag color={SCOPE_COLOR[rule.scopeType] ?? "default"}>{rule.scopeType}</Tag>
            </div>
            <div style={{ marginTop: 4 }}>
              <code style={{ fontSize: 13 }}>{rule.ipRange}</code>
            </div>
            <Space size={4} wrap style={{ marginTop: 4 }}>
              {rule.exceptionActions.map((a) => (
                <Tag key={a} color={actionTagColor(a)}>
                  {a}
                </Tag>
              ))}
            </Space>
            {rule.reason && (
              <div style={{ marginTop: 4, fontSize: 12, color: "#6b7280" }}>{rule.reason}</div>
            )}
            {rule.expiresAt && (
              <div style={{ marginTop: 4, fontSize: 12, color: "#6b7280" }}>
                有效期至 {dayjs(rule.expiresAt).format("YYYY-MM-DD")}
              </div>
            )}
          </div>
        </List.Item>
      )}
    />
  );
};

export const ExceptionMatchList = React.memo(ListInner);
ExceptionMatchList.displayName = "ExceptionMatchList";
