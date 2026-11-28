/**
 * brandStore — 站点品牌信息（brand.* sys_config）前端消费层
 *
 * 数据流：App 启动时 fetchPublic() 拉一次免认证公开端点（后端缓存 5 分钟），
 * 登录页 / 侧边栏 / document.title 都从本 store 读；请求失败静默回退
 * DEFAULT_BRANDING（与 migration_259 seed 同值），登录页永不空白。
 *
 * 品牌事实源 = 数据库 sys_config；本 store 只是运行时快照。
 * 管理端（系统设置-站点品牌）保存成功后调用 refresh() 让全局立即可见。
 */
import { create } from "zustand";
import {
  brandingApi,
  DEFAULT_BRANDING,
  normalizeBranding,
  type BrandingConfig,
} from "@/lib/brandingApi";

interface BrandState {
  /** 当前品牌配置（加载失败 = DEFAULT_BRANDING） */
  config: BrandingConfig;
  /** 是否已完成首次拉取（无论成败） */
  loaded: boolean;
  /** 拉取公开品牌信息（幂等；失败静默回退） */
  fetchPublic: () => Promise<void>;
  /** 管理端保存后强制刷新（下轮 fetch 走网络） */
  refresh: () => Promise<void>;
}

export const useBrandStore = create<BrandState>((set) => ({
  config: DEFAULT_BRANDING,
  loaded: false,

  fetchPublic: async () => {
    try {
      const result = await brandingApi.getPublic();
      set({ config: normalizeBranding(result.data), loaded: true });
    } catch {
      // 后端不可达：保持 DEFAULT_BRANDING，不阻断 UI
      set({ config: DEFAULT_BRANDING, loaded: true });
    }
  },

  refresh: async () => {
    set({ loaded: false });
    await useBrandStore.getState().fetchPublic();
  },
}));
