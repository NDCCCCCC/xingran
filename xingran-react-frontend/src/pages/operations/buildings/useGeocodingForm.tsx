/**
 * 楼宇表单地理编码状态管理 Hook
 */

import { useState, useCallback } from "react";
import { useGeocoding } from "@/pages/operations/building-spaces-3d/hooks/useGeocoding";

interface GeocodingResult {
  longitude?: number;
  latitude?: number;
  formattedAddress?: string;
}

interface GeocodingFormState {
  geocodingLoading: boolean;
  geocodingResult: GeocodingResult | null;
  geocodingWarning: string | null;
  handleGeocode: (address: string) => Promise<void>;
  resetGeocodingState: () => void;
  setGeocodingResultFromRecord: (record: {
    longitude?: number;
    latitude?: number;
    address?: string;
  }) => void;
}

export function useGeocodingForm(): GeocodingFormState {
  const [geocodingLoading, setGeocodingLoading] = useState(false);
  const [geocodingResult, setGeocodingResult] = useState<GeocodingResult | null>(null);
  const [geocodingWarning, setGeocodingWarning] = useState<string | null>(null);

  const { geocode } = useGeocoding();

  const handleGeocode = useCallback(
    async (address: string) => {
      if (!address?.trim()) {
        setGeocodingResult(null);
        setGeocodingWarning(null);
        return;
      }

      setGeocodingLoading(true);
      setGeocodingWarning(null);

      try {
        const result = await geocode(address);

        if (result) {
          setGeocodingResult({
            longitude: result.longitude,
            latitude: result.latitude,
            formattedAddress: result.formattedAddress,
          });
          setGeocodingWarning(null);
        } else {
          setGeocodingWarning("地址解析失败，将保存不含经纬度的信息");
          setGeocodingResult(null);
        }
      } catch (_error) {
        setGeocodingWarning("地址解析出错，将保存不含经纬度的信息");
        setGeocodingResult(null);
      } finally {
        setGeocodingLoading(false);
      }
    },
    [geocode]
  );

  const resetGeocodingState = useCallback(() => {
    setGeocodingResult(null);
    setGeocodingWarning(null);
    setGeocodingLoading(false);
  }, []);

  const setGeocodingResultFromRecord = useCallback(
    (record: { longitude?: number; latitude?: number; address?: string }) => {
      if (record?.longitude && record?.latitude) {
        setGeocodingResult({
          longitude: record.longitude,
          latitude: record.latitude,
        });
      }
    },
    []
  );

  return {
    geocodingLoading,
    geocodingResult,
    geocodingWarning,
    handleGeocode,
    resetGeocodingState,
    setGeocodingResultFromRecord,
  };
}
