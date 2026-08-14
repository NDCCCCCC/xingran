/**
 * 全局搜索组件
 * 支持Cmd+K / Ctrl+K快捷键打开
 */

import { useState, useEffect, useCallback, useMemo } from "react";
import { Modal, Input, List, Typography, Tag, Divider } from "antd";
import { useNavigate } from "react-router-dom";
import type { FC } from "react";
import {
  DASHBOARD,
  MONITOR_DASHBOARD,
  SYSTEM_USER,
  SYSTEM_ROLE,
  SYSTEM_MENU,
  NETWORK_DEVICES,
  DUTY_MY_DUTY,
  WORKORDER_ORDERS,
} from "@/constants/routes";
import {
  SearchOutlined,
  DashboardOutlined,
  UserOutlined,
  SettingOutlined,
  MonitorOutlined,
  CloudServerOutlined,
  CalendarOutlined,
  FileTextOutlined,
  AppstoreOutlined,
  ArrowRightOutlined,
} from "@ant-design/icons";

const { Search } = Input;
const { Text } = Typography;

interface SearchResult {
  id: string;
  title: string;
  description: string;
  icon: import("react").ReactNode;
  path: string;
  category: string;
  keywords: string[];
}

// 搜索结果配置
const searchResults: SearchResult[] = [
  {
    id: "dashboard",
    title: "仪表盘",
    description: "数据概览与统计分析",
    icon: <DashboardOutlined />,
    path: DASHBOARD,
    category: "页面",
    keywords: ["dashboard", "仪表盘", "首页", "数据"],
  },
  {
    id: "user",
    title: "用户管理",
    description: "管理系统用户信息",
    icon: <UserOutlined />,
    path: SYSTEM_USER,
    category: "系统管理",
    keywords: ["user", "用户", "用户管理", "系统"],
  },
  {
    id: "role",
    title: "角色管理",
    description: "管理系统角色权限",
    icon: <SettingOutlined />,
    path: SYSTEM_ROLE,
    category: "系统管理",
    keywords: ["role", "角色", "角色管理", "权限"],
  },
  {
    id: "menu",
    title: "菜单管理",
    description: "管理系统菜单配置",
    icon: <AppstoreOutlined />,
    path: SYSTEM_MENU,
    category: "系统管理",
    keywords: ["menu", "菜单", "菜单管理", "配置"],
  },
  {
    id: "monitor",
    title: "系统监控",
    description: "服务器性能监控",
    icon: <MonitorOutlined />,
    path: MONITOR_DASHBOARD,
    category: "监控",
    keywords: ["monitor", "监控", "服务器", "性能"],
  },
  {
    id: "devices",
    title: "设备管理",
    description: "网络设备管理",
    icon: <CloudServerOutlined />,
    path: NETWORK_DEVICES,
    category: "网络",
    keywords: ["device", "设备", "网络", "路由器", "交换机"],
  },
  {
    id: "duty",
    title: "我的值班",
    description: "值班安排与日历",
    icon: <CalendarOutlined />,
    path: DUTY_MY_DUTY,
    category: "值班",
    keywords: ["duty", "值班", "日历", "排班"],
  },
  {
    id: "workorder",
    title: "工单管理",
    description: "工单查看与处理",
    icon: <FileTextOutlined />,
    path: WORKORDER_ORDERS,
    category: "工单",
    keywords: ["workorder", "工单", "任务", "问题"],
  },
];

const GlobalSearch: FC = () => {
  const [visible, setVisible] = useState(false);
  const [searchValue, setSearchValue] = useState("");
  const [selectedIndex, setSelectedIndex] = useState(0);
  const navigate = useNavigate();

  // 过滤搜索结果
  const filteredResults = useMemo(() => {
    if (!searchValue.trim()) {
      return searchResults;
    }

    const query = searchValue.toLowerCase();
    return searchResults.filter(
      (result) =>
        result.title.toLowerCase().includes(query) ||
        result.description.toLowerCase().includes(query) ||
        result.keywords.some((keyword) => keyword.toLowerCase().includes(query))
    );
  }, [searchValue]);

  // 处理结果点击
  const handleResultClick = useCallback((result: SearchResult) => {
    navigate(result.path);
    setVisible(false);
    setSearchValue("");
  }, [navigate]);

  // 处理键盘事件
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    // Cmd+K or Ctrl+K
    if ((e.metaKey || e.ctrlKey) && e.key === "k") {
      e.preventDefault();
      setVisible((prev) => !prev);
      setSearchValue("");
      setSelectedIndex(0);
    }

    // 在搜索框打开时处理键盘导航
    if (visible) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSelectedIndex((prev) =>
          prev < filteredResults.length - 1 ? prev + 1 : prev
        );
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setSelectedIndex((prev) => (prev > 0 ? prev - 1 : 0));
      } else if (e.key === "Enter") {
        e.preventDefault();
        if (filteredResults[selectedIndex]) {
          handleResultClick(filteredResults[selectedIndex]);
        }
      } else if (e.key === "Escape") {
        e.preventDefault();
        setVisible(false);
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- filteredResults.length is redundant with filteredResults
  }, [visible, filteredResults.length, selectedIndex, handleResultClick, filteredResults]);

  // 注册键盘事件监听
  useEffect(() => {
    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [handleKeyDown]);

  return (
    <>
      {/* 搜索提示 - 鼠标悬停显示 */}
      <div
        style={{
          position: "fixed",
          bottom: "20px",
          right: "20px",
          background: "var(--theme-bg-surface)",
          border: "1px solid var(--theme-border-primary)",
          borderRadius: "var(--theme-radius-lg)",
          padding: "8px 16px",
          display: "flex",
          alignItems: "center",
          gap: "8px",
          cursor: "pointer",
          boxShadow: "var(--theme-shadow-md)",
          zIndex: 1000,
          transition: "all var(--theme-transition-base)",
        }}
        onClick={() => setVisible(true)}
        onMouseEnter={(e) => {
          e.currentTarget.style.transform = "translateY(-2px)";
          e.currentTarget.style.boxShadow = "var(--theme-shadow-lg)";
        }}
        onMouseLeave={(e) => {
          e.currentTarget.style.transform = "translateY(0)";
          e.currentTarget.style.boxShadow = "var(--theme-shadow-md)";
        }}
      >
        <SearchOutlined style={{ fontSize: "16px", color: "var(--theme-text-secondary)" }} />
        <Text style={{ fontSize: "14px", color: "var(--theme-text-secondary)" }}>搜索</Text>
        <Tag
          style={{
            margin: 0,
            fontSize: "12px",
            borderRadius: "6px",
            background: "var(--theme-bg-tertiary)",
            borderColor: "var(--theme-border-primary)",
            color: "var(--theme-text-secondary)",
          }}
        >
          ⌘K
        </Tag>
      </div>

      {/* 搜索模态框 */}
      <Modal
        open={visible}
        onCancel={() => setVisible(false)}
        footer={null}
        centered
        width={600}
        styles={{
          body: { padding: "16px" },
          content: {
            background: "var(--theme-bg-surface)",
            borderRadius: "var(--theme-radius-xl)",
            border: "1px solid var(--theme-border-primary)",
          },
        } as Record<string, unknown>}
        closeIcon={null}
        style={{ top: 100 }}
      >
        <div style={{ padding: "16px" }}>
          {/* 搜索输入框 */}
          <Search
            placeholder="搜索页面、命令、菜单..."
            value={searchValue}
            onChange={(e) => {
              setSearchValue(e.target.value);
              setSelectedIndex(0);
            }}
            autoFocus
            size="large"
            style={{
              marginBottom: "16px",
            }}
            prefix={<SearchOutlined style={{ color: "var(--theme-text-secondary)" }} />}
          />

          {/* 快捷键提示 */}
          <div
            style={{
              display: "flex",
              gap: "16px",
              marginBottom: "12px",
              fontSize: "12px",
              color: "var(--theme-text-tertiary)",
            }}
          >
            <span>
              <Tag style={{ margin: "0 4px 0 0", fontSize: "11px" }}>↑↓</Tag>
              导航
            </span>
            <span>
              <Tag style={{ margin: "0 4px 0 0", fontSize: "11px" }}>↵</Tag>
              打开
            </span>
            <span>
              <Tag style={{ margin: "0 4px 0 0", fontSize: "11px" }}>esc</Tag>
              关闭
            </span>
          </div>

          <Divider style={{ margin: "8px 0 12px 0" }} />

          {/* 搜索结果列表 */}
          <List
            dataSource={filteredResults}
            renderItem={(result, index) => (
              <List.Item
                key={result.id}
                onClick={() => handleResultClick(result)}
                style={{
                  padding: "12px",
                  borderRadius: "var(--theme-radius-md)",
                  cursor: "pointer",
                  background:
                    index === selectedIndex
                      ? "var(--theme-primary-50)"
                      : "transparent",
                  transition: "all var(--theme-transition-base)",
                  border:
                    index === selectedIndex
                      ? "1px solid var(--theme-primary-200)"
                      : "1px solid transparent",
                }}
                onMouseEnter={() => setSelectedIndex(index)}
              >
                <List.Item.Meta
                  avatar={
                    <div
                      style={{
                        width: "40px",
                        height: "40px",
                        borderRadius: "var(--theme-radius-md)",
                        background: "var(--theme-bg-secondary)",
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                        fontSize: "18px",
                        color: "var(--theme-primary-500)",
                      }}
                    >
                      {result.icon}
                    </div>
                  }
                  title={
                    <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
                      <Text strong style={{ fontSize: "14px" }}>
                        {result.title}
                      </Text>
                      <Tag
                        style={{
                          margin: 0,
                          fontSize: "11px",
                          borderRadius: "4px",
                        }}
                      >
                        {result.category}
                      </Tag>
                    </div>
                  }
                  description={
                    <Text style={{ fontSize: "12px", color: "var(--theme-text-secondary)" }}>
                      {result.description}
                    </Text>
                  }
                />
                {index === selectedIndex && (
                  <ArrowRightOutlined style={{ color: "var(--theme-primary-500)" }} />
                )}
              </List.Item>
            )}
          />
        </div>
      </Modal>
    </>
  );
};

export default GlobalSearch;
