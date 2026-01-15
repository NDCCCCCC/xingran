/**
 * 地址输入组件
 * 支持地址解析，输入详细地址后自动获取经纬度坐标
 */

import { useState, useCallback, useEffect } from 'react';
import { Input, Select, Button, Space, Alert, Spin } from 'antd';
import { SearchOutlined, EnvironmentOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { useGeocoding } from '../hooks/useGeocoding';
import { HUBEI_CITIES } from '@/utils/baidu-map';

const { TextArea } = Input;

export interface AddressValue {
  cityCode: string;
  cityName: string;
  address: string;
  longitude?: number;
  latitude?: number;
}

interface AddressInputProps {
  value?: AddressValue;
  onChange?: (value: AddressValue) => void;
  disabled?: boolean;
}

const AddressInput: React.FC<AddressInputProps> = ({ value, onChange, disabled }) => {
  const [localValue, setLocalValue] = useState<AddressValue>(
    value || {
      cityCode: '',
      cityName: '',
      address: '',
    }
  );
  const { geocode, loading, error } = useGeocoding();

  useEffect(() => {
    if (value) {
      setLocalValue(value);
    }
  }, [value]);

  const handleCityChange = useCallback(
    (cityCode: string) => {
      const city = HUBEI_CITIES[cityCode as keyof typeof HUBEI_CITIES];
      const newValue: AddressValue = {
        ...localValue,
        cityCode,
        cityName: city?.name || '',
        longitude: undefined,
        latitude: undefined,
      };
      setLocalValue(newValue);
      onChange?.(newValue);
    },
    [localValue, onChange]
  );

  const handleAddressChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
      const newValue: AddressValue = {
        ...localValue,
        address: e.target.value,
        longitude: undefined,
        latitude: undefined,
      };
      setLocalValue(newValue);
      onChange?.(newValue);
    },
    [localValue, onChange]
  );

  const handleResolve = useCallback(async () => {
    if (!localValue.address || localValue.address.trim().length === 0) {
      return;
    }

    const result = await geocode(localValue.address, localValue.cityName);

    if (result) {
      const newValue: AddressValue = {
        ...localValue,
        longitude: result.longitude,
        latitude: result.latitude,
        address: result.formattedAddress || localValue.address,
      };
      setLocalValue(newValue);
      onChange?.(newValue);
    }
  }, [localValue, geocode, onChange]);

  const hasCoordinates = localValue.longitude && localValue.latitude;

  return (
    <div>
      <Space direction="vertical" style={{ width: '100%' }}>
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
          >
            {Object.entries(HUBEI_CITIES).map(([code, city]) => (
              <Select.Option key={code} value={code}>
                {city.name}
              </Select.Option>
            ))}
          </Select>
        </Space>

        <div>
          <div style={{ marginBottom: 8 }}>
            <EnvironmentOutlined /> 详细地址：
          </div>
          <Space.Compact style={{ width: '100%' }}>
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
              style={{ height: 'auto' }}
            >
              解析
            </Button>
          </Space.Compact>
        </div>

        {error && (
          <Alert
            message="解析失败"
            description={error.message}
            type="error"
            showIcon
            closable
          />
        )}

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
                  经度：{localValue.longitude?.toFixed(6)}，纬度：{localValue.latitude?.toFixed(6)}
                </div>
                {localValue.address && (
                  <div style={{ fontSize: 12, color: '#666', marginTop: 4 }}>
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
