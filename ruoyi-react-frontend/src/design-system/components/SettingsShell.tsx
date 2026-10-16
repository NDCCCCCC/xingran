/**
 * Phase 70 · SettingsShell 共用骨架组件（D-01/D-03/D-04/D-05 落地）
 *
 * 职责：
 *   - 桌面（≥lg）：Layout + Sider(220) + Content，左侧分类导航白卡 + 右侧内容白卡。
 *   - 窄屏（<lg）：顶部 Segmented block + 内容区全宽。
 *   - 激活分类 = URL `?{paramName}=` 参数（useSearchParams 唯一真相源），非法/缺失值
 *     回退 defaultCat；切换用 `setSearchParams(..., { replace: true })` 不污染 history 栈。
 *
 * Tab 副作用说明（无需适配）：
 *   - src/components/layout/shared/useRouteTabs.ts 的 tab key = location.pathname，
 *     searchParams 变更不会产生新 RouteTab（Pitfall 3 已验证）。
 *   - 故本组件不触 tabsStore，分类注册表由调用方以模块级常量传入以满足
 *     CLAUDE.md「useEffect 依赖稳定」纪律与「引用稳定」假设。
 *
 * 样式契约（CSS 类消费清单，本组件只读不写）：
 *   - .xr-settings-sider / .xr-settings-nav / .xr-settings-nav-item(-active) /
 *     .xr-settings-nav-icon / .xr-settings-content
 */

import type { FC, ReactNode } from "react";
import { Layout, Segmented, Grid } from "antd";
import { useSearchParams } from "react-router-dom";

const { Sider, Content } = Layout;
const { useBreakpoint } = Grid;

export interface SettingsCategory {
  /** 唯一键（= URL ?cat= 参数值），必须与 categories 数组内其他项不同 */
  key: string;
  /** 导航项显示文本 */
  label: string;
  /** 导航项图标（建议 @ant-design/icons） */
  icon: ReactNode;
  /** 该分类对应的内容（由调用方页面提供） */
  content: ReactNode;
  /** 可选：表单类分类限宽（px）。表格/网格类缺省撑满（D-02） */
  maxWidth?: number;
}

export interface SettingsShellProps {
  /** 分类注册表（模块级常量或 useMemo 稳定引用） */
  categories: SettingsCategory[];
  /** 非法/缺失 ?cat= 时的回退分类 key */
  defaultCat: string;
  /** URL 参数名，缺省 "cat" */
  paramName?: string;
}

const SIDER_WIDTH = 220;

export const SettingsShell: FC<SettingsShellProps> = (props) => {
  const { categories, defaultCat, paramName = "cat" } = props;

  const [searchParams, setSearchParams] = useSearchParams();

  // D-03：URL 参数为唯一真相源；非法值经白名单校验后回退 defaultCat（无 setState，渲染期计算）
  const raw = searchParams.get(paramName);
  const activeCat = categories.some((c) => c.key === raw) ? (raw as string) : defaultCat;
  const activeCategory = categories.find((c) => c.key === activeCat) ?? categories[0];

  // D-03：replace:true 不污染 history 栈、不产生新 RouteTab
  const setCat = (key: string): void => {
    setSearchParams({ [paramName]: key }, { replace: true });
  };

  // D-04：≥lg 桌面布局 / <lg 顶部 Segmented 降级
  const screens = useBreakpoint();
  const isDesktop = !!screens.lg;

  if (!isDesktop) {
    return (
      <div>
        <div style={{ marginBottom: 16 }}>
          <Segmented
            block
            value={activeCat}
            onChange={(v) => setCat(String(v))}
            options={categories.map((c) => ({
              label: (
                <span style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                  <span aria-hidden="true">{c.icon}</span>
                  <span>{c.label}</span>
                </span>
              ),
              value: c.key,
            }))}
          />
        </div>
        <div className="xr-settings-content">{activeCategory?.content}</div>
      </div>
    );
  }

  return (
    <Layout
      style={{
        display: "flex",
        alignItems: "stretch",
        background: "transparent",
        gap: 16,
      }}
    >
      <Sider width={SIDER_WIDTH} className="xr-settings-sider">
        <nav className="xr-settings-nav" aria-label="设置分类">
          {categories.map((c) => {
            const isActive = c.key === activeCat;
            return (
              <button
                key={c.key}
                type="button"
                className={
                  isActive
                    ? "xr-settings-nav-item xr-settings-nav-item-active"
                    : "xr-settings-nav-item"
                }
                aria-current={isActive ? "true" : undefined}
                onClick={() => setCat(c.key)}
              >
                <span className="xr-settings-nav-icon" aria-hidden="true">
                  {c.icon}
                </span>
                <span>{c.label}</span>
              </button>
            );
          })}
        </nav>
      </Sider>
      <Content style={{ padding: 0, background: "transparent" }}>
        <div className="xr-settings-content">
          {activeCategory?.maxWidth ? (
            <div style={{ maxWidth: activeCategory.maxWidth }}>{activeCategory.content}</div>
          ) : (
            activeCategory?.content
          )}
        </div>
      </Content>
    </Layout>
  );
};

export default SettingsShell;
