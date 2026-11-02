/**
 * PageTitle 页头组件（v1.23 原型复刻）
 *
 * 基准：.planning/sketch/system-user-demo.html（用户确认稿）
 * 格式：<h1 class="page-title">前缀<span class="dot">·</span>后缀</h1>
 * 样式：.page-head / .page-title / .dot / .page-sub / .page-actions 见 index.css
 *
 * 用法：
 *   <PageTitle pre="系统" post="用户" sub="账号 · 部门 · 角色三位一体的权限底座"
 *     actions={<><Button>导入</Button><Button type="primary">新增</Button></>} />
 */

import type { FC, ReactNode } from "react";

interface PageTitleProps {
  /** 标题前缀（金色 dot 左侧） */
  pre: string;
  /** 标题后缀（金色 dot 右侧，可空 → 无 dot） */
  post?: string;
  /** 副标题（page-sub） */
  sub?: string;
  /** 右侧操作区（page-actions，margin-left: auto） */
  actions?: ReactNode;
}

const PageTitle: FC<PageTitleProps> = ({ pre, post, sub, actions }) => {
  return (
    <div className="page-head">
      <div>
        <h1 className="page-title">
          {pre}
          {post && <span className="dot">·</span>}
          {post}
        </h1>
        {sub && <p className="page-sub">{sub}</p>}
      </div>
      {actions && <div className="page-actions">{actions}</div>}
    </div>
  );
};

export default PageTitle;
