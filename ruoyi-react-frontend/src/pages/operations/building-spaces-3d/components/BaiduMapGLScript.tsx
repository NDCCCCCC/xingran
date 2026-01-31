/**
 * 百度地图 WebGL 版本脚本加载器
 * 支持 BMapGL (WebGL) 和 BMap (普通版本)
 */

let loadGLPromise: Promise<void> | null = null;
let loadNormalPromise: Promise<void> | null = null;

/**
 * 加载百度地图 WebGL 版本脚本
 */
export const loadBaiduMapGLScript = (ak: string): Promise<void> => {
  if (loadGLPromise) {
    return loadGLPromise;
  }

  loadGLPromise = new Promise((resolve, reject) => {
    // 检查是否已加载 BMapGL
    if ((window as any).BMapGL) {
      resolve();
      return;
    }

    const script = document.createElement('script');
    script.type = 'text/javascript';
    // 使用 WebGL 版本 API
    script.src = `https://api.map.baidu.com/api?type=webgl&v=1.0&ak=${ak}&callback=initBMapGL`;
    script.async = true;
    script.onerror = () => {
      reject(new Error('百度地图GL脚本加载失败'));
    };

    // 设置全局回调函数
    (window as any).initBMapGL = () => {
      console.log('✅ 百度地图GL脚本加载成功');
      resolve();
    };

    document.head.appendChild(script);
  });

  return loadGLPromise;
};

/**
 * 加载百度地图普通版本脚本（向后兼容）
 */
export const loadBaiduMapScript = (ak: string): Promise<void> => {
  if (loadNormalPromise) {
    return loadNormalPromise;
  }

  loadNormalPromise = new Promise((resolve, reject) => {
    // 检查是否已加载 BMap
    if ((window as any).BMap) {
      resolve();
      return;
    }

    const script = document.createElement('script');
    script.type = 'text/javascript';
    script.src = `https://api.map.baidu.com/api?v=3.0&ak=${ak}&callback=init`;
    script.async = true;
    script.onerror = () => {
      reject(new Error('百度地图脚本加载失败'));
    };

    // 设置全局回调函数
    (window as any).init = () => {
      console.log('✅ 百度地图脚本加载成功');
      resolve();
    };

    document.head.appendChild(script);
  });

  return loadNormalPromise;
};

/**
 * 检查百度地图是否已加载
 */
export const isBaiduMapLoaded = (): boolean => {
  return !!(window as any).BMap || !!(window as any).BMapGL;
};

/**
 * 检查百度地图GL版本是否已加载
 */
export const isBaiduMapGLLoaded = (): boolean => {
  return !!(window as any).BMapGL;
};
