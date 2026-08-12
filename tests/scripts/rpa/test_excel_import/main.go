
//go:build ignore
// 降级保留: 引用已废弃的 core.NewCore / db.CloseDB,修复需重构初始化逻辑并核对 operations 包 API,
// 暂维持 ignore 不参与 go build。如需启用,改用 config.Load() + db.NewDatabase() 模式(参考 scripts/mac/cleanup)。

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
	"github.com/xingran-next/xingran-go-backend/internal/services/operations"
)

// 测试Excel导入功能的主函数
func main() {
	// 初始化Core
	coreInstance, err := core.NewCore()
	if err != nil {
		log.Fatalf("初始化Core失败: %v", err)
	}
	defer func() {
		if coreInstance.DB != nil {
			db.CloseDB(coreInstance.DB.GetDB())
		}
	}()

	fmt.Println("=== Excel导入功能测试 ===")
	fmt.Println()

	// 1. 测试ReferenceResolver
	fmt.Println("1. 测试ReferenceResolver...")
	testReferenceResolver(coreInstance)
	fmt.Println()

	// 2. 测试Excel配置
	fmt.Println("2. 测试Excel配置...")
	testExcelConfig()
	fmt.Println()

	// 3. 测试数据库连接
	fmt.Println("3. 测试数据库连接...")
	testDatabaseConnection(coreInstance)
	fmt.Println()

	fmt.Println("=== 测试完成 ===")
	fmt.Println()
	fmt.Println("下一步：")
	fmt.Println("1. 执行数据库迁移: psql -U postgres -d xingran_next -f internal/core/db/migrations/080_add_dept_code_field.sql")
	fmt.Println("2. 运行应用并测试Excel导入功能")
	fmt.Println("3. 准备测试Excel文件进行实际导入测试")
}

func testReferenceResolver(coreInstance *core.Core) {
	resolver := operations.NewReferenceResolver(coreInstance.DB.GetDB())
	if resolver == nil {
		fmt.Println("  ❌ ReferenceResolver创建失败")
		return
	}
	fmt.Println("  ✅ ReferenceResolver创建成功")
}

func testExcelConfig() {
	types := operations.GetAllEntityTypes()
	fmt.Printf("  支持的实体类型: %v\n", types)

	for _, entityType := range types {
		config, exists := operations.GetExcelConfig(entityType)
		if !exists {
			fmt.Printf("  ❌ %s 配置不存在\n", entityType)
			continue
		}
		fmt.Printf("  ✅ %s: %s (表: %s)\n", entityType, config.EntityName, config.TableName)
	}
}

func testDatabaseConnection(coreInstance *core.Core) {
	db := coreInstance.DB.GetDB()
	var result int
	err := db.Raw("SELECT 1").Scan(&result).Error
	if err != nil {
		fmt.Printf("  ❌ 数据库连接失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("  ✅ 数据库连接正常")
}
