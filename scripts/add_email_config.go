package main
import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
)
func main() {
	db, _ := sql.Open("postgres", "host=10.62.10.34 port=5432 user=postgres password=Cpic1234 dbname=ruoyi sslmode=disable")
	defer db.Close()
	// 先检查是否已有配置
	var count int
	db.QueryRow("SELECT COUNT(*) FROM sys_email_config WHERE del_flag = 0").Scan(&count)
	if count > 0 {
		fmt.Printf("已存在 %d 条邮件配置，系统只允许一条。请先删除现有配置。\n", count)
		return
	}
	// 生成ID
	id := uuid.New().String()
	// 加密密码
	encryptedPassword, err := services.EncryptPassword("Cpic@1234", "")
	if err != nil {
		fmt.Printf("密码加密失败: %v\n", err)
		return
	}
	// 插入配置
	query := `
		INSERT INTO sys_email_config (
			id, config_name, host, port, username, password, 
			from_name, use_ssl, is_default, status, 
			created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
	`
	_, err = db.Exec(query,
		id,                    // $1 id
		"管理员邮件",           // $2 config_name
		"21.0.120.22",         // $3 host
		25,                    // $4 port
		"guanliyuan-014@cpic.com.cn", // $5 username
		encryptedPassword,     // $6 password (encrypted)
		"数智化平台",           // $7 from_name
		false,                 // $8 use_ssl
		true,                  // $9 is_default
		0,                     // $10 status (0=正常)
		"system",              // $11 created_by
	)
	if err != nil {
		fmt.Printf("插入失败: %v\n", err)
		return
	}
	fmt.Println("邮件配置添加成功！")
	fmt.Printf("配置ID: %s\n", id)
	fmt.Println("\n配置信息:")
	fmt.Println("  配置名称: 管理员邮件")
	fmt.Println("  SMTP服务器: 21.0.120.22:25")
	fmt.Println("  邮箱账号: guanliyuan-014@cpic.com.cn")
	fmt.Println("  发件名称: 数智化平台")
	fmt.Println("  SSL: 关闭")
	fmt.Println("  状态: 正常")
}
