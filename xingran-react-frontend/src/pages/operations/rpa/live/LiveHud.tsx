/**
 * Phase 155 / LIVEVIEW-04：实时画面叠加层（HUD）
 *
 * 定位：**情境感知**而非装饰 —— 运维在看画面时必须同时知道「跑到第几步、
 * 在做什么动作、刚发生了什么、模型拿到的变量被脱成什么样、现在是人还是机器
 * 在操作」。因此 HUD 是绝对定位在 canvas 之上的常驻层，而不是可折叠的附加面板。
 *
 * 纯展示组件：只读 props，不自己发请求（所有动作经回调上抛给页面）。
 */

import { useMemo } from "react";
import { Space, Tag, Tooltip, Typography } from "antd";
import { AppSelect } from "@/components/ui";
import {
  actionTypeLabel,
  LIVE_MODE_OPTIONS,
  liveModeLabel,
  type FrameStats,
} from "./liveFrameParser";
import type { LiveContext, LiveMode } from "./types";

const { Text } = Typography;

/** 变量最多展示条数（其余收进「更多」气泡） */
export const HUD_MAX_VARIABLES = 8;

export interface LiveHudProps {
  /** 叠加层情境（来自 rpa_liveview_state / state 端点 / 帧内 context） */
  context: LiveContext | null;
  /** 当前协作模式（Select 受控值） */
  mode: LiveMode;
  /** 切换模式回调（页面负责二次确认 + 落审计） */
  onModeChange: (mode: LiveMode) => void;
  /** 帧率 / 带宽（可选，缺省时显示 0） */
  stats?: FrameStats;
  /** 是否已拿到首帧（false 时给"等待画面"提示） */
  hasFrame?: boolean;
}

export function LiveHud({ context, mode, onModeChange, stats, hasFrame = false }: LiveHudProps) {
  const takeoverBy = context?.takeoverBy ?? "";
  const variables = context?.variables;

  // 变量最多 8 条，余量合并成「更多」气泡（脱敏值原样展示，不做任何还原猜测）
  const variableEntries = useMemo(() => Array.from(Object.entries(variables ?? {})), [variables]);
  const shownVariables = variableEntries.slice(0, HUD_MAX_VARIABLES);
  const restVariables = variableEntries.slice(HUD_MAX_VARIABLES);

  const step = context?.step ?? 0;
  const totalSteps = context?.totalSteps ?? 0;

  return (
    <div
      className="rpa-live-hud"
      style={{
        position: "absolute",
        inset: 0,
        pointerEvents: "none", // 不吃鼠标事件（画面交互必须穿透到 canvas）
        padding: 12,
        display: "flex",
        flexDirection: "column",
        justifyContent: "space-between",
        gap: 8,
      }}
    >
      {/* ============ 顶部：角标 + 步骤/动作 + 模式 ============ */}
      <div style={{ display: "flex", flexWrap: "wrap", alignItems: "center", gap: 8 }}>
        {takeoverBy ? (
          <span
            data-testid="live-hud-badge"
            style={{
              background: "var(--theme-error-bg)",
              color: "var(--theme-error)",
              border: "1px solid var(--theme-error)",
              borderRadius: 4,
              padding: "1px 8px",
              fontSize: 12,
              fontWeight: 600,
            }}
          >
            已接管 · {takeoverBy}
          </span>
        ) : (
          <span
            data-testid="live-hud-badge"
            style={{
              background: "var(--theme-success-bg)",
              color: "var(--theme-success)",
              border: "1px solid var(--theme-success)",
              borderRadius: 4,
              padding: "1px 8px",
              fontSize: 12,
              fontWeight: 600,
            }}
          >
            自动执行
          </span>
        )}

        <span
          data-testid="live-hud-step"
          style={{
            background: "rgba(0, 0, 0, 0.6)",
            color: "var(--theme-bg-surface)",
            borderRadius: 4,
            padding: "1px 8px",
            fontSize: 12,
          }}
        >
          步骤 {step}/{totalSteps}
        </span>

        <span
          data-testid="live-hud-action"
          style={{
            background: "rgba(0, 0, 0, 0.6)",
            color: "var(--theme-bg-surface)",
            borderRadius: 4,
            padding: "1px 8px",
            fontSize: 12,
          }}
        >
          {actionTypeLabel(context?.actionType)}
        </span>

        {context?.status && (
          <span
            style={{
              background: "rgba(0, 0, 0, 0.6)",
              color: "var(--theme-bg-surface)",
              borderRadius: 4,
              padding: "1px 8px",
              fontSize: 12,
            }}
          >
            {context.status}
          </span>
        )}

        {/* 模式切换：pointerEvents 显式打开（HUD 整体穿透） */}
        <div style={{ pointerEvents: "auto", marginLeft: "auto", minWidth: 150 }}>
          <AppSelect
            aria-label="协作模式"
            size="small"
            value={mode}
            showSearch={false}
            options={LIVE_MODE_OPTIONS}
            onChange={(value) => onModeChange(value as LiveMode)}
            style={{ width: 150 }}
          />
        </div>
      </div>

      {/* ============ 底部：最近日志 + 脱敏变量 + 帧率 ============ */}
      <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
        {context?.lastLog && (
          <div
            style={{
              background: "rgba(0, 0, 0, 0.6)",
              borderRadius: 4,
              padding: "4px 8px",
              maxWidth: "100%",
            }}
          >
            <Tooltip title={context.lastLog} placement="topLeft">
              <Text
                style={{
                  color: "var(--theme-bg-surface)",
                  fontSize: 12,
                  display: "block",
                  whiteSpace: "nowrap",
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                }}
              >
                {context.lastLog}
              </Text>
            </Tooltip>
          </div>
        )}

        {variableEntries.length > 0 && (
          <div
            style={{
              background: "rgba(0, 0, 0, 0.6)",
              borderRadius: 4,
              padding: "4px 8px",
              display: "flex",
              flexWrap: "wrap",
              gap: 6,
              alignItems: "center",
            }}
          >
            <Text style={{ color: "var(--theme-bg-surface)", fontSize: 12 }}>变量：</Text>
            {shownVariables.map(([key, value]) => (
              <Tag key={key} style={{ margin: 0, fontSize: 11 }}>
                {key}={value}
              </Tag>
            ))}
            {restVariables.length > 0 && (
              <Tooltip
                placement="topLeft"
                title={restVariables.map(([key, value]) => (
                  <div key={key}>
                    {key}={value}
                  </div>
                ))}
              >
                <Tag style={{ margin: 0, fontSize: 11, cursor: "help" }} color="default">
                  更多 {restVariables.length} 条
                </Tag>
              </Tooltip>
            )}
          </div>
        )}

        <Space size={8} style={{ pointerEvents: "none" }}>
          {!hasFrame && (
            <Text style={{ color: "var(--theme-bg-surface)", fontSize: 11 }}>等待画面…</Text>
          )}
          {stats && (
            <Text style={{ color: "var(--theme-bg-surface)", fontSize: 11 }}>
              {stats.fps} fps / {stats.mbps} Mbps
            </Text>
          )}
          <Text style={{ color: "var(--theme-bg-surface)", fontSize: 11 }}>
            模式：{liveModeLabel(mode)}
          </Text>
        </Space>
      </div>
    </div>
  );
}

export default LiveHud;
