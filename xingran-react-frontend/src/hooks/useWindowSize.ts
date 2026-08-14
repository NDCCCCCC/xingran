/**
 * useWindowSize - 窗口大小Hook
 *
 * 监听窗口大小变化
 */

import { useState, useEffect } from "react";

interface WindowSize {
  width: number;
  height: number;
}

/**
 * 窗口大小Hook
 */
export function useWindowSize(): WindowSize {
  const [windowSize, setWindowSize] = useState<WindowSize>({
    width: window.innerWidth,
    height: window.innerHeight,
  });

  useEffect(() => {
    const handleResize = () => {
      setWindowSize({
        width: window.innerWidth,
        height: window.innerHeight,
      });
    };

    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  return windowSize;
}
