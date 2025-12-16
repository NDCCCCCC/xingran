import React from 'react';
import { Breadcrumb } from 'antd';
import { useLocation, Link } from 'react-router-dom';

const BreadcrumbComponent: React.FC = () => {
  const location = useLocation();
  const pathSnippets = location.pathname.split('/').filter((i) => i);

  const breadcrumbNameMap: Record<string, string> = {
    dashboard: '仪表盘',
    system: '系统管理',
    users: '用户管理',
    roles: '角色管理',
    menus: '菜单管理',
    depts: '部门管理',
    posts: '岗位管理',
    dicts: '字典管理',
    configs: '参数配置',
    notices: '通知公告',
    monitor: '系统监控',
    online: '在线用户',
    operlog: '操作日志',
    loginlog: '登录日志',
  };

  const items = pathSnippets.map((snippet, index) => {
    const url = `/${pathSnippets.slice(0, index + 1).join('/')}`;
    const name = breadcrumbNameMap[snippet] || snippet;

    return {
      key: url,
      title: index === pathSnippets.length - 1 ? (
        <span>{name}</span>
      ) : (
        <Link to={url}>{name}</Link>
      ),
    };
  });

  return (
    <div className="px-6 py-3 bg-white border-b border-gray-100">
      <Breadcrumb
        items={items}
        className="text-sm"
      />
    </div>
  );
};

export default BreadcrumbComponent;