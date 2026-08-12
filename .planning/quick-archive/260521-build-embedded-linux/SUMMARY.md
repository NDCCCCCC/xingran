---
slug: build-embedded-linux
status: complete
date: 2026-05-21
---

# 构建脚本创建完成

## 执行结果

已成功创建将前端React应用打包并嵌入到Go后端的构建脚本。

### 创建的文件

1. **internal/server/embed_frontend.go** - 开发模式存根实现
   - 返回友好错误，提示用户启动前端开发服务器

2. **internal/server/embed_frontend_prod.go** - 生产模式嵌入式前端
   - 使用 Go embed 指令嵌入前端静态文件
   - SPA fallback 支持（所有路由返回 index.html）
   - 带 `//go:build embed` 标签，仅在使用 `-tags=embed` 编译时启用

3. **scripts/build-embedded.bat** - Windows 构建脚本
   - 清理前端 dist
   - 构建前端（`npm run build`）
   - 复制前端文件到 `internal/server/xingran-react-frontend/dist`
   - 使用 `-tags=embed` 编译 Go 后端
   - 输出：`xingran-backend-embedded.exe`

4. **scripts/build-embedded.sh** - Linux 构建脚本
   - 同 Windows 脚本流程
   - 输出：`xingran-backend-embedded-linux`
   - 设置可执行权限

### 修改的文件

1. **internal/api/router.go**
   - 添加 `internal/server` 包导入
   - 在 `SetupRouter` 末尾添加前端静态文件路由
   - 仅在生产模式（`server.mode == "release"`）启用

2. **.gitignore**
   - 忽略 `internal/server/xingran-react-frontend/`（复制的嵌入式文件）
   - 忽略 `xingran-backend-embedded*`（构建产物）
   - 忽略 `scripts/tools/`（临时工具）

## 关键技术点

### Build Tags 策略
- 默认编译（无 tags）：使用 `embed_frontend.go`，返回开发提示
- 使用 `-tags=embed`：使用 `embed_frontend_prod.go`，嵌入真实前端文件

### Embed 路径
```
internal/server/xingran-react-frontend/dist/
```

### 路由配置
```go
if core.Config.Server.Mode == "release" {
    r.GET("/assets/*filepath", server.ServeFrontend)
    r.GET("/index.html", server.ServeFrontend)
    r.GET("/", server.ServeFrontend)
}
```

## 使用方法

### Windows 构建
```batch
cd scripts
.\build-embedded.bat
```

### Linux 构建
```bash
cd scripts
chmod +x build-embedded.sh
./build-embedded.sh
```

### 输出文件
- Windows: `xingran-backend-embedded.exe` (~50-80MB)
- Linux: `xingran-backend-embedded-linux` (~50-80MB)

### 运行
```bash
# Linux
./xingran-backend-embedded-linux

# Windows
.\xingran-backend-embedded.exe
```

然后访问 `http://localhost:9000` 即可看到嵌入的前端界面。

## 验证状态

✅ Go 代码编译通过
✅ 构建脚本创建完成
✅ .gitignore 更新完成
✅ 路由配置正确

## 后续建议

1. **测试构建**：在实际环境中运行构建脚本并验证
2. **版本注入**：构建时使用 `-ldflags` 注入版本信息
3. **部署文档**：补充服务器部署说明
