/**
 * 用户设置卡片组件（v1.24 P82 自 categories.tsx 拆出：
 * react-refresh 要求常量注册表文件不含组件定义）
 *
 * 结构：
 *   - 行式设置项（D-06）：每行 label + 描述 + 右对齐控件，Antd Card 分组包裹。
 *     v1.56 目录化后分类标题由 SettingsShell 的区块标题（<h2>）承担，Card 不再重复 title。
 *     （消费 70-02 已落地的 .xr-settings-card-row* CSS 契约）。
 *   - 明暗模式 = 分段卡片选择器（IC-2，.xr-settings-segmented-card*，button + radio 语义）。
 *
 * 保存交互（IC-1 即改即存裁定）：每行控件 onChange 直接调用 settingsStore
 * 单字段 action（updateTheme / updateLayout / updateDataPageSize），成功轻提示
 * 「已保存」；失败 message.error 提示 —— settingsStore.updatePreferences 是
 * 「先服务端成功、再本地提交」，失败时本地 state 未变，受控控件（值全部取自
 * store preferences）天然回滚，无需额外回滚代码。
 */

import { useCallback, useRef, useState, type FC, type KeyboardEvent } from "react";
import { App, Button, Card, Switch } from "antd";
import { ReloadOutlined } from "@ant-design/icons";
// Phase 71 (W1)：裸 Select → AppSelect。两处均为 options 形态的固定枚举（3 项 / 4 项，
// 均 < 8 项 → showSearch 自动为 false）；再显式 showSearch={false} 表达「设置项禁搜」业务意图。
import { AppSelect } from "@/components/ui";
import { useSettingsStore } from "@/store/settingsStore";
import { refreshMenuCache } from "@/store/menuStore";
import { STORAGE_KEYS } from "@/constants/storage";
import type { ColorMode, DensityMode } from "@/types/config";

// ---------- 模块级选项常量（DensitySwitcher.tsx 先例：选项/注册表不随渲染重建） ----------
// 密度选项值 compact/comfortable/spacious 与 settingsStore 权威字段一致（THEME-03 不回归）
const DENSITY_OPTIONS: { value: DensityMode; label: string }[] = [
  { value: "compact", label: "紧凑" },
  { value: "comfortable", label: "舒适" },
  { value: "spacious", label: "宽松" },
];

// 分页选项 10/20/50/100 与 settingsStore 权威字段一致（原值保留）
const PAGE_SIZE_SELECT_OPTIONS: { value: number; label: string }[] = [10, 20, 50, 100].map(
  (size) => ({ value: size, label: `${size} 条/页` })
);

// ---------- IC-1 即改即存统一包装：成功轻提示 / 失败错误提示（UI-SPEC 错误格式） ----------

const useSaveSetting = () => {
  const { message } = App.useApp();
  return useCallback(
    async (fn: () => Promise<void>) => {
      try {
        await fn();
        message.success("已保存");
      } catch {
        message.error("保存设置失败，请重试");
      }
    },
    [message]
  );
};

// ---------- 界面设置：明暗模式分段卡片选择器（IC-2） ----------

export const AppearanceSettingsCard: FC = () => {
  const preferences = useSettingsStore((s) => s.preferences);
  const updateTheme = useSettingsStore((s) => s.updateTheme);
  const handleUpdate = useSaveSetting();
  const lightCardRef = useRef<HTMLButtonElement>(null);
  const darkCardRef = useRef<HTMLButtonElement>(null);

  const mode = preferences.theme.mode;

  const selectMode = (next: ColorMode): void => {
    handleUpdate(() => updateTheme({ mode: next }));
  };

  // IC-2 可达性：radiogroup 语义，←/→ 在两卡间切换并移动焦点
  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>): void => {
    if (e.key !== "ArrowLeft" && e.key !== "ArrowRight") return;
    e.preventDefault();
    const next: ColorMode = e.key === "ArrowRight" ? "dark" : "light";
    (next === "dark" ? darkCardRef : lightCardRef).current?.focus();
    if (next !== mode) {
      selectMode(next);
    }
  };

  const cardClass = (active: boolean): string =>
    active
      ? "xr-settings-segmented-card xr-settings-segmented-card-active"
      : "xr-settings-segmented-card";

  return (
    <Card>
      <div className="xr-settings-card-row">
        <div>
          <div className="xr-settings-card-row-label">明暗模式</div>
          <div className="xr-settings-card-row-desc">选择界面的颜色模式，深色模式适合夜间使用</div>
        </div>
        <div className="xr-settings-card-row-control">
          <div
            className="xr-settings-segmented"
            role="radiogroup"
            aria-label="明暗模式"
            onKeyDown={handleKeyDown}
          >
            <button
              ref={lightCardRef}
              type="button"
              role="radio"
              aria-checked={mode === "light"}
              className={cardClass(mode === "light")}
              onClick={() => selectMode("light")}
            >
              <span className="xr-settings-segmented-card-preview xr-settings-segmented-card-preview-light">
                <span className="preview-side" />
                <span className="preview-body">
                  <span className="preview-bar" />
                  <span className="preview-bar short" />
                </span>
              </span>
              <span className="xr-settings-segmented-card-label">浅色模式</span>
            </button>
            <button
              ref={darkCardRef}
              type="button"
              role="radio"
              aria-checked={mode === "dark"}
              className={cardClass(mode === "dark")}
              onClick={() => selectMode("dark")}
            >
              <span className="xr-settings-segmented-card-preview xr-settings-segmented-card-preview-dark">
                <span className="preview-side" />
                <span className="preview-body">
                  <span className="preview-bar" />
                  <span className="preview-bar short" />
                </span>
              </span>
              <span className="xr-settings-segmented-card-label">深色模式</span>
            </button>
          </div>
        </div>
      </div>
    </Card>
  );
};

// ---------- 布局设置：密度模式 + 侧栏折叠 ----------

export const LayoutSettingsCard: FC = () => {
  const preferences = useSettingsStore((s) => s.preferences);
  const updateLayout = useSettingsStore((s) => s.updateLayout);
  const handleUpdate = useSaveSetting();

  return (
    <Card>
      <div className="xr-settings-card-row">
        <div>
          <div className="xr-settings-card-row-label">密度模式</div>
          <div className="xr-settings-card-row-desc">控制列表与表单的紧凑程度</div>
        </div>
        <div className="xr-settings-card-row-control">
          <AppSelect
            showSearch={false}
            style={{ width: 160 }}
            value={preferences.layout.density}
            options={DENSITY_OPTIONS}
            // AppSelect 泛型被 ComponentProps 擦除 → onChange 值为 unknown，收敛回业务类型
            onChange={(value) =>
              handleUpdate(() => updateLayout({ density: value as DensityMode }))
            }
          />
        </div>
      </div>
      <div className="xr-settings-card-row">
        <div>
          <div className="xr-settings-card-row-label">默认折叠侧边栏</div>
          <div className="xr-settings-card-row-desc">折叠侧边栏以获得更大的内容区域</div>
        </div>
        <div className="xr-settings-card-row-control">
          <Switch
            checked={preferences.layout.sidebar.collapsed}
            onChange={(checked) =>
              // updateLayout 对 layout 是浅合并：必须传完整 sidebar 对象，
              // 否则既有 width/collapsedWidth 字段会被覆盖丢失
              handleUpdate(() =>
                updateLayout({
                  sidebar: { ...preferences.layout.sidebar, collapsed: checked },
                })
              )
            }
          />
        </div>
      </div>
    </Card>
  );
};

// ---------- 数据设置：默认分页大小 ----------

export const DataSettingsCard: FC = () => {
  const preferences = useSettingsStore((s) => s.preferences);
  const updateDataPageSize = useSettingsStore((s) => s.updateDataPageSize);
  const handleUpdate = useSaveSetting();

  return (
    <Card>
      <div className="xr-settings-card-row">
        <div>
          <div className="xr-settings-card-row-label">默认分页大小</div>
          <div className="xr-settings-card-row-desc">列表页面默认每页显示的数据条数</div>
        </div>
        <div className="xr-settings-card-row-control">
          <AppSelect
            showSearch={false}
            style={{ width: 160 }}
            value={preferences.data.defaultPageSize}
            options={PAGE_SIZE_SELECT_OPTIONS}
            onChange={(value) => handleUpdate(() => updateDataPageSize(value as number))}
          />
        </div>
      </div>
      <PermissionRefreshRow />
    </Card>
  );
};

// ---------- 数据设置：权限集刷新（管理端改权限后无需重新登录） ----------

/**
 * 刷新权限集与菜单缓存。
 *
 * 为什么需要入口：权限集在前端有 30 分钟 sessionStorage 快照
 * （DynamicRoutes 的 MENU_CACHE），后端 menu:user:perms:{userID} 也是 30 分钟
 * TTL 缓存。管理员刚在角色授权页补了权限码，当前会话看到的仍是旧快照，
 * 表现为"按钮灰着"，用户只能重新登录或等过期。
 *
 * 本行 = 手动解药：清 sessionStorage 快照 + 强制拉 API（fetchAll(true) 内部
 * 走真实 API 绕 TTL 缓存并回写新快照）。菜单树与权限集一起刷新，
 * 与 DynamicRoutes 的 hydrate-then-revalidate 口径一致。
 */
const PermissionRefreshRow: FC = () => {
  const [refreshing, setRefreshing] = useState(false);
  const { message } = App.useApp();

  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    try {
      // 先清 sessionStorage 快照：否则 fetchAll 的 hydrate 会先喂旧值，
      // 且硬刷新后 DynamicRoutes 仍会从旧快照 restore。
      try {
        sessionStorage.removeItem(STORAGE_KEYS.MENU_CACHE);
      } catch {
        // 隐私模式 —— 静默
      }
      // refreshMenuCache = fetchAll(true)：强制绕过 TTLMenuCache 走真实 API，
      // 完成后由 store 回写内存缓存；sessionStorage 快照已在上面清掉，
      // 下次硬刷新时 DynamicRoutes 会重新写新快照。
      await refreshMenuCache();
      message.success("权限已刷新，页面按钮已按最新权限更新");
    } catch {
      message.error("刷新权限失败，请检查网络后重试");
    } finally {
      setRefreshing(false);
    }
  }, [message]);

  return (
    <div className="xr-settings-card-row">
      <div>
        <div className="xr-settings-card-row-label">刷新权限</div>
        <div className="xr-settings-card-row-desc">
          管理员调整角色权限后，点此立即生效，无需重新登录
        </div>
      </div>
      <div className="xr-settings-card-row-control">
        <Button icon={<ReloadOutlined />} loading={refreshing} onClick={handleRefresh}>
          刷新
        </Button>
      </div>
    </div>
  );
};
