/**
 * 导入恢复弹窗：选择备份 zip → 二次确认 → 上传并触发恢复 → 交给父页面轮询。
 *
 * 上传范式对齐 duty/holidays/index.tsx 的 <Upload customRequest>；
 * 二次确认走 confirmDelete（全仓唯一命令式危险操作入口，lint 禁止裸 Modal.confirm），
 * 文案明确「覆盖现有全部数据 + 系统自动生成恢复前备份」。
 *
 * 前置提示（不依赖失败）：父页面把 capabilities() 结果传入，当 dbType === "sqlite"
 * 时常驻显示「SQLite 整库恢复需停服离线执行」说明与离线命令入口，避免用户直到
 * 失败才在 errorMessage 里看到命令（后端 restorer.go 的降级路径）。
 */
import { App, Alert, Modal, Typography, Upload } from "antd";
import { InboxOutlined } from "@ant-design/icons";
import type { FC } from "react";
import { confirmDelete } from "@/components/shared";
import { dbBackupApi } from "@/lib/dbBackupApi";
import type { Capabilities } from "@/types/backup";
import type { AsyncTask } from "../types";
import { RESTORE_CONFIRM_TEXT, SQLITE_OFFLINE_COMMAND, SQLITE_OFFLINE_NOTICE } from "../types";

interface RestoreModalProps {
  open: boolean;
  taskRunning: boolean;
  onClose: () => void;
  onStartTask: (task: AsyncTask) => void;
  /** 能力预检结果：dbType === "sqlite" 时常驻展示离线恢复提示 */
  capabilities: Capabilities | null;
}

const RestoreModal: FC<RestoreModalProps> = ({
  open,
  taskRunning,
  onClose,
  onStartTask,
  capabilities,
}) => {
  const { message } = App.useApp();
  const isSQLite = capabilities?.dbType === "sqlite";

  /** 选择文件后：先二次确认，确认通过才真正上传（customRequest 不立即发请求） */
  const handleCustomRequest = (options: {
    file: File | unknown;
    onSuccess?: (data: unknown) => void;
    onError?: (error: Error) => void;
  }) => {
    const file = options.file as File;
    if (!file || !(file instanceof File)) {
      options.onError?.(new Error("未选择有效文件"));
      return;
    }
    if (!file.name.toLowerCase().endsWith(".zip")) {
      message.warning("请选择 .zip 格式的备份文件");
      options.onError?.(new Error("文件格式不支持"));
      return;
    }

    confirmDelete({
      title: "确认导入恢复",
      target: `备份文件（${file.name}）`,
      actionLabel: "开始恢复",
      actionContext: "导入恢复",
      contentText: RESTORE_CONFIRM_TEXT,
      // 恢复是异步任务，此处只表示「任务已提交」，真实结果由父页面轮询反馈
      successText: "恢复任务已提交，请在进度条查看执行结果",
      onDelete: async () => {
        const { taskId } = await dbBackupApi.restore(file);
        if (!taskId) {
          throw new Error("未获取到任务 ID，无法跟踪恢复进度");
        }
        onStartTask({ taskId, kind: "restore" });
        options.onSuccess?.({ taskId });
        onClose();
      },
    });
    // 注意：confirmDelete 的 onOk 失败时保持确认框打开并 handleApiError，
    // 这里不主动调 onError（避免 Upload 过早把本次选择判成失败）。
  };

  return (
    <Modal
      title="导入恢复"
      open={open}
      onCancel={onClose}
      footer={null}
      destroyOnHidden
      width={560}
    >
      {isSQLite && (
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 16 }}
          title="SQLite 整库恢复需停服离线执行"
          description={
            <div>
              <div>{SQLITE_OFFLINE_NOTICE}</div>
              <div style={{ marginTop: 8 }}>离线命令入口：</div>
              <Typography.Text code copyable={{ text: SQLITE_OFFLINE_COMMAND }}>
                {SQLITE_OFFLINE_COMMAND}
              </Typography.Text>
            </div>
          }
        />
      )}
      <Alert
        type="warning"
        showIcon
        style={{ marginBottom: 16 }}
        title="高风险操作"
        description="恢复将覆盖现有全部数据、期间不可写入、恢复后需重新登录，操作前系统会自动生成一份恢复前备份。请确认已选择正确的备份文件。"
      />
      <Upload.Dragger
        accept=".zip"
        showUploadList={false}
        disabled={taskRunning}
        customRequest={(options) =>
          handleCustomRequest({
            file: options.file,
            onSuccess: options.onSuccess as (data: unknown) => void,
            onError: options.onError as (error: Error) => void,
          })
        }
      >
        <p className="ant-upload-drag-icon">
          <InboxOutlined />
        </p>
        <p className="ant-upload-text">点击或拖拽备份 .zip 文件到此处</p>
        <p className="ant-upload-hint">仅支持系统导出的 .zip 备份包，选择后需二次确认</p>
      </Upload.Dragger>
      {taskRunning && (
        <Alert
          type="info"
          showIcon
          style={{ marginTop: 16 }}
          title="已有任务执行中"
          description="请等待当前备份/恢复任务结束后再发起新的恢复。"
        />
      )}
    </Modal>
  );
};

export default RestoreModal;
