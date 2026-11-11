/**
 * 备份计划面板：启用开关 / CronSelector 周期 / 保留份数 / 存储后端 + 条件配置字段。
 *
 * 敏感字段（oss.accessKey / oss.secretKey / ftp.password）约定：
 * - GET /plan 回填时，若后端已存值则展示为 "******"（脱敏占位）；
 * - 提交时若值仍为 "******"，改写为 ""（空串）；与后端约定 "" 表示「保留原值、不覆盖」
 *   （后端 SavePlan 亦做保留）。绝不把占位串写回配置，也不误清空真实凭据。
 */
import { useCallback, useEffect, useState, type FC } from "react";
import {
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Space,
  Switch,
} from "antd";
import CronSelector from "@/components/CronSelector";
// Input.Password 为成员表达式形态，不在 Phase 65 <Input> selector 命中面（沿用 settings 页范式）
import { AppInput, AppSelect } from "@/components/ui";
import { useActionPermission } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { dbBackupApi } from "@/lib/dbBackupApi";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import type { Plan, StorageType } from "@/types/backup";
import {
  DEFAULT_PLAN,
  SENSITIVE_KEEP_ORIGINAL,
  SENSITIVE_PLACEHOLDER,
  STORAGE_OPTIONS,
} from "../types";

/** 表单值形状：计划嵌套字段展开为扁平 name，便于 Form.Item 条件渲染 */
interface PlanFormValues {
  enabled: boolean;
  cron: string;
  retention: number;
  storage: StorageType;
  ossEndpoint: string;
  ossBucket: string;
  ossAccessKey: string;
  ossSecretKey: string;
  ossUseSSL: boolean;
  ftpHost: string;
  ftpPort: number;
  ftpUser: string;
  ftpPassword: string;
  ftpRemoteDir: string;
  smbPath: string;
}

/** Plan → 表单值：敏感字段有值则回填脱敏占位符 */
function toFormValues(plan: Plan): PlanFormValues {
  const masked = (value: string) => (value ? SENSITIVE_PLACEHOLDER : "");
  return {
    enabled: plan.enabled,
    cron: plan.cron || DEFAULT_PLAN.cron,
    retention: plan.retention ?? DEFAULT_PLAN.retention,
    storage: plan.storage ?? "local",
    ossEndpoint: plan.oss?.endpoint ?? "",
    ossBucket: plan.oss?.bucket ?? "",
    ossAccessKey: masked(plan.oss?.accessKey ?? ""),
    ossSecretKey: masked(plan.oss?.secretKey ?? ""),
    ossUseSSL: plan.oss?.useSSL ?? true,
    ftpHost: plan.ftp?.host ?? "",
    ftpPort: plan.ftp?.port ?? 21,
    ftpUser: plan.ftp?.user ?? "",
    ftpPassword: masked(plan.ftp?.password ?? ""),
    ftpRemoteDir: plan.ftp?.remoteDir ?? "/",
    smbPath: plan.smb?.path ?? "",
  };
}

/** 表单值 → Plan：敏感字段若为脱敏占位符，则发 "" 表示保留原值 */
function toPlanPayload(values: PlanFormValues): Plan {
  const keepOrValue = (value: string): string =>
    value === SENSITIVE_PLACEHOLDER ? SENSITIVE_KEEP_ORIGINAL : value;
  return {
    enabled: values.enabled,
    cron: values.cron,
    retention: values.retention,
    storage: values.storage,
    oss: {
      endpoint: values.ossEndpoint,
      bucket: values.ossBucket,
      accessKey: keepOrValue(values.ossAccessKey),
      secretKey: keepOrValue(values.ossSecretKey),
      useSSL: values.ossUseSSL,
    },
    ftp: {
      host: values.ftpHost,
      port: values.ftpPort,
      user: values.ftpUser,
      password: keepOrValue(values.ftpPassword),
      remoteDir: values.ftpRemoteDir,
    },
    smb: {
      path: values.smbPath,
    },
  };
}

const BackupPlanTab: FC = () => {
  const [form] = Form.useForm<PlanFormValues>();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const storage = Form.useWatch("storage", form) as StorageType | undefined;

  // 保存按钮门控（三态：ready / loading / error）—— 不自造 hasPermission
  const permGate = useActionPermission();
  const permSave = permGate.decide(PERMISSION_CODES.systemBackup.export);

  /** 读取计划并回填表单（敏感字段脱敏展示） */
  const loadPlan = useCallback(async () => {
    setLoading(true);
    try {
      const plan = await dbBackupApi.getPlan();
      form.setFieldsValue(toFormValues(plan));
    } catch (error) {
      handleApiError(error, "加载备份计划");
      form.setFieldsValue(toFormValues(DEFAULT_PLAN));
    } finally {
      setLoading(false);
    }
  }, [form]);

  useEffect(() => {
    void loadPlan();
  }, [loadPlan]);

  /** 保存：validateFields → 组装 Plan（敏感占位转 ""）→ POST */
  const handleSave = useCallback(async () => {
    let values: PlanFormValues;
    try {
      values = await form.validateFields();
    } catch {
      return; // 表单校验错误由 antd 自行展示
    }
    setSaving(true);
    try {
      await dbBackupApi.savePlan(toPlanPayload(values));
      handleSuccess("保存备份计划", true);
      // 重新读取，敏感字段再次以脱敏占位回填（避免界面残留明文）
      await loadPlan();
    } catch (error) {
      handleApiError(error, "保存备份计划");
    } finally {
      setSaving(false);
    }
  }, [form, loadPlan]);

  return (
    <Card loading={loading}>
      <Form<PlanFormValues>
        form={form}
        layout="horizontal"
        labelCol={{ span: 5 }}
        wrapperCol={{ span: 15 }}
        initialValues={toFormValues(DEFAULT_PLAN)}
        style={{ maxWidth: 760 }}
      >
        <Form.Item name="enabled" label="启用定期备份" valuePropName="checked">
          <Switch checkedChildren="启用" unCheckedChildren="停用" />
        </Form.Item>

        {/* 6 字段含秒、Quartz 风格 '?' 兼容 robfig/cron/v3 */}
        <Form.Item
          name="cron"
          label="执行周期"
          rules={[{ required: true, message: "请配置 Cron 表达式" }]}
          extra="6 字段（秒 分 时 日 月 周），时区 Asia/Shanghai；日/周字段可用 ? 兼容 Quartz 写法"
        >
          <CronSelector />
        </Form.Item>

        <Form.Item
          name="retention"
          label="保留份数"
          rules={[{ required: true, message: "请输入保留份数" }]}
          extra="超出份数的历史备份将在下次备份后自动清理"
        >
          <InputNumber min={1} max={365} precision={0} style={{ width: 160 }} />
        </Form.Item>

        <Form.Item name="storage" label="存储后端" rules={[{ required: true }]}>
          <AppSelect
            showSearch={false}
            options={STORAGE_OPTIONS.map((item) => ({
              value: item.value,
              label: item.label,
            }))}
          />
        </Form.Item>

        {storage === "oss" && (
          <>
            <Form.Item name="ossEndpoint" label="OSS Endpoint">
              <AppInput placeholder="如 oss-cn-hangzhou.aliyuncs.com" />
            </Form.Item>
            <Form.Item name="ossBucket" label="OSS Bucket">
              <AppInput placeholder="桶名" />
            </Form.Item>
            <Form.Item
              name="ossAccessKey"
              label="AccessKey"
              extra="已保存的值显示为 ******；保持不动则提交空串，后端保留原值"
            >
              <AppInput placeholder="AccessKey" />
            </Form.Item>
            <Form.Item name="ossSecretKey" label="SecretKey">
              <Input.Password placeholder="SecretKey" />
            </Form.Item>
            <Form.Item name="ossUseSSL" label="使用 SSL" valuePropName="checked">
              <Switch checkedChildren="是" unCheckedChildren="否" />
            </Form.Item>
          </>
        )}

        {storage === "ftp" && (
          <>
            <Form.Item name="ftpHost" label="FTP 主机">
              <AppInput placeholder="IP 或域名" />
            </Form.Item>
            <Form.Item name="ftpPort" label="FTP 端口">
              <InputNumber min={1} max={65535} precision={0} style={{ width: 160 }} />
            </Form.Item>
            <Form.Item name="ftpUser" label="FTP 用户名">
              <AppInput placeholder="用户名" />
            </Form.Item>
            <Form.Item
              name="ftpPassword"
              label="FTP 密码"
              extra="已保存的值显示为 ******；保持不动则提交空串，后端保留原值"
            >
              <Input.Password placeholder="密码" />
            </Form.Item>
            <Form.Item name="ftpRemoteDir" label="远端目录">
              <AppInput placeholder="如 /backup" />
            </Form.Item>
          </>
        )}

        {storage === "smb" && (
          <Form.Item
            name="smbPath"
            label="SMB 本地目录"
            extra="需操作系统已挂载共享，填写挂载后的本地目录绝对路径"
          >
            <AppInput placeholder="如 /mnt/backup 或 D:\\backup" />
          </Form.Item>
        )}

        <Form.Item wrapperCol={{ offset: 5, span: 15 }}>
          <Space>
            {permSave.visible && (
              <Button
                type="primary"
                loading={saving}
                disabled={permSave.disabled}
                title={permSave.hint}
                onClick={() => void handleSave()}
              >
                保存
              </Button>
            )}
            <Button onClick={() => void loadPlan()}>重新读取</Button>
          </Space>
          {!permSave.visible && (
            <div style={{ marginTop: 8 }}>当前账号无「保存备份计划」权限</div>
          )}
        </Form.Item>
      </Form>
    </Card>
  );
};

export default BackupPlanTab;
