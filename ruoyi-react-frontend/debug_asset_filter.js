// 在浏览器控制台中执行此脚本来调试资产筛选

// 1. 拦截 fetch 请求来查看实际发送的参数
const originalFetch = window.fetch;
window.fetch = function(...args) {
    console.log('🔍 Fetch Request:', args[0], args[1]);

    // 如果是资产列表请求
    if (args[0] && args[0].includes('/ops/asset/list')) {
        console.log('📦 Asset List Request Body:', args[1]?.body);
        try {
            const body = JSON.parse(args[1]?.body);
            console.table(body);
            console.log('✓ deviceTypeName 值:', body.deviceTypeName);
            console.log('✓ 所有参数:', Object.keys(body));
        } catch (e) {
            console.error('❌ 解析请求体失败:', e);
        }
    }

    return originalFetch.apply(this, args);
};

console.log('✅ 调试脚本已加载！现在选择设备类型并点击搜索，控制台会显示请求详情。');

// 2. 也可以手动检查表单值
// console.log('当前表单值:', document.querySelector('[name="deviceTypeName"]')?.value);
