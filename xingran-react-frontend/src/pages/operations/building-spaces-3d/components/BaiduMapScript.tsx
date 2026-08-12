/**
 * 百度地图脚本加载器
 * 支持普通版本和 WebGL 版本
 */

let loadGLPromise: Promise<void> | null = null;
let loadNormalPromise: Promise<void> | null = null;

/**
 * 加载百度地图 WebGL 版本脚本
 */
export function loadBaiduMapGLScript(ak: string): Promise<void> {
  if (loadGLPromise) {
    return loadGLPromise;
  }

  loadGLPromise = new Promise((resolve, reject) => {
    if (window.BMapGL) {
      resolve();
      return;
    }

    const script = document.createElement("script");
    script.type = "text/javascript";
    script.src = `https://api.map.baidu.com/api?type=webgl&v=1.0&ak=${ak}&callback=initBMapGL`;
    script.async = true;
    script.onerror = () => reject(new Error("百度地图GL脚本加载失败"));

    window.initBMapGL = () => {
      resolve();
    };

    document.head.appendChild(script);
  });

  return loadGLPromise;
}

/**
 * 加载百度地图普通版本脚本
 */
export function loadBaiduMapScript(ak: string): Promise<void> {
  if (loadNormalPromise) {
    return loadNormalPromise;
  }

  loadNormalPromise = new Promise((resolve, reject) => {
    if (window.BMap) {
      resolve();
      return;
    }

    const script = document.createElement("script");
    script.type = "text/javascript";
    script.src = `https://api.map.baidu.com/api?v=3.0&ak=${ak}&callback=init`;
    script.async = true;
    script.onerror = () => reject(new Error("百度地图脚本加载失败"));

    window.init = () => {
      resolve();
    };

    document.head.appendChild(script);
  });

  return loadNormalPromise;
}

/**
 * 检查百度地图是否已加载（任一版本）
 */
export function isBaiduMapLoaded(): boolean {
  return !!window.BMap || !!window.BMapGL;
}

/**
 * 检查百度地图 GL 版本是否已加载
 */
export function isBaiduMapGLLoaded(): boolean {
  return !!window.BMapGL;
}
