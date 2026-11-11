/**
 * 系统设置「备份与恢复」分类区块（单页平铺版）。
 *
 * 来源：原独立菜单页 src/pages/system/backup/index.tsx（quick 261110-tbq，CONTRACT.md 第 1/5 节）。
 * 本次把整页逻辑原样迁入设置页区块，子组件（列表 / 计划 / 恢复弹窗）继续复用 backup/components，
 * 旧路由文件 backup/index.tsx 退化为薄壳 re-export，保证旧链接与既有测试行为等价。
 *
 * 区块内容：Tabs「备份列表 / 备份计划」+ 全页异步任务进度条。
 * - 列表 / 计划：由 BackupListTab、BackupPlanTab 负责（含各自统计卡与表单）。
 * - 触发导出/恢复后拿到 taskId，本组件统一每 2s 轮询 task(id)，终态停止并刷新列表。
 * - 按钮权限门控（system:backup:export / system:backup:restore）在 BackupListTab 内，
 *   本区块不重复门控，也不改变 API 契约。
 *
 * 平铺布局说明：区块标题与描述由 SettingsShell 的 <section> 提供，本组件只返回内容，
 * 不再套 Card 标题；按「单页平铺」约定，区块内不新增 StatCards 统计卡。
 */
import { useCallback, useEffect, useMemo, useState, type FC } from "react";
import { useLocation } from "react-router-dom";
import { App, Alert, Card, Progress, Space, Tabs, Tag } from "antd";
import { DatabaseOutlined, ScheduleOutlined } from "@ant-design/icons";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import { dbBackupApi } from "@/lib/dbBackupApi";
import { handleApiError } from "@/utils/errorHandler";
import type { TaskView } from "@/types/backup";
import type { AsyncTask, BackupTabKey } from "../backup/types";
import { TASK_STATUS_COLOR, TASK_STATUS_TEXT } from "../backup/types";
import BackupListTab from "../backup/components/BackupListTab";
import BackupPlanTab from "../backup/components/BackupPlanTab";

/** 轮询间隔（毫秒） */
const POLL_INTERVAL_MS = 2000;

const BackupRestoreSettings: FC = () => {
  const { message } = App.useApp();
  const location = useLocation();

  // 记忆当前 Tab（切走/刷新后仍停留在上次面板）。
  // keyPrefix 仍取 location.pathname（迁入设置页后即设置页 pathname），行为与原页等价。
  const [activeTab, setActiveTab] = usePersistedStateController<BackupTabKey>({
    keyPrefix: location.pathname,
    keySuffix: "activeTab",
    defaultValue: "list",
  });

  // 由子组件触发的异步任务（导出 / 恢复）。taskId 稳定后才会重建定时器。
  const [asyncTask, setAsyncTask] = useState<AsyncTask | null>(null);
  const [taskView, setTaskView] = useState<TaskView | null>(null);
  // 终态刷新用递增 token（作为列表刷新的稳定信号，避免把回调塞进轮询依赖）
  const [listRefreshToken, setListRefreshToken] = useState(0);

  /** 触发一个异步任务：立即进入轮询（终态由轮询 effect 处理） */
  const startTask = useCallback((task: AsyncTask) => {
    setTaskView(null);
    setAsyncTask(task);
  }, []);

  /** 任务结束（成功/失败）后的统一收口：停轮询 + 提示 + 刷新列表 */
  const handleTaskSettled = useCallback(
    (view: TaskView) => {
      setAsyncTask(null);
      if (view.status === "success") {
        message.success(
          view.taskType === "restore" ? "恢复完成，列表已刷新" : "备份完成，列表已刷新"
        );
      } else {
        message.error(view.errorMessage || "任务执行失败，请查看操作日志");
      }
      setListRefreshToken((token) => token + 1);
    },
    [message]
  );

  // 兼容历史残留：本 pathname 曾属于旧「系统设置」Tabs 页（settings-page），
  // sessionStorage 里可能还存着 email/api/captcha 等旧 Tab key。持久化值不属于
  // 本区块的 Tab 时回退到 list，避免 Tabs 无匹配项而显示空白面板。
  const safeActiveTab: BackupTabKey = activeTab === "plan" ? "plan" : "list";

  const taskId = asyncTask?.taskId;
  const taskKind = asyncTask?.kind;

  // 轮询：依赖仅含稳定值（taskId / handleTaskSettled），不会因渲染反复重建定时器。
  // taskKind 只用于进度条文案，不进依赖，避免无谓重建。
  useEffect(() => {
    if (!taskId) return;

    let cancelled = false;

    const poll = async () => {
      try {
        const view = await dbBackupApi.task(taskId);
        if (cancelled) return;
        setTaskView(view);
        if (view.status === "success" || view.status === "failed") {
          handleTaskSettled(view);
        }
      } catch (error) {
        if (cancelled) return;
        // 查询失败也停止轮询，避免无限报错循环
        handleApiError(error, "查询备份任务进度");
        setAsyncTask(null);
      }
    };

    // 立即查一次，之后每 2s 轮询
    void poll();
    const timer = window.setInterval(() => void poll(), POLL_INTERVAL_MS);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [taskId, handleTaskSettled]);

  const progressCard = useMemo(() => {
    if (!asyncTask || !taskView) return null;
    const status = taskView.status;
    const isTerminal = status === "success" || status === "failed";
    const percent = Math.max(0, Math.min(100, taskView.progress ?? 0));
    return (
      <Card size="small" style={{ marginBottom: 16 }}>
        <Space orientation="vertical" style={{ width: "100%" }} size="small">
          <Space size="medium" wrap>
            <span>异步任务：{taskKind === "restore" ? "导入恢复" : "立即备份"}</span>
            <Tag color={TASK_STATUS_COLOR[status] ?? "default"}>
              {TASK_STATUS_TEXT[status] ?? status}
            </Tag>
            <span>阶段：{taskView.stage || "-"}</span>
            <span>任务 ID：{taskView.id}</span>
          </Space>
          <Progress
            percent={percent}
            status={status === "failed" ? "exception" : status === "success" ? "success" : "active"}
          />
          {status === "failed" && taskView.errorMessage && (
            <Alert type="error" showIcon title="任务失败" description={taskView.errorMessage} />
          )}
          {!isTerminal && (
            <Alert
              type="info"
              showIcon
              title="任务执行中"
              description="请勿关闭页面，任务完成后会自动刷新备份列表。"
            />
          )}
        </Space>
      </Card>
    );
  }, [asyncTask, taskView, taskKind]);

  const tabItems = useMemo(
    () => [
      {
        key: "list" as const,
        label: (
          <span>
            <DatabaseOutlined />
            备份列表
          </span>
        ),
        children: (
          <BackupListTab
            onStartTask={startTask}
            refreshToken={listRefreshToken}
            taskRunning={asyncTask !== null}
          />
        ),
      },
      {
        key: "plan" as const,
        label: (
          <span>
            <ScheduleOutlined />
            备份计划
          </span>
        ),
        children: <BackupPlanTab />,
      },
    ],
    [startTask, listRefreshToken, asyncTask]
  );

  return (
    <div>
      {progressCard}
      <Tabs
        activeKey={safeActiveTab}
        onChange={(key) => setActiveTab(key as BackupTabKey)}
        items={tabItems}
      />
    </div>
  );
};

export default BackupRestoreSettings;
