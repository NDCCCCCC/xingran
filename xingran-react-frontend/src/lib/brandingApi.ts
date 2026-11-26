/**
 * 站点品牌 API（brand.* 命名空间）
 *
 * 端点:
 *   GET  /system/branding/public  免认证聚合读取（登录页/侧边栏/标题用）
 *   GET  /system/branding         管理端读取（需 system:config:list/view）
 *   POST /system/branding/update  批量更新（strict system:config:edit）
 *   POST /system/branding/logo    上传 Logo（multipart, strict system:config:edit）
 */
import { get, post, postFormData } from "@/lib/api";

/** 品牌配置聚合视图（与后端 BrandingConfig 对齐） */
export interface BrandingConfig {
  siteName: string;
  siteMark: string;
  logoUrl: string;
  tagline: string;
  copyright: string;
  subtitle: string;
}

/**
 * 前端兜底默认值（与 migration_259 seed / 后端 defaultBrandingConfig 一致）。
 * 后端不可达或 public 端点失败时使用，保证登录页永不空白。
 */
export const DEFAULT_BRANDING: BrandingConfig = {
  siteName: "数字化平台",
  siteMark: "数",
  logoUrl: "",
  tagline: "以国密算法为基座,融合自动化与 AI 调度,让每一次资源调度都可观测、可审计、可信赖。",
  copyright: "© 2026 数字化平台",
  subtitle: "请使用您的账号登录以访问运维控制台",
};

/** 品牌信息规范化：空字段回退默认值 */
export function normalizeBranding(raw: Partial<BrandingConfig> | null | undefined): BrandingConfig {
  const merged = { ...DEFAULT_BRANDING, ...(raw ?? {}) };
  return {
    siteName: merged.siteName || DEFAULT_BRANDING.siteName,
    siteMark: merged.siteMark || DEFAULT_BRANDING.siteMark,
    logoUrl: merged.logoUrl || "",
    tagline: merged.tagline || DEFAULT_BRANDING.tagline,
    copyright: merged.copyright || DEFAULT_BRANDING.copyright,
    subtitle: merged.subtitle || DEFAULT_BRANDING.subtitle,
  };
}

export const brandingApi = {
  /** 免认证公开读取（CacheProvider 缓存 5 分钟） */
  getPublic: () => get<BrandingConfig>("/system/branding/public"),

  /** 管理端读取全部品牌配置 */
  get: () => get<BrandingConfig>("/system/branding"),

  /** 批量更新（config_key → value；只接受 brand.* 键） */
  update: (values: Record<string, string>) =>
    post<{ message: string }>("/system/branding/update", values),

  /** 上传 Logo（≤2MB, png/jpg/svg/webp/ico），返回新的 logoUrl（带 ?v= 破缓存） */
  uploadLogo: (file: File) => {
    const formData = new FormData();
    formData.append("file", file);
    return postFormData<{ logoUrl: string }>("/system/branding/logo", formData);
  },
};
