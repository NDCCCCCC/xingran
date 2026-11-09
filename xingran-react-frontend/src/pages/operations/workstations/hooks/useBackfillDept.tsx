/**
 * useBackfillDept — 工位管理页「按人员反推部门」按钮的交互逻辑（quick-261106-wsb）。
 *
 * 流程（用户锁定：预览 + 确认两步）：
 *   1. 点击 → dryRun=true 预览（不写库）
 *   2. Modal.confirm 展示候选/待填补/待纠正/跳过分布 + 前若干条样例，
 *      并带「纠正不一致」Checkbox（默认关闭）
 *   3. 勾选 Checkbox → 重新 dry-run 刷新预览（保证预览与执行口径一致）
 *   4. 确认 → dryRun=false 执行 → 成功提示 → onSuccess 刷新列表/统计
 *
 * 「纠正不一致」会覆盖已填写的部门，而「工位部门 ≠ 人员部门」常是有意的
 * 跨机构配置，故默认关闭、需显式勾选、确认文案明确警示覆盖条数。
 *
 * 用 antd 命令式 Modal.confirm + 模块级 message 桥接，避免给页面引入
 * App.useApp() context 依赖（现有 render 测试的 wrapper 已支持，但零侵入更稳）。
 */

import { useCallback, useState } from "react";
import { Modal, Checkbox, Table, Typography, Space, Alert } from "antd";
import type { ColumnsType } from "antd/es/table";
import { workstationApi } from "@/lib/opsApi";
import type { BackfillDeptResult, BackfillDeptSample } from "@/types";
import { getAppMessage } from "@/utils/antdMessage";
import { handleApiError } from "@/utils/errorHandler";

const { Text } = Typography;

const actionLabel: Record<BackfillDeptSample["action"], string> = {
  fill: "填补空缺",
  correct: "纠正不一致",
  skip: "跳过",
};

const sampleColumns: ColumnsType<BackfillDeptSample> = [
  {
    title: "工位",
    dataIndex: "workstationName",
    key: "workstationName",
    width: 120,
    ellipsis: true,
  },
  {
    title: "现部门",
    dataIndex: "currentDeptName",
    key: "currentDeptName",
    width: 120,
    ellipsis: true,
    render: (v?: string) => v || "-",
  },
  {
    title: "目标部门",
    dataIndex: "targetDeptName",
    key: "targetDeptName",
    width: 120,
    ellipsis: true,
    render: (v?: string) => v || "-",
  },
  {
    title: "人员",
    dataIndex: "userName",
    key: "userName",
    width: 90,
    ellipsis: true,
    render: (v?: string) => v || "-",
  },
  {
    title: "动作",
    dataIndex: "action",
    key: "action",
    width: 100,
    render: (v: BackfillDeptSample["action"]) => actionLabel[v] ?? v,
  },
];

export interface UseBackfillDeptReturn {
  loading: boolean;
  open: () => Promise<void>;
}

export function useBackfillDept(onSuccess: () => void): UseBackfillDeptReturn {
  const [loading, setLoading] = useState(false);

  const open = useCallback(async () => {
    if (loading) return;
    setLoading(true);

    // 预览状态放在闭包里，Modal.confirm 的 content 通过 rerender 刷新。
    let fixMismatch = false;
    let preview: BackfillDeptResult | null = null;
    let refreshing = false;
    const modalRef: { current?: ReturnType<typeof Modal.confirm> } = {};

    const runPreview = async (): Promise<boolean> => {
      try {
        preview = await workstationApi.backfillDeptFromUser({ dryRun: true, fixMismatch });
        return true;
      } catch (error) {
        handleApiError(error, "预览反推结果");
        return false;
      }
    };

    if (!(await runPreview())) {
      setLoading(false);
      return;
    }

    // 用 modalRef.current.update() 重渲染内容（Checkbox 勾选后刷新预览）。

    // Phase 150：不再消费范围快照规模，确认框文案不含全库规模（DECISION-①）；
    // scopeTier / scopeDeptNames / scopeFailClosed 仅作可见集内提示与 fail-closed
    // 门控。132 的四类「本次范围：…（共 N 条）」文案整体作废，改为不含任何数值
    // 的可见集内表述；文案仍挂进 modalRef.current.update() 刷新链路。
    const rangeText = (): string => {
      const p = preview;
      const tier = p?.scopeTier ?? "All";
      if (p?.scopeFailClosed || tier === "FailClosed") {
        return "本次范围：无（数据权限异常，请联系管理员）";
      }
      if (tier === "All") {
        return "本次范围：全部工位";
      }
      if (tier === "Self") {
        return "本次范围：仅本人关联工位";
      }
      // Custom/Dept/DeptChild：不含「共 N 条」/「等 M 个部门」等任何全库规模数值。
      return "本次范围：当前数据权限可见的部门及子部门";
    };

    const okButtonProps = () => ({
      disabled: (preview?.scopeFailClosed ?? false) || preview?.scopeTier === "FailClosed",
    });

    const renderContent = () => (
      <Space orientation="vertical" size="medium" style={{ width: "100%" }}>
        <div>
          <Text strong>{rangeText()}</Text>
        </div>
        <div>
          <Text>共匹配到 </Text>
          <Text strong>{preview?.candidates ?? 0}</Text>
          <Text> 个绑定了有效用户的工位：</Text>
        </div>
        <div>
          <Text>将填补空缺部门 </Text>
          <Text strong style={{ color: "var(--theme-success)" }}>
            {preview?.toFill ?? 0}
          </Text>
          <Text> 条；跳过（已一致） </Text>
          <Text strong>{preview?.skippedAlreadyValid ?? 0}</Text>
          <Text> 条。</Text>
        </div>
        {fixMismatch && (
          <Alert
            type="warning"
            showIcon
            title={`将覆盖 ${preview?.toCorrect ?? 0} 条已填写的部门`}
            description="「工位部门 ≠ 人员部门」可能是有意的跨机构配置，请仅在确认这些部门确实填错时勾选。"
          />
        )}
        <Checkbox
          checked={fixMismatch}
          disabled={refreshing}
          onChange={async (e) => {
            fixMismatch = e.target.checked;
            refreshing = true;
            modalRef.current?.update({ content: renderContent(), okButtonProps: okButtonProps() });
            const ok = await runPreview();
            refreshing = false;
            if (!ok) {
              fixMismatch = !fixMismatch;
            }
            modalRef.current?.update({
              content: renderContent(),
              okText: okText(),
              okButtonProps: okButtonProps(),
            });
          }}
        >
          <Text type="danger">
            ⚠ 同时纠正「已有部门但与人员部门不一致」的工位（会覆盖已填部门）
          </Text>
        </Checkbox>
        {(preview?.samples?.length ?? 0) > 0 && (
          <Table<BackfillDeptSample>
            columns={sampleColumns}
            dataSource={preview?.samples ?? []}
            rowKey="workstationId"
            size="small"
            pagination={false}
            scroll={{ y: 220 }}
          />
        )}
      </Space>
    );

    const okText = () =>
      fixMismatch ? `确认执行（含纠正 ${preview?.toCorrect ?? 0} 条）` : "确认执行";

    // 非删除类确认（批量回填预览 + 危险开关二次确认），ConfirmDelete 语义不匹配；
    // 按 eslint 契约显式豁免并注明原因（同 FormModal.tsx 的既有豁免口径）。
    // eslint-disable-next-line no-restricted-syntax
    modalRef.current = Modal.confirm({
      title: "按人员反推部门 · 预览",
      width: 720,
      content: renderContent(),
      okText: okText(),
      okButtonProps: okButtonProps(),
      cancelText: "取消",
      mask: { closable: false },
      onOk: async () => {
        try {
          const result = await workstationApi.backfillDeptFromUser({ dryRun: false, fixMismatch });
          getAppMessage().success(
            `反推完成：填补 ${result.toFill} 条${result.toCorrect > 0 ? `、纠正 ${result.toCorrect} 条` : ""}`
          );
          onSuccess();
        } catch (error) {
          handleApiError(error, "按人员反推部门");
          // 不 rethrow：让 Modal 正常关闭（afterClose 复位 loading），
          // 避免 onOk 抛错后弹窗滞留导致按钮永久 loading（审计 P2-9）。
        }
      },
      onCancel: () => {
        setLoading(false);
      },
      afterClose: () => {
        setLoading(false);
      },
    });
  }, [loading, onSuccess]);

  return { loading, open };
}
