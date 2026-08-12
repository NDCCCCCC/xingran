import { useState, useCallback } from "react";
import { useGeocoding } from "@/pages/operations/building-spaces-3d/hooks/useGeocoding";

interface GeocodingResult {
  longitude: number;
  latitude: number;
  formattedAddress?: string;
}

interface GeocodingState {
  loading: boolean;
  result: GeocodingResult | null;
  warning: string | null;
}

/**
 * 楼宇地理编码 Hook
 * 处理地址解析为经纬度的逻辑
 */
export function useBuildingGeocoding() {
  const { geocode } = useGeocoding();
  const [state, setState] = useState<GeocodingState>({
    loading: false,
    result: null,
    warning: null,
  });

  const resolveAddress = useCallback(async (address: string): Promise<GeocodingResult | null> => {
    if (!address?.trim()) {
      setState({ loading: false, result: null, warning: null });
      return null;
    }

    setState(prev => ({ ...prev, loading: true, warning: null }));

    try {
      const result = await geocode(address);

      if (result) {
        const coords: GeocodingResult = {
          longitude: result.longitude,
          latitude: result.latitude,
          formattedAddress: result.formattedAddress,
        };
        setState({ loading: false, result: coords, warning: null });
        return coords;
      } else {
        setState({
          loading: false,
          result: null,
          warning: "地址解析失败，将保存不含经纬度的信息",
        });
        return null;
      }
    } catch (_error) {
      setState({
        loading: false,
        result: null,
        warning: "地址解析出错，将保存不含经纬度的信息",
      });
      return null;
    }
  }, [geocode]);

  const reset = useCallback(() => {
    setState({ loading: false, result: null, warning: null });
  }, []);

  const setResult = useCallback((result: GeocodingResult | null) => {
    setState(prev => ({ ...prev, result }));
  }, []);

  return {
    ...state,
    resolveAddress,
    reset,
    setResult,
  };
}
