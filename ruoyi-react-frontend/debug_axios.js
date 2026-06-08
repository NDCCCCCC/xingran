// Axios 请求拦截调试脚本
// 在浏览器控制台中执行此代码

(function() {
    // 找到 axios 并添加拦截器
    const axiosInterceptors = window.axios || window.__axios__;

    // 添加请求拦截器
    if (window.axios) {
        window.axios.interceptors.request.use(
            (config) => {
                if (config.url && config.url.includes('/ops/asset/list')) {
                    console.log('🔍 Axios请求URL:', config.url);
                    console.log('📦 请求方法:', config.method);
                    console.log('📋 请求数据:', config.data);

                    if (config.data) {
                        try {
                            const parsedData = typeof config.data === 'string'
                                ? JSON.parse(config.data)
                                : config.data;
                            console.table(parsedData);
                            console.log('✅ deviceTypeName 值:', parsedData.deviceTypeName);
                        } catch (e) {
                            console.log('📋 原始数据:', config.data);
                        }
                    }
                }
                return config;
            },
            (error) => {
                console.error('❌ 请求错误:', error);
                return Promise.reject(error);
            }
        );
        console.log('✅ Axios拦截器已安装！现在请选择设备类型并点击搜索。');
    } else {
        console.log('❌ 未找到 axios，请尝试其他方法。');
    }

    // 备用：检查 React DevTools 或 Redux store
    console.log('💡 提示：你也可以直接在 Network 面板中查看请求详情');
})();
