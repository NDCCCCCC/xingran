/**
 * 同步漂移告警 — 配置表单（Phase 144 ALRT-03）
 *
 * 9 个 sys_config 旋钮：
 *   1 enabled            总开关
 *   2 tolerance          容差
 *   3 throttleMinutes    节流分钟
 *   4 noticeEnabled      站内通知
 *   5 wsEnabled          WS 实时广播
 *   6 webhookEnabled     webhook 推送
 *   7 webhookConfigId    webhook 配置（下拉选已有 configType=webhook 配置）
 *   8 webhookTemplate    webhook 模板
 *   9 baseline           基线（只读回显，不参与提交）
 *
 * 约束：
 *   - webhook 下拉走 getAPINotificationConfigList({configType:'webhook'})，只选已有配置，
 *     不新建配置 CRUD 页；**不展示明文 secret、不在前端日志打印 apiUrl / URL**。
 *   - 保存走 updateConfig（wrapped post）+「:edit」权限门控（三态 useActionPermission）。
 *   - 错误处理统一走 handleApiError（唯一错误出口），不手写 message.error。
 *   - useEffect 依赖用 primitive / 稳定引用，禁止对象字面量进依赖（防无限循环）。
 */
import { useCallback, useEffect, useMemo, useState } from "react";
import type { FC } from "react";
import { Alert, Button, Card, Form, InputNumber, Space, Switch, Tooltip } from "antd";
import { InfoCircleOutlined, SaveOutlined } from "@ant-design/icons";
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import { useActionPermission } from "@/components/shared/useActionPermission";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import { updateConfig, type ADDriftAlertConfig } from "@/lib/adDriftAlertApi";
import { getAPINotificationConfigList } from "@/lib/notificationConfigApi";

export interface ConfigFormProps {
  /** 当前配置（父组件 getConfig() 注入，引用来自 useState，稳定） */
  config: ADDriftAlertConfig | null;
  loading?: boolean;
  /** 保存成功回调：父组件据此刷新状态卡 */
  onSaved?: (next: ADDriftAlertConfig) => void;
}

interface WebhookOption {
  id: string;
  configName: string;
}

interface ConfigFormValues {
  enabled: boolean;
  tolerance: number;
  throttleMinutes: number;
  noticeEnabled: boolean;
  wsEnabled: boolean;
  webhookEnabled: boolean;
  webhookConfigId: string;
  webhookTemplate: string;
}

const ConfigForm: FC<ConfigFormProps> = ({ config, loading = false, onSaved }) => {
  const [form] = Form.useForm<ConfigFormValues>();
  const [saving, setSaving] = useState(false);
  const [webhookOptions, setWebhookOptions] = useState<WebhookOption[]>([]);
  const [webhookLoadError, setWebhookLoadError] = useState<unknown>(null);

  // 操作级权限门控（三态：未就绪 → 渲染但 disabled + 提示）
  const gate = useActionPermission();
  const editDecision = gate.decide(PERMISSION_CODES.adDriftAlert.edit);

  // 加载 webhook 配置下拉选项。
  // 空依赖 = mount 时一次；不把对象字面量放进依赖。
  // 只取 configName / id 两个非敏感字段，绝不读取 / 打印 apiUrl、headers、authConfig
  // （避免明文 secret 落前端日志或进内存展示）。
  const loadWebhookOptions = useCallback(async () => {
    try {
      const result = await getAPINotificationConfigList({ configType: "webhook" });
      const raw = result.data as { list?: Array<{ id?: string; configName?: string }> } | undefined;
      const list = Array.isArray(raw?.list) ? raw.list : [];
      setWebhookOptions(
        list.map((item) => ({
          id: String(item.id ?? ""),
          configName: String(item.configName ?? item.id ?? ""),
        }))
      );
      setWebhookLoadError(null);
    } catch (error) {
      // 唯一错误出口：handleApiError（不手写 message.error，不打印 URL）
      setWebhookLoadError(error);
      handleApiError(error, "加载 Webhook 配置", false);
      setWebhookOptions([]);
    }
  }, []);

  useEffect(() => {
    void loadWebhookOptions();
  }, [loadWebhookOptions]);

  // config prop 变化时回填表单。config 来自父组件 useState（引用稳定，非每次 render
  // 新建的对象字面量）；用 primitive 派生字段做依赖，彻底避开对象依赖陷阱。
  const enabled = config?.enabled;
  const tolerance = config?.tolerance;
  const throttleMinutes = config?.throttleMinutes;
  const noticeEnabled = config?.noticeEnabled;
  const wsEnabled = config?.wsEnabled;
  const webhookEnabled = config?.webhookEnabled;
  const webhookConfigId = config?.webhookConfigId;
  const webhookTemplate = config?.webhookTemplate;
  useEffect(() => {
    if (config) {
      form.setFieldsValue({
        enabled: Boolean(enabled),
        tolerance: Number(tolerance ?? 0),
        throttleMinutes: Number(throttleMinutes ?? 0),
        noticeEnabled: Boolean(noticeEnabled),
        wsEnabled: Boolean(wsEnabled),
        webhookEnabled: Boolean(webhookEnabled),
        webhookConfigId: webhookConfigId ?? "",
        webhookTemplate: webhookTemplate ?? "",
      });
    }
  }, [
    config,
    form,
    enabled,
    tolerance,
    throttleMinutes,
    noticeEnabled,
    wsEnabled,
    webhookEnabled,
    webhookConfigId,
    webhookTemplate,
  ]);

  const selectOptions = useMemo(
    () => [
      { label: "使用默认配置", value: "" },
      ...webhookOptions.map((o) => ({ label: o.configName, value: o.id })),
    ],
    [webhookOptions]
  );

  const handleSubmit = useCallback(
    async (values: ConfigFormValues) => {
      setSaving(true);
      try {
        // baseline 只读展示，不在表单 values 里；提交时从 config 原样带回
        const payload: ADDriftAlertConfig = {
          enabled: values.enabled,
          tolerance: values.tolerance,
          throttleMinutes: values.throttleMinutes,
          noticeEnabled: values.noticeEnabled,
          wsEnabled: values.wsEnabled,
          webhookEnabled: values.webhookEnabled,
          webhookConfigId: values.webhookConfigId ?? "",
          webhookTemplate: values.webhookTemplate ?? "",
          baseline: config?.baseline ?? "",
        };
        const next = await updateConfig(payload);
        handleSuccess("保存配置");
        onSaved?.(next ?? payload);
      } catch (error) {
        // 唯一错误出口
        handleApiError(error, "保存配置");
      } finally {
        setSaving(false);
      }
    },
    [config, onSaved]
  );

  const saveDisabled = editDecision.disabled || saving || loading;

  return (
    <Card title="告警配置" loading={loading} size="small">
      {webhookLoadError !== null && (
        <Alert
          type="warning"
          showIcon
          title="Webhook 配置列表加载失败，可手动填写配置 ID 或稍后重试"
          style={{ marginBottom: 12 }}
        />
      )}
      <Form<ConfigFormValues>
        form={form}
        layout="vertical"
        onFinish={handleSubmit}
        disabled={editDecision.disabled}
        initialValues={{
          enabled: true,
          tolerance: 0,
          throttleMinutes: 60,
          noticeEnabled: true,
          wsEnabled: true,
          webhookEnabled: false,
          webhookConfigId: "",
          webhookTemplate: "",
        }}
      >
        <Space orientation="vertical" size="medium" style={{ width: "100%" }}>
          <Form.Item
            name="enabled"
            label="启用告警（总开关）"
            valuePropName="checked"
            extra="关闭后定时任务不再产生告警"
          >
            <Switch />
          </Form.Item>

          <Space size="large" wrap>
            <Form.Item name="tolerance" label="容差（计数变化阈值）">
              <InputNumber min={0} max={100000} precision={0} />
            </Form.Item>
            <Form.Item name="throttleMinutes" label="节流窗口（分钟）">
              <InputNumber min={0} max={10080} precision={0} addonAfter="min" />
            </Form.Item>
          </Space>

          <Space size="large" wrap>
            <Form.Item name="noticeEnabled" label="站内通知（铃铛）" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="wsEnabled" label="WS 实时推送" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="webhookEnabled" label="Webhook 推送" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Space>

          <Form.Item
            name="webhookConfigId"
            label={
              <span>
                Webhook 配置&nbsp;
                <Tooltip title="仅选择已存在的 Webhook 类型通知配置；不在此展示或打印密钥/URL">
                  <InfoCircleOutlined />
                </Tooltip>
              </span>
            }
          >
            <AppSelect options={selectOptions} placeholder="使用默认配置" allowClear={false} />
          </Form.Item>

          <Form.Item name="webhookTemplate" label="Webhook 模板（可选）">
            <AppTextArea
              rows={2}
              placeholder="自定义推送模板；留空使用后端默认模板（不填明文密钥）"
            />
          </Form.Item>

          <Form.Item label="当前基线（只读）">
            <AppInput readOnly value={config?.baseline ?? "-"} aria-label="当前基线" />
          </Form.Item>

          {editDecision.visible && (
            <Form.Item>
              <Button
                type="primary"
                htmlType="submit"
                icon={<SaveOutlined />}
                loading={saving}
                disabled={saveDisabled}
                aria-label="保存告警配置"
                title={editDecision.hint ?? "保存告警配置"}
              >
                保存配置
              </Button>
              {editDecision.disabled && editDecision.hint && (
                <span
                  style={{
                    marginInlineStart: 8,
                    fontSize: 12,
                    color: "var(--theme-text-tertiary)",
                  }}
                >
                  {editDecision.hint}
                </span>
              )}
            </Form.Item>
          )}
        </Space>
      </Form>
    </Card>
  );
};

export default ConfigForm;
