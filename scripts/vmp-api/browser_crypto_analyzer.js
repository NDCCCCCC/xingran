/**
 * VMP密码加密分析工具 - 浏览器Console脚本
 *
 * 使用方法:
 * 1. 打开VMP登录页面 (https://10.62.0.72/login.pl)
 * 2. 打开开发者工具 (F12)
 * 3. 切换到Console标签
 * 4. 复制并粘贴此脚本
 * 5. 查看输出的加密相关信息
 */

(function() {
    console.log('='.repeat(60));
    console.log('VMP密码加密分析工具');
    console.log('='.repeat(60));

    // 1. 查找全局加密相关对象
    console.log('\n[步骤1] 查找全局加密对象:');
    const encryptKeywords = ['encrypt', 'rsa', 'sm2', 'crypto', 'jsencrypt'];
    const foundObjects = {};

    for (const keyword of encryptKeywords) {
        const matches = Object.keys(window).filter(k => k.toLowerCase().includes(keyword));
        if (matches.length > 0) {
            foundObjects[keyword] = matches;
            console.log(`  ${keyword}:`, matches);
        }
    }

    // 2. 查找JSEncrypt实例
    console.log('\n[步骤2] 查找JSEncrypt:');
    if (window.JSEncrypt) {
        console.log('  ✓ 找到 JSEncrypt');
        console.log('  版本:', JSEncrypt.prototype.version || '未知');

        // 查找页面中的实例
        const instances = [];
        let element = document.querySelector('*');
        while (element) {
            for (const key in element) {
                if (element[key] && element[key].constructor &&
                    element[key].constructor.name === 'JSEncrypt') {
                    instances.push(element[key]);
                }
            }
            element = element.parentElement;
        }

        if (instances.length > 0) {
            console.log('  找到', instances.length, '个JSEncrypt实例');
            const firstInstance = instances[0];

            // 尝试获取公钥
            try {
                const publicKey = firstInstance.getPublicKey();
                if (publicKey) {
                    console.log('  ✓ 成功获取公钥:');
                    console.log(publicKey.substring(0, 100) + '...');
                    window.vmpPublicKey = publicKey;
                    console.log('  公钥已保存到: window.vmpPublicKey');
                }
            } catch (e) {
                console.log('  ✗ 无法获取公钥:', e.message);
            }
        }
    } else {
        console.log('  ✗ 未找到 JSEncrypt');
    }

    // 3. 查找crypto-js
    console.log('\n[步骤3] 查找crypto-js:');
    if (window.CryptoJS) {
        console.log('  ✓ 找到 CryptoJS');
        console.log('  可用算法:', Object.keys(CryptoJS.algo || {}));
    } else {
        console.log('  ✗ 未找到 CryptoJS');
    }

    // 4. 查找其他加密库
    console.log('\n[步骤4] 查找其他加密库:');
    const libs = ['Forge', 'OpenSSL', 'sjcl', 'nacl'];
    for (const lib of libs) {
        if (window[lib]) {
            console.log(`  ✓ 找到 ${lib}`);
        }
    }

    // 5. 监听表单提交
    console.log('\n[步骤5] 设置密码拦截:');
    const passwordInput = document.querySelector('input[type="password"]');
    if (passwordInput) {
        console.log('  ✓ 找到密码输入框');

        // 保存原始表单提交
        const form = passwordInput.form;
        if (form) {
            form.addEventListener('submit', function(e) {
                const plainPassword = passwordInput.value;
                console.log('\n' + '='.repeat(60));
                console.log('[捕获] 表单提交事件');
                console.log('='.repeat(60));
                console.log('明文密码:', plainPassword);

                // 查找加密后的密码
                const inputs = form.querySelectorAll('input[type="hidden"]');
                for (const input of inputs) {
                    if (input.name === 'password' || input.name.includes('password')) {
                        console.log('加密密码:', input.value);
                        console.log('密文长度:', input.value.length);
                        window.vmpEncryptedPassword = input.value;
                        console.log('✓ 加密密码已保存到: window.vmpEncryptedPassword');
                    }
                }

                // 不阻止提交，让登录继续
            }, true);
            console.log('  ✓ 已设置表单拦截器');
            console.log('  提示: 输入用户名密码后点击登录，Console会显示加密信息');
        }
    } else {
        console.log('  ✗ 未找到密码输入框');
    }

    // 6. 测试加密功能
    console.log('\n[步骤6] 测试加密功能:');
    window.testVMPLogin = function(username, password) {
        console.log('\n' + '='.repeat(60));
        console.log('[测试] 模拟登录加密');
        console.log('='.repeat(60));
        console.log('用户名:', username);
        console.log('密码:', password);

        // 尝试使用找到的加密实例
        if (window.vmpPublicKey && window.JSEncrypt) {
            const encrypt = new JSEncrypt();
            encrypt.setPublicKey(window.vmpPublicKey);
            const encrypted = encrypt.encrypt(password);
            console.log('\n加密结果:');
            console.log(encrypted);
            console.log('\n密文长度:', encrypted ? encrypted.length : 0);
            console.log('密文格式:', /^[0-9a-fA-F]+$/.test(encrypted) ? '十六进制' : '其他');
            return encrypted;
        } else {
            console.log('\n✗ 无法测试: 未找到公钥或JSEncrypt');
            return null;
        }
    };
    console.log('  ✓ 测试函数已创建: window.testVMPLogin("admin", "password")');

    // 7. 导出结果
    console.log('\n[步骤7] 导出结果:');
    window.exportVMPInfo = function() {
        const info = {
            publicKey: window.vmpPublicKey || null,
            encryptedPassword: window.vmpEncryptedPassword || null,
            hasJSEncrypt: !!window.JSEncrypt,
            hasCryptoJS: !!window.CryptoJS,
            timestamp: new Date().toISOString()
        };
        console.log('\n导出信息:');
        console.log(JSON.stringify(info, null, 2));
        return info;
    };
    console.log('  ✓ 导出函数已创建: window.exportVMPInfo()');

    console.log('\n' + '='.repeat(60));
    console.log('分析完成!');
    console.log('='.repeat(60));
    console.log('\n使用说明:');
    console.log('1. 输入用户名和密码，点击登录');
    console.log('2. 查看Console中的加密信息');
    console.log('3. 使用 window.exportVMPInfo() 导出结果');
    console.log('4. 或使用 window.testVMPLogin("admin", "test") 测试加密');

})();
