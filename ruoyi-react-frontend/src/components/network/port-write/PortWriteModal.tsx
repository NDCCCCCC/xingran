/**
 * PortWriteModal — Phase 53 W4 单端口写操作统一 Modal (D-01)
 *
 * 一个 Modal 覆盖 5 个 action (shutdown / undo_shutdown / description / dot1x_enable / dot1x_disable),
 * 通过 action prop 切换 Modal 标题 (ACTION_TITLE[action]) / 是否显示"新描述"字段 (D-03 description 特例) /
 * wrapper 调用。reason 字段为预置 Select + 自定义 TextArea 双层结构 (D-02),__custom__ sentinel 切换。
 *
 * 设计约束 (53-02 PLAN):
 * - D-01: 不用 Modal.confirm (reason/description 输入需 Form, confirm 不够); 不建 5 个独立 Modal
 * - D-03: description action 时"新描述"必填 + reason 可空; 其他 4 action 时 reason 必填
 * - D-10: 成功后 Toast 含"查看审计日志"链接 navigate('/monitor/logs?module=端口管理')
 * - LANDMINE #5: wrapper reject 由 post() 拦截器统一弹 Toast, 本组件不再传 errorMessage / 不再 message.error
 * - T-53-06: 全部用 antd 组件渲染, 禁止 dangerouslySetInnerHTML (React 默认转义文本节点)
 *
 * 来源: 53-02-PLAN.md Task 1, 53-PATTERNS.md line 122-198 骨架代码
 */

import { useEffect, useState } from "react";
import { Modal, Form, Select, Input, App } from "antd";
import type { MessageInstance } from "antd/es/message/interface";
import { useNavigate } from "react-router-dom";
import type { DevicePortStatus, PortWriteAction } from "@/types/network";
import {
  writeShutdown,
  writeUndoShutdown,
  writeDescription,
  writeDot1xEnable,
  writeDot1xDisable,
} from "@/lib/api/networkApi";
import {
  PRESET_REASONS,
  ACTION_TITLE,
  REASON_MIN,
  REASON_MAX,
  DESCRIPTION_MAX,
  REASON_CUSTOM_SENTINEL,
  composeReason,
  validateReasonOptional,
  validateReasonRequired,
} from "./constants";
/**
 * D-10 审计跳转 URL — route path 是 /monitor/logs (非 /monitor/operlog),
 * module= 后为中文模块名 (URL-encoded)。BulkWriteDrawer 复用同一路径。
 */
export const AUDIT_LOG_PATH = "/monitor/logs?module=" + encodeURIComponent("端口管理");

export interface PortWriteModalProps {
  open: boolean;
  action: PortWriteAction;
  portRecord: DevicePortStatus | null;
  onClose: () => void;
  onSuccess: () => void;
}

/**
 * D-10 审计跳转 Toast helper (单端口 + 批量共用)
 *
 * antd message.success 不支持链接, 用 message.open({ content: <ReactNode> }) + 链接 onClick navigate。
 * 端口管理是 Phase 52 handler ModulePortWrite 常量 (port_write_handler.go:25), 与 sys_oper_log.title 列匹配。
 *
 * ★ 2026-07-08 Bug #1 修复:
 * 原实现用 react-router <Link to=...> + onClick navigate。antd message 用 React Portal 把
 * content 渲染到 document.body（不在 React Router 树内），<Link> 内部 useContext(RouterContext)
 * 拿到 null → throw 'Cannot destructure basename of null useContext' → React 整树 unmount
 * → 整个端口管理页空白（地址栏 URL 正确但页面 unmounted）。
 *
 * 改用原生 <a href> + onClick 拦截:
 *   - <a> 不依赖任何 React Context，portal 渲染安全
 *   - e.preventDefault() 阻止浏览器硬跳转
 *   - message.destroy() 路由切换前关 toast，避免 portal 在另一个路由下残留 DOM
 *   - navigate() 走 SPA history.pushState，页面不刷新
 */
export function showAuditLinkToast(message: MessageInstance, navigate: (path: string) => void): void {
  const handleClick = (e: React.MouseEvent<HTMLAnchorElement>): void => {
    e.preventDefault();
    message.destroy();
    navigate(AUDIT_LOG_PATH);
  };
  message.open({
    type: "success",
    duration: 5,
    content: (
      <span>
        操作成功，
        <a href={AUDIT_LOG_PATH} onClick={handleClick}>
          查看审计日志
        </a>
      </span>
    ),
  });
}

/**
 * D-01 单端口写操作统一 Modal
 *
 * 端口列表页"操作"列的 5 个 menu item 都打开同一个 Modal, 通过 action prop 切换。
 * action 变化时 form.resetFields 防上次 reason/description 残留 (useEffect deps 全稳定 ref)。
 */
export function PortWriteModal({
  open,
  action,
  portRecord,
  onClose,
  onSuccess,
}: PortWriteModalProps) {
  const [form] = Form.useForm();
  const { message } = App.useApp();
  const navigate = useNavigate();
  // IN-03: submitting 防写操作重复提交 (wrapper await 期间用户可重复点"确认执行")
  const [submitting, setSubmitting] = useState(false);

  // action 变化时 reset 表单 (防上次 reason/description 残留)
  // deps 全稳定: open/action 原始值, form 来自 useForm 稳定 ref — CLAUDE.md useEffect 纪律满足
  useEffect(() => {
    if (open) form.resetFields();
  }, [open, action, form]);

  const handleOk = async (): Promise<void> => {
    setSubmitting(true);
    try {
      let values: Record<string, unknown>;
      try {
        values = await form.validateFields();
      } catch (err) {
        // validateFields 失败抛 errorFields, antd 自动标红字段, 不需 Toast
        if (err && typeof err === "object" && "errorFields" in err) return;
        // WR-01: 非预期校验异常不再 throw 冒泡到 antd onOk 形成未处理 Promise rejection
        console.error("[PortWriteModal] validateFields unexpected error:", err);
        return;
      }

      // 组装最终 reason (预设项直接用 value; custom 项用 reasonText)
      const reason = composeReason(values.reasonSelect, values.reasonText);

      if (!portRecord) {
        // 防御性: portRecord 应永远非空 (ActionButtons onClick 一定带 record)
        return;
      }

      // 按 action 分支调对应 wrapper (LANDMINE #5: 不传 errorMessage / 不 message.error, post() 拦截器已弹)
      if (action === "shutdown") {
        await writeShutdown(portRecord.id, reason ?? "");
      } else if (action === "undo_shutdown") {
        await writeUndoShutdown(portRecord.id, reason ?? "");
      } else if (action === "description") {
        const description = typeof values.description === "string" ? values.description : "";
        await writeDescription(portRecord.id, description, reason ?? undefined);
      } else if (action === "dot1x_enable") {
        await writeDot1xEnable(portRecord.id, reason ?? "");
      } else if (action === "dot1x_disable") {
        await writeDot1xDisable(portRecord.id, reason ?? "");
      }

      // D-10 成功 Toast 含"查看审计日志"链接
      showAuditLinkToast(message, navigate);

      form.resetFields();
      onSuccess();
      onClose();
    } finally {
      setSubmitting(false);
    }
  };

  const reasonRules =
    action === "description"
      ? [
          {
            validator: (rule: unknown, value: unknown) =>
              validateReasonOptional(rule, value, form),
          },
        ]
      : [
          {
            required: true,
            validator: (rule: unknown, value: unknown) =>
              validateReasonRequired(rule, value, form),
          },
        ];

  return (
    <Modal
      title={`${ACTION_TITLE[action]} - ${portRecord?.interfaceName ?? ""}`}
      open={open}
      onOk={handleOk}
      onCancel={onClose}
      destroyOnHidden
      width={520}
      okText="确认执行"
      cancelText="取消"
      okButtonProps={{ loading: submitting }}
    >
      <Form form={form} layout="vertical">
        {/* D-03 description action 特例 — 显示"新描述"必填输入框 */}
        {action === "description" && (
          <Form.Item
            name="description"
            label="新描述"
            rules={[
              { required: true, message: "请输入新端口描述" },
              { max: DESCRIPTION_MAX, message: `描述不超过 ${DESCRIPTION_MAX} 字符` },
            ]}
          >
            <Input maxLength={DESCRIPTION_MAX} showCount placeholder="请输入新端口描述" />
          </Form.Item>
        )}

        {/* D-02 reason 字段 — 外层 reasonSelect Select + 内层 reasonText TextArea (仅 __custom__ 时展开) */}
        <Form.Item name="reasonSelect" label="操作原因" rules={reasonRules}>
          <Select
            placeholder="请选择操作原因"
            options={PRESET_REASONS.map((opt) => ({ label: opt.label, value: opt.value }))}
          />
        </Form.Item>

        {/* shouldUpdate 监听 reasonSelect 值, 仅选 __custom__ 时展开 TextArea */}
        <Form.Item shouldUpdate noStyle>
          {({ getFieldValue }) =>
            getFieldValue("reasonSelect") === REASON_CUSTOM_SENTINEL ? (
              <Form.Item
                name="reasonText"
                label="自定义原因"
                rules={[{ max: REASON_MAX, message: `操作原因不超过 ${REASON_MAX} 个字符` }]}
              >
                <Input.TextArea
                  rows={2}
                  maxLength={REASON_MAX}
                  showCount
                  placeholder={`请输入操作原因（${REASON_MIN}-${REASON_MAX} 字符）`}
                />
              </Form.Item>
            ) : null
          }
        </Form.Item>
      </Form>
    </Modal>
  );
}

export default PortWriteModal;
