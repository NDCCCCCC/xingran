# 后端技术选型和API设计文档

## 1. 技术选型

### 1.1 核心框架选择

#### 1.1.1 Go语言选择理由

| 优势 | 说明 | 对比Python |
|------|------|------------|
| **高性能** | 原生并发支持，编译型语言 | 性能提升5-10倍 |
| **内存效率** | 低内存占用，GC优化 | 内存占用减少50% |
| **部署简单** | 单一可执行文件 | 无需环境依赖 |
| **并发模型** | Goroutine轻量级协程 | 替代复杂的异步框架 |
| **开发效率** | 简洁语法，快速编译 | 编译速度更快 |

**版本选择：Go 1.23+**
- 最新的稳定版本（1.23.2），性能持续优化
- 增强的语言特性（range over integer、type inference改进）
- 更好的工具链支持（go vet、go fmt改进）
- 性能提升，编译速度和运行时性能优化
- 长期支持版本 |

#### 1.1.2 Web框架对比

| 框架 | 性能 | 生态 | 文档 | 学习成本 | 推荐度 |
|------|------|------|------|---------|--------|
| **Gin** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| Echo | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| Fiber | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ |
| Chi | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ |

**最终选择：Gin框架**
- 最活跃的社区和最丰富的中间件生态
- 性能优异，文档完善
- 学习成本低，开发效率高

### 1.2 数据库技术选型

#### 1.2.1 PostgreSQL vs MySQL

| 特性 | PostgreSQL 18.1 | MySQL 8.0 | 优势方 |
|------|---------------|-----------|--------|
| **性能** | I/O子系统优化，3倍性能提升 | 简单查询性能好 | PostgreSQL |
| **功能** | JSON支持、窗口函数、CTE | JSON支持有限 | PostgreSQL |
| **事务** | 完全的ACID支持 | ACID支持 | PostgreSQL |
| **扩展性** | 丰富的扩展插件 | 扩展性一般 | PostgreSQL |
| **生态** | 日益壮大 | 最广泛的关系型数据库 | MySQL |
| **运维** | 版本升级兼容性改进 | 运维成熟 | PostgreSQL |

**最终选择：PostgreSQL 18.1+**
- I/O子系统优化，性能提升3倍
- 新增索引类型，提升查询性能
- 版本升级兼容性改进，升级更平滑
- 增强的JSON支持和并发性能优化
- 完整的事务支持，数据一致性有保障

#### 1.2.2 ORM框架选择

| 框架 | 性能 | 功能 | 学习成本 | 社区活跃度 | 推荐度 |
|------|------|------|---------|------------|--------|
| **GORM** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| Squirrel | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| sqlx | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ |
| sqlboiler | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐ | ⭐⭐⭐ |

**最终选择：GORM v1.25**
- 功能最完善的ORM框架
- 支持自动迁移、关联、事务
- 活跃的社区，持续的更新维护

### 1.3 缓存技术选型

#### 1.3.1 Redis选择理由

| 特性 | Redis | Memcached | 优势方 |
|------|-------|-----------|--------|
| **数据结构** | 丰富的数据类型 | 仅简单键值 | Redis |
| **持久化** | RDB/AOF持久化 | 无持久化 | Redis |
| **集群** | 原生集群支持 | 客户端分片 | Redis |
| **功能** | 发布订阅、Lua脚本 | 基础功能 | Redis |
| **性能** | 单线程高性能 | 多线程高性能 | 平手 |

**版本选择：Redis 7.4+**
- 性能优化，内存效率提升35%
- 增强的集群功能和自动故障转移
- 更好的持久化性能（AOF重写优化）
- 支持客户端缓存功能
- 更丰富的数据类型操作
- 改进的内存分析和诊断工具

#### 1.3.2 Go Redis客户端

| 客户端 | 性能 | 功能 | 社区 | 推荐度 |
|--------|------|------|------|--------|
| **go-redis** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| redigo | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ |
| rueidis | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐ |

**最终选择：go-redis v9**
- 官方维护，功能最全面
- 支持集群、哨兵、管道
- 丰富的中间件支持

### 1.4 消息队列选型

| 队列 | 性能 | 可靠性 | 复杂度 | 运维成本 | 适用场景 |
|------|------|--------|--------|---------|---------|
| **NATS** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 内部消息、低延迟 |
| RabbitMQ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | 复杂路由、事务 |
| Kafka | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐ | 大数据、流处理 |
| Redis | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | 简单队列、缓存 |

**最终选择：NATS JetStream**
- 极高的性能和低延迟
- 云原生设计，运维简单
- 支持持久化和ACK机制
- 轻量级部署，资源占用少

### 1.5 监控和日志

| 组件 | 选择 | 版本 | 用途 |
|------|------|------|------|
| **监控** | Prometheus | 2.40+ | 指标收集和存储 |
| **可视化** | Grafana | 9.3+ | 监控仪表盘 |
| **日志** | ELK Stack | 8.5+ | 日志收集和分析 |
| **链路追踪** | Jaeger | 1.40+ | 分布式追踪 |
| **APM** | New Relic | 最新版 | 应用性能监控 |

## 2. 项目结构设计

### 2.1 目录结构

```
ruoyi-next/
├── cmd/                          # 应用入口
│   ├── server/                   # Web服务
│   │   └── main.go              # 服务启动入口
│   ├── worker/                   # 后台任务
│   │   └── main.go              # 任务处理器入口
│   └── migrate/                  # 数据库迁移
│       └── main.go              # 迁移工具
├── internal/                     # 内部代码
│   ├── api/                      # API层
│   │   ├── v1/                   # API v1版本
│   │   │   ├── auth/            # 认证相关API
│   │   │   │   ├── auth.go      # 认证接口
│   │   │   │   └── router.go    # 路由注册
│   │   │   ├── system/          # 系统管理API
│   │   │   │   ├── user/        # 用户管理
│   │   │   │   ├── role/        # 角色管理
│   │   │   │   ├── menu/        # 菜单管理
│   │   │   │   ├── dept/        # 部门管理
│   │   │   │   └── config/      # 配置管理
│   │   │   ├── monitor/         # 监控相关API
│   │   │   │   ├── server/      # 服务监控
│   │   │   │   ├── cache/       # 缓存监控
│   │   │   │   └── log/         # 日志监控
│   │   │   └── tool/            # 系统工具API
│   │   │       ├── gen/         # 代码生成
│   │   │       └── build/       # 在线构建
│   │   └── middleware/          # 中间件
│   │       ├── auth.go         # 认证中间件
│   │       ├── logger.go       # 日志中间件
│   │       ├── rate.go         # 限流中间件
│   │       └── cors.go         # 跨域中间件
│   ├── service/                 # 业务服务层
│   │   ├── auth/               # 认证服务
│   │   │   ├── service.go      # 认证业务逻辑
│   │   │   └── interface.go    # 服务接口定义
│   │   ├── system/             # 系统服务
│   │   │   ├── user/           # 用户服务
│   │   │   ├── role/           # 角色服务
│   │   │   ├── menu/           # 菜单服务
│   │   │   └── dept/           # 部门服务
│   │   └── monitor/            # 监控服务
│   ├── repository/              # 数据访问层
│   │   ├── user/               # 用户数据访问
│   │   │   ├── repo.go         # 用户仓储
│   │   │   └── interface.go    # 仓储接口
│   │   ├── role/               # 角色数据访问
│   │   └── cache/              # 缓存仓储
│   ├── model/                  # 数据模型
│   │   ├── entity/             # 数据库实体
│   │   │   ├── user.go         # 用户实体
│   │   │   ├── role.go         # 角色实体
│   │   │   └── menu.go         # 菜单实体
│   │   ├── dto/                # 数据传输对象
│   │   │   ├── auth/           # 认证DTO
│   │   │   └── system/         # 系统DTO
│   │   └── vo/                 # 视图对象
│   ├── domain/                 # 领域模型
│   │   ├── auth/               # 认证领域
│   │   │   ├── user.go         # 用户领域模型
│   │   │   └── permission.go   # 权限领域模型
│   │   └── system/             # 系统领域
│   └── infrastructure/         # 基础设施
│       ├── config/             # 配置管理
│       ├── database/           # 数据库连接
│       ├── cache/              # 缓存连接
│       ├── mq/                 # 消息队列
│       ├── logger/             # 日志系统
│       └── crypto/             # 加密工具
├── pkg/                        # 公共包
│   ├── cache/                  # 缓存封装
│   │   ├── redis.go           # Redis封装
│   │   └── memory.go          # 内存缓存
│   ├── config/                 # 配置管理
│   │   ├── config.go          # 配置结构
│   │   └── loader.go          # 配置加载器
│   ├── crypto/                 # 加密工具
│   │   ├── sm2.go             # SM2加密
│   │   ├── sm3.go             # SM3摘要
│   │   ├── sm4.go             # SM4加密
│   │   └── jwt.go             # JWT工具
│   ├── logger/                 # 日志工具
│   │   ├── zap.go             # Zap封装
│   │   └── middleware.go      # 日志中间件
│   ├── utils/                  # 通用工具
│   │   ├── string.go          # 字符串工具
│   │   ├── time.go            # 时间工具
│   │   ├── id.go              # ID生成
│   │   └── page.go            # 分页工具
│   ├── validator/              # 验证器
│   │   ├── validator.go       # 验证器封装
│   │   └── rules.go           # 验证规则
│   ├── response/               # 响应封装
│   │   ├── response.go        # 响应结构
│   │   └── code.go            # 状态码
│   └── errors/                 # 错误处理
│       ├── errors.go          # 错误定义
│       └── handler.go         # 错误处理器
├── scripts/                    # 脚本文件
│   ├── build.sh               # 构建脚本
│   ├── deploy.sh              # 部署脚本
│   ├── init-db.sql            # 数据库初始化
│   └── seed-data.sql          # 种子数据
├── configs/                    # 配置文件
│   ├── config.yaml            # 主配置
│   ├── config-dev.yaml        # 开发配置
│   ├── config-prod.yaml       # 生产配置
│   └── config-test.yaml       # 测试配置
├── migrations/                 # 数据库迁移
│   ├── migrations/            # 迁移文件
│   └── seeds/                 # 种子数据
├── docs/                       # 项目文档
├── tests/                      # 测试文件
│   ├── integration/           # 集成测试
│   ├── unit/                  # 单元测试
│   └── e2e/                   # 端到端测试
├── docker/                     # Docker相关
│   ├── Dockerfile             # Docker镜像
│   ├── docker-compose.yml     # 开发环境
│   └── docker-compose.prod.yml # 生产环境
├── deploy/                     # 部署相关
│   ├── k8s/                   # Kubernetes配置
│   ├── helm/                  # Helm Charts
│   └── terraform/             # 基础设施代码
├── go.mod                      # Go模块文件
├── go.sum                      # 依赖锁定文件
├── Makefile                    # 构建命令
├── README.md                   # 项目说明
└── CHANGELOG.md                # 变更日志
```

### 2.2 分层架构设计

#### 2.2.1 API层设计原则

1. **统一响应格式**
```go
type Response struct {
    Code    int         `json:"code"`    // 业务状态码
    Message string      `json:"message"` // 消息提示
    Data    interface{} `json:"data"`    // 响应数据
    TraceID string      `json:"trace_id"` // 链路追踪ID
}
```

2. **统一的错误处理**
```go
type ErrorResponse struct {
    Code      int       `json:"code"`
    Message   string    `json:"message"`
    Details   string    `json:"details,omitempty"`
    Timestamp time.Time `json:"timestamp"`
    TraceID   string    `json:"trace_id"`
}
```

3. **统一的参数验证**
```go
type LoginRequest struct {
    Username string `json:"username" binding:"required,min=3,max=20"`
    Password string `json:"password" binding:"required,min=6,max=20"`
    Captcha  string `json:"captcha" binding:"required,len=6"`
}
```

#### 2.2.2 服务层设计模式

```go
// 服务接口定义
type UserService interface {
    Create(ctx context.Context, req *CreateUserRequest) (*UserVO, error)
    Update(ctx context.Context, req *UpdateUserRequest) error
    Delete(ctx context.Context, id string) error
    Get(ctx context.Context, id string) (*UserVO, error)
    List(ctx context.Context, req *ListUserRequest) (*ListUserResponse, error)
}

// 服务实现
type userService struct {
    repo   repository.UserRepository
    cache  cache.UserCache
    logger logger.Logger
}

func NewUserService(
    repo repository.UserRepository,
    cache cache.UserCache,
    logger logger.Logger,
) UserService {
    return &userService{
        repo:   repo,
        cache:  cache,
        logger: logger,
    }
}
```

#### 2.2.3 数据访问层设计

```go
// 仓储接口
type UserRepository interface {
    Create(ctx context.Context, user *entity.User) error
    Update(ctx context.Context, user *entity.User) error
    Delete(ctx context.Context, id string) error
    GetByID(ctx context.Context, id string) (*entity.User, error)
    GetByUsername(ctx context.Context, username string) (*entity.User, error)
    List(ctx context.Context, filter *UserFilter) ([]*entity.User, int64, error)
}

// 仓储实现
type userRepository struct {
    db *gorm.DB
    cache cache.Cache
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
    if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
        return err
    }
    // 清理缓存
    r.cache.Delete(ctx, "user:list")
    return nil
}
```

## 3. API设计规范

### 3.1 RESTful API设计

#### 3.1.1 URL设计规范

| 资源 | HTTP方法 | URL | 说明 |
|------|----------|-----|------|
| 用户列表 | POST | /api/v1/users/list | 查询用户列表（支持分页、过滤） |
| 创建用户 | POST | /api/v1/users | 创建新用户 |
| 获取用户 | POST | /api/v1/users/{id} | 获取用户详情 |
| 更新用户 | POST | /api/v1/users/{id}/update | 更新用户信息 |
| 删除用户 | POST | /api/v1/users/{id}/delete | 删除用户 |
| 批量删除 | POST | /api/v1/users/batch-delete | 批量删除用户 |
| 用户状态 | POST | /api/v1/users/{id}/status | 更改用户状态 |

**注意：所有API请求使用POST方法，通过请求体传递参数**

#### 3.1.2 请求/响应格式

**统一请求格式**
```json
{
    "request_id": "uuid-v4",
    "timestamp": "2024-01-01T12:00:00Z",
    "data": {
        // 业务数据
    }
}
```

**成功响应格式**
```json
{
    "code": 0,
    "message": "success",
    "data": {
        // 业务数据
    },
    "timestamp": "2024-01-01T12:00:00Z",
    "request_id": "uuid-v4"
}
```

**错误响应格式**
```json
{
    "code": 10001,
    "message": "用户名或密码错误",
    "details": "用户不存在或密码不正确",
    "timestamp": "2024-01-01T12:00:00Z",
    "request_id": "uuid-v4"
}
```

### 3.2 API版本管理

#### 3.2.1 版本策略

- **URL路径版本**：`/api/v1/`, `/api/v2/`
- **向后兼容**：新版本保持对旧版本的兼容
- **废弃通知**：提前3个月通知API废弃

#### 3.2.2 版本迁移

```go
// v1版本API
func v1Handler(c *gin.Context) {
    // v1实现
}

// v2版本API
func v2Handler(c *gin.Context) {
    // v2实现，兼容v1请求
    if isV1Request(c) {
        return v1Handler(c)
    }
    // v2新功能
}
```

### 3.3 认证和授权

#### 3.3.1 JWT Token设计

```go
type JWTClaims struct {
    UserID   string   `json:"user_id"`
    Username string   `json:"username"`
    Roles    []string `json:"roles"`
    jwt.StandardClaims
}

// 生成Token
func GenerateToken(claims *JWTClaims) (string, error) {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(config.GetString("jwt.secret")))
}
```

#### 3.3.2 权限控制

```go
// 权限装饰器
func RequirePermission(permission string) gin.HandlerFunc {
    return func(c *gin.Context) {
        user := getCurrentUser(c)
        if !hasPermission(user, permission) {
            response.Error(c, response.CodeNoPermission, "无权限访问")
            c.Abort()
            return
        }
        c.Next()
    }
}

// 使用示例
router.POST("/api/v1/users",
    auth.RequireLogin(),
    auth.RequirePermission("system:user:create"),
    createUser,
)
```

### 3.4 API文档

#### 3.4.1 Swagger集成

```go
// @title RuoYi-Next API
// @version 1.0
// @description 企业级权限管理系统API
// @termsOfService http://swagger.io/terms/

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
    r := gin.New()

    // Swagger文档路由
    r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

    // API路由
    v1 := r.Group("/api/v1")
    {
        auth := v1.Group("/auth")
        {
            auth.POST("/login", authLogin)
            auth.POST("/logout", authLogout)
            auth.POST("/refresh", authRefresh)
        }
    }
}
```

#### 3.4.2 API文档规范

```go
// @Summary 用户登录
// @Description 用户登录接口，返回JWT Token
// @Tags 认证管理
// @Accept json
// @Produce json
// @Param request body LoginRequest true "登录请求"
// @Success 200 {object} Response{data=LoginResponse}
// @Failure 400 {object} Response
// @Failure 401 {object} Response
// @Router /api/v1/auth/login [post]
func authLogin(c *gin.Context) {
    // 实现
}
```

## 4. 数据库设计

### 4.1 数据库连接配置

```go
type DatabaseConfig struct {
    Host         string `yaml:"host"`
    Port         int    `yaml:"port"`
    User         string `yaml:"user"`
    Password     string `yaml:"password"`
    DBName       string `yaml:"db_name"`
    SSLMode      string `yaml:"ssl_mode"`
    MaxOpenConns int    `yaml:"max_open_conns"`
    MaxIdleConns int    `yaml:"max_idle_conns"`
    MaxLifetime  int    `yaml:"max_lifetime"`
}

// 初始化数据库
func InitDB(cfg *DatabaseConfig) (*gorm.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
        cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode,
    )

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
        NamingStrategy: schema.NamingStrategy{
            SingularTable: true,
            NoLowerCase:   true,
        },
    })

    if err != nil {
        return nil, err
    }

    sqlDB, err := db.DB()
    if err != nil {
        return nil, err
    }

    sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
    sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
    sqlDB.SetConnMaxLifetime(time.Duration(cfg.MaxLifetime) * time.Second)

    return db, nil
}
```

### 4.2 数据模型设计

```go
// 基础模型
type BaseModel struct {
    ID        string         `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
    CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
    DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
    CreatedBy string         `json:"created_by" gorm:"size:64"`
    UpdatedBy string         `json:"updated_by" gorm:"size:64"`
    Version  int            `json:"version" gorm:"default:1"` // 乐观锁
}

// 用户模型
type User struct {
    BaseModel
    Username    string         `json:"username" gorm:"uniqueIndex;size:64;not null"`
    Password    string         `json:"-" gorm:"size:128;not null"`
    Salt        string         `json:"-" gorm:"size:64;not null"`
    Nickname    string         `json:"nickname" gorm:"size:64"`
    Email       string         `json:"email" gorm:"size:128"`
    Phone       string         `json:"phone" gorm:"size:32"`
    Avatar      string         `json:"avatar" gorm:"size:255"`
    Gender      int            `json:"gender" gorm:"default:0"` // 0:未知 1:男 2:女
    Status      int            `json:"status" gorm:"default:1"` // 0:禁用 1:启用
    DeptID      string         `json:"dept_id" gorm:"size:64"`
    Dept        *Dept          `json:"dept,omitempty" gorm:"foreignKey:DeptID"`
    Roles       []Role         `json:"roles,omitempty" gorm:"many2many:user_roles;"`
    LoginIP     string         `json:"login_ip" gorm:"size:64"`
    LoginTime   *time.Time     `json:"login_time"`
    Remark      string         `json:"remark" gorm:"size:512"`
}
```

### 4.3 数据库迁移

```go
// 迁移文件示例
package migrations

import (
    "gorm.io/gorm"
)

func Migrate_20240101000001(db *gorm.DB) error {
    // 创建用户表
    err := db.AutoMigrate(&User{})
    if err != nil {
        return err
    }

    // 创建索引
    if err := db.Exec("CREATE INDEX idx_user_username ON users (username)").Error; err != nil {
        return err
    }

    // 插入初始数据
    admin := &User{
        Username: "admin",
        Nickname: "超级管理员",
        Email:    "admin@ruoyi.com",
        Status:   1,
    }
    admin.Password = hashPassword("admin123", admin.Salt)

    return db.Create(admin).Error
}
```

## 5. 缓存设计

### 5.1 缓存架构

```go
type Cache interface {
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    Increment(ctx context.Context, key string) (int64, error)
}

// 多级缓存
type MultiLevelCache struct {
    l1Cache Cache  // 本地缓存
    l2Cache Cache  // Redis缓存
}

func (c *MultiLevelCache) Get(ctx context.Context, key string) (string, error) {
    // 先查本地缓存
    if value, err := c.l1Cache.Get(ctx, key); err == nil {
        return value, nil
    }

    // 查Redis缓存
    value, err := c.l2Cache.Get(ctx, key)
    if err != nil {
        return "", err
    }

    // 回写本地缓存
    c.l1Cache.Set(ctx, key, value, 5*time.Minute)

    return value, nil
}
```

### 5.2 缓存策略

```go
type CacheStrategy interface {
    GetKey(params ...interface{}) string
    GetExpiration() time.Duration
    ShouldRefresh(data interface{}) bool
}

// 用户缓存策略
type UserCacheStrategy struct{}

func (s *UserCacheStrategy) GetKey(params ...interface{}) string {
    userID := params[0].(string)
    return fmt.Sprintf("user:%s", userID)
}

func (s *UserCacheStrategy) GetExpiration() time.Duration {
    return 30 * time.Minute
}

func (s *UserCacheStrategy) ShouldRefresh(data interface{}) bool {
    // 根据数据判断是否需要刷新
    return true
}
```

### 5.3 缓存使用

```go
type UserRepository struct {
    db    *gorm.DB
    cache Cache
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*User, error) {
    // 尝试从缓存获取
    cacheKey := fmt.Sprintf("user:%s", id)
    var user User

    if err := r.cache.Get(ctx, cacheKey, &user); err == nil {
        return &user, nil
    }

    // 从数据库查询
    if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
        return nil, err
    }

    // 写入缓存
    r.cache.Set(ctx, cacheKey, &user, 30*time.Minute)

    return &user, nil
}
```

## 6. 中间件设计

### 6.1 认证中间件

```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            response.Error(c, response.CodeUnauthorized, "未提供认证信息")
            c.Abort()
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := validateToken(tokenString)
        if err != nil {
            response.Error(c, response.CodeUnauthorized, "无效的Token")
            c.Abort()
            return
        }

        // 将用户信息存入上下文
        c.Set("user_id", claims.UserID)
        c.Set("username", claims.Username)
        c.Set("roles", claims.Roles)

        c.Next()
    }
}
```

### 6.2 日志中间件

```go
func LoggerMiddleware() gin.HandlerFunc {
    return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        logger.Info("Request",
            logger.String("method", param.Method),
            logger.String("path", param.Path),
            logger.Int("status", param.StatusCode),
            logger.Duration("latency", param.Latency),
            logger.String("client_ip", param.ClientIP),
            logger.String("user_agent", param.Request.UserAgent()),
            logger.String("request_id", param.Request.Header.Get("X-Request-ID")),
        )
        return ""
    })
}
```

### 6.3 限流中间件

```go
func RateLimitMiddleware() gin.HandlerFunc {
    limiter := rate.NewLimiter(rate.Limit(100), 200) // 100req/s, burst 200

    return func(c *gin.Context) {
        if !limiter.Allow() {
            response.Error(c, response.CodeTooManyRequests, "请求过于频繁")
            c.Abort()
            return
        }
        c.Next()
    }
}
```

### 6.4 CORS中间件

```go
func CORSMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Access-Control-Allow-Origin", "*")
        c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
        c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
        c.Header("Access-Control-Max-Age", "86400")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }

        c.Next()
    }
}
```

## 7. 配置管理

### 7.1 配置结构

```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Database DatabaseConfig `yaml:"database"`
    Redis    RedisConfig    `yaml:"redis"`
    JWT      JWTConfig      `yaml:"jwt"`
    Logger   LoggerConfig   `yaml:"logger"`
    App      AppConfig      `yaml:"app"`
}

type ServerConfig struct {
    Port         int           `yaml:"port"`
    Mode         string        `yaml:"mode"` // debug, release, test
    ReadTimeout  time.Duration `yaml:"read_timeout"`
    WriteTimeout time.Duration `yaml:"write_timeout"`
    IdleTimeout  time.Duration `yaml:"idle_timeout"`
}
```

### 7.2 配置加载

```go
func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var config Config
    err = yaml.Unmarshal(data, &config)
    if err != nil {
        return nil, err
    }

    // 环境变量覆盖
    if os.Getenv("PORT") != "" {
        config.Server.Port = parseInt(os.Getenv("PORT"))
    }

    return &config, nil
}
```

## 8. 错误处理

### 8.1 错误定义

```go
type BusinessError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}

func (e *BusinessError) Error() string {
    return e.Message
}

// 预定义错误
var (
    ErrUserNotFound      = &BusinessError{Code: 10001, Message: "用户不存在"}
    ErrPasswordIncorrect = &BusinessError{Code: 10002, Message: "密码错误"}
    ErrUserDisabled      = &BusinessError{Code: 10003, Message: "用户已被禁用"}
)
```

### 8.2 错误处理中间件

```go
func ErrorMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                var businessErr *BusinessError
                if errors.As(err, &businessErr) {
                    response.Error(c, businessErr.Code, businessErr.Message)
                } else {
                    logger.Error("Panic recovered",
                        logger.Any("error", err),
                        logger.String("stack", debug.Stack()),
                    )
                    response.Error(c, response.CodeInternalServer, "内部服务器错误")
                }
                c.Abort()
            }
        }()
        c.Next()
    }
}
```

## 9. 测试策略

### 9.1 单元测试

```go
func TestUserService_Create(t *testing.T) {
    // 准备
    mockRepo := &MockUserRepository{}
    mockCache := &MockUserCache{}
    service := NewUserService(mockRepo, mockCache, logger.New())

    req := &CreateUserRequest{
        Username: "testuser",
        Password: "password123",
        Email:    "test@example.com",
    }

    // 设置Mock期望
    mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.User")).Return(nil)

    // 执行
    result, err := service.Create(context.Background(), req)

    // 断言
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, req.Username, result.Username)
    mockRepo.AssertExpectations(t)
}
```

### 9.2 集成测试

```go
func TestUserAPI_Create(t *testing.T) {
    // 测试数据库
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)

    // 测试服务器
    router := setupTestRouter(db)

    // 测试数据
    req := CreateUserRequest{
        Username: "testuser",
        Password: "password123",
        Email:    "test@example.com",
    }

    // 发送请求
    w := httptest.NewRecorder()
    body, _ := json.Marshal(req)
    httpReq, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(body))
    httpReq.Header.Set("Content-Type", "application/json")

    router.ServeHTTP(w, httpReq)

    // 验证响应
    assert.Equal(t, 200, w.Code)

    var response Response
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.Equal(t, 0, response.Code)
}
```

## 10. 性能优化

### 10.1 数据库优化

```go
// 批量插入优化
func (r *userRepository) BatchCreate(ctx context.Context, users []*entity.User) error {
    batchSize := 100
    for i := 0; i < len(users); i += batchSize {
        end := i + batchSize
        if end > len(users) {
            end = len(users)
        }

        batch := users[i:end]
        if err := r.db.WithContext(ctx).CreateInBatches(batch, len(batch)).Error; err != nil {
            return err
        }
    }
    return nil
}

// 查询优化
func (r *userRepository) ListWithPreload(ctx context.Context, filter *UserFilter) ([]*entity.User, error) {
    var users []*entity.User
    query := r.db.WithContext(ctx).Model(&entity.User{})

    // 预加载关联数据
    query = query.Preload("Dept").Preload("Roles")

    // 条件过滤
    if filter.DeptID != "" {
        query = query.Where("dept_id = ?", filter.DeptID)
    }
    if filter.Status != nil {
        query = query.Where("status = ?", *filter.Status)
    }

    err := query.Find(&users).Error
    return users, err
}
```

### 10.2 并发优化

```go
// 并发控制
type WorkerPool struct {
    workers    int
    jobQueue   chan Job
    workerPool chan chan Job
    quit       chan bool
}

func NewWorkerPool(workers int) *WorkerPool {
    return &WorkerPool{
        workers:    workers,
        jobQueue:   make(chan Job, 1000),
        workerPool: make(chan chan Job, workers),
        quit:       make(chan bool),
    }
}

func (p *WorkerPool) Start() {
    for i := 0; i < p.workers; i++ {
        worker := NewWorker(p.workerPool)
        worker.Start()
    }

    go p.dispatch()
}

func (p *WorkerPool) Submit(job Job) {
    p.jobQueue <- job
}
```

## 11. 监控和指标

### 11.1 Prometheus指标

```go
var (
    // HTTP请求计数器
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status"},
    )

    // HTTP请求延迟
    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request latency",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path"},
    )

    // 数据库查询指标
    dbQueryDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "db_query_duration_seconds",
            Help:    "Database query latency",
            Buckets: prometheus.DefBuckets,
        },
        []string{"operation", "table"},
    )
)

// 指标中间件
func PrometheusMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        c.Next()

        duration := time.Since(start).Seconds()

        httpRequestsTotal.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
            strconv.Itoa(c.Writer.Status()),
        ).Inc()

        httpRequestDuration.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
        ).Observe(duration)
    }
}
```

### 11.2 健康检查

```go
type HealthChecker struct {
    db    *gorm.DB
    redis redis.Cmdable
}

func (h *HealthChecker) Check(ctx context.Context) *HealthStatus {
    status := &HealthStatus{
        Status:    "healthy",
        Timestamp: time.Now(),
        Checks:    make(map[string]CheckResult),
    }

    // 数据库检查
    if sqlDB, err := h.db.DB(); err == nil {
        if err := sqlDB.Ping(); err == nil {
            status.Checks["database"] = CheckResult{Status: "healthy"}
        } else {
            status.Checks["database"] = CheckResult{Status: "unhealthy", Error: err.Error()}
            status.Status = "unhealthy"
        }
    }

    // Redis检查
    if _, err := h.redis.Ping(ctx).Result(); err == nil {
        status.Checks["redis"] = CheckResult{Status: "healthy"}
    } else {
        status.Checks["redis"] = CheckResult{Status: "unhealthy", Error: err.Error()}
        status.Status = "unhealthy"
    }

    return status
}
```

## 12. 总结

本后端技术选型和API设计文档详细描述了RuoYi-Next项目的技术架构、实现规范和最佳实践。主要特点包括：

1. **现代化技术栈**：Go + Gin + GORM + PostgreSQL + Redis
2. **清晰的分层架构**：API层、服务层、仓储层分离
3. **统一的API设计**：RESTful风格，统一的请求响应格式
4. **完善的中间件**：认证、日志、限流、错误处理
5. **高效的缓存策略**：多级缓存，合理的缓存策略
6. **全面的测试覆盖**：单元测试、集成测试、API测试
7. **完整的监控体系**：Prometheus + Grafana监控
8. **国密安全支持**：SM2/SM3/SM4加密算法

通过这些设计，系统将具备高性能、高可用、易扩展的特点，能够满足企业级应用的需求。