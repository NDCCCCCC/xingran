/**
 * 百度地图地址解析 Hook
 * 通过后端 API 调用百度地图 Geocoding API 将地址转换为经纬度坐标
 * 避免前端直接调用百度地图 API 时的 CORS 问题
 * 支持缓存以减少 API 调用
 */

import { useState, useCallback } from "react";
import { post } from "@/lib/api";
import { getGeocodingCache } from "@/utils/geocodingCache";

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
      if (!address || address.trim().length === 0) {
        setError({ code: -1, message: "地址不能为空" });
        return null;
      }

      setLoading(true);
      setError(null);

      try {
        // 生成缓存键
        const cacheKey = geocodingCache.generateKey({
          type: "geocode",
          address: address.trim(),
          city: city || "",
        });

        // 尝试从缓存获取
        const cached = geocodingCache.get(cacheKey);
        if (cached) {
          setLoading(false);
          return cached;
        }

        // 缓存未命中，调用后端 API
        const response = await post<GeocodingResponse>("/ops/building/geocode", {
          address: address.trim(),
          city: city || "",
        });

        if (response.data) {
          const geocodingResult: GeocodingResult = {
            longitude: response.data.longitude,
            latitude: response.data.latitude,
            formattedAddress: response.data.formattedAddress || address,
            province: response.data.province || "",
            city: response.data.city || "",
            district: response.data.district || "",
            street: response.data.street || "",
          };

          // 存入缓存
          geocodingCache.set(cacheKey, geocodingResult);

          return geocodingResult;
        } else {
          // 解析失败
          setError({
            code: -1,
            message: "地址解析失败",
          });
          return null;
        }
      } catch (err) {
        setError({
          code: -1,
          message: err instanceof Error ? err.message : "网络请求失败",
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
      setLoading(true);
      setError(null);

      try {
        // 生成缓存键（坐标保留6位小数，精度约0.1米）
        const cacheKey = geocodingCache.generateKey({
          type: "reverse_geocode",
          lng: longitude.toFixed(6),
          lat: latitude.toFixed(6),
        });

        // 尝试从缓存获取
        const cached = geocodingCache.get(cacheKey);
        if (cached) {
          setLoading(false);
          return cached;
        }

        // TODO: 后端暂时不支持逆地址解析，这里保留接口但返回空值
        // 如果需要，可以在后端添加逆地址解析的 API 端点
        setLoading(false);
        return null;
      } catch (err) {
        setError({
          code: -1,
          message: err instanceof Error ? err.message : "网络请求失败",
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

// 后端 API 响应类型
interface GeocodingResponse {
  longitude: number;
  latitude: number;
  formattedAddress?: string;
  province?: string;
  city?: string;
  district?: string;
  street?: string;
}
