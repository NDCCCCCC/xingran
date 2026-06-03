# VMP密码加密解析指南

## 快速开始

### 步骤1: 访问VMP登录页面

打开浏览器，访问: `https://10.62.0.72/login.pl`

### 步骤2: 打开开发者工具

按 `F12` 键，或右键点击页面 → "检查"

### 步骤3: 运行分析脚本

1. 切换到 `Console` 标签
2. 复制 `browser_crypto_analyzer.js` 的内容
3. 粘贴到Console中
4. 按 `Enter` 执行

### 步骤4: 分析输出

脚本会自动分析并输出：
- 全局加密对象
- JSEncrypt实例
- 公钥信息
- 加密库列表

### 步骤5: 捕获加密密码

1. 在登录页面输入用户名和密码
2. 点击"登录"按钮
3. Console会显示加密前后的密码
4. 查看输出的 `window.vmpEncryptedPassword`

## 手动查找公钥

如果自动分析失败，可以手动查找：

### 方法1: 查看页面源代码

1. 右键点击页面 → "查看网页源代码"
2. 搜索关键词: `publicKey`, `-----BEGIN PUBLIC KEY-----`, `rsa`

### 方法2: 查看Network请求

1. 切换到 `Network` 标签
2. 刷新页面
3. 查看JS文件 (`.js`)
4. 逐个查看包含 `encrypt` 或 `crypto` 的文件

### 方法3: 直接查找公钥

在Console中执行:

```javascript
// 查找所有script标签
document.querySelectorAll('script').forEach((script, i) => {
    console.log(`Script ${i}:`, script.src);
});

// 如果是内联脚本，查看内容
document.querySelectorAll('script:not([src])').forEach((script, i) => {
    console.log(`Inline Script ${i}:`, script.textContent.substring(0, 200));
});
```

## 常见加密库识别

### JSEncrypt

```javascript
// 特征代码
var rsa = new JSEncrypt();
rsa.setPublicKey(publicKey);
var encrypted = rsa.encrypt(password);
```

### crypto-js

```javascript
// 特征代码
var encrypted = CryptoJS.AES.encrypt(password, key);
var encrypted = CryptoJS.RSA.encrypt(password, publicKey);
```

### Forge

```javascript
// 特征代码
var keyPair = forge.pki.rsa.generateKeyPair(2048);
var encrypted = keyPair.publicKey.encrypt(password);
```

## 实现Python加密

### RSA加密示例

```python
from Crypto.PublicKey import RSA
from Crypto.Cipher import PKCS1_v1_5
import base64

def encrypt_password(password, public_key_pem):
    """
    RSA加密密码
    
    Args:
        password: 明文密码
        public_key_pem: PEM格式公钥
    """
    # 加载公钥
    rsa_key = RSA.import_key(public_key_pem)
    cipher = PKCS1_v1_5.new(rsa_key)
    
    # 加密
    encrypted = cipher.encrypt(password.encode('utf-8'))
    
    # 转换为十六进制
    return encrypted.hex()

# 使用示例
public_key = """-----BEGIN PUBLIC KEY-----
从浏览器获取的公钥
-----END PUBLIC KEY-----"""

encrypted = encrypt_password("sangfor@2020", public_key)
print(f"加密结果: {encrypted}")
print(f"密文长度: {len(encrypted)}")
```

## 临时解决方案

在未实现加密前，可以使用以下方法：

### 方法1: 使用现有Cookie

```bash
# 从浏览器获取Cookie
set VMP_AUTH_COOKIE=<LoginAuthCookie值>
python verify_vmp_api.py
```

### 方法2: 使用浏览器拦截

```python
# 使用Selenium自动化浏览器获取加密密码
from selenium import webdriver
from selenium.webdriver.common.by import By

def get_encrypted_password(username, password):
    driver = webdriver.Chrome()
    driver.get("https://10.62.0.72/login.pl")
    
    # 输入凭证
    driver.find_element(By.ID, "username").send_keys(username)
    driver.find_element(By.ID, "password").send_keys(password)
    
    # 拦截网络请求
    # (需要配置mitmproxy或类似工具)
    
    # 提交表单
    # driver.find_element(By.TAG_NAME, "form").submit()
    
    encrypted = None  # 从拦截的请求中获取
    driver.quit()
    return encrypted
```

## 下一步

1. **运行分析脚本**: 获取加密相关信息
2. **识别加密算法**: 确定使用的加密库和方式
3. **提取公钥**: 获取用于加密的公钥
4. **实现Python版本**: 移植到vmp_login.py
5. **测试验证**: 确保加密结果与前端一致

## 故障排查

### 问题1: 找不到公钥

**可能原因**:
- 公钥通过AJAX动态加载
- 公钥硬编码在混淆的JS中

**解决方案**:
- 使用Network标签查看API响应
- 使用美化工具处理混淆的JS代码

### 问题2: 加密结果不一致

**可能原因**:
- 使用了不同的填充方式
- 字符编码不一致

**解决方案**:
- 检查填充方式 (PKCS1_v1_5, OAEP, etc.)
- 确保使用UTF-8编码

### 问题3: 脚本无法执行

**可能原因**:
- 浏览器安全策略限制
- 页面使用了Content Security Policy

**解决方案**:
- 在Console中允许脚本执行
- 使用浏览器扩展执行自定义脚本
