package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	// 数据库连接配置
	host := "10.62.10.34"
	port := 5432
	user := "postgres"
	password := "Cpic1234"
	dbname := "ruoyi"

	// 构造连接字符串
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// 连接数据库
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatal("数据库连接测试失败:", err)
	}

	fmt.Println("成功连接到数据库")

	// 生成 UUID
	dictTypeID := uuid.New().String()
	deviceDataIDs := []string{
		uuid.New().String(),
		uuid.New().String(),
		uuid.New().String(),
		uuid.New().String(),
		uuid.New().String(),
	}

	// SQL 语句
	sqlStatements := []string{
		// 插入字典类型
		fmt.Sprintf(`INSERT INTO "sys_dict_type" ("id", "dict_name", "dict_type", "status", "remark", "created_at", "updated_at")
		VALUES ('%s', '网络设备类型', 'network_device_type', 0, '网络设备类型映射', NOW(), NOW())
		ON CONFLICT ("dict_type") DO NOTHING;`, dictTypeID),

		// 插入字典数据
		fmt.Sprintf(`INSERT INTO "sys_dict_data" ("id", "dict_sort", "dict_label", "dict_value", "dict_type", "css_class", "list_class", "is_default", "status", "remark", "created_at", "updated_at")
		VALUES ('%s', 1, '路由器', 'router', 'network_device_type', NULL, 'default', false, 0, '路由器设备', NOW(), NOW())
		ON CONFLICT DO NOTHING;`, deviceDataIDs[0]),

		fmt.Sprintf(`INSERT INTO "sys_dict_data" ("id", "dict_sort", "dict_label", "dict_value", "dict_type", "css_class", "list_class", "is_default", "status", "remark", "created_at", "updated_at")
		VALUES ('%s', 2, '交换机', 'switch', 'network_device_type', NULL, 'default', false, 0, '交换机设备', NOW(), NOW())
		ON CONFLICT DO NOTHING;`, deviceDataIDs[1]),

		fmt.Sprintf(`INSERT INTO "sys_dict_data" ("id", "dict_sort", "dict_label", "dict_value", "dict_type", "css_class", "list_class", "is_default", "status", "remark", "created_at", "updated_at")
		VALUES ('%s', 3, '防火墙', 'firewall', 'network_device_type', NULL, 'default', false, 0, '防火墙设备', NOW(), NOW())
		ON CONFLICT DO NOTHING;`, deviceDataIDs[2]),

		fmt.Sprintf(`INSERT INTO "sys_dict_data" ("id", "dict_sort", "dict_label", "dict_value", "dict_type", "css_class", "list_class", "is_default", "status", "remark", "created_at", "updated_at")
		VALUES ('%s', 4, '无线AP', 'ap', 'network_device_type', NULL, 'default', false, 0, '无线接入点', NOW(), NOW())
		ON CONFLICT DO NOTHING;`, deviceDataIDs[3]),

		fmt.Sprintf(`INSERT INTO "sys_dict_data" ("id", "dict_sort", "dict_label", "dict_value", "dict_type", "css_class", "list_class", "is_default", "status", "remark", "created_at", "updated_at")
		VALUES ('%s', 5, '负载均衡器', 'loadbalancer', 'network_device_type', NULL, 'default', false, 0, '负载均衡设备', NOW(), NOW())
		ON CONFLICT DO NOTHING;`, deviceDataIDs[4]),
	}

	// 执行 SQL
	for i, stmt := range sqlStatements {
		_, err := db.Exec(stmt)
		if err != nil {
			log.Printf("执行第 %d 条语句失败: %v\n", i+1, err)
			log.Printf("SQL: %s\n", stmt)
			continue
		}
		fmt.Printf("成功执行第 %d 条语句\n", i+1)
	}

	fmt.Println("\n迁移执行完成！")

	// 验证结果
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sys_dict_type WHERE dict_type = 'network_device_type'").Scan(&count)
	if err != nil {
		log.Fatal("验证失败:", err)
	}
	fmt.Printf("字典类型记录数: %d\n", count)

	err = db.QueryRow("SELECT COUNT(*) FROM sys_dict_data WHERE dict_type = 'network_device_type'").Scan(&count)
	if err != nil {
		log.Fatal("验证失败:", err)
	}
	fmt.Printf("字典数据记录数: %d\n", count)
}
