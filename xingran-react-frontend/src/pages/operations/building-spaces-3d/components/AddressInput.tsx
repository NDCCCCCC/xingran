/**
 * 地址输入组件
 * 支持地址解析，输入详细地址后自动获取经纬度坐标
 */

import { useState, useCallback, useEffect } from "react";
import { Input, Select, Button, Space, Alert, Spin } from "antd";
import { SearchOutlined, EnvironmentOutlined, CheckCircleOutlined } from "@ant-design/icons";
import { useGeocoding } from "../hooks/useGeocoding";
import { HUBEI_CITIES } from "@/utils/baidu-map";
import type { AddressValue } from "./types";

const { TextArea } = Input;

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

  const updateValue = useCallback((updates: Partial<AddressValue>) => {
    const newValue: AddressValue = {
      ...localValue,
      ...updates,
      longitude: updates.longitude ?? undefined,
      latitude: updates.latitude ?? undefined,
    };
    setLocalValue(newValue);
    onChange?.(newValue);
  }, [localValue, onChange]);

  const handleCityChange = useCallback((cityCode: string) => {
    const city = HUBEI_CITIES[cityCode as keyof typeof HUBEI_CITIES];
    updateValue({
      cityCode,
      cityName: city?.name || "",
      longitude: undefined,
      latitude: undefined,
    });
  }, [updateValue]);

  const handleAddressChange = useCallback((e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    updateValue({
      address: e.target.value,
      longitude: undefined,
      latitude: undefined,
    });
  }, [updateValue]);

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
      <Space direction="vertical" style={{ width: "100%" }}>
        {/* 城市选择 */}
        <Space>
          <EnvironmentOutlined />
          <span>城市：</span>
          <Select
            style={{ width: 200 }}
            placeholder="请选择城市"
            value={localValue.cityCode || undefined}
            onChange={handleCityChange}
            disabled={disabled}
            showSearch
            optionFilterProp="children"
            options={cityOptions}
           onSearch={() => {}}/>
        </Space>

        {/* 地址输入 */}
        <div>
          <div style={{ marginBottom: 8 }}>
            <EnvironmentOutlined /> 详细地址：
          </div>
          <Space.Compact style={{ width: "100%" }}>
            <TextArea
              rows={2}
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
          <Alert
            message="解析失败"
            description={error.message}
            type="error"
            showIcon
            closable
          />
        )}

        {/* 成功提示 */}
        {hasCoordinates && (
          <Alert
            message={
              <Space>
                <CheckCircleOutlined />
                <span>地址解析成功</span>
              </Space>
            }
            description={
              <div>
                <div>
                  经度：{localValue.longitude?.toFixed(6)}，
                  纬度：{localValue.latitude?.toFixed(6)}
                </div>
                {localValue.address && (
                  <div style={{ fontSize: 12, color: "var(--theme-text-tertiary, #666)", marginTop: 4 }}>
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
