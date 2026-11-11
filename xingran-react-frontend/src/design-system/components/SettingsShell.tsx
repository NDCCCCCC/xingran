/**
 * SettingsShell 共用骨架组件（Phase 70 创建；v1.56 目录化改造）
 *
 * 职责（目录化 = 文档目录 + 单页平铺）：
 *   - 桌面（≥lg）：左侧目录 Sider（sticky 白卡）+ 右侧内容白卡；右侧**全部**分类
 *     纵向平铺，每个分类是一个带标题的锚点区块（`<section id="{key}">`）。
 *     左目录点击 = 滚动到对应区块（scrollIntoView）+ URL 同步（?cat= 与 #hash 双写），
 *     URL 可分享：带 `?cat=xxx` 或 `#xxx` 打开页面会直接定位到该区块。
 *   - 窄屏（<lg）：顶部 Segmented 目录 + 同样平铺的全部区块（点击同样滚动）。
 *
 * 为什么从「一次显示一个分类」改为「平铺 + 锚点」：
 *   旧实现每个分类各自带统计卡 + 搜索 + 分页，同一时刻只能看到一屏，
 *   信息密度过低且跨分类对比需要反复切换。平铺后一屏可纵览，目录退化为导航。
 *
 * 权限：分类可选 `permission`（权限码）。门控走 useActionPermission 三态语义 ——
 *   权限集未就绪时目录项与区块仍渲染（避免闪烁），就绪且无权限时目录项与区块一并隐藏。
 *
 * Tab 副作用说明（无需适配）：
 *   - src/components/layout/shared/useRouteTabs.ts 的 tab key = location.pathname，
 *     searchParams / hash 变更不会产生新 RouteTab。
 *   - 故本组件不触 tabsStore，分类注册表由调用方以模块级常量传入以满足
 *     CLAUDE.md「useEffect 依赖稳定」纪律与「引用稳定」假设。
 *
 * 样式契约（CSS 类消费清单，本组件只读不写）：
 *   - .xr-settings-sider / .xr-settings-nav / .xr-settings-nav-item(-active) /
 *     .xr-settings-nav-icon / .xr-settings-content / .xr-settings-section(-title/-desc/-icon)
 */

import { useEffect, useMemo, useRef, type FC, type ReactNode } from "react";
import { Layout, Segmented, Grid } from "antd";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { useActionPermission } from "@/components/shared/useActionPermission";

const { Sider, Content } = Layout;
const { useBreakpoint } = Grid;

export interface SettingsCategory {
  /** 唯一键（= URL ?cat= 参数值 / 区块锚点 id），必须与 categories 数组内其他项不同 */
  key: string;
  /** 目录项与区块标题的显示文本 */
  label: string;
  /** 目录项与区块标题的图标（建议 @ant-design/icons） */
  icon: ReactNode;
  /** 该分类对应的内容（由调用方页面提供） */
  content: ReactNode;
  /** 可选：表单类分类限宽（px）。表格/网格类缺省撑满 */
  maxWidth?: number;
  /** 可选：区块标题下的说明文案 */
  description?: string;
  /**
   * 可选：分类权限码（如 "ops:manager:rule:list"）。
   * 未就绪（loading/error）时仍渲染；就绪且无权限 → 目录项与区块均隐藏。
   */
  permission?: string;
}

export interface SettingsShellProps {
  /** 分类注册表（模块级常量或 useMemo 稳定引用） */
  categories: SettingsCategory[];
  /** 非法/缺失 URL 参数时的回退分类 key */
  defaultCat: string;
  /** URL 查询参数名，缺省 "cat" */
  paramName?: string;
}

const SIDER_WIDTH = 232;

export const SettingsShell: FC<SettingsShellProps> = (props) => {
  const { categories, defaultCat, paramName = "cat" } = props;

  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const location = useLocation();
  const gate = useActionPermission();

  // 权限过滤：loading/error 时 decide().visible 仍为 true（不闪烁）；ready 且无权限才隐藏
  const visibleCategories = useMemo(
    () => categories.filter((c) => !c.permission || gate.decide(c.permission).visible),
    [categories, gate]
  );

  // 唯一真相源：优先 #hash（锚点语义），回退 ?cat=（历史链接兼容），非法值回退 defaultCat
  const hashKey = location.hash.replace(/^#/, "");
  const raw = hashKey || searchParams.get(paramName);
  const fallbackKey =
    visibleCategories.find((c) => c.key === defaultCat)?.key ?? visibleCategories[0]?.key ?? "";
  const activeCat = visibleCategories.some((c) => c.key === raw) ? (raw as string) : fallbackKey;

  // 区块 DOM 引用（用于 scrollIntoView）
  const sectionRefs = useRef<Record<string, HTMLElement | null>>({});

  const scrollToCat = (key: string, behavior: ScrollBehavior = "smooth"): void => {
    const el = sectionRefs.current[key];
    // jsdom / 老浏览器可能没有 scrollIntoView 实现，调用前判空
    if (el && typeof el.scrollIntoView === "function") {
      el.scrollIntoView({ behavior, block: "start" });
    }
  };

  // 目录点击：URL 双写（?cat= 兼容旧链接 + #hash 锚点可分享）+ 平滑滚动
  const setCat = (key: string): void => {
    navigate({ search: `?${paramName}=${key}`, hash: key }, { replace: true });
    scrollToCat(key);
  };

  // URL 指定的分类变化（进入页面 / 外部链接 / 目录点击）→ 定位到区块。
  // 首次定位用 auto，避免页面一进来就长距离平滑滚动。
  const firstScrollRef = useRef(true);
  useEffect(() => {
    if (!activeCat) return;
    scrollToCat(activeCat, firstScrollRef.current ? "auto" : "smooth");
    firstScrollRef.current = false;
    // scrollToCat 只读 ref，无需进依赖
  }, [activeCat]);

  // D-04：≥lg 桌面目录布局 / <lg 顶部 Segmented 目录（内容同样平铺）
  const screens = useBreakpoint();
  const isDesktop = !!screens.lg;

  const sections = visibleCategories.map((c) => {
    const sectionId = `settings-${c.key}`;
    const titleId = `${sectionId}-title`;
    return (
      <section
        key={c.key}
        id={c.key}
        ref={(el) => {
          sectionRefs.current[c.key] = el;
        }}
        className="xr-settings-section"
        aria-labelledby={titleId}
        data-settings-section={c.key}
      >
        <h2 id={titleId} className="xr-settings-section-title">
          <span className="xr-settings-section-icon" aria-hidden="true">
            {c.icon}
          </span>
          <span>{c.label}</span>
        </h2>
        {c.description ? <p className="xr-settings-section-desc">{c.description}</p> : null}
        {c.maxWidth ? <div style={{ maxWidth: c.maxWidth }}>{c.content}</div> : c.content}
      </section>
    );
  });

  const directory = visibleCategories.map((c) => {
    const isActive = c.key === activeCat;
    return { key: c.key, label: c.label, icon: c.icon, isActive };
  });

  if (!isDesktop) {
    return (
      <div>
        <div style={{ marginBottom: 16 }}>
          <Segmented
            block
            value={activeCat}
            onChange={(v) => setCat(String(v))}
            options={directory.map((d) => ({
              label: (
                <span style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                  <span aria-hidden="true">{d.icon}</span>
                  <span>{d.label}</span>
                </span>
              ),
              value: d.key,
            }))}
          />
        </div>
        <div className="xr-settings-content">{sections}</div>
      </div>
    );
  }

  return (
    <Layout
      style={{
        display: "flex",
        alignItems: "flex-start",
        background: "transparent",
        gap: 16,
      }}
    >
      <Sider width={SIDER_WIDTH} className="xr-settings-sider">
        <nav className="xr-settings-nav" aria-label="设置目录">
          {directory.map((d) => (
            <button
              key={d.key}
              type="button"
              className={
                d.isActive
                  ? "xr-settings-nav-item xr-settings-nav-item-active"
                  : "xr-settings-nav-item"
              }
              aria-current={d.isActive ? "true" : undefined}
              onClick={() => setCat(d.key)}
            >
              <span className="xr-settings-nav-icon" aria-hidden="true">
                {d.icon}
              </span>
              <span>{d.label}</span>
            </button>
          ))}
        </nav>
      </Sider>
      <Content style={{ padding: 0, background: "transparent", minWidth: 0 }}>
        <div className="xr-settings-content">{sections}</div>
      </Content>
    </Layout>
  );
};

export default SettingsShell;
