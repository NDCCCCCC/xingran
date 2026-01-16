/**
 * 百度地图脚本加载器
 */

let loadPromise: Promise<void> | null = null;

export const loadBaiduMapScript = (ak: string): Promise<void> => {
  if (loadPromise) {
    return loadPromise;
  }

  loadPromise = new Promise((resolve, reject) => {
    if (window.BMap) {
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
      resolve();
    };

    document.head.appendChild(script);
  });

  return loadPromise;
};

export const isBaiduMapLoaded = (): boolean => {
  return !!(window as any).BMap;
};
