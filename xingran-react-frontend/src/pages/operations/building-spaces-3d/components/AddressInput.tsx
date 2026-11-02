/**
 * 地址输入组件
 * 支持地址解析，输入详细地址后自动获取经纬度坐标
 */

import { useState, useCallback, useEffect } from "react";
import { Button, Space, Alert, Spin } from "antd";
import { SearchOutlined, EnvironmentOutlined, CheckCircleOutlined } from "@ant-design/icons";
import { useGeocoding } from "../hooks/useGeocoding";
import { HUBEI_CITIES } from "@/utils/baidu-map";
import type { AddressValue } from "./types";
// Phase 71 (W3)：裸 Select/TextArea → 契约组件（只换 UI 组件，不碰地址解析逻辑）。
// 城市下拉原 optionFilterProp="children" + options 形态 → 交给 AppSelect 统一下沉
// optionFilterProp="label"（cityOptions 的 label 真实存在，过滤语义等价）；
// 详细地址是受控输入框 → 显式 allowClear={false}，保持原有「无清空按钮」交互。
import { AppSelect, AppTextArea } from "@/components/ui";

interface AddressInputProps {
  value?: AddressValue;
  onChange?: (value: AddressValue) => void;
  disabled?: boolean;
}

const DEFAULT_VALUE: AddressValue = {
  cityCode: "",
  cityName: "",
  address: "",
};

const AddressInput: React.FC<AddressInputProps> = ({ value, onChange, disabled }) => {
  const [localValue, setLocalValue] = useState<AddressValue>(value || DEFAULT_VALUE);
  const { geocode, loading, error } = useGeocoding();

  useEffect(() => {
    if (value) {
      const timer = setTimeout(() => setLocalValue(value), 0);
      return () => clearTimeout(timer);
    }
  }, [value]);

  const updateValue = useCallback(
    (updates: Partial<AddressValue>) => {
      const newValue: AddressValue = {
        ...localValue,
        ...updates,
        longitude: updates.longitude ?? undefined,
        latitude: updates.latitude ?? undefined,
      };
      setLocalValue(newValue);
      onChange?.(newValue);
    },
    [localValue, onChange]
  );

  const handleCityChange = useCallback(
    (cityCode: string) => {
      const city = HUBEI_CITIES[cityCode as keyof typeof HUBEI_CITIES];
      updateValue({
        cityCode,
        cityName: city?.name || "",
        longitude: undefined,
        latitude: undefined,
      });
    },
    [updateValue]
  );

  const handleAddressChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      updateValue({
        address: e.target.value,
        longitude: undefined,
        latitude: undefined,
      });
    },
    [updateValue]
  );

  const handleResolve = useCallback(async () => {
    if (!localValue.address?.trim()) {
      return;
    }

    const result = await geocode(localValue.address, localValue.cityName);

    if (result) {
      updateValue({
        longitude: result.longitude,
        latitude: result.latitude,
        address: result.formattedAddress || localValue.address,
      });
    }
  }, [localValue, geocode, updateValue]);

  const hasCoordinates = localValue.longitude && localValue.latitude;
  const cityOptions = Object.entries(HUBEI_CITIES).map(([code, city]) => ({
    label: city.name,
    value: code,
  }));

  return (
    <div>
      <Space orientation="vertical" style={{ width: "100%" }}>
        {/* 城市选择 */}
        <Space>
          <EnvironmentOutlined />
          <span>城市：</span>
          <AppSelect
            allowClear
            style={{ width: 200 }}
            placeholder="请选择城市"
            value={localValue.cityCode || undefined}
            // AppSelect 的 onChange 值为 unknown；原 allowClear 清空时同样落到
            // handleCityChange 的空值路径（city 查不到 → cityName ""），收敛成 "" 等价
            onChange={(value) => handleCityChange((value as string) ?? "")}
            disabled={disabled}
            showSearch
            options={cityOptions}
            onSearch={() => {}}
          />
        </Space>

        {/* 地址输入 */}
        <div>
          <div style={{ marginBottom: 8 }}>
            <EnvironmentOutlined /> 详细地址：
          </div>
          <Space.Compact style={{ width: "100%" }}>
            <AppTextArea
              rows={2}
              allowClear={false}
              placeholder="请输入详细地址，如：武汉市洪山区珞喻路1037号"
              value={localValue.address}
              onChange={handleAddressChange}
              disabled={disabled}
            />
            <Button
              type="primary"
              icon={loading ? <Spin size="small" /> : <SearchOutlined />}
              onClick={handleResolve}
              disabled={disabled || !localValue.address}
              style={{ height: "auto" }}
            >
              解析
            </Button>
          </Space.Compact>
        </div>

        {/* 错误提示 */}
        {error && (
          <Alert title="解析失败" description={error.message} type="error" showIcon closable />
        )}

        {/* 成功提示 */}
        {hasCoordinates && (
          <Alert
            title={
              <Space>
                <CheckCircleOutlined />
                <span>地址解析成功</span>
              </Space>
            }
            description={
              <div>
                <div>
                  经度：{localValue.longitude?.toFixed(6)}， 纬度：{localValue.latitude?.toFixed(6)}
                </div>
                {localValue.address && (
                  <div
                    style={{
                      fontSize: 12,
                      color: "var(--theme-text-tertiary, #666)",
                      marginTop: 4,
                    }}
                  >
                    {localValue.address}
                  </div>
                )}
              </div>
            }
            type="success"
            showIcon
            closable
          />
        )}
      </Space>
    </div>
  );
};

export default AddressInput;
