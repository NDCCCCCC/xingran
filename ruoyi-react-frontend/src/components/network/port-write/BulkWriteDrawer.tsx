/**
 * BulkWriteDrawer — Phase 53 W4 批量端口操作 Drawer (D-04..D-07)
 *
 * select → executing → result 状态机:
 * - select: 只读汇总 selectedPorts + action Select + reason Select/TextArea + 跨设备预校验 (Alert)
 * - executing: indeterminate Spin, 不伪造 X/Y (D-05/ROADMAP #2 纠正)
 * - result: 三 Statistic 卡片 (成功/跳过/失败) + 失败明细 Table + 跳过折叠 + 重试按钮 (D-06)
 *
 * 设计约束 (53-02 PLAN):
 * - D-04: Drawer 不重选, 直接读父组件 selectedRowKeys 过滤出的 selectedPorts
 * - D-05: indeterminate Spin, 不用 Progress 伪造 X/Y (后端 batch 同步阻塞, 无实时进度)
 * - D-06: 重试只取 batchResult.failed.map(p => p.portId), 不重试 skipped
 * - D-07: onExecutingChange 上抛父组件, 父组件禁用刷新+采集按钮 (LANDMINE #4 同类竞态)
 * - D-10: 全部成功 (failed.length===0) 时弹含审计链接的 Toast, 链接目标 '/monitor/logs?module=端口管理' (复用 PortWriteModal 的 showAuditLinkToast helper)
 * - LANDMINE #3: HTTP 200 + status:failed 是正常 resolve, batchResult.failed/skipped/succeeded 从 body 读, 不靠 .catch
 * - T-53-06: 全部 antd 组件渲染, 禁止 dangerouslySetInnerHTML
 * - T-53-07: wrapper 只发白名单字段 (deviceId/action/portIds/description), 不 spread 整个 port record
 *
 * 来源: 53-02-PLAN.md Task 2, 53-PATTERNS.md line 202-289 骨架代码
 */

import { useEffect, useMemo, useState } from "react";
import {
  Drawer,
  Form,
  Button,
  Space,
  Alert,
  Row,
  Col,
  Card,
  Statistic,
  Spin,
  Table,
  Tag,
  Collapse,
  Typography,
  App,
} from "antd";
// Phase 71 (W4)：裸 Input/Select/Input.TextArea → 契约组件。
// ⚠️ 本文件承载 v1.19 批量写 drawer 冻结交互（提交中状态机 / batchInProgress），
// 本波仅换 UI 组件名，不改任何状态/条件渲染/disabled 逻辑。
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import { useNavigate } from "react-router-dom";
import type {
  DevicePortStatus,
  PortResult,
  PortWriteAction,
  BatchWriteRequest,
  BatchResult,
  PortWritePreviewResult,
  PortWritePreviewPort,
} from "@/types/network";
import { batchWritePorts, previewBatchWritePorts } from "@/lib/api/networkApi";
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
import { showAuditLinkToast } from "./PortWriteModal";
import { handleApiError } from "@/utils/errorHandler";

// v1.51 Phase 121 PORTW-02: 两段式写确认 — select → preview(dry-run) → executing → result
type DrawerPhase = "select" | "preview" | "executing" | "result";

// 避免 Object.keys() as 断言在新增 ACTION_TITLE key 时静默失同步)
const ACTION_OPTIONS: { label: string; value: PortWriteAction }[] = [
  { label: ACTION_TITLE.shutdown, value: "shutdown" },
  { label: ACTION_TITLE.undo_shutdown, value: "undo_shutdown" },
  { label: ACTION_TITLE.description, value: "description" },
  { label: ACTION_TITLE.dot1x_enable, value: "dot1x_enable" },
  { label: ACTION_TITLE.dot1x_disable, value: "dot1x_disable" },
];

export interface BulkWriteDrawerProps {
  open: boolean;
  /** D-04: 由父组件读 selectedRowKeys 过滤 portStatus 得到, Drawer 不重选 */
  selectedPorts: DevicePortStatus[];
  onClose: () => void;
  onSuccess: () => void;
  /** D-07: 上抛 batchInProgress 给父组件, 父组件禁用刷新+采集按钮 (LANDMINE #4) */
  onExecutingChange?: (inProgress: boolean) => void;
}

export function BulkWriteDrawer({
  open,
  selectedPorts,
  onClose,
  onSuccess,
  onExecutingChange,
}: BulkWriteDrawerProps) {
  const [form] = Form.useForm();
  const { message } = App.useApp();
  const navigate = useNavigate();
  const [phase, setPhase] = useState<DrawerPhase>("select");
  const [batchResult, setBatchResult] = useState<BatchResult | null>(null);
  /** PORTW-02: dry-run 预览结果（preview 阶段渲染, 确认后进入真实写） */
  const [previewResult, setPreviewResult] = useState<PortWritePreviewResult | null>(null);
  /** Phase 123 错误语义: 409 端口占用 / 503 预检失败 → 常驻 Alert 分流（Toast 之外补充上下文） */
  const [writeError, setWriteError] = useState<{ code?: number; message: string } | null>(null);
  /** 保留上次提交的 action/description/deviceId/interfaceMap, 用于失败重试沿用 (D-06)
   * CR-01/CR-02: deviceId 必须在首次提交时快照, retry 时用缓存值 —— 不能从 selectedPorts 重读,
   *   否则 onSuccess 触发父级 loadPortStatus 后 selectedPorts 引用漂移, deviceId 错位会让
   *   后端 batch_orchestrator fallback 用错误 deviceId 走 SSH (跨设备误操作风险)。
   * WR-03: interfaceMap 同理快照, result 视图失败明细表的接口名不随父级刷新失真。 */
  const [lastAction, setLastAction] = useState<PortWriteAction>("shutdown");
  const [lastDescription, setLastDescription] = useState<string | undefined>(undefined);
  const [lastDeviceId, setLastDeviceId] = useState<string>("");
  const [lastInterfaceMap, setLastInterfaceMap] = useState<Map<string, string>>(new Map());

  // open 变 false 时 reset phase + batchResult, 防下次打开残留
  // CLAUDE.md useEffect 纪律: deps 全稳定 (open/phase/onExecutingChange 原始值或稳定 setter)
  useEffect(() => {
    if (!open) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setPhase("select");

      setBatchResult(null);
      setPreviewResult(null);
      setWriteError(null);
      form.resetFields();
    }
  }, [open, form]);

  // D-07: phase 变化时 onExecutingChange 上抛父组件
  useEffect(() => {
    onExecutingChange?.(phase === "executing");
  }, [phase, onExecutingChange]);

  // 跨设备预校验 (T-53-07 前端预校验, 后端 ErrMixedDevices 是真相源)
  const uniqueDeviceIds = useMemo(() => {
    const set = new Set<string>();
    selectedPorts.forEach((p) => set.add(p.deviceId));
    return Array.from(set);
  }, [selectedPorts]);

  const isMixedDevices = uniqueDeviceIds.length > 1;

  /** 组装 BatchWriteRequest (T-53-07: 白名单字段, 不 spread 整个 port record)
   * CR-01: deviceId 改为显式参数 —— 首次提交传当前 uniqueDeviceIds[0], retry 传缓存的 lastDeviceId,
   *   杜绝从漂移的 selectedPorts 快照重读 deviceId。 */
  const buildRequest = (
    deviceId: string,
    action: PortWriteAction,
    portIds: string[],
    description?: string
  ): BatchWriteRequest => ({
    deviceId,
    action,
    portIds,
    ...(action === "description" && description !== undefined ? { description } : {}),
  });

  /** D-06 重试只取 failed, 不重试 skipped (skipped = 设备已是目标态, 重试无意义) */
  const handleRetryFailed = async (): Promise<void> => {
    if (!batchResult || batchResult.failed.length === 0) return;
    const failedIds = batchResult.failed.map((p) => p.portId);
    // CR-01/CR-02: retry 用首次提交时缓存的 lastDeviceId, 不从 selectedPorts 重读
    const req = buildRequest(lastDeviceId, lastAction, failedIds, lastDescription);
    setPhase("executing");
    try {
      // LANDMINE #3: batchWritePorts 失败是正常 resolve, 不用 .catch 分类
      const newResult = await batchWritePorts(req);
      setBatchResult(newResult);
      setPhase("result");
      if (newResult.failed.length === 0) {
        // D-10 全部成功 (含重试后) 弹审计 Toast
        showAuditLinkToast(message, navigate);
        onSuccess();
      } else if (newResult.succeeded.length > 0) {
        // 部分成功, 刷新父组件列表让用户看到已生效的端口, 但不弹 Toast (Drawer 还在结果视图)
        onSuccess();
      }
    } catch (err) {
      // 网络错误等才走 catch, message.error 由 post() 拦截器已弹 (LANDMINE #5)
      // Phase 123: 409 端口占用 / 503 预检失败 → 常驻 Alert 补充上下文（持锁者/预检信息在 message 内）
      const code = (err as { code?: number })?.code;
      if (code === 409 || code === 503) {
        setWriteError({ code, message: (err as Error).message });
      }
      setPhase("select");
    }
  };

  /** PORTW-02 第一段: select 提交 → dry-run 预览（不下发任何设备命令）。
   * 校验通过后快照提交参数（与 CR-01/WR-03 同理），preview 失败留在 select 由拦截器弹 Toast。 */
  const handlePreview = async (): Promise<void> => {
    setWriteError(null);
    let values: Record<string, unknown>;
    try {
      values = await form.validateFields();
    } catch (err) {
      // validateFields 校验失败抛含 errorFields 的对象, antd 自动标红字段, 直接 return
      if (err && typeof err === "object" && "errorFields" in err) return;
      // WR-01: 非预期校验异常不再 throw 冒泡到 antd onOk/onClick 形成未处理 Promise rejection
      handleApiError(err, "表单校验", false);
      return;
    }

    const action = values.action as PortWriteAction;
    const reason = composeReason(values.reasonSelect, values.reasonText);
    const description = typeof values.description === "string" ? values.description : undefined;

    // 必填校验 (action 总是必填, reason 在非 description action 时必填, validateFields 已通过这里再兜底)
    if (action !== "description" && (reason === null || reason.length === 0)) {
      message.error("请选择或输入操作原因");
      return;
    }

    const portIds = selectedPorts.map((p) => p.id);
    const deviceId = uniqueDeviceIds[0] ?? "";
    // CR-01/WR-03: 提交时快照 deviceId + interfaceMap, 后续 retry/result 全部基于快照,
    // 不受 onSuccess 触发的父级 loadPortStatus 刷新导致的 selectedPorts 引用漂移影响
    const interfaceMap = new Map<string, string>();
    selectedPorts.forEach((p) => interfaceMap.set(p.id, p.interfaceName));

    setLastAction(action);
    setLastDescription(description);
    setLastDeviceId(deviceId);
    setLastInterfaceMap(interfaceMap);
    setPhase("preview");
    try {
      const preview = await previewBatchWritePorts(
        buildRequest(deviceId, action, portIds, description)
      );
      setPreviewResult(preview);
    } catch {
      // 400 参数 / 404 不存在等, post() 拦截器已弹 Toast, 返回 select 让用户修正
      setPhase("select");
    }
  };

  /** PORTW-02 第二段: preview 确认 → 真实批量写（沿用预览时快照的参数, 不再重复校验）。 */
  const handleConfirmWrite = async (): Promise<void> => {
    const portIds = selectedPorts.map((p) => p.id);
    const req = buildRequest(lastDeviceId, lastAction, portIds, lastDescription);

    setPhase("executing");
    try {
      // LANDMINE #3: batchWritePorts HTTP 200 + status:failed 是正常 resolve
      const result = await batchWritePorts(req);
      setBatchResult(result);
      setPhase("result");
      if (result.failed.length === 0) {
        // D-10 全部成功 (succeeded + skipped) 弹审计 Toast
        showAuditLinkToast(message, navigate);
        onSuccess();
      } else if (result.succeeded.length > 0) {
        // 部分成功, 刷新父组件列表让已生效端口可见
        onSuccess();
      }
    } catch (err) {
      // 网络错误等才走 catch, post() 拦截器已弹 Toast
      // Phase 123: 409 端口占用 / 503 预检失败 → 常驻 Alert 补充上下文
      const code = (err as { code?: number })?.code;
      if (code === 409 || code === 503) {
        setWriteError({ code, message: (err as Error).message });
      }
      setPhase("preview");
    }
  };

  // executing 阶段禁用 Drawer 关闭 (避免用户中途关 Drawer 留下孤儿 batch 状态)
  const isExecuting = phase === "executing";

  return (
    <Drawer
      title="批量配置端口"
      open={open}
      onClose={isExecuting ? () => {} : onClose}
      size={720}
      destroyOnHidden
      mask={{ closable: !isExecuting }}
      closable={!isExecuting}
    >
      {/* Phase 123 错误语义: 409/503 常驻 Alert（持锁者/预检详情后端带在 message 内） */}
      {writeError && !isExecuting && (
        <Alert
          type={writeError.code === 409 ? "warning" : "error"}
          showIcon
          style={{ marginBottom: 16 }}
          title={writeError.code === 409 ? "端口正在被其他写操作处理" : "设备不可达（预检失败）"}
          description={writeError.message}
          closable
          onClose={() => setWriteError(null)}
        />
      )}

      {phase === "select" && (
        <SelectView
          form={form}
          selectedPorts={selectedPorts}
          uniqueDeviceCount={uniqueDeviceIds.length}
          isMixedDevices={isMixedDevices}
          onSubmit={handlePreview}
        />
      )}

      {/* PORTW-02: dry-run 预览视图（确认后才进入真实写） */}
      {phase === "preview" && previewResult && (
        <PreviewView
          preview={previewResult}
          interfaceMap={lastInterfaceMap}
          onConfirm={handleConfirmWrite}
          onBack={() => setPhase("select")}
        />
      )}

      {/* D-05 indeterminate Spin, 不伪造 X/Y (ROADMAP #2 纠正) */}
      {phase === "executing" && (
        <div style={{ textAlign: "center", padding: "60px 0" }}>
          <Spin size="large" tip="正在批量配置...（预计 ~1s/端口）">
            <div style={{ padding: "30px" }} />
          </Spin>
        </div>
      )}

      {phase === "result" && batchResult && (
        <ResultView
          result={batchResult}
          // WR-03: 用提交时快照的 interfaceMap, 不随父级 selectedPorts 漂移失真
          interfaceMap={lastInterfaceMap}
          onRetry={handleRetryFailed}
        />
      )}

      {/* Phase 123: 写失败且回滚也失败 → 常驻告警（不允许一闪而过）, 引导查看审计页 */}
      {phase === "result" && batchResult?.failed.some((p) => p.extra?.rollback === "rollback_failed") && (
        <Alert
          type="error"
          showIcon
          style={{ marginTop: 16 }}
          title="严重：部分端口写失败且自动回滚也失败"
          description={
            <Space direction="vertical" size={4}>
              {batchResult.failed
                .filter((p) => p.extra?.rollback === "rollback_failed")
                .map((p) => (
                  <Typography.Text key={p.portId} type="danger" code>
                    {lastInterfaceMap.get(p.portId) ?? p.portId}:{" "}
                    {p.extra?.rollbackError ?? p.error ?? "未知回滚错误"}
                  </Typography.Text>
                ))}
              <Typography.Text>
                设备可能处于中间状态，请立即人工核查，并到端口写审计页核对变更前后快照。
              </Typography.Text>
            </Space>
          }
          action={
            <Button
              size="small"
              danger
              onClick={() => {
                onClose();
                navigate("/network/port-audits");
              }}
            >
              查看端口写审计
            </Button>
          }
        />
      )}
    </Drawer>
  );
}

// ==================== Select View (子组件, 简化主组件 render) ====================

interface SelectViewProps {
  form: ReturnType<typeof Form.useForm>[0];
  selectedPorts: DevicePortStatus[];
  uniqueDeviceCount: number;
  isMixedDevices: boolean;
  onSubmit: () => Promise<void>;
}

function SelectView({
  form,
  selectedPorts,
  uniqueDeviceCount,
  isMixedDevices,
  onSubmit,
}: SelectViewProps) {
  // 55-01 WR-02: 用 Form.useWatch 实时跟踪当前 action, 给 reasonSelect.rules 动态选择
  // validateReasonOptional (description) 或 validateReasonRequired (其他 4 action)。
  // SelectView 本就因 getFieldValue 在 shouldUpdate 重渲染, useWatch 不引入额外开销。
  const action = Form.useWatch("action", form) as PortWriteAction | undefined;

  return (
    <Form form={form} layout="vertical">
      {/* 只读汇总: 已选 N 端口 / 唯一设备数 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={12}>
          <Card>
            <Statistic title="已选端口" value={selectedPorts.length} />
          </Card>
        </Col>
        <Col span={12}>
          <Card>
            <Statistic title="唯一设备数" value={uniqueDeviceCount} />
          </Card>
        </Col>
      </Row>

      {/* 跨设备预校验: 后端 ErrMixedDevices 是真相源, 前端先 Alert 提示并禁用提交 */}
      {isMixedDevices && (
        <Alert
          type="error"
          showIcon
          title="批量必须同设备"
          description={`检测到 ${uniqueDeviceCount} 个不同设备, 后端会拒绝跨设备批量。请重新勾选, 确保所选端口属于同一设备 (same device)。`}
          style={{ marginBottom: 16 }}
        />
      )}

      <Form.Item
        name="action"
        label="操作类型"
        rules={[{ required: true, message: "请选择操作类型" }]}
      >
        {/* fixed option list, no server search needed（AppSelect 不受 local/no-large-dropdown-list
            的 Select 名拦截，原 eslint-disable 已随迁移失效，故移除） */}
        <AppSelect placeholder="请选择操作类型" options={ACTION_OPTIONS} />
      </Form.Item>

      {/* D-03 description action 时多一个"新描述"必填字段 */}
      <Form.Item shouldUpdate noStyle>
        {({ getFieldValue }) =>
          getFieldValue("action") === "description" ? (
            <Form.Item
              name="description"
              label="新描述"
              rules={[
                { required: true, message: "请输入新端口描述" },
                { max: DESCRIPTION_MAX, message: `描述不超过 ${DESCRIPTION_MAX} 字符` },
              ]}
            >
              <AppInput maxLength={DESCRIPTION_MAX} showCount placeholder="请输入新端口描述" />
            </Form.Item>
          ) : null
        }
      </Form.Item>

      {/* 55-01 WR-02: reasonSelect.rules 按 action 动态选择 validator,
          从 form 参数跨字段读取 reasonText, 触发完整长度校验 (含 REASON_MIN=5 下限) */}
      <Form.Item
        name="reasonSelect"
        label="操作原因"
        rules={
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
              ]
        }
      >
        {/* fixed option list, no server search needed（同上，disable 注释已随迁移失效） */}
        <AppSelect
          placeholder="请选择操作原因"
          options={PRESET_REASONS.map((opt) => ({ label: opt.label, value: opt.value }))}
        />
      </Form.Item>

      {/* 55-01 WR-02: reasonText.rules 同时含 REASON_MIN 下限 (此前只有 max),
          与 PRESET_REASONS value 字符数 ≥ REASON_MIN 自洽约束一致 */}
      <Form.Item shouldUpdate noStyle>
        {({ getFieldValue }) =>
          getFieldValue("reasonSelect") === REASON_CUSTOM_SENTINEL ? (
            <Form.Item
              name="reasonText"
              label="自定义原因"
              rules={[
                { min: REASON_MIN, message: `操作原因至少 ${REASON_MIN} 个字符` },
                { max: REASON_MAX, message: `操作原因不超过 ${REASON_MAX} 个字符` },
              ]}
            >
              <AppTextArea
                rows={2}
                maxLength={REASON_MAX}
                showCount
                placeholder={`请输入操作原因（${REASON_MIN}-${REASON_MAX} 字符）`}
              />
            </Form.Item>
          ) : null
        }
      </Form.Item>

      <Form.Item>
        <Button
          type="primary"
          onClick={onSubmit}
          disabled={selectedPorts.length === 0 || isMixedDevices}
        >
          预览命令（dry-run）
        </Button>
      </Form.Item>
    </Form>
  );
}

// ==================== Preview View (PORTW-02 两段式第一段, dry-run) ====================

/** 风险等级 → antd Tag 色板（token 语义色, 无手写暗色覆盖） */
const RISK_TAG_COLOR: Record<"critical" | "medium" | "low", string> = {
  critical: "red",
  medium: "orange",
  low: "green",
};

const RISK_LABEL: Record<"critical" | "medium" | "low", string> = {
  critical: "高危",
  medium: "中危",
  low: "低危",
};

interface PreviewViewProps {
  preview: PortWritePreviewResult;
  /** WR-03: 提交时快照的 portId → interfaceName 映射 */
  interfaceMap: Map<string, string>;
  onConfirm: () => Promise<void>;
  onBack: () => void;
}

/**
 * PreviewView — dry-run 预览视图（Phase 121 PORTW-02）
 * - Alert 常驻标注「dry-run 预览，未下发任何设备命令」（验收 6, 用后端 summary）
 * - 按端口分组: Table + expandedRowRender 展开命令序列
 * - 风险色标 critical/medium/low（antd Tag 预设色, 零手写 CSS）
 * - skipped 端口显式标注 noOp（pre-state 已匹配, 下发时将被跳过）
 */
function PreviewView({ preview, interfaceMap, onConfirm, onBack }: PreviewViewProps) {
  const willApply = preview.ports.filter((p) => p.status === "preview");
  const skipped = preview.ports.filter((p) => p.status === "skipped");

  return (
    <div>
      {/* 验收 6: dry-run 常驻提示 */}
      <Alert
        type="warning"
        showIcon
        title="dry-run 预览，未下发任何设备命令"
        description={preview.summary}
        style={{ marginBottom: 16 }}
      />

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card>
            <Statistic title="设备" value={preview.deviceName} />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic title="将下发端口" value={willApply.length} />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic title="跳过 (noOp)" value={skipped.length} />
          </Card>
        </Col>
      </Row>

      {/* 整体风险等级 */}
      <Space style={{ marginBottom: 16 }}>
        <span>整体风险等级：</span>
        <Tag color={RISK_TAG_COLOR[preview.riskLevel]}>{RISK_LABEL[preview.riskLevel]}</Tag>
        <Typography.Text type="secondary">
          {preview.vendor} {preview.model} · {ACTION_TITLE[preview.action]}
        </Typography.Text>
      </Space>

      {/* 按端口分组: 展开行显示命令序列 */}
      <Table<PortWritePreviewPort>
        dataSource={preview.ports}
        rowKey="portId"
        size="small"
        pagination={false}
        style={{ marginBottom: 16 }}
        expandable={{
          defaultExpandAllRows: true,
          expandedRowRender: (port) =>
            port.commands.length === 0 ? (
              <Typography.Text type="secondary">
                无命令 —— pre-state 已匹配目标态，下发时将被跳过
              </Typography.Text>
            ) : (
              <ul style={{ margin: 0, paddingLeft: 16 }}>
                {port.commands.map((cmd) => (
                  <li key={cmd.index} style={{ marginBottom: 4 }}>
                    <Tag color={RISK_TAG_COLOR[cmd.riskLevel]}>{RISK_LABEL[cmd.riskLevel]}</Tag>
                    <Typography.Text code>{cmd.command}</Typography.Text>
                    {cmd.note && (
                      <Typography.Text type="secondary" style={{ marginLeft: 8 }}>
                        {cmd.note}
                      </Typography.Text>
                    )}
                  </li>
                ))}
              </ul>
            ),
        }}
        columns={[
          {
            title: "接口名",
            key: "interfaceName",
            width: 140,
            render: (_, port) => interfaceMap.get(port.portId) ?? port.interfaceName,
          },
          {
            title: "状态",
            key: "status",
            width: 120,
            render: (_, port) =>
              port.status === "skipped" ? (
                <Tag color="default">跳过 (noOp)</Tag>
              ) : (
                <Tag color="blue">将下发</Tag>
              ),
          },
          {
            title: "命令数",
            key: "commandCount",
            width: 80,
            render: (_, port) => port.commands.length,
          },
        ]}
      />

      {/* 第二段确认: 高危动作用 danger 按钮强化 */}
      <Space>
        <Button onClick={onBack}>返回修改</Button>
        <Button
          type="primary"
          danger={preview.riskLevel === "critical"}
          onClick={() => void onConfirm()}
        >
          确认下发（真实写）
        </Button>
      </Space>
    </div>
  );
}

// ==================== Result View (子组件) ====================

interface ResultViewProps {
  result: BatchResult;
  /** WR-03: 提交时快照的 portId → interfaceName 映射 (父组件 setLastInterfaceMap 缓存),
   * 不随父级 selectedPorts 引用漂移失真。 */
  interfaceMap: Map<string, string>;
  onRetry: () => Promise<void>;
}

function ResultView({ result, interfaceMap, onRetry }: ResultViewProps) {
  return (
    <div>
      {/* 三 Statistic 卡片: ✓ 成功 / ⚠ 跳过 / ✗ 失败 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card>
            <Statistic
              title="✓ 成功"
              value={result.succeeded.length}
              valueStyle={{ color: "var(--theme-success)" }}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="⚠ 跳过"
              value={result.skipped.length}
              valueStyle={{ color: "var(--theme-text-secondary)" }}
            />
            {result.skipped.length > 0 && (
              <Tag color="default" style={{ marginTop: 8 }}>
                无需操作
              </Tag>
            )}
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="✗ 失败"
              value={result.failed.length}
              valueStyle={{ color: "var(--theme-error)" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 失败明细 Table (LANDMINE #3: dataSource=batchResult.failed 从 body 读, 不是 .catch) */}
      {result.failed.length > 0 && (
        <>
          <Typography.Title level={5}>失败明细</Typography.Title>
          <Table<PortResult>
            dataSource={result.failed}
            rowKey="portId"
            size="small"
            pagination={false}
            expandable={{
              expandedRowRender: (port) => (
                <Typography.Text type="secondary" code>
                  {port.commandSent || "(无命令记录)"}
                </Typography.Text>
              ),
            }}
            columns={[
              {
                title: "接口名",
                key: "interfaceName",
                width: 150,
                render: (_, port) => interfaceMap.get(port.portId) ?? port.portId,
              },
              {
                title: "错误原因",
                dataIndex: "error",
                key: "error",
                ellipsis: true,
              },
            ]}
            style={{ marginBottom: 16 }}
          />
        </>
      )}

      {/* 跳过明细折叠 (D-05 默认收起) */}
      {result.skipped.length > 0 && (
        <Collapse
          style={{ marginBottom: 16 }}
          items={[
            {
              key: "skipped",
              label: `跳过明细 (${result.skipped.length})`,
              children: (
                <ul style={{ margin: 0, paddingLeft: 20 }}>
                  {result.skipped.map((p) => (
                    <li key={p.portId}>
                      {interfaceMap.get(p.portId) ?? p.portId}
                      {p.currentState ? ` — 当前态: ${p.currentState}` : ""}
                    </li>
                  ))}
                </ul>
              ),
            },
          ]}
        />
      )}

      {/* D-06 重试按钮 — 只取 batchResult.failed (上面 dataSource 同源), disabled 当 failed.length === 0 */}
      <Space>
        <Button type="primary" disabled={result.failed.length === 0} onClick={onRetry}>
          重试失败端口 ({result.failed.length})
        </Button>
      </Space>
    </div>
  );
}

export default BulkWriteDrawer;
