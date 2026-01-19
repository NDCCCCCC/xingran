/**
 * 百度地图地址解析 Hook
 * 使用百度地图 Geocoding API 将地址转换为经纬度坐标
 * 支持缓存以减少 API 调用
 */

import { useState, useCallback } from 'react';
import { getGeocodingCache } from '@/utils/geocodingCache';

// 获取百度地图 AK
const BAIDU_MAP_AK = import.meta.env.VITE_BAIDU_MAP_AK || '';

// 缓存实例
const geocodingCache = getGeocodingCache<GeocodingResult>();

export interface GeocodingResult {
  longitude: number;
  latitude: number;
  formattedAddress: string;
  province: string;
  city: string;
  district: string;
  street: string;
}

export interface GeocodingError {
  code: number;
  message: string;
}

export const useGeocoding = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<GeocodingError | null>(null);

  /**
   * 地址解析：将详细地址转换为经纬度坐标
   * @param address 详细地址，如 "武汉市洪山区珞喻路1037号"
   * @param city 城市名称，如 "武汉市"（可选，提高准确度）
   */
  const geocode = useCallback(
    async (address: string, city?: string): Promise<GeocodingResult | null> => {
      if (!BAIDU_MAP_AK) {
        setError({
          code: -1,
          message: '未配置百度地图 AK，请在环境变量中设置 VITE_BAIDU_MAP_AK',
        });
        return null;
      }

      if (!address || address.trim().length === 0) {
        setError({ code: -1, message: '地址不能为空' });
        return null;
      }

      setLoading(true);
      setError(null);

      try {
        // 生成缓存键
        const cacheKey = geocodingCache.generateKey({
          type: 'geocode',
          address: address.trim(),
          city: city || '',
        });

        // 尝试从缓存获取
        const cached = geocodingCache.get(cacheKey);
        if (cached) {
          setLoading(false);
          return cached;
        }

        // 缓存未命中，调用 API
        // 构建请求 URL
        const params = new URLSearchParams({
          address: address.trim(),
          output: 'json',
          ak: BAIDU_MAP_AK,
        });

        if (city) {
          params.append('city', city);
        }

        const url = `https://api.map.baidu.com/geocoding/v3/?${params.toString()}`;

        const response = await fetch(url);
        const data = await response.json();

        if (data.status === 0) {
          // 解析成功
          const result = data.result;
          const location = result.location;

          const geocodingResult: GeocodingResult = {
            longitude: location.lng,
            latitude: location.lat,
            formattedAddress: result.formatted_address || address,
            province: result.addressComponent?.province || '',
            city: result.addressComponent?.city || '',
            district: result.addressComponent?.district || '',
            street: result.addressComponent?.street || '',
          };

          // 存入缓存
          geocodingCache.set(cacheKey, geocodingResult);

          return geocodingResult;
        } else {
          // 解析失败
          setError({
            code: data.status,
            message: data.message || '地址解析失败',
          });
          return null;
        }
      } catch (err) {
        setError({
          code: -1,
          message: err instanceof Error ? err.message : '网络请求失败',
        });
        return null;
      } finally {
        setLoading(false);
      }
    },
    []
  );

  /**
   * 逆地址解析：将经纬度坐标转换为详细地址
   * @param longitude 经度
   * @param latitude 纬度
   */
  const reverseGeocode = useCallback(
    async (longitude: number, latitude: number): Promise<GeocodingResult | null> => {
      if (!BAIDU_MAP_AK) {
        setError({
          code: -1,
          message: '未配置百度地图 AK，请在环境变量中设置 VITE_BAIDU_MAP_AK',
        });
        return null;
      }

      setLoading(true);
      setError(null);

      try {
        // 生成缓存键（坐标保留6位小数，精度约0.1米）
        const cacheKey = geocodingCache.generateKey({
          type: 'reverse_geocode',
          lng: longitude.toFixed(6),
          lat: latitude.toFixed(6),
        });

        // 尝试从缓存获取
        const cached = geocodingCache.get(cacheKey);
        if (cached) {
          setLoading(false);
          return cached;
        }

        // 缓存未命中，调用 API
        // 构建请求 URL
        const params = new URLSearchParams({
          location: `${latitude},${longitude}`,
          output: 'json',
          ak: BAIDU_MAP_AK,
        });

        const url = `https://api.map.baidu.com/reverse_geocoding/v3/?${params.toString()}`;

        const response = await fetch(url);
        const data = await response.json();

        if (data.status === 0) {
          // 解析成功
          const result = data.result;

          const geocodingResult: GeocodingResult = {
            longitude,
            latitude,
            formattedAddress: result.formatted_address,
            province: result.addressComponent?.province || '',
            city: result.addressComponent?.city || '',
            district: result.addressComponent?.district || '',
            street: result.addressComponent?.street || '',
          };

          // 存入缓存
          geocodingCache.set(cacheKey, geocodingResult);

          return geocodingResult;
        } else {
          // 解析失败
          setError({
            code: data.status,
            message: data.message || '逆地址解析失败',
          });
          return null;
        }
      } catch (err) {
        setError({
          code: -1,
          message: err instanceof Error ? err.message : '网络请求失败',
        });
        return null;
      } finally {
        setLoading(false);
      }
    },
    []
  );

  return {
    geocode,
    reverseGeocode,
    loading,
    error,
  };
};
