/**
 * 图标选择器组件
 * 弹窗式图标选择，支持搜索和预览
 */

import { useState, useMemo, useEffect } from "react";
import type { FC } from "react";
import { App, Input, Modal, Tabs } from "antd";
import { SearchOutlined } from "@ant-design/icons";
import { getIconComponent, iconCategories, iconDescriptionMap } from "@/utils/iconUtils";

interface IconSelectProps {
  value?: string;
  onChange?: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
}

const IconSelect: FC<IconSelectProps> = ({
  value,
  onChange,
  placeholder = "请选择图标",
  disabled = false,
}) => {
  const { message } = App.useApp();
  const [visible, setVisible] = useState(false);
  const [searchText, setSearchText] = useState("");
  const [selectedIcon, setSelectedIcon] = useState<string | undefined>(value);

  // 当外部 value 变化时同步内部状态
  useEffect(() => {
    setSelectedIcon(value);
  }, [value]);

  // 过滤图标
  const filteredCategories = useMemo(() => {
    if (!searchText) {
      return iconCategories;
    }

    const lowerSearch = searchText.toLowerCase();
    const result: Record<string, string[]> = {};

    for (const [category, icons] of Object.entries(iconCategories)) {
      const filtered = icons.filter((icon) => {
        // 匹配图标名称
        if (icon.toLowerCase().includes(lowerSearch)) {
          return true;
        }
        // 匹配中文描述
        const description = iconDescriptionMap[icon];
        if (description && description.includes(searchText)) {
          return true;
        }
        return false;
      });

      if (filtered.length > 0) {
        result[category] = filtered;
      }
    }

    return result;
  }, [searchText]);

  // 计算匹配到的图标总数
  const totalMatchedIcons = useMemo(() => {
    return Object.values(filteredCategories).reduce((sum, icons) => sum + icons.length, 0);
  }, [filteredCategories]);

  // 处理图标选择
  const handleIconClick = (iconName: string) => {
    setSelectedIcon(iconName);
  };

  // 确认选择
  const handleOk = () => {
    if (selectedIcon) {
      onChange?.(selectedIcon);
      setVisible(false);
      setSearchText("");
    } else {
      message.warning("请先选择一个图标");
    }
  };

  // 取消选择
  const handleCancel = () => {
    setSelectedIcon(value); // 恢复到原来的值
    setVisible(false);
    setSearchText("");
  };

  // 清空选择
  const handleClear = () => {
    setSelectedIcon(undefined);
    onChange?.("");
    setVisible(false);
    setSearchText("");
  };

  return (
    <>
      {/* 输入框 */}
      <Input
        value={value ? `${value} (${iconDescriptionMap[value] || value})` : ""}
        readOnly
        disabled={disabled}
        onClick={() => !disabled && setVisible(true)}
        placeholder={value ? "" : placeholder}
        suffix={value ? getIconComponent(value) : <span />}
        style={{ cursor: disabled ? "not-allowed" : "pointer" }}
      />

      {/* 图标选择弹窗 */}
      <Modal
        title="选择图标"
        open={visible}
        onOk={handleOk}
        onCancel={handleCancel}
        width={800}
        okText="确定"
        cancelText="取消"
        okButtonProps={{ disabled: !selectedIcon }}
      >
        {/* 搜索框 */}
        <Input
          placeholder="搜索图标名称或描述..."
          prefix={<SearchOutlined />}
          value={searchText}
          onChange={(e) => setSearchText(e.target.value)}
          style={{ marginBottom: 16 }}
          allowClear
        />

        {/* 搜索结果统计 */}
        {searchText && (
          <div style={{ marginBottom: 12, color: "var(--theme-text-tertiary, #999)" }}>
            找到 <strong>{totalMatchedIcons}</strong> 个匹配的图标
          </div>
        )}

        {/* 图标分类展示 */}
        <div style={{ maxHeight: 400, overflowY: "auto" }}>
          {Object.keys(filteredCategories).length === 0 ? (
            <div
              style={{
                textAlign: "center",
                padding: 40,
                color: "var(--theme-text-tertiary, #999)",
              }}
            >
              未找到匹配的图标
            </div>
          ) : (
            <Tabs
              defaultActiveKey={Object.keys(filteredCategories)[0]}
              items={Object.entries(filteredCategories).map(([category, icons]) => ({
                label: `${category} (${icons.length})`,
                key: category,
                children: (
                  <div
                    style={{
                      display: "grid",
                      gridTemplateColumns: "repeat(6, 1fr)",
                      gap: 12,
                    }}
                  >
                    {icons.map((iconName) => {
                      const isSelected = selectedIcon === iconName;
                      return (
                        <div
                          key={iconName}
                          onClick={() => handleIconClick(iconName)}
                          style={{
                            border: `1px solid ${isSelected ? "var(--theme-info, #337ab0)" : "#dbd7ce"}`,
                            borderRadius: 4,
                            padding: 12,
                            textAlign: "center",
                            cursor: "pointer",
                            backgroundColor: isSelected ? "#e6f7ff" : "#fff",
                            transition: "all 0.2s",
                          }}
                          onMouseEnter={(e) => {
                            if (!isSelected) {
                              e.currentTarget.style.borderColor = "var(--theme-info, #337ab0)";
                              e.currentTarget.style.backgroundColor = "#e9efeb";
                            }
                          }}
                          onMouseLeave={(e) => {
                            if (!isSelected) {
                              e.currentTarget.style.borderColor = "#dbd7ce";
                              e.currentTarget.style.backgroundColor = "#fff";
                            }
                          }}
                        >
                          <div style={{ fontSize: 24, marginBottom: 8 }}>
                            {getIconComponent(iconName)}
                          </div>
                          <div
                            style={{
                              fontSize: 12,
                              color: "var(--theme-text-tertiary, #666)",
                              overflow: "hidden",
                              textOverflow: "ellipsis",
                              whiteSpace: "nowrap",
                            }}
                            title={iconName}
                          >
                            {iconDescriptionMap[iconName] || iconName}
                          </div>
                          {isSelected && (
                            <div
                              style={{
                                position: "absolute",
                                top: 4,
                                right: 4,
                                width: 16,
                                height: 16,
                                borderRadius: "50%",
                                backgroundColor: "var(--theme-info, #337ab0)",
                                color: "#fff",
                                fontSize: 10,
                                display: "flex",
                                alignItems: "center",
                                justifyContent: "center",
                              }}
                            >
                              ✓
                            </div>
                          )}
                        </div>
                      );
                    })}
                  </div>
                ),
              }))}
            />
          )}
        </div>

        {/* 当前选中 */}
        {selectedIcon && (
          <div
            style={{
              marginTop: 16,
              padding: 12,
              backgroundColor: "#f5f5f5",
              borderRadius: 4,
              display: "flex",
              alignItems: "center",
              gap: 12,
            }}
          >
            <span>当前选中：</span>
            <span style={{ fontSize: 24 }}>{getIconComponent(selectedIcon)}</span>
            <span style={{ fontWeight: 500 }}>
              {selectedIcon} ({iconDescriptionMap[selectedIcon] || selectedIcon})
            </span>
          </div>
        )}

        {/* 清空按钮 */}
        {value && (
          <div style={{ marginTop: 12, textAlign: "center" }}>
            <a onClick={handleClear} style={{ color: "var(--theme-error, #ba3630)" }}>
              清空已选图标
            </a>
          </div>
        )}
      </Modal>
    </>
  );
};

export default IconSelect;
