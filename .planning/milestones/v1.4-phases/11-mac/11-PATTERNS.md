# Phase 11: MAC地址采集优化 - 过滤设备间互联端口 - Pattern Map

**Mapped:** 2026-05-09
**Files analyzed:** 10 (6 new, 2 modified, 2 shared patterns)
**Analogs found:** 9 / 10

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/services/lldp/lldp_service.go` | service | request-response | `internal/services/mac_collection_service.go` | exact |
| `internal/services/lldp/lldp_parser.go` | utility | transform | `internal/services/portcollection/parser.go` | exact |
| `internal/services/lldp/port_classifier.go` | utility | transform | `internal/services/mac_collection_service.go` | role-match |
| `internal/services/topology/topology_service.go` | service | CRUD | `internal/services/system/config_service.go` | role-match |
| `internal/services/topology/filter_rules.go` | service | CRUD | `internal/services/system/config_service.go` | exact |
| `internal/models/device_lldp_info.go` | model | CRUD | `internal/models/device_mac_address.go` | exact |
| `internal/models/port_classification.go` | model | CRUD | `internal/models/device_mac_address.go` | exact |
| `internal/services/mac_collection_service.go` | service | CRUD | **(same file)** | enhancement |
| `internal/device/scrapli_wrapper.go` | utility | request-response | **(same file)** | enhancement |
| `pkg/cache/lldp_cache.go` | utility | caching | `internal/services/portcollection/template_cache.go` | role-match |

## Pattern Assignments

### `internal/services/lldp/lldp_service.go` (service, request-response)

**Analog:** `internal/services/mac_collection_service.go` (lines 18-38)

**Imports pattern** (lines 1-16):
```go
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
	"gorm.io/gorm"
)
```

**Service struct pattern** (lines 18-24):
```go
// MACCollectionService MAC地址采集服务
type MACCollectionService struct {
	db            *gorm.DB
	executor      *device.DeviceExecutor
	maxConcurrent int
}

// NewMACCollectionService 创建MAC采集服务
func NewMACCollectionService(db *gorm.DB, executor *device.DeviceExecutor) *MACCollectionService {
	svc := &MACCollectionService{
		db:            db,
		executor:      executor,
		maxConcurrent: 10, // 默认值
	}
	// 从数据库加载配置
	svc.loadConfigFromDB()
	return svc
}
```

**Device collection pattern** (lines 50-90):
```go
// CollectAllDevices 采集所有设备的MAC地址表
func (s *MACCollectionService) CollectAllDevices(ctx context.Context) ([]*MACCollectionResult, error) {
	// 获取所有在线设备
	var devices []models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("status = ?", models.DeviceStatusOnline).Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("查询设备列表失败: %w", err)
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("没有在线设备")
	}

	var results []*MACCollectionResult
	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, s.maxConcurrent) // 使用动态并发数

	for _, dev := range devices {
		wg.Add(1)
		go func(device models.NetworkDevice) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 为每个设备创建独立的 context（带超时控制）
			// 这样单个设备的问题不会影响其他设备
			deviceCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()

			result := s.collectDeviceMAC(deviceCtx, &device)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(dev)
	}

	wg.Wait()

	return results, nil
}
```

**Vendor-specific command pattern** (lines 196-209):
```go
// getMACCommand 根据厂商获取MAC地址表命令
func (s *MACCollectionService) getMACCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display mac-address",
		models.VendorH3C:    "display mac-address",
		models.VendorRuijie: "show mac-address-table",
		models.VendorMaipu:  "show mac-address-table",
	}

	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "show mac-address-table" // 默认命令
}
```

**Panic recovery pattern** (lines 104-109):
```go
// 添加 panic recovery，防止 scrapligo 库的 panic 导致程序崩溃
defer func() {
	if r := recover(); r != nil {
		applogger.Infof("[MAC采集] 设备 %s 发生 panic: %v", device.DeviceName, r)
	}
}()
```

---

### `internal/services/lldp/lldp_parser.go` (utility, transform)

**Analog:** `internal/services/portcollection/parser.go`

**Template cache pattern** (from `template_cache.go`, lines 9-49):
```go
package portcollection

import (
	"sync"

	"github.com/xingran-next/xingran-go-backend/internal/templates"
)

// TemplateCache 模板缓存
type TemplateCache struct {
	cache map[string]*templates.FSM
	mu    sync.RWMutex
}

// NewTemplateCache 创建模板缓存
func NewTemplateCache() *TemplateCache {
	return &TemplateCache{
		cache: make(map[string]*templates.FSM),
	}
}

// Get 获取缓存中的模板
func (c *TemplateCache) Get(templatePath string) (*templates.FSM, error) {
	c.mu.RLock()
	fsm, ok := c.cache[templatePath]
	c.mu.RUnlock()

	if ok {
		return fsm, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查
	fsm, ok = c.cache[templatePath]
	if ok {
		return fsm, nil
	}

	// 加载并缓存模板
	fsm, err := templates.ParseTemplate(templatePath)
	if err != nil {
		return nil, err
	}

	c.cache[templatePath] = fsm
	return fsm, nil
}
```

**TextFSM template usage pattern** (from `collection.go`, lines 148-152):
```go
// 批量获取802.1X状态
dot1xMap, err := getAllDot1xStatus(wrapper, device.Vendor, s.templateCache)
if err != nil {
	applogger.Infof("[警告] 获取802.1X状态失败: %v", err)
	dot1xMap = make(map[string]Dot1xInfo)
}
```

---

### `internal/services/lldp/port_classifier.go` (utility, transform)

**Analog:** `internal/services/mac_collection_service.go` (lines 244-323)

**Port parsing logic pattern** (lines 244-323):
```go
// parseMACLine 解析MAC地址行
func (s *MACCollectionService) parseMACLine(line string, vendor models.DeviceVendor) (MACAddressEntry, bool) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return MACAddressEntry{}, false
	}

	var entry MACAddressEntry

	switch vendor {
	case models.VendorHuawei, models.VendorH3C:
		// Huawei/H3C 格式: MAC Address VLAN Interface... Type
		// 实际输出示例需要确认，可能格式为:
		// d89e.f327.2d19 100 GigabitEthernet0/1/1 Dynamic
		// 或者包含更多空格的变体
		if len(fields) >= 4 {
			entry.MACAddress = fields[0]
			// 解析VLAN ID (fields[1])
			if vlanID, err := strconv.Atoi(fields[1]); err == nil {
				entry.VLANID = vlanID
			}

			// MAC类型是最后一个字段
			entry.MACType = fields[len(fields)-1]

			// 智能识别接口名称：从fields[2]开始，到最后一个非类型字段
			// 常见的MAC类型：Dynamic, Static, Secure
			interfaceParts := []string{}
			for i := 2; i < len(fields)-1; i++ {
				// 跳过已知的MAC类型字段
				lowerField := strings.ToLower(fields[i])
				if lowerField == "dynamic" || lowerField == "static" || lowerField == "secure" {
					continue
				}
				interfaceParts = append(interfaceParts, fields[i])
			}

			if len(interfaceParts) > 0 {
				entry.InterfaceName = strings.Join(interfaceParts, " ")
			} else {
				// fallback: 使用fields[2]
				entry.InterfaceName = fields[2]
			}

			// 清理接口名称中的时间戳格式（如 "2026-5-9 0:51:07"）
			entry.InterfaceName = s.cleanTimestampFromInterface(entry.InterfaceName)
		}

	case models.VendorRuijie, models.VendorMaipu:
		// Ruijie/Maipu 格式: VLAN MAC Address Type Interface...
		if len(fields) >= 4 {
			// 解析VLAN ID (fields[0])
			if vlanID, err := strconv.Atoi(fields[0]); err == nil {
				entry.VLANID = vlanID
			}
			entry.MACAddress = fields[1]
			entry.MACType = fields[2]

			// 接口名称从fields[3]开始
			interfaceParts := []string{}
			for i := 3; i < len(fields); i++ {
				interfaceParts = append(interfaceParts, fields[i])
			}

			if len(interfaceParts) > 0 {
				entry.InterfaceName = strings.Join(interfaceParts, " ")
			} else {
				entry.InterfaceName = fields[3]
			}

			// 清理接口名称中的时间戳格式（如 "2026-5-9 0:51:07"）
			entry.InterfaceName = s.cleanTimestampFromInterface(entry.InterfaceName)
			}
		}
	}

	if entry.MACAddress != "" {
		return entry, true
	}
	return MACAddressEntry{}, false
}
```

**Interface name normalization pattern** (from `portcollection/utils.go`):
```go
// normalizeInterfaceName 标准化接口名称
// 处理不同厂商的接口名称格式差异
func normalizeInterfaceName(name string) string {
	// 转换为小写
	name = strings.ToLower(name)

	// 替换常见的缩写
	replacer := strings.NewReplacer(
		"gigabitethernet", "gi",
		"fastethernet", "fa",
		"tengigabitethernet", "te",
		"ethernet", "eth",
		" hundredgig", "hundredgige",
		"fortygig", "fortygige",
		"twenty-fivegig", "twenty-fivegige",
		"/", "",
		" ", "",
	)

	return replacer.Replace(name)
}
```

---

### `internal/services/topology/topology_service.go` (service, CRUD)

**Analog:** `internal/services/system/config_service.go` (lines 12-32)

**Service interface pattern** (lines 12-32):
```go
// ConfigService 参数配置服务接口
type ConfigService interface {
	Create(ctx context.Context, req *requests.ConfigCreateRequest) error
	Update(ctx context.Context, req *requests.ConfigUpdateRequest) error
	Delete(ctx context.Context, id string) error
	BatchDelete(ctx context.Context, ids []string) error
	GetByID(ctx context.Context, id string) (*models.Config, error)
	GetByKey(ctx context.Context, configKey string) (*models.Config, error)
	List(ctx context.Context, params requests.ConfigListParams) (*PageResult, error)
	RefreshCache(ctx context.Context) error
}

// configService 参数配置服务实现
type configService struct {
	db *gorm.DB
}

// NewConfigService 创建参数配置服务实例
func NewConfigService(db *gorm.DB) ConfigService {
	return &configService{db: db}
}
```

**CRUD create pattern** (lines 36-57):
```go
func (s *configService) Create(ctx context.Context, req *requests.ConfigCreateRequest) error {
	// 检查配置键是否已存在
	var existConfig models.Config
	if err := s.db.WithContext(ctx).Where("config_key = ?", req.ConfigKey).First(&existConfig).Error; err == nil {
		return fmt.Errorf("配置键已存在")
	}

	config := models.Config{
		ConfigName:  req.ConfigName,
		ConfigKey:   req.ConfigKey,
		ConfigValue: req.ConfigValue,
		ConfigType:  req.ConfigType,
		IsSystem:    models.ConfigIsSystem(req.IsSystem),
		Remark:      toStringPtrStr(req.Remark),
	}

	if err := s.db.WithContext(ctx).Create(&config).Error; err != nil {
		return fmt.Errorf("创建参数配置失败: %w", err)
	}

	return nil
}
```

---

### `internal/services/topology/filter_rules.go` (service, CRUD)

**Analog:** `internal/services/system/config_service.go` (same as above)

**Use the same CRUD patterns as topology_service.go since both handle database entities with similar operations.**

---

### `internal/models/device_lldp_info.go` (model, CRUD)

**Analog:** `internal/models/device_mac_address.go` (full file)

**Model struct pattern** (lines 1-43):
```go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MACType MAC地址类型枚举
type MACType string

const (
	MACTypeDynamic MACType = "dynamic" // 动态学习
	MACTypeStatic  MACType = "static"  // 静态配置
	MACTypeSecure  MACType = "secure"  // 安全MAC
)

// DeviceMACAddress 设备MAC地址模型
type DeviceMACAddress struct {
	ID            string    `gorm:"type:uuid;primary_key" json:"id"`
	DeviceID      string    `gorm:"type:uuid;not null" json:"deviceId"`
	MACAddress    string    `gorm:"size:30;not null" json:"macAddress"`
	InterfaceName string    `gorm:"size:100;not null" json:"interfaceName"`
	VLANID        *int      `json:"vlanId,omitempty"`
	MACType       MACType   `gorm:"size:20" json:"macType,omitempty"`
	CollectedAt   time.Time `gorm:"not null" json:"collectedAt"`
	CreatedAt     time.Time `json:"createdAt"`
}

// TableName 设置表名
func (DeviceMACAddress) TableName() string {
	return "sys_device_mac_address"
}

// BeforeCreate GORM 钩子：在创建记录前自动生成 UUID
func (d *DeviceMACAddress) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" || d.ID == "00000000-0000-0000-0000-000000000000" {
		d.ID = uuid.New().String()
	}
	return nil
}
```

---

### `internal/models/port_classification.go` (model, CRUD)

**Analog:** `internal/models/device_mac_address.go` (same as above)

**Use the same model pattern with enum types and UUID generation.**

---

### `internal/services/mac_collection_service.go` (service, CRUD) - ENHANCEMENT

**Analog:** Same file (enhancement pattern)

**Enhancement location:** Lines 103-186 (collectDeviceMAC method)

**Add LLDP discovery before MAC collection** (insert after line 115):
```go
// Step 1: Discover LLDP neighbors (best-effort)
lldpNeighbors := make(map[string]bool)
if lldpSvc := s.getLLDPService(); lldpSvc != nil {
	neighbors, err := lldpSvc.DiscoverNeighbors(ctx, device)
	if err != nil {
		applogger.Warnf("[MAC采集] %s: LLDP发现失败 (仅使用MAC数过滤): %v", device.DeviceName, err)
	} else {
		for iface := range neighbors {
			lldpNeighbors[iface] = true
		}
	}
}
```

**Add filtering logic** (insert after MAC parsing, before database save):
```go
// Step 5: Apply filtering rules
threshold := s.getMACThreshold(device)
filtered := 0
var filteredMACAddresses []MACAddressEntry

for _, mac := range macAddresses {
	normalizedIface := normalizeInterfaceName(mac.InterfaceName)

	// Filter: LLDP neighbor port
	if lldpNeighbors[normalizedIface] {
		filtered++
		continue
	}

	// Filter: MAC count exceeds threshold
	if macCountByInterface[normalizedIface] > threshold {
		filtered++
		continue
	}

	filteredMACAddresses = append(filteredMACAddresses, mac)
}

applogger.Infof("[MAC采集] %s: 总MAC=%d, 过滤=%d, 保留=%d",
	device.DeviceName, len(macAddresses), filtered, len(filteredMACAddresses))
```

**Add threshold configuration method** (new method):
```go
func (s *MACCollectionService) getMACThreshold(device *models.NetworkDevice) int {
	// Default thresholds by device type
	thresholds := map[models.DeviceType]int{
		models.DeviceTypeRouter:       500,
		models.DeviceTypeSwitch:       10,
		models.DeviceTypeFirewall:     100,
		models.DeviceTypeLoadBalancer: 50,
	}

	if threshold, ok := thresholds[device.DeviceType]; ok {
		return threshold
	}
	return 10 // Default
}
```

---

### `internal/device/scrapli_wrapper.go` (utility, request-response) - ENHANCEMENT

**Analog:** Same file (enhancement pattern)

**Add LLDP command support** (similar to existing GetCommandForVendor pattern):
```go
// GetLLDPCommand 根据厂商获取LLDP命令
func GetLLDPCommand(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display lldp neighbor brief",
		models.VendorH3C:    "display lldp neighbor brief",
		models.VendorRuijie: "show lldp neighbors",
		models.VendorMaipu:  "show lldp neighbors",
	}
	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "show lldp neighbors" // Default
}
```

---

### `pkg/cache/lldp_cache.go` (utility, caching)

**Analog:** `internal/services/portcollection/template_cache.go` (lines 9-49)

**Reuse the same template cache pattern but for LLDP data caching:**
```go
package cache

import (
	"sync"
	"time"
)

// LLDPCache LLDP数据缓存
type LLDPCache struct {
	cache map[string]*LLDPCacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

// LLDPCacheEntry 缓存条目
type LLDPCacheEntry struct {
	Neighbors   map[string]*LLDPNeighbor
	CachedAt    time.Time
}

// NewLLDPCache 创建LLDP缓存
func NewLLDPCache(ttl time.Duration) *LLDPCache {
	return &LLDPCache{
		cache: make(map[string]*LLDPCacheEntry),
		ttl:   ttl,
	}
}

// Get 获取缓存中的LLDP数据
func (c *LLDPCache) Get(deviceID string) (map[string]*LLDPNeighbor, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.cache[deviceID]
	if !ok {
		return nil, false
	}

	// 检查是否过期
	if time.Since(entry.CachedAt) > c.ttl {
		return nil, false
	}

	return entry.Neighbors, true
}

// Set 设置LLDP缓存
func (c *LLDPCache) Set(deviceID string, neighbors map[string]*LLDPNeighbor) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[deviceID] = &LLDPCacheEntry{
		Neighbors: neighbors,
		CachedAt:  time.Now(),
	}
}

// Delete 删除缓存
func (c *LLDPCache) Delete(deviceID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, deviceID)
}
```

---

## Shared Patterns

### Connection Pool Pattern
**Source:** `internal/device/connection_pool.go` (lines 16-150)
**Apply to:** All LLDP discovery operations

```go
// Get connection from pool
pool := s.executor.GetScheduler().GetConnectionPool()
conn, err := pool.GetConnection(ctx, device.ID)
if err != nil {
	return nil, err
}
defer conn.Release()

if !conn.Acquire() {
	return nil, fmt.Errorf("无法获取连接锁")
}
defer conn.Release()

wrapper := conn.GetWrapper()
if wrapper == nil {
	return nil, fmt.Errorf("wrapper 为空")
}
```

### Context with Timeout Pattern
**Source:** `internal/services/mac_collection_service.go` (lines 76-77)
**Apply to:** All device command execution

```go
// 为每个设备创建独立的 context（带超时控制）
deviceCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
defer cancel()
```

### Panic Recovery Pattern
**Source:** `internal/services/mac_collection_service.go` (lines 104-109)
**Apply to:** All scrapligo command execution

```go
defer func() {
	if r := recover(); r != nil {
		applogger.Infof("[服务名] 设备 %s 发生 panic: %v", device.DeviceName, r)
	}
}()
```

### Best-Effort Error Handling Pattern
**Source:** `internal/services/portcollection/collection.go` (lines 148-152)
**Apply to:** LLDP discovery and optional features

```go
// 批量获取802.1X状态
dot1xMap, err := getAllDot1xStatus(wrapper, device.Vendor, s.templateCache)
if err != nil {
	applogger.Infof("[警告] 获取802.1X状态失败: %v", err)
	dot1xMap = make(map[string]Dot1xInfo)
}
```

### Vendor-Specific Command Pattern
**Source:** `internal/services/mac_collection_service.go` (lines 196-209)
**Apply to:** All network device commands

```go
func getCommandForVendor(vendor models.DeviceVendor) string {
	commands := map[models.DeviceVendor]string{
		models.VendorHuawei: "display xxx",
		models.VendorH3C:    "display xxx",
		models.VendorRuijie: "show xxx",
		models.VendorMaipu:  "show xxx",
	}
	if cmd, ok := commands[vendor]; ok {
		return cmd
	}
	return "show xxx" // 默认命令
}
```

### GORM Batch Operations Pattern
**Source:** `internal/services/mac_collection_service.go` (lines 139-182)
**Apply to:** All batch database operations

```go
// 1. 先删除旧记录
if err := s.db.WithContext(ctx).
	Where("device_id = ?", device.ID).
	Delete(&models.DeviceMACAddress{}).Error; err != nil {
	return fmt.Errorf("删除旧记录失败: %v", err)
}

// 2. 构建新记录列表
var records []*models.DeviceMACAddress
for _, data := range dataList {
	record := &models.DeviceMACAddress{
		DeviceID: device.ID,
		// ... 其他字段
	}
	records = append(records, record)
}

// 3. 批量插入
if len(records) > 0 {
	if err := s.db.WithContext(ctx).
		Create(&records).Error; err != nil {
		return fmt.Errorf("批量插入失败: %v", err)
	}
}
```

### Service Interface Pattern
**Source:** `internal/services/system/config_service.go` (lines 12-32)
**Apply to:** All new service layers

```go
// XxxService xxx服务接口
type XxxService interface {
	Create(ctx context.Context, req *CreateRequest) error
	Update(ctx context.Context, req *UpdateRequest) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*Model, error)
	List(ctx context.Context, params ListParams) (*PageResult, error)
}

// xxxService xxx服务实现
type xxxService struct {
	db *gorm.DB
}

// NewXxxService 创建xxx服务实例
func NewXxxService(db *gorm.DB) XxxService {
	return &xxxService{db: db}
}
```

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| **TextFSM templates for LLDP** | template | transform | No LLDP templates exist yet; need to create templates for Huawei/H3C/Ruijie/Maipu |

**Template locations:**
- Create in `templates/lldp/` directory
- Follow existing template naming: `lldp_huawei.textfsm`, `lldp_ruijie.textfsm`, etc.
- Reference existing port templates in `templates/` for format guidance

---

## Metadata

**Analog search scope:**
- `internal/services/` (all service implementations)
- `internal/models/` (all data models)
- `internal/device/` (connection pool and executor patterns)
- `pkg/cache/` (caching interfaces and implementations)

**Files scanned:** 25
**Pattern extraction date:** 2026-05-09

**Key insights:**
1. **Service pattern consistency:** All services follow interface + private implementation pattern
2. **Collection pattern reuse:** MAC collection and port collection share identical structure
3. **Template caching proven:** Port collection's template cache pattern is production-ready
4. **Connection pool mature:** Device connection pool handles concurrent access safely
5. **Vendor command pattern:** Established pattern for vendor-specific commands
6. **Error handling consistent:** Best-effort pattern with logging prevents cascading failures

**Implementation priority:**
1. Start with LLDP service (reuses MAC collection pattern exactly)
2. Add parser (reuse template cache from port collection)
3. Create models (follow device_mac_address.go pattern)
4. Enhance MAC collection (insert filtering into existing flow)
5. Add filter rules service (follow config_service.go pattern)
6. Create LLDP cache (adapt template cache for data caching)
