/**
 * 站点品牌设置（系统设置-站点品牌）
 *
 * 消费 /system/branding 端点族（brand.* sys_config 命名空间）：
 *   - 表单：站点名称 / Logo 方块字 / 登录页宣传语 / 版权信息 / 登录副标题
 *   - Logo 上传：≤2MB 图片，POST /system/branding/logo（multipart），
 *     成功后 brandStore.refresh() 让登录页/侧边栏/标题立即生效
 *
 * 权限：读挂 system:config:list（与系统设置菜单一致），写 strict system:config:edit，
 * 按钮显隐走 useMenuStore.permissions（对齐 MACHistoryPage 惯例）。
 */
import { useEffect, useState, type FC } from "react";
import { App, Button, Card, Form, Space, Upload, Alert } from "antd";
import { UploadOutlined } from "@ant-design/icons";
import { AppInput, AppTextArea } from "@/components/ui";
import { xingranBrand } from "@/design-system/tokens/colors";
import { brandingApi, type BrandingConfig } from "@/lib/brandingApi";
import { queryKeys } from "@/lib/queryKeys";
import { useBrandStore } from "@/store/brandStore";
import { useMenuStore } from "@/store/menuStore";
import { useQuery, useQueryClient } from "@tanstack/react-query";

const FORM_ITEMS: Array<{
  name: string;
  label: string;
  extra?: string;
  area?: boolean;
}> = [
  {
    name: "brand.site_name",
    label: "站点名称",
    extra: "显示在侧边栏字标、登录页与浏览器标签页标题",
  },
  { name: "brand.site_mark", label: "Logo 方块字", extra: "未上传 Logo 图片时显示的单个汉字" },
  { name: "brand.site_subtitle", label: "登录副标题", extra: "登录表单标题下方的副标题" },
  {
    name: "brand.login_tagline",
    label: "登录页宣传语",
    extra: "登录页左侧品牌面板的宣传语",
    area: true,
  },
  { name: "brand.copyright", label: "版权信息", extra: "登录页页脚版权行" },
];

const BrandSettings: FC = () => {
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const queryClient = useQueryClient();
  const refreshBrand = useBrandStore((s) => s.refresh);
  const permissions = useMenuStore((s) => s.permissions);
  const canEdit = permissions.includes("system:config:edit");
  const [uploading, setUploading] = useState(false);

  // 管理端读取（与公开端点分离：公开端点有 5 分钟缓存，管理页要看最新值）
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.branding.admin(),
    queryFn: async () => (await brandingApi.get()).data,
  });

  useEffect(() => {
    if (data) {
      form.setFieldsValue({
        "brand.site_name": data.siteName,
        "brand.site_mark": data.siteMark,
        "brand.site_subtitle": data.subtitle,
        "brand.login_tagline": data.tagline,
        "brand.copyright": data.copyright,
      });
    }
  }, [data, form]);

  const handleSubmit = async (values: Record<string, string>) => {
    try {
      await brandingApi.update(values);
      message.success("品牌配置已保存");
      await refreshBrand(); // 登录页/侧边栏/标签页标题立即生效
    } catch (err) {
      message.error((err as Error)?.message ?? "保存失败");
    }
  };

  const handleUpload = async (file: File) => {
    setUploading(true);
    try {
      const result = await brandingApi.uploadLogo(file);
      message.success("Logo 已上传");
      await refreshBrand();
      // 重新拉取管理端配置，让上方预览换上新的 logoUrl（带 ?v= 破缓存）
      await queryClient.invalidateQueries({ queryKey: queryKeys.branding.admin() });
      return result.data?.logoUrl;
    } catch (err) {
      message.error((err as Error)?.message ?? "上传失败");
      return null;
    } finally {
      setUploading(false);
    }
  };

  const current: BrandingConfig = data ?? {
    siteName: "",
    siteMark: "",
    logoUrl: "",
    tagline: "",
    copyright: "",
    subtitle: "",
  };

  return (
    <Space orientation="vertical" size={16} style={{ width: "100%" }}>
      {!canEdit && (
        <Alert type="info" showIcon title="当前账号无 system:config:edit 权限，仅可查看品牌配置" />
      )}

      <Card title="站点 Logo" size="small">
        <Space align="center" size={24}>
          <div
            style={{
              width: 48,
              height: 48,
              borderRadius: 8,
              overflow: "hidden",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              background: xingranBrand.greenPrimary,
              color: xingranBrand.onDark.white,
              fontSize: 24,
              flexShrink: 0,
            }}
          >
            {current.logoUrl ? (
              <img
                key={current.logoUrl}
                src={current.logoUrl}
                alt="Logo"
                style={{ width: "100%", height: "100%", objectFit: "contain" }}
              />
            ) : (
              current.siteMark || "数"
            )}
          </div>
          <Upload
            accept=".png,.jpg,.jpeg,.svg,.webp,.ico"
            maxCount={1}
            showUploadList={false}
            beforeUpload={(file) => {
              if (file.size > 2 * 1024 * 1024) {
                message.error("Logo 图片不能超过 2MB");
                return Upload.LIST_IGNORE;
              }
              void handleUpload(file);
              return false; // 手动上传（走 postFormData，带认证；multipart 不做请求体加密）
            }}
            disabled={!canEdit || uploading}
          >
            <Button icon={<UploadOutlined />} loading={uploading} disabled={!canEdit}>
              上传图片
            </Button>
          </Upload>
          <span style={{ color: "var(--theme-text-secondary)", fontSize: 13 }}>
            支持 png / jpg / svg / webp / ico，≤2MB；上传后立即生效
          </span>
        </Space>
      </Card>

      <Card title="品牌文案" size="small" loading={isLoading}>
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          {FORM_ITEMS.map((item) => (
            <Form.Item
              key={item.name}
              name={item.name}
              label={item.label}
              extra={item.extra}
              rules={
                item.name === "brand.site_name"
                  ? [{ required: true, message: "站点名称不能为空" }]
                  : undefined
              }
            >
              {item.area ? (
                <AppTextArea rows={2} maxLength={200} showCount disabled={!canEdit} />
              ) : (
                <AppInput maxLength={100} disabled={!canEdit} />
              )}
            </Form.Item>
          ))}
          {canEdit && (
            <Form.Item>
              <Button type="primary" htmlType="submit">
                保存
              </Button>
            </Form.Item>
          )}
        </Form>
      </Card>
    </Space>
  );
};

export default BrandSettings;
