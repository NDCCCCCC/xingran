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

import { useEffect, useMemo, useState, useRef } from "react";
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
  Switch,
  Progress,
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
import { batchWritePorts, previewBatchWritePorts, listPortWriteAudits } from "@/lib/api/networkApi";
import { useNoticeStore } from "@/store/noticeStore";
import {
  PRESET_REASONS,
  ACTION_TITLE,
  REASON_MIN,
  REASON_MAX,
  DESCRIPTION_MAX,
  REASON_CUSTOM_SENTINEL,
  TRUNK_ALLOWED_MODE_OPTIONS,
  TRUNK_REPLACE_HINT,
  TRUNK_HYBRID_OVERLAP_HINT,
  SKIP_PREFLIGHT_HINT,
  PORTSEC_MAC_HINT,
  PHYSICAL_SPEED_OPTIONS,
  PHYSICAL_SPEED_MODE_AUTO,
  PHYSICAL_DUPLEX_OPTIONS,
  PHYSICAL_MTU_MIN,
  PHYSICAL_MTU_MAX,
  PHYSICAL_NO_ROLLBACK_HINT,
  PHYSICAL_EVIDENCE_LABEL,
  composeReason,
  validateReasonOptional,
  validateReasonRequired,
  validateVlanListText,
  parseVlanList,
  parseMacAddress,
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
  // v1.51 Phase 124 (9835291c): trunk/hybrid 三动作
  { label: ACTION_TITLE.trunk_set_native, value: "trunk_set_native" },
  { label: ACTION_TITLE.trunk_set_allowed, value: "trunk_set_allowed" },
  { label: ACTION_TITLE.trunk_set_hybrid, value: "trunk_set_hybrid" },
  // v1.51 Phase 125 (006ece28): portsec 五动作
  { label: ACTION_TITLE.portsec_enable, value: "portsec_enable" },
  { label: ACTION_TITLE.portsec_disable, value: "portsec_disable" },
  { label: ACTION_TITLE.portsec_max_mac, value: "portsec_max_mac" },
  { label: ACTION_TITLE.portsec_sticky, value: "portsec_sticky" },
  { label: ACTION_TITLE.portsec_static_bind, value: "portsec_static_bind" },
  // v1.54 Phase 143 (PORT-01/02): 物理参数三动作
  { label: ACTION_TITLE.set_speed, value: "set_speed" },
  { label: ACTION_TITLE.set_duplex, value: "set_duplex" },
  { label: ACTION_TITLE.set_mtu, value: "set_mtu" },
];

/** Phase 122 (PORTW-03): WS 实时进度聚合状态 */
interface LiveProgress {
  batchId: string | null;
  total: number;
  events: {
    portId: string;
    interfaceName: string;
    index: number;
    status: string;
    message?: string;
  }[];
  summary: { succeeded: number; failed: number; skipped: number } | null;
  completedMessage: string | null;
  lastEventAt: number;
}

/** trunk/portsec/physical 动作的批量写附加参数（T-53-07 白名单的 Phase 124/125/143 延伸） */
interface TrunkParams {
  nativeVlan?: number;
  allowedVlans?: number[];
  mode?: "add" | "remove" | "replace";
  hybridTagged?: number[];
  hybridUntagged?: number[];
  // Phase 125 portsec
  maxMacCount?: number;
  macAddress?: string; // 归一后 AA-BB-CC-DD-EE-FF
  staticBindVlanId?: number; // portsec_static_bind 的 VLAN（1-4094）
  // v1.54 Phase 143 物理参数（复用同一份参数快照，确认写/重试与预览同参）
  speed?: number;
  speedMode?: string;
  duplex?: string;
  mtu?: number;
}

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
  /** Phase 124: trunk 附加参数快照（retry/确认写与首次预览同参, CR-01 同理不重读表单） */
  const [lastTrunkParams, setLastTrunkParams] = useState<TrunkParams | undefined>(undefined);
  /** Phase 124 补强: portIds 同样快照——预览与确认之间父级刷新使 selectedPorts 漂移时,
   * 确认写必须落在预览过的那批端口上（否则 dry-run 与真实写内容错位）。 */
  const [lastPortIds, setLastPortIds] = useState<string[]>([]);
  /** Phase 125: skipPreflight 快照（确认写/重试与预览同参） */
  const [lastSkipPreflight, setLastSkipPreflight] = useState<boolean | undefined>(undefined);

  // ===== v1.51 Phase 122 (PORTW-03): WS 实时进度状态 =====
  const [live, setLive] = useState<LiveProgress | null>(null);
  const liveRef = useRef(live);
  liveRef.current = live;

  // WS 事件订阅：executing 阶段挂监听（复用 NoticeHub 连接的重连逻辑, 契约 ece40502）
  useEffect(() => {
    if (phase !== "executing") return;
    const unsubscribe = useNoticeStore.getState().onPortWriteProgress((event) => {
      const cur = liveRef.current;
      if (!cur) return;
      // batchId 锁定：首个事件确定批次, 后续事件仅当 batchId 匹配才接受
      if (cur.batchId && event.batchId !== cur.batchId) return;
      if (event.type === "port_write_progress") {
        setLive({
          ...cur,
          batchId: event.batchId,
          total: event.total,
          events: [
            ...cur.events.filter((e: { portId: string }) => e.portId !== event.portId),
            {
              portId: event.portId,
              interfaceName: event.interfaceName,
              index: event.index,
              status: event.status,
              message: event.message,
            },
          ],
          lastEventAt: Date.now(),
        });
      } else {
        // port_write_completed: 终态 summary（同步 BatchResult 仍为结果视图真相源）
        let summary: { succeeded: number; failed: number; skipped: number } | null = null;
        try {
          summary = JSON.parse(event.summary);
        } catch {
          summary = null;
        }
        setLive({
          ...cur,
          batchId: event.batchId,
          summary,
          completedMessage: event.message,
          lastEventAt: Date.now(),
        });
      }
    });
    return unsubscribe;
  }, [phase]);

  // 断线对账（契约: WS 不补发, audit 是真相源）：executing 且已锁定 batchId 时,
  // 每 15s 检查一次——距最后事件 >20s（断线丢事件/重连后）即拉 audit 刷新进度快照。
  // 契约 9a1ea4d7: 优先 batchId 等值过滤（精确）; 历史批次（migration 232 前落库）
  // batchId 为空时退回 deviceId+pageSize 近似兜底（两路径并存）。
  useEffect(() => {
    if (phase !== "executing") return;
    const timer = setInterval(() => {
      const cur = liveRef.current;
      if (!cur?.batchId || cur.summary) return; // 未锁定批次或已终态, 不对账
      if (Date.now() - cur.lastEventAt < 20000) return;
      listPortWriteAudits({
        current: 1,
        pageSize: Math.min(cur.total, 100),
        batchId: cur.batchId,
        deviceId: lastDeviceId || undefined, // 历史批次兜底（batchId 过滤为空时生效）
      })
        .then((res) => {
          const rows = (res.list ?? []).filter((r) => r.commandSent && r.status !== undefined);
          if (rows.length === 0) return;
          setLive((prev) =>
            prev
              ? {
                  ...prev,
                  events: rows.map((r, i) => ({
                    portId: r.portId,
                    interfaceName: r.portId,
                    index: i + 1,
                    status: r.status,
                    message: r.failureReason ?? undefined,
                  })),
                  lastEventAt: Date.now(),
                }
              : prev
          );
        })
        .catch(() => {
          // 对账失败静默（同步 BatchResult 兜底, 不白屏）
        });
    }, 15000);
    return () => clearInterval(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [phase]);

  // open 变 false 时 reset phase + batchResult, 防下次打开残留
  // CLAUDE.md useEffect 纪律: deps 全稳定 (open/phase/onExecutingChange 原始值或稳定 setter)
  useEffect(() => {
    if (!open) {
      setPhase("select");

      setBatchResult(null);
      setPreviewResult(null);
      setWriteError(null);
      setLive(null);
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
   *   杜绝从漂移的 selectedPorts 快照重读 deviceId。
   * Phase 124: trunkParams 仅 trunk 动作携带（白名单字段直传, 后端按 action 校验）。 */
  const buildRequest = (
    deviceId: string,
    action: PortWriteAction,
    portIds: string[],
    description?: string,
    trunkParams?: TrunkParams,
    skipPreflight?: boolean
  ): BatchWriteRequest => ({
    deviceId,
    action,
    portIds,
    ...(action === "description" && description !== undefined ? { description } : {}),
    ...(action === "trunk_set_native" && trunkParams?.nativeVlan !== undefined
      ? { nativeVlan: trunkParams.nativeVlan }
      : {}),
    ...(action === "trunk_set_allowed" && trunkParams
      ? {
          allowedVlans: trunkParams.allowedVlans,
          mode: trunkParams.mode ?? "replace",
        }
      : {}),
    ...(action === "trunk_set_hybrid" && trunkParams
      ? {
          hybridTagged: trunkParams.hybridTagged,
          hybridUntagged: trunkParams.hybridUntagged,
        }
      : {}),
    // v1.51 Phase 125 portsec（portsec_enable/disable/sticky 无附加参数）
    ...(action === "portsec_max_mac" && trunkParams?.maxMacCount !== undefined
      ? { maxMacCount: trunkParams.maxMacCount }
      : {}),
    ...(action === "portsec_static_bind" && trunkParams
      ? {
          macAddress: trunkParams.macAddress,
          vlanId: trunkParams.staticBindVlanId,
        }
      : {}),
    // v1.54 Phase 143 物理参数（按 action 白名单携带, 后端按型号能力画像校验）
    ...(action === "set_speed" && trunkParams
      ? trunkParams.speedMode === PHYSICAL_SPEED_MODE_AUTO
        ? { speedMode: PHYSICAL_SPEED_MODE_AUTO }
        : { speed: trunkParams.speed }
      : {}),
    ...(action === "set_duplex" && trunkParams ? { duplex: trunkParams.duplex } : {}),
    ...(action === "set_mtu" && trunkParams ? { mtu: trunkParams.mtu } : {}),
    ...(skipPreflight ? { skipPreflight: true } : {}),
  });

  /** D-06 重试只取 failed, 不重试 skipped (skipped = 设备已是目标态, 重试无意义) */
  const handleRetryFailed = async (): Promise<void> => {
    if (!batchResult || batchResult.failed.length === 0) return;
    const failedIds = batchResult.failed.map((p) => p.portId);
    // CR-01/CR-02: retry 用首次提交时缓存的 lastDeviceId, 不从 selectedPorts 重读
    // Phase 124: trunk 参数同样走快照（lastTrunkParams）; Phase 122: retry 走 runBatch（带实时进度）
    const req = buildRequest(
      lastDeviceId,
      lastAction,
      failedIds,
      lastDescription,
      lastTrunkParams,
      lastSkipPreflight
    );
    await runBatch(req);
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
    // Phase 124: 按 action 收集 trunk 附加参数（表单 rules 已校验, 此处仅解析/归一）
    let trunkParams: TrunkParams | undefined;
    if (action === "trunk_set_native") {
      trunkParams = { nativeVlan: Number(values.nativeVlan) };
    } else if (action === "trunk_set_allowed") {
      trunkParams = {
        allowedVlans: parseVlanList(String(values.allowedVlansText)) ?? [],
        mode: (values.trunkMode as "add" | "remove" | "replace") ?? "replace",
      };
    } else if (action === "trunk_set_hybrid") {
      trunkParams = {
        hybridTagged: values.hybridTaggedText
          ? (parseVlanList(String(values.hybridTaggedText)) ?? [])
          : [],
        hybridUntagged: values.hybridUntaggedText
          ? (parseVlanList(String(values.hybridUntaggedText)) ?? [])
          : [],
      };
    } else if (action === "portsec_max_mac") {
      trunkParams = { maxMacCount: Number(values.maxMacCount) };
    } else if (action === "portsec_static_bind") {
      trunkParams = {
        macAddress: parseMacAddress(String(values.staticMacAddress)) ?? undefined,
        staticBindVlanId: Number(values.staticBindVlanId),
      };
    } else if (action === "set_speed") {
      // v1.54 Phase 143: auto 走 speedMode 哨兵, 否则带数字速率
      trunkParams =
        values.speedMode === PHYSICAL_SPEED_MODE_AUTO
          ? { speedMode: PHYSICAL_SPEED_MODE_AUTO }
          : { speed: Number(values.speed) };
    } else if (action === "set_duplex") {
      trunkParams = { duplex: String(values.duplex) };
    } else if (action === "set_mtu") {
      trunkParams = { mtu: Number(values.mtu) };
    }

    // Phase 125: skipPreflight 开关（表单可达即持 network:port:write —— 菜单 perms 已门控路由,
    // 无需二次 decide, 无真相源问题）。语义：仅真实写请求携带, dry-run 预览不受影响
    //（预览展示命令序列是防误操作核心, 与在线预检是两回事, 保留预览流程）。
    const skipPreflight = values.skipPreflight === true ? true : undefined;

    // CR-01/WR-03: 提交时快照 deviceId + interfaceMap, 后续 retry/result 全部基于快照,
    // 不受 onSuccess 触发的父级 loadPortStatus 刷新导致的 selectedPorts 引用漂移影响
    const interfaceMap = new Map<string, string>();
    selectedPorts.forEach((p) => interfaceMap.set(p.id, p.interfaceName));

    setLastAction(action);
    setLastDescription(description);
    setLastDeviceId(deviceId);
    setLastInterfaceMap(interfaceMap);
    setLastTrunkParams(trunkParams);
    setLastPortIds(portIds);
    setLastSkipPreflight(skipPreflight);
    setPhase("preview");
    try {
      const preview = await previewBatchWritePorts(
        buildRequest(deviceId, action, portIds, description, trunkParams)
      );
      setPreviewResult(preview);
    } catch {
      // 400 参数 / 404 不存在等, post() 拦截器已弹 Toast, 返回 select 让用户修正
      setPhase("select");
    }
  };

  /** PORTW-02 第二段: preview 确认 → 真实批量写（沿用预览时快照的参数, 不再重复校验）。 */
  const handleConfirmWrite = async (): Promise<void> => {
    // Phase 124: portIds 用预览时快照（预览与确认间父级刷新导致的漂移防护）
    const req = buildRequest(
      lastDeviceId,
      lastAction,
      lastPortIds,
      lastDescription,
      lastTrunkParams,
      lastSkipPreflight
    );
    await runBatch(req);
  };

  /** Phase 122 (PORTW-03): 执行批次 + WS 实时进度订阅 + 断线对账。
   * confirm 与 retry 共用（retry 换 req 的 portIds 为 failed 子集, 进度 total 同步子集数）。 */
  const runBatch = async (req: BatchWriteRequest): Promise<void> => {
    // 初始化实时进度状态（batchId 待首个 WS 事件或同步响应锁定）
    setLive({
      batchId: null,
      total: req.portIds.length,
      events: [],
      summary: null,
      completedMessage: null,
      lastEventAt: Date.now(),
    });
    setPhase("executing");
    try {
      // LANDMINE #3: batchWritePorts HTTP 200 + status:failed 是正常 resolve
      const result = await batchWritePorts(req);
      setBatchResult(result);
      // 同步响应锁定 batchId（WS 事件先到时 listener 已按事件值锁定, 此处幂等）
      if (result.batchId) {
        setLive((prev) => (prev && !prev.batchId ? { ...prev, batchId: result.batchId! } : prev));
      }
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
    } finally {
      // 批次结束清空实时进度（result 视图以 batchResult 为真相源）
      setLive(null);
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

      {/* Phase 122 (PORTW-03): executing 态 —— WS 订阅驱动的真实进度（契约 ece40502）。
          原 D-05「indeterminate 不伪造 X/Y」约束被 122 解除：后端逐端口实时推送使
          index/total 为真实值而非伪造；WS 不可用/事件未达时回退 indeterminate Spin
          （batch 同步响应的 BatchResult 兜底, 不白屏）。 */}
      {phase === "executing" && live && live.events.length > 0 && <ExecutingView live={live} />}
      {phase === "executing" && (!live || live.events.length === 0) && (
        <div style={{ textAlign: "center", padding: "60px 0" }}>
          <Spin size="large" tip="正在批量配置...（预计 ~1s/端口）">
            <div style={{ padding: "30px" }} />
          </Spin>
          <Typography.Text type="secondary" style={{ marginTop: 16, display: "block" }}>
            实时进度通道未就绪，完成后将直接显示结果
          </Typography.Text>
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
      {phase === "result" &&
        batchResult?.failed.some((p) => p.extra?.rollback === "rollback_failed") && (
          <Alert
            type="error"
            showIcon
            style={{ marginTop: 16 }}
            title="严重：部分端口写失败且自动回滚也失败"
            description={
              <Space orientation="vertical" size={4}>
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

// ==================== Executing View (Phase 122 PORTW-03, WS 实时进度) ====================

const LIVE_STATUS_TAG: Record<string, { color: string; label: string }> = {
  succeeded: { color: "green", label: "成功" },
  failed: { color: "red", label: "失败" },
  skipped: { color: "default", label: "跳过" },
};

/**
 * ExecutingView — WS 订阅驱动的真实进度（契约 ece40502）
 * - Progress: index/total 真实百分比（D-05 约束被 122 解除）
 * - 端口列表逐行点亮（按事件到达顺序, status Tag 区分 succeeded/failed/skipped）
 * - completed 终态: summary 三计数 + fail-fast 中断文案
 * - 交互选择说明：列表逐行点亮而非仅当前行——批量操作中「哪些已好/哪些失败/哪些
 *   被跳过」是用户最关心的全景, 滚动单行会丢失败上下文
 */
function ExecutingView({ live }: { live: LiveProgress }) {
  const done = live.events.length;
  const percent = live.total > 0 ? Math.round((done / live.total) * 100) : 0;
  return (
    <div>
      <Progress
        percent={live.summary ? 100 : percent}
        status={live.summary ? "normal" : "active"}
        style={{ marginBottom: 16 }}
      />
      {live.summary && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={8}>
            <Statistic title="✓ 成功" value={live.summary.succeeded} />
          </Col>
          <Col span={8}>
            <Statistic title="✗ 失败" value={live.summary.failed} />
          </Col>
          <Col span={8}>
            <Statistic title="⚠ 跳过" value={live.summary.skipped} />
          </Col>
        </Row>
      )}
      {live.completedMessage && (
        <Typography.Text strong style={{ display: "block", marginBottom: 16 }}>
          {live.completedMessage}
        </Typography.Text>
      )}
      <Table
        dataSource={live.events}
        rowKey="portId"
        size="small"
        pagination={false}
        columns={[
          { title: "接口名", dataIndex: "interfaceName", key: "interfaceName", width: 180 },
          {
            title: "状态",
            dataIndex: "status",
            key: "status",
            width: 100,
            render: (v: string) => {
              const meta = LIVE_STATUS_TAG[v] ?? { color: "default", label: v };
              return <Tag color={meta.color}>{meta.label}</Tag>;
            },
          },
          { title: "详情", dataIndex: "message", key: "message", ellipsis: true },
        ]}
      />
    </div>
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
  // validateReasonOptional (description) 或 validateReasonRequired (其他 action)。
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
            的 Select 名拦截，原 eslint-disable 已随迁移失效，故移除）
            Phase 125: virtual={false} —— 动作数扩至 13 后虚拟滚动导致尾部选项不进 DOM,
            13 项全量渲染无性能负担, 且对自动化/可访问性友好 */}
        <AppSelect placeholder="请选择操作类型" options={ACTION_OPTIONS} virtual={false} />
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

      {/* ===== v1.51 Phase 124: trunk/hybrid 动态参数表单 ===== */}
      <Form.Item shouldUpdate noStyle>
        {({ getFieldValue }) => {
          const action = getFieldValue("action");
          if (action === "trunk_set_native") {
            return (
              <Form.Item
                name="nativeVlan"
                label="Native VLAN (PVID)"
                rules={[
                  { required: true, message: "请输入 Native VLAN" },
                  {
                    validator: (_rule: unknown, value: unknown) => {
                      const n = Number(value);
                      if (value === undefined || value === null || value === "") {
                        return Promise.resolve();
                      }
                      if (!Number.isInteger(n) || n < 1 || n > 4094) {
                        return Promise.reject(new Error("VLAN 取值范围 1-4094"));
                      }
                      return Promise.resolve();
                    },
                  },
                ]}
              >
                <AppInput type="number" placeholder="1-4094" />
              </Form.Item>
            );
          }
          if (action === "trunk_set_allowed") {
            return (
              <>
                <Form.Item
                  name="allowedVlansText"
                  label="放行 VLAN 列表"
                  extra={TRUNK_REPLACE_HINT}
                  rules={[
                    { required: true, message: "请输入 VLAN 列表" },
                    {
                      validator: (_rule: unknown, value: unknown) => {
                        if (!value) return Promise.resolve();
                        const parsed = parseVlanList(String(value));
                        if (parsed === null) {
                          return Promise.reject(
                            new Error("格式：逗号/空格分隔的 1-4094 整数（自动去重）")
                          );
                        }
                        return Promise.resolve();
                      },
                    },
                  ]}
                >
                  <AppInput placeholder="如：10,20,30（逗号或空格分隔）" />
                </Form.Item>
                <Form.Item
                  name="trunkMode"
                  label="模式"
                  initialValue="replace"
                  rules={[{ required: true, message: "请选择模式" }]}
                >
                  <AppSelect
                    options={TRUNK_ALLOWED_MODE_OPTIONS.map((o) => ({ ...o }))}
                    showSearch={false}
                  />
                </Form.Item>
              </>
            );
          }
          if (action === "trunk_set_hybrid") {
            return (
              <>
                <Form.Item
                  name="hybridTaggedText"
                  label="Tagged VLAN 列表"
                  extra={TRUNK_HYBRID_OVERLAP_HINT}
                  rules={[
                    {
                      validator: (_rule: unknown, value: unknown) => validateVlanListText(value),
                    },
                  ]}
                >
                  <AppInput placeholder="如：30,40（可留空）" />
                </Form.Item>
                <Form.Item
                  name="hybridUntaggedText"
                  label="Untagged VLAN 列表"
                  rules={[
                    {
                      validator: (_rule: unknown, value: unknown) => validateVlanListText(value),
                    },
                    // 前端预校验: 两组不可重叠（后端 ErrHybridOverlap 400 兜底）
                    {
                      validator: () => {
                        const tagged = parseVlanList(
                          String(form.getFieldValue("hybridTaggedText") ?? "")
                        );
                        const untagged = parseVlanList(
                          String(form.getFieldValue("hybridUntaggedText") ?? "")
                        );
                        if (tagged && untagged && tagged.some((v) => untagged.includes(v))) {
                          return Promise.reject(
                            new Error("tagged 与 untagged VLAN 存在重叠，请修正后重试")
                          );
                        }
                        return Promise.resolve();
                      },
                    },
                  ]}
                >
                  <AppInput placeholder="如：10,20（可留空）" />
                </Form.Item>
              </>
            );
          }
          if (action === "portsec_max_mac") {
            return (
              <Form.Item
                name="maxMacCount"
                label="MAC 学习上限"
                rules={[
                  { required: true, message: "请输入 MAC 学习上限" },
                  {
                    validator: (_rule: unknown, value: unknown) => {
                      const n = Number(value);
                      if (value === undefined || value === null || value === "") {
                        return Promise.resolve();
                      }
                      if (!Number.isInteger(n) || n < 1 || n > 4096) {
                        return Promise.reject(new Error("取值范围 1-4096"));
                      }
                      return Promise.resolve();
                    },
                  },
                ]}
              >
                <AppInput type="number" placeholder="1-4096" />
              </Form.Item>
            );
          }
          if (action === "set_speed") {
            return (
              <>
                <Form.Item
                  name="speedMode"
                  label="速率模式"
                  initialValue={PHYSICAL_SPEED_MODE_AUTO}
                  rules={[{ required: true, message: "请选择速率模式" }]}
                >
                  <AppSelect
                    options={[
                      { label: "自协商 (auto)", value: PHYSICAL_SPEED_MODE_AUTO },
                      { label: "指定速率", value: "manual" },
                    ]}
                    showSearch={false}
                  />
                </Form.Item>
                <Form.Item shouldUpdate noStyle>
                  {({ getFieldValue }) =>
                    getFieldValue("speedMode") !== PHYSICAL_SPEED_MODE_AUTO ? (
                      <Form.Item
                        name="speed"
                        label="速率 (Mbps)"
                        rules={[{ required: true, message: "请选择速率" }]}
                      >
                        <AppSelect
                          options={PHYSICAL_SPEED_OPTIONS.map((o) => ({ ...o }))}
                          showSearch={false}
                        />
                      </Form.Item>
                    ) : null
                  }
                </Form.Item>
              </>
            );
          }
          if (action === "set_duplex") {
            return (
              <Form.Item
                name="duplex"
                label="双工模式"
                rules={[{ required: true, message: "请选择双工模式" }]}
              >
                <AppSelect
                  options={PHYSICAL_DUPLEX_OPTIONS.map((o) => ({ ...o }))}
                  showSearch={false}
                />
              </Form.Item>
            );
          }
          if (action === "set_mtu") {
            return (
              <Form.Item
                name="mtu"
                label={`MTU (${PHYSICAL_MTU_MIN}-${PHYSICAL_MTU_MAX})`}
                extra={PHYSICAL_NO_ROLLBACK_HINT}
                rules={[
                  { required: true, message: "请输入 MTU" },
                  {
                    validator: (_rule: unknown, value: unknown) => {
                      const n = Number(value);
                      if (value === undefined || value === null || value === "") {
                        return Promise.resolve();
                      }
                      if (!Number.isInteger(n) || n < PHYSICAL_MTU_MIN || n > PHYSICAL_MTU_MAX) {
                        return Promise.reject(
                          new Error(`MTU 取值范围 ${PHYSICAL_MTU_MIN}-${PHYSICAL_MTU_MAX}`)
                        );
                      }
                      return Promise.resolve();
                    },
                  },
                ]}
              >
                <AppInput type="number" placeholder={`${PHYSICAL_MTU_MIN}-${PHYSICAL_MTU_MAX}`} />
              </Form.Item>
            );
          }
          if (action === "portsec_static_bind") {
            return (
              <>
                <Form.Item
                  name="staticMacAddress"
                  label="静态 MAC 地址"
                  extra={PORTSEC_MAC_HINT}
                  rules={[
                    { required: true, message: "请输入 MAC 地址" },
                    {
                      validator: (_rule: unknown, value: unknown) => {
                        if (!value) return Promise.resolve();
                        if (parseMacAddress(String(value)) === null) {
                          return Promise.reject(
                            new Error("格式非法：支持 AA-BB-CC-DD-EE-FF / AA:BB:…/ aabb.ccdd.eeff")
                          );
                        }
                        return Promise.resolve();
                      },
                    },
                  ]}
                >
                  <AppInput placeholder="AA-BB-CC-DD-EE-FF" />
                </Form.Item>
                <Form.Item
                  name="staticBindVlanId"
                  label="绑定 VLAN"
                  rules={[
                    { required: true, message: "请输入绑定 VLAN" },
                    {
                      validator: (_rule: unknown, value: unknown) => {
                        const n = Number(value);
                        if (value === undefined || value === null || value === "") {
                          return Promise.resolve();
                        }
                        if (!Number.isInteger(n) || n < 1 || n > 4094) {
                          return Promise.reject(new Error("VLAN 取值范围 1-4094"));
                        }
                        return Promise.resolve();
                      },
                    },
                  ]}
                >
                  <AppInput type="number" placeholder="1-4094" />
                </Form.Item>
              </>
            );
          }
          return null;
        }}
      </Form.Item>

      {/* ===== v1.51 Phase 125: skipPreflight 开关（lead 裁决: 沿用 network:port:write,
          面板可达即持权 —— 菜单 perms 已门控路由, 无需二次 decide/新增权限码） ===== */}
      <Form.Item
        name="skipPreflight"
        valuePropName="checked"
        label="跳过在线预检"
        extra={SKIP_PREFLIGHT_HINT}
        style={{ marginBottom: 8 }}
      >
        <Switch />
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

      {/* 整体风险等级 + Phase 143 取证级别徽标 */}
      <Space style={{ marginBottom: 16 }}>
        <span>整体风险等级：</span>
        <Tag color={RISK_TAG_COLOR[preview.riskLevel]}>{RISK_LABEL[preview.riskLevel]}</Tag>
        <Typography.Text type="secondary">
          {preview.vendor} {preview.model} · {ACTION_TITLE[preview.action]}
        </Typography.Text>
        {/* 默认放行 ≠ 有证据：unverified 诚实标注「未取证」，不冒充真机结论 */}
        {preview.evidenceLevel && (
          <Tag color={preview.evidenceLevel === "unverified" ? "default" : "blue"}>
            取证级别：{PHYSICAL_EVIDENCE_LABEL[preview.evidenceLevel] ?? preview.evidenceLevel}
          </Tag>
        )}
      </Space>
      {preview.evidenceLevel === "unverified" && (
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          title="未取证（默认放行）"
          description="该物理参数命令形态按厂商/全局默认画像渲染，尚未针对该型号真机逐一取证；默认放行不代表已验证。批量物理参数无自动回滚，失败需人工按设备 running-config / 采集快照与 audit.command_sent 恢复（audit.before_value 仅含端口状态，不含物理原值）。"
        />
      )}

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
