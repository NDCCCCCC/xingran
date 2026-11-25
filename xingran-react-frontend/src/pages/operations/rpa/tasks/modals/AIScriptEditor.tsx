/**
 * AI 脚本编辑器组件
 * 使用 AI 从自然语言描述生成 RPA 脚本
 */

import { useState, useCallback, useEffect, useRef } from "react";
import {
  App,
  Modal,
  Form,
  Button,
  Steps,
  Card,
  Space,
  Alert,
  Spin,
  Tag,
  Divider,
  Typography,
} from "antd";
import {
  RobotOutlined,
  ThunderboltOutlined,
  CheckOutlined,
  LoadingOutlined,
} from "@ant-design/icons";
import type { Action, ActionType } from "@/types/rpa";
import { ACTION_TYPE_OPTIONS, ACTION_TYPE_TEXT_MAP } from "../../constants";
import { api } from "@/lib/api";
// Phase 71 (W3)：裸 TextArea → AppTextArea。
// 任务描述是受控 Form.Item（disabled 依赖生成状态），显式 allowClear={false}
// 保持原有「无清空按钮」交互（清空会绕过 disabled 的生成态约束）。
import { AppTextArea } from "@/components/ui";

const { Text } = Typography;

export interface AIScriptEditorProps {
  open: boolean;
  onClose: () => void;
  onConfirm: (actions: Action[]) => void;
  /**
   * NOCODE-03：AI 不可用时降级到「录制」。
   *
   * 不传时按钮仍会显示，但点击只给出「请从新建向导选择录制」的提示 ——
   * 保证组件在任何挂载点都有可操作出路（绝不出现点了没反应的死按钮）。
   */
  onFallbackToRecord?: () => void;
}

// CONTRACT-05（D15）：这里原本有一份 mockGeneratedActions 常量 + setTimeout(1500)
// 假延迟 —— 后端 11 个 AI 端点因此「零调用」，用户看到的步骤永远是写死的 5 条演示。
// 现已删除，改为实调 POST /rpa/ai/generate（见下方 handleGenerate 与 mapAIActions）。

/** 后端 POST /rpa/ai/generate 返回的单个动作（models/rpa ScriptAction 形状）。 */
interface AIScriptActionDTO {
  type?: string;
  selector?: string;
  value?: string;
  attributes?: Record<string, unknown>;
  timeout?: number;
  retry?: number;
}

interface AIGenerateResponseDTO {
  actions?: AIScriptActionDTO[];
  explanation?: string;
  confidence?: number;
}

// AI_GENERATE_TIMEOUT_MS 前端超时预算。后端 AIClient 自身超时是 30s，
// 前端留出余量到 35s；超时后必须给出「可重试」的明确态，而不是无限转圈。
const AI_GENERATE_TIMEOUT_MS = 35000;

/** 已知动作类型白名单（与 constants.tsx ACTION_TYPE_OPTIONS 同源）。 */
const KNOWN_ACTION_TYPES: readonly string[] = ACTION_TYPE_OPTIONS.map((o) => o.value);

/**
 * mapAIActions 把后端 ScriptAction DTO 映射成前端 Action。
 *
 * 契约要点（NOCODE-03 的校验在服务端做；前端只做防御性丢弃）：
 *   - 未知 action.type 直接丢弃（不让非法动作进入编辑态）
 *   - selector 文本 → {type:"css"}（后端 ScriptAction 只有字符串选择器）
 *   - attributes.description → description
 */
export function mapAIActions(dtos: AIScriptActionDTO[] | undefined): Action[] {
  if (!dtos || dtos.length === 0) return [];
  return dtos
    .filter((d) => d && typeof d.type === "string" && KNOWN_ACTION_TYPES.includes(d.type))
    .map((d, index) => ({
      id: `ai_action_${index + 1}`,
      type: d.type as ActionType,
      description: (d.attributes?.description as string) || ACTION_TYPE_TEXT_MAP[d.type as string] || d.type,
      selector: { type: "css", value: d.selector || "" },
      params: { value: d.value ?? "" },
      timeout: d.timeout ?? 30000,
      retry: d.retry ?? 0,
    }));
}

/** AI 生成失败/超时/空结果/不可用四态。 */
type GenerateErrorKind = "failed" | "timeout" | "empty" | "unavailable";

/**
 * isAIDisabledResponse 判定后端是否在报「智能生成不可用」。
 *
 * 后端 NOCODE-03 的三态契约（ai_handler.go GenerateScript）：
 *   - 400 + 含「不可用」→ AI 未启用（应引导录制，不是重试）
 *   - 400 + 含「第 N 步」→ 非法动作（应改描述重试）
 *   - 500 → 超时/网络（应稍后重试）
 *
 * 这里优先看 HTTP 状态 + message 关键词；两者都拿不到时保守判为非不可用
 * （宁可让用户重试一次，也不要误把可重试的故障说成「功能没开」）。
 */
export function isAIDisabledResponse(error: unknown): boolean {
  const err = error as
    | {
        response?: { status?: number; data?: { message?: string } };
        status?: number;
        message?: string;
      }
    | undefined;
  if (!err) return false;

  const status = err.response?.status ?? err.status;
  const message =
    err.response?.data?.message ??
    (typeof err.message === "string" ? err.message : "");

  // 后端约定：AI 未启用返回 400 且文案明确说「不可用」
  if (status === 400 && /不可用/.test(message)) return true;
  // 兜底：即使拿不到状态码，明确的「不可用」文案也足以判定
  return /智能生成当前不可用|AI 脚本生成未启用/.test(message);
}

export function AIScriptEditor({
  open,
  onClose,
  onConfirm,
  onFallbackToRecord,
}: AIScriptEditorProps) {
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [generatedActions, setGeneratedActions] = useState<Action[]>([]);
  // CONTRACT-05：失败/超时/空结果必须有可见、可重试的态，而不是只有一条 toast。
  const [generateError, setGenerateError] = useState<{
    kind: GenerateErrorKind;
    text: string;
  } | null>(null);
  const [explanation, setExplanation] = useState<string>("");
  const abortRef = useRef<AbortController | null>(null);

  const handleGenerate = useCallback(async () => {
    const description = form.getFieldValue("description");
    if (!description?.trim()) {
      message.warning("请输入任务描述");
      return;
    }

    setLoading(true);
    setGenerateError(null);
    setGeneratedActions([]);
    setExplanation("");

    // 前端侧超时预算：后端 AIClient 自己也会超时（30s），这里兜住
    // 「请求挂住 / 代理不回包」的场景，保证 UI 不会无限转圈。
    const controller = new AbortController();
    abortRef.current = controller;
    const timeoutId = setTimeout(() => controller.abort(), AI_GENERATE_TIMEOUT_MS);

    try {
      // CONTRACT-05：实调后端 POST /rpa/ai/generate（不再 setTimeout + mock）。
      // 用 api 实例（而非 post 封装）是因为需要透传 AbortSignal 以便超时中断。
      const result = await api.post<AIGenerateResponseDTO>(
        "/rpa/ai/generate",
        { description },
        { signal: controller.signal }
      );

      const mapped = mapAIActions(result.data?.actions);
      if (mapped.length === 0) {
        // 空结果态：后端 200 但没有任何可执行动作（或全部被白名单过滤掉）
        setGenerateError({
          kind: "empty",
          text: "AI 未返回可执行的动作步骤，请补充更具体的任务描述后重试。",
        });
        message.warning("AI 未返回可执行动作");
        return;
      }

      setGeneratedActions(mapped);
      setExplanation(result.data?.explanation ?? "");
      message.success("AI 生成脚本成功！");
    } catch (error) {
      const aborted =
        controller.signal.aborted ||
        (error as { name?: string } | undefined)?.name === "AbortError" ||
        (error as { code?: string } | undefined)?.code === "ERR_CANCELED";
      if (aborted) {
        setGenerateError({
          kind: "timeout",
          text: `AI 生成超时（超过 ${Math.round(AI_GENERATE_TIMEOUT_MS / 1000)} 秒）。可以稍后重试，也可以直接改用「录制一段操作」——录制不依赖智能生成，随时可用。`,
        });
        message.error("AI 生成超时，请重试");
      } else if (isAIDisabledResponse(error)) {
        // NOCODE-03 关键路径：AI 不可用**不是死路**，必须当场给出录制入口。
        setGenerateError({
          kind: "unavailable",
          text: "智能生成当前未开启。不用等管理员：直接用「录制一段操作」把你的操作记下来，一样能生成任务。",
        });
        message.warning("智能生成未开启，可改用录制");
      } else {
        const detail =
          (error as { message?: string } | undefined)?.message ||
          (error as { response?: { data?: { message?: string } } })?.response?.data?.message ||
          "未知错误";
        setGenerateError({ kind: "failed", text: `AI 生成失败：${detail}` });
        message.error("AI 生成失败，请重试");
      }
    } finally {
      clearTimeout(timeoutId);
      abortRef.current = null;
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [form]);

  // 取消/关闭时中断在途请求，避免组件卸载后 setState。
  useEffect(() => {
    if (!open) {
      abortRef.current?.abort();
      abortRef.current = null;
      setGenerateError(null);
      setExplanation("");
    }
  }, [open]);

  const handleConfirm = useCallback(() => {
    if (generatedActions.length === 0) {
      message.warning("请先生成脚本");
      return;
    }
    onConfirm(generatedActions);
    form.resetFields();
    setGeneratedActions([]);
    setGenerateError(null);
    setExplanation("");
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [generatedActions, onConfirm, form]);

  const handleCancel = useCallback(() => {
    abortRef.current?.abort();
    abortRef.current = null;
    form.resetFields();
    setGeneratedActions([]);
    setGenerateError(null);
    setExplanation("");
    onClose();
  }, [form, onClose]);

  /**
   * handleFallbackToRecord NOCODE-03 降级入口。
   *
   * 语义：AI 走不通时，把用户**已经写好的中文描述**带进录制流程（不丢弃用户
   * 输入），然后关闭本弹窗。这样用户的努力不会白费，也不会被困在「AI 报错 →
   * 重试 → 再报错」的循环里。
   */
  const handleFallbackToRecord = useCallback(() => {
    abortRef.current?.abort();
    abortRef.current = null;
    const description = String(form.getFieldValue("description") ?? "").trim();
    form.resetFields();
    setGeneratedActions([]);
    setGenerateError(null);
    setExplanation("");
    if (onFallbackToRecord) {
      onFallbackToRecord();
      return;
    }
    // 无回调时的兜底：明确告知下一步该去哪，而不是静默关闭。
    message.info(
      description
        ? "已记下你的描述。请从「新增任务」向导选择「录制一段操作」，把这段流程录下来。"
        : "请从「新增任务」向导选择「录制一段操作」来创建任务。"
    );
    onClose();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, [form, onClose, onFallbackToRecord]);

  return (
    <Modal
      title={
        <Space>
          <RobotOutlined />
          <span>AI 脚本生成器</span>
        </Space>
      }
      open={open}
      onCancel={handleCancel}
      width={800}
      footer={
        <Space>
          <Button onClick={handleCancel} disabled={loading}>
            取消
          </Button>
          <Button
            type="primary"
            icon={<ThunderboltOutlined />}
            loading={loading}
            onClick={handleGenerate}
            disabled={generatedActions.length > 0}
          >
            {loading ? "生成中..." : "生成脚本"}
          </Button>
          <Button
            type="primary"
            icon={<CheckOutlined />}
            onClick={handleConfirm}
            disabled={generatedActions.length === 0}
          >
            确认使用
          </Button>
        </Space>
      }
    >
      <div style={{ padding: "16px 0" }}>
        <Alert
          title="使用自然语言描述您的自动化任务，AI 将为您生成相应的脚本步骤"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />

        <Form form={form} layout="vertical">
          <Form.Item
            name="description"
            label="任务描述"
            rules={[{ required: true, message: "请描述您的自动化任务" }]}
          >
            <AppTextArea
              rows={4}
              allowClear={false}
              placeholder="例如：打开登录页面，输入用户名和密码，点击登录按钮，验证登录成功"
              disabled={loading || generatedActions.length > 0}
            />
          </Form.Item>
        </Form>

        {generatedActions.length > 0 && (
          <>
            <Divider>生成的脚本步骤</Divider>
            <Steps
              direction="vertical"
              current={-1}
              items={generatedActions.map((action, index) => ({
                title: (
                  <Space>
                    <Tag color="blue">{ACTION_TYPE_TEXT_MAP[action.type] || action.type}</Tag>
                    <Text strong>{action.description}</Text>
                  </Space>
                ),
                description: (
                  <Card size="small" style={{ marginTop: 8 }}>
                    <Space orientation="vertical" style={{ width: "100%" }}>
                      {action.selector?.value && (
                        <Text code style={{ fontSize: 12 }}>
                          选择器: {action.selector.value}
                        </Text>
                      )}
                      {action.params?.value !== undefined && action.params?.value !== null && (
                        <Text code style={{ fontSize: 12 }}>
                          值: {String(action.params.value)}
                        </Text>
                      )}
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        超时: {action.timeout}ms
                      </Text>
                    </Space>
                  </Card>
                ),
                icon: index < generatedActions.length ? <CheckOutlined /> : undefined,
              }))}
            />

            <Alert
              title="请检查生成的脚本步骤，确认无误后点击「确认使用」按钮"
              type="success"
              showIcon
              style={{ marginTop: 16 }}
            />
          </>
        )}

        {loading && (
          <div style={{ textAlign: "center", padding: "40px 0" }}>
            <Spin
              indicator={<LoadingOutlined style={{ fontSize: 48 }} spin />}
              tip="AI 正在生成脚本，请稍候..."
            />
          </div>
        )}

        {/* NOCODE-03：四态的可见提示。
            三态（失败/超时/空结果）都给「重试生成」；「不可用」额外给
            「改用录制」——这是关键体验路径，不能只给一个报错就完事。 */}
        {!loading && generateError && (
          <Alert
            title={
              generateError.kind === "timeout"
                ? "AI 生成超时"
                : generateError.kind === "empty"
                  ? "AI 未返回可执行动作"
                  : generateError.kind === "unavailable"
                    ? "智能生成未开启"
                    : "AI 生成失败"
            }
            description={
              <Space orientation="vertical" style={{ width: "100%" }}>
                <Text type={generateError.kind === "unavailable" ? undefined : "danger"}>
                  {generateError.text}
                </Text>
                <Space>
                  {generateError.kind !== "unavailable" && (
                    <Button size="small" onClick={handleGenerate} icon={<ThunderboltOutlined />}>
                      重试生成
                    </Button>
                  )}
                  {/* 不可用时把「录制」做成主按钮：这是唯一能继续的路 */}
                  <Button
                    size="small"
                    type={generateError.kind === "unavailable" ? "primary" : "default"}
                    onClick={handleFallbackToRecord}
                    data-testid="ai-fallback-record"
                  >
                    改用录制
                  </Button>
                </Space>
              </Space>
            }
            type={generateError.kind === "empty" ? "warning" : generateError.kind === "unavailable" ? "info" : "error"}
            showIcon
            style={{ marginTop: 16 }}
            data-testid="ai-generate-error"
          />
        )}

        {!loading && !generateError && explanation && (
          <Alert
            title="AI 说明"
            description={explanation}
            type="info"
            showIcon
            style={{ marginTop: 16 }}
            data-testid="ai-generate-explanation"
          />
        )}
      </div>
    </Modal>
  );
}
