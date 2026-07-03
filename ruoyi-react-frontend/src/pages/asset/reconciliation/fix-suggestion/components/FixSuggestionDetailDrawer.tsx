/**
 * 修复建议详情 Drawer (Phase 46 R5)
 *
 * 3 Tab:
 *   - 冲突摘要:raw_snapshot + conflict_type / severity / confidenceScore
 *   - 修复详情:时间轴 + 当前 vs 建议 user_id + pre_fix_user_id + 7d 倒计时
 *   - 历史变更:同 exception_id 的所有 fix_suggestion 记录
 */

import { useEffect } from "react";
import { Drawer, Tabs, Descriptions, Tag, Timeline, Empty, Spin } from "antd";
import { useQuery } from "@tanstack/react-query";
import { fixSuggestionApi, type FixSuggestionListItem, type FixStatus } from "@/lib/assetApi";
import { queryKeys } from "@/lib/queryKeys";

interface FixSuggestionDetailDrawerProps {
  open: boolean;
  suggestionId: string | null;
  onClose: () => void;
}

const fixStatusColor: Record<FixStatus, string> = {
  pending: "gold",
  accepted: "blue",
  rejected: "default",
  applied: "green",
  rolled_back: "magenta",
  failed: "red",
};

const fixStatusLabel: Record<FixStatus, string> = {
  pending: "待处理",
  accepted: "已接受",
  rejected: "已拒绝",
  applied: "已应用",
  rolled_back: "已回滚",
  failed: "失败",
};

/**
 * 7d 倒计时显示(46-02 Task 3 / D-C2 视觉强化)
 *
 * 返回结构化剩余时间(天数 / 小时 / 分钟)便于渲染为彩色 Tag。
 * remainingMs <= 0 表示已过 7d 窗口,按钮将隐藏。
 */
interface CountdownInfo {
  remainingMs: number;
  remainingDays: number;
  remainingHours: number;
  remainingMinutes: number;
  isExpired: boolean;
}

function calcCountdown(target: string | null): CountdownInfo {
  if (!target) {
    return { remainingMs: 0, remainingDays: 0, remainingHours: 0, remainingMinutes: 0, isExpired: true };
  }
  const t = new Date(target).getTime();
  const now = Date.now();
  const remainingMs = t - now;
  if (remainingMs <= 0) {
    return { remainingMs: 0, remainingDays: 0, remainingHours: 0, remainingMinutes: 0, isExpired: true };
  }
  const remainingDays = Math.floor(remainingMs / (24 * 60 * 60 * 1000));
  const remainingHours = Math.floor((remainingMs % (24 * 60 * 60 * 1000)) / (60 * 60 * 1000));
  const remainingMinutes = Math.floor((remainingMs % (60 * 60 * 1000)) / (60 * 1000));
  return { remainingMs, remainingDays, remainingHours, remainingMinutes, isExpired: false };
}

export const FixSuggestionDetailDrawer = ({
  open,
  suggestionId,
  onClose,
}: FixSuggestionDetailDrawerProps) => {
  const { data, isLoading } = useQuery({
    queryKey: suggestionId
      ? queryKeys.reconciliation.fixSuggestionDetail(suggestionId)
      : ["reconciliation", "fix-suggestion-detail", "none"],
    queryFn: () => fixSuggestionApi.getById(suggestionId!),
    enabled: !!suggestionId && open,
  });

  useEffect(() => {
    if (!open) return;
  }, [open]);

  const sugg: FixSuggestionListItem | undefined = data?.suggestion;

  return (
    <Drawer
      title="修复建议详情"
      open={open}
      onClose={onClose}
      width={720}
      destroyOnClose
    >
      <Spin spinning={isLoading}>
        {sugg ? (
          <Tabs
            defaultActiveKey="summary"
            items={[
              {
                key: "summary",
                label: "冲突摘要",
                children: (
                  <Descriptions bordered column={1} size="small">
                    <Descriptions.Item label="资产编号">{sugg.assetCode}</Descriptions.Item>
                    <Descriptions.Item label="冲突类型">
                      <Tag color="orange">Type {sugg.conflictType}</Tag>
                    </Descriptions.Item>
                    <Descriptions.Item label="置信度">
                      {(sugg.confidenceScore * 100).toFixed(0)}%
                    </Descriptions.Item>
                    <Descriptions.Item label="检测时间">
                      {data?.exception?.detectedAt ?? "-"}
                    </Descriptions.Item>
                    <Descriptions.Item label="异常严重度">
                      {data?.exception?.severity ?? "-"}
                    </Descriptions.Item>
                    <Descriptions.Item label="建议原因">
                      {sugg.reason}
                    </Descriptions.Item>
                  </Descriptions>
                ),
              },
              {
                key: "detail",
                label: "修复详情",
                children: (
                  <>
                    <Descriptions bordered column={1} size="small" style={{ marginBottom: 16 }}>
                      <Descriptions.Item label="当前 ops_asset.user_id">
                        {sugg.currentUserId ?? <span style={{ color: "#999" }}>-</span>}
                      </Descriptions.Item>
                      <Descriptions.Item label="建议 user_id">{sugg.suggestedUserId}</Descriptions.Item>
                      <Descriptions.Item label="建议人 username">
                        {sugg.suggestedUsername ?? "-"}
                      </Descriptions.Item>
                      <Descriptions.Item label="应用前 user_id(pre_fix)">
                        {sugg.preFixUserId ?? <span style={{ color: "#999" }}>未应用</span>}
                      </Descriptions.Item>
                      <Descriptions.Item label="回滚窗口截止">
                        {sugg.rollbackWindowUntil ? (
                          <>
                            <span>{sugg.rollbackWindowUntil}</span>
                            {(() => {
                              const cd = calcCountdown(sugg.rollbackWindowUntil);
                              if (cd.isExpired) {
                                return (
                                  <Tag color="default" style={{ marginLeft: 8 }}>
                                    已超 7d 窗口
                                  </Tag>
                                );
                              }
                              // 颜色:剩余 < 1d 红色紧急;< 3d 橙色;>= 3d 蓝色
                              const color = cd.remainingDays < 1 ? "red" : cd.remainingDays < 3 ? "orange" : "blue";
                              return (
                                <Tag color={color} style={{ marginLeft: 8 }}>
                                  剩余 {cd.remainingDays}d {cd.remainingHours}h {cd.remainingMinutes}m 可回滚
                                </Tag>
                              );
                            })()}
                          </>
                        ) : (
                          "-"
                        )}
                      </Descriptions.Item>
                    </Descriptions>

                    <Timeline
                      items={[
                        {
                          color: "gray",
                          children: (
                            <>
                              <div>创建建议</div>
                              <div style={{ fontSize: 12, color: "#999" }}>{sugg.createdAt}</div>
                            </>
                          ),
                        },
                        sugg.acceptedAt
                          ? {
                              color: "blue",
                              children: (
                                <>
                                  <div>接受</div>
                                  <div style={{ fontSize: 12, color: "#999" }}>{sugg.acceptedAt}</div>
                                </>
                              ),
                            }
                          : null,
                        sugg.appliedAt
                          ? {
                              color: "green",
                              children: (
                                <>
                                  <div>应用</div>
                                  <div style={{ fontSize: 12, color: "#999" }}>{sugg.appliedAt}</div>
                                </>
                              ),
                            }
                          : null,
                        sugg.rolledBackAt
                          ? {
                              color: "magenta",
                              children: (
                                <>
                                  <div>回滚</div>
                                  <div style={{ fontSize: 12, color: "#999" }}>{sugg.rolledBackAt}</div>
                                </>
                              ),
                            }
                          : null,
                      ].filter(Boolean) as Array<{ color: string; children: React.ReactNode }>}
                    />
                  </>
                ),
              },
              {
                key: "history",
                label: "历史变更",
                children:
                  data?.history && data.history.length > 1 ? (
                    <Timeline
                      items={data.history.map((h) => ({
                        color: fixStatusColor[h.fixStatus],
                        children: (
                          <>
                            <div>
                              <Tag color={fixStatusColor[h.fixStatus]}>{fixStatusLabel[h.fixStatus]}</Tag>
                              <span style={{ color: "#999", fontSize: 12 }}>{h.id}</span>
                            </div>
                            <div style={{ fontSize: 12, color: "#999" }}>{h.createdAt}</div>
                            {h.rejectionReason && (
                              <div style={{ fontSize: 12, color: "red" }}>拒绝原因:{h.rejectionReason}</div>
                            )}
                            {h.rollbackReason && (
                              <div style={{ fontSize: 12, color: "red" }}>回滚原因:{h.rollbackReason}</div>
                            )}
                          </>
                        ),
                      }))}
                    />
                  ) : (
                    <Empty description="无历史记录" />
                  ),
              },
            ]}
          />
        ) : (
          !isLoading && <Empty description="无数据" />
        )}
      </Spin>
    </Drawer>
  );
};
