/**
 * AnchorPreviewPanel —— 角色/岗位「机构授权生效范围」推导预览面板
 *
 * 契约来源：`.planning/notes/261118-anchor-scope-contract.md` §六
 *
 * 交互：按钮「预览生效范围」→ 调 `previewRoleAnchor` / `previewPostAnchor`，
 * 展示 `已解析 N 人 / 未解析 M 人 / 共 T 人`，未解析项列成表格（账号 + 原因）。
 * 未解析 > 0 时用 `Alert type="warning"` 明确提示这些人不会获得机构范围。
 *
 * 降级策略（后端预览接口可能尚未上线）：
 * - 请求失败**不抛到页面**，catch 后落 `error` 态，面板内提示"预览接口不可用"；
 * - 角色/岗位未配置锚点（mode=0）时给出说明，不发请求；
 * - 重复点击复用同一按钮的 loading，不叠加并发请求（内部以请求序号做竞态守卫）。
 */
import { useCallback, useEffect, useRef, useState } from "react";
import { Alert, Button, Descriptions, Space, Table, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { EyeOutlined } from "@ant-design/icons";
import { previewPostAnchor, previewRoleAnchor } from "@/lib/systemAnchorApi";
import {
  ANCHOR_MODE_LABELS,
  anchorModeLabel,
  isAnchorLevelParam,
  orgLevelLabel,
} from "@/constants/anchor";
import type { AnchorPreviewItem, AnchorPreviewResult } from "@/types/system";

export type AnchorPreviewKind = "role" | "post";

export interface AnchorPreviewPanelProps {
  /** 角色 or 岗位 */
  kind: AnchorPreviewKind;
  /** 角色/岗位 ID（无值时按钮禁用） */
  targetId?: string;
  /**
   * 当前表单里选中的锚点（用于在未配置时直接提示"未启用机构授权"，
   * 避免对着一条必然空结果的记录发请求）。
   */
  mode?: number | null;
  param?: string | null;
  /** 禁用（如无 view 权限） */
  disabled?: boolean;
}

/** 把 (mode,param) 渲染成人类可读的锚点摘要 */
function describeAnchor(mode?: number | null, param?: string | null): string {
  if (!mode) return anchorModeLabel(0);
  if (mode === 5) {
    return isAnchorLevelParam(param)
      ? `${ANCHOR_MODE_LABELS[5]}：${orgLevelLabel(Number(param))}`
      : ANCHOR_MODE_LABELS[5];
  }
  if (mode === 2) {
    return param ? `${ANCHOR_MODE_LABELS[2]}：指定部门` : ANCHOR_MODE_LABELS[2];
  }
  return anchorModeLabel(mode);
}

const unresolvedColumns: ColumnsType<AnchorPreviewItem> = [
  {
    title: "账号",
    dataIndex: "username",
    key: "username",
    width: 160,
    render: (username: string | undefined, record) => (
      <span className="xr-cell-id">{username || record.userId || "—"}</span>
    ),
  },
  {
    title: "姓名",
    dataIndex: "nickname",
    key: "nickname",
    width: 120,
    render: (nickname: string | undefined) => nickname || "—",
  },
  {
    title: "未解析原因",
    dataIndex: "reason",
    key: "reason",
    render: (reason: string | undefined) => reason || "未命中锚点",
  },
];

export function AnchorPreviewPanel({
  kind,
  targetId,
  mode,
  param,
  disabled = false,
}: AnchorPreviewPanelProps) {
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<AnchorPreviewResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  // 竞态守卫：连续点击时只接受最后一次请求的结果
  const requestSeq = useRef(0);
  useEffect(
    () => () => {
      requestSeq.current += 1;
    },
    []
  );

  // 目标或锚点变化 → 清掉旧的预览结果（避免展示另一条记录的结论）
  const signature = `${targetId ?? ""}|${mode ?? 0}|${param ?? ""}`;
  useEffect(() => {
    setResult(null);
    setError(null);
  }, [signature]);

  const handlePreview = useCallback(async () => {
    if (!targetId) return;
    const seq = ++requestSeq.current;
    setLoading(true);
    setError(null);
    try {
      const next =
        kind === "role" ? await previewRoleAnchor(targetId) : await previewPostAnchor(targetId);
      if (seq !== requestSeq.current) return;
      setResult(next);
    } catch (e) {
      if (seq !== requestSeq.current) return;
      // 优雅降级：不让预览失败打断页面（后端接口可能尚未上线）
      console.warn(`[AnchorPreviewPanel] ${kind} 锚点预览失败（已降级）:`, e);
      setResult(null);
      setError("预览接口暂不可用（后端可能尚未上线），本面板不影响保存与列表功能。");
    } finally {
      if (seq === requestSeq.current) setLoading(false);
    }
  }, [kind, targetId]);

  const anchorConfigured = typeof mode === "number" && mode !== 0;
  const unresolved = result?.unresolved ?? [];
  const hasUnresolved = (result?.unresolvedCount ?? 0) > 0 || unresolved.length > 0;

  return (
    <div style={{ marginTop: 8 }}>
      <Space wrap>
        <Button
          size="small"
          icon={<EyeOutlined />}
          onClick={handlePreview}
          loading={loading}
          disabled={disabled || !targetId}
          title={disabled ? "当前账号无查看权限" : undefined}
        >
          预览生效范围
        </Button>
        {typeof mode === "number" && (
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            当前锚点：{describeAnchor(mode, param)}
          </Typography.Text>
        )}
      </Space>

      {!anchorConfigured && (
        <Alert
          style={{ marginTop: 8 }}
          type="info"
          showIcon
          title="未启用机构授权：当前记录没有配置机构锚点（模式=不参与），预览只会得到空结果。"
        />
      )}

      {error && <Alert style={{ marginTop: 8 }} type="warning" showIcon title={error} />}

      {result && !error && (
        <div style={{ marginTop: 12 }}>
          <Descriptions size="small" column={3} bordered={false}>
            <Descriptions.Item label="已解析">
              <Tag color="green">{result.resolvedCount} 人</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="未解析">
              <Tag color={result.unresolvedCount > 0 ? "orange" : "default"}>
                {result.unresolvedCount} 人
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="任职人合计">{result.totalCount} 人</Descriptions.Item>
          </Descriptions>

          {hasUnresolved && (
            <Alert
              style={{ marginTop: 8 }}
              type="warning"
              showIcon
              title={`有 ${result.unresolvedCount} 人未解析到锚点机构`}
              description="这些人不会获得机构范围，将按原有角色范围生效（不会回退到其本部门，也不会因此收窄既有可见范围）。"
            />
          )}

          {(result.warnings ?? []).length > 0 && (
            <Alert
              style={{ marginTop: 8 }}
              type="info"
              showIcon
              title="提示"
              description={(result.warnings ?? []).join("；")}
            />
          )}

          {hasUnresolved && (
            <Table<AnchorPreviewItem>
              style={{ marginTop: 8 }}
              size="small"
              rowKey={(record) => record.userId}
              columns={unresolvedColumns}
              dataSource={unresolved}
              pagination={false}
              locale={{ emptyText: "无未解析任职人" }}
            />
          )}
        </div>
      )}
    </div>
  );
}

export default AnchorPreviewPanel;
