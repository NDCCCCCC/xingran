# VMP密码加密分析

## 加密特征

### 已知信息
- **明文密码**: `sangfor@2020` (10字符)
- **密文长度**: 512字符 (十六进制)
- **密文格式**: 纯十六进制字符串

### 密文示例
```
295141767757762b08c2002a7a65fe61baf7ee99aa7b907d23684f4803404a0efef8a33b9cd42e5b6a874694b603b6b2b86ef87a2a166dca169e5ae24a65f48ec959d209b7d04ee1c214502f0514009e91ac0c1f508dc1412ff3a1ca374599a9c2142c96f19c5312d02f10bc5b5ab0d5444db322dc45ec17ddb5f222b35d51f8d661e83bc872687b911c6ddbf17858183d90250865b79dd1ab26324287c8f788688a5e2c3047c557c4bfd68270e60807a09f641ca0f36fd1975f652d2ead3fa748c114dc8116fd22f073a28ce68fdb7660b6ec2c2bc782c2fa2776ec9db169c14704bf62cb7fbce52d15dd6a0474dc6ae8ca08a56055026ce1b6fae71b0b5d
```

## 可能的加密方式

### 1. RSA-2048 加密 (最可能)

**特征匹配**:
- 512个十六进制字符 = 2048位 = 256字节
- RSA-2048密文长度正好是256字节
- 转换为十六进制字符串表示: 256 * 2 = 512字符

**加密流程**:
```javascript
// 伪代码
function encryptPassword(password, publicKey) {
    // 1. 将密码转换为字节
    const passwordBytes = utf8.encode(password);
    
    // 2. 使用RSA公钥加密
    const encrypted = rsa.encrypt(passwordBytes, publicKey);
    
    // 3. 转换为十六进制字符串
    return bytesToHex(encrypted);
}
```

**Python实现方案**:
```python
from Crypto.PublicKey import RSA
from Crypto.Cipher import PKCS1_v1_5
import binascii

def encrypt_password_rsa(password, public_key_pem):
    """
    使用RSA-2048公钥加密密码
    
    Args:
        password: 明文密码
        public_key_pem: PEM格式的公钥
    
    Returns:
        十六进制加密字符串
    """
    # 加载公钥
    rsa_key = RSA.importKey(public_key_pem)
    cipher = PKCS1_v1_5.new(rsa_key)
    
    # 加密密码
    encrypted = cipher.encrypt(password.encode('utf-8'))
    
    # 转换为十六进制
    return binascii.hexlify(encrypted).decode('ascii')

# 示例使用
public_key = """-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
-----END PUBLIC KEY-----"""
encrypted = encrypt_password_rsa("sangfor@2020", public_key)
```

### 2. SM2 国密加密 (深信服可能使用)

**特征匹配**:
- SM2椭圆曲线加密也能产生512字符的密文
- 深信服作为国内厂商，可能使用国密算法

**加密流程**:
```python
# 需要安装: pip install gmssl
from gmssl import sm2
import binascii

def encrypt_password_sm2(password, public_key_hex):
    """
    使用SM2公钥加密密码
    
    Args:
        password: 明文密码
        public_key_hex: 十六进制公钥
    
    Returns:
        十六进制加密字符串
    """
    sm2_crypt = sm2.CryptSM2(
        public_key=public_key_hex, 
        private_key='',
        mode=sm2.EncryptMode
    )
    
    encrypted = sm2_crypt.encrypt(password.encode('utf-8'))
    return encrypted.hex()
```

### 3. 混合加密 (RSA + AES)

**可能的流程**:
1. 生成随机AES密钥 (32字节)
2. 使用AES加密密码
3. 使用RSA加密AES密钥
4. 组合: RSA加密的密钥 + AES加密的密码

## 如何获取公钥

### 方法1: 分析前端JS代码

1. 打开VMP登录页面
2. 打开浏览器开发者工具 (F12)
3. 查看Sources标签中的JavaScript文件
4. 搜索关键词: `encrypt`, `password`, `RSA`, `publicKey`
5. 找到加密函数和公钥定义

**可能的关键词**:
- `RSA.encrypt`
- `publicKey`
- `encryptPassword`
- `doEncrypt`
- `JSEncrypt`

### 方法2: 网络抓包

1. 打开开发者工具的Network标签
2. 在登录页面输入用户名和密码
3. 查看POST /vapi/extjs/access/ticket请求
4. 查看是否有之前的API调用返回公钥

### 方法3: 浏览器Console调试

```javascript
// 在登录页面的Console中执行
// 查找全局对象中的加密相关函数
console.log(Object.keys(window).filter(k => k.toLowerCase().includes('encrypt')));

// 或者查找RSA相关
console.log(Object.keys(window).filter(k => k.toLowerCase().includes('rsa')));

// 查找JSEncrypt实例
console.log(window.encrypter);
console.log(window.encrypter.getPublicKey());
```

## 实现建议

### 方案A: 获取真实公钥 (推荐)

1. 从VMP服务器的前端代码中提取公钥
2. 实现相应的加密算法
3. 更新vmp_login.py脚本

### 方案B: 使用浏览器自动化

```python
from selenium import webdriver
from selenium.webdriver.common.by import By

def get_encrypted_password_via_browser(username, password):
    """
    使用浏览器自动化获取加密后的密码
    
    优点: 不需要分析加密算法
    缺点: 需要安装浏览器驱动，速度较慢
    """
    driver = webdriver.Chrome()
    driver.get("https://10.62.0.72/login.pl")
    
    # 输入用户名和密码
    driver.find_element(By.ID, "username").send_keys(username)
    driver.find_element(By.ID, "password").send_keys(password)
    
    # 拦截网络请求获取加密后的密码
    # (需要配置浏览器代理或使用mitmproxy)
    
    driver.quit()
```

### 方案C: 使用现有Cookie (临时方案)

```bash
# 跳过加密，直接使用从浏览器获取的Cookie
set VMP_AUTH_COOKIE=<从浏览器复制的Cookie>
python verify_vmp_api.py
```

## 下一步行动建议

1. **分析前端代码**: 登录 https://10.62.0.72，查看Sources中的JS文件
2. **查找公钥**: 搜索`publicKey`、`RSA`等关键词
3. **测试加密**: 找到加密函数后，在Console中测试
4. **实现Python版本**: 将前端加密逻辑移植到Python

## 需要的信息

要实现完整的加密，我需要：

1. **前端JS代码**: 登录页面加载的JavaScript文件
2. **公钥**: 用于加密的RSA/SM2公钥
3. **加密库**: 前端使用的加密库名称 (JSEncrypt, crypto-js, 等)

**请提供以下信息以继续分析**:
- 登录页面的URL: https://10.62.0.72/login.pl
- 是否可以访问前端代码？
- 是否有浏览器的开发者工具访问权限？
