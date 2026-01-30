package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"

	"github.com/go-ldap/ldap/v3"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// 模拟测试LDAP分页搜索
func main() {
	// 这里需要配置实际的AD服务器信息
	config := &models.ADConfig{
		ServerAddress: "your-ad-server.com",
		ServerPort:    636,
		UseSSL:        true,
		BaseDN:        "DC=example,DC=com",
		DomainName:    "example.com",
		AdminUsername: "admin",
		AdminPassword: "password",
	}

	// 测试分页搜索
	if err := testPagingSearch(config); err != nil {
		log.Fatalf("测试失败: %v", err)
	}
}

func testPagingSearch(config *models.ADConfig) error {
	// 连接AD
	var conn *ldap.Conn
	var err error

	address := fmt.Sprintf("%s:%d", config.ServerAddress, config.ServerPort)
	if config.UseSSL {
		conn, err = ldap.DialTLS("tcp", address, &tls.Config{
			InsecureSkipVerify: true,
		})
	} else {
		conn, err = ldap.Dial("tcp", address)
	}

	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer conn.Close()

	// 绑定
	cleanDomain := strings.TrimSuffix(config.DomainName, fmt.Sprintf("@%s", config.DomainName))
	if cleanDomain == "" {
		cleanDomain = config.DomainName
	}

	bindUsername := fmt.Sprintf("%s@%s", config.AdminUsername, cleanDomain)
	if err := conn.Bind(bindUsername, config.AdminPassword); err != nil {
		return fmt.Errorf("绑定失败: %w", err)
	}

	fmt.Println("✓ 连接成功")
	fmt.Println()

	// 测试OU搜索的分页
	fmt.Println("=== 测试OU搜索（分页）===")
	if err := testOUSearch(conn, config.BaseDN); err != nil {
		return err
	}

	// 测试用户搜索的分页
	fmt.Println("\n=== 测试用户搜索（分页）===")
	if err := testUserSearch(conn, config.BaseDN); err != nil {
		return err
	}

	return nil
}

func testOUSearch(conn *ldap.Conn, baseDN string) error {
	return testPagingSearchInternal(conn, baseDN,
		"(objectClass=organizationalUnit)",
		[]string{"dn", "ou", "description"},
		"OU",
	)
}

func testUserSearch(conn *ldap.Conn, baseDN string) error {
	return testPagingSearchInternal(conn, baseDN,
		"(objectClass=user)",
		[]string{"dn", "sAMAccountName", "displayName"},
		"用户",
	)
}

func testPagingSearchInternal(conn *ldap.Conn, baseDN, filter string, attributes []string, typeName string) error {
	const pageSize = 1000
	var allEntries []*ldap.Entry
	var pageCount int

	pagingControl := ldap.NewControlPaging(pageSize)

	for {
		pageCount++
		searchRequest := ldap.NewSearchRequest(
			baseDN,
			ldap.ScopeWholeSubtree,
			ldap.NeverDerefAliases,
			0,     // SizeLimit: 0 表示无限制
			0,     // TimeLimit: 0 表示无限制
			false, // TypesOnly
			filter,
			attributes,
			[]ldap.Control{pagingControl},
		)

		sr, err := conn.Search(searchRequest)
		if err != nil {
			// 检查是否是 SizeLimit Exceeded 错误
			if ldap.IsErrorWithCode(err, ldap.LDAPResultSizeLimitExceeded) {
				fmt.Printf("✗ 服务器返回 SizeLimit Exceeded 错误\n")
				fmt.Printf("  这表明服务器达到了 MaxPageSize 限制\n")
				return fmt.Errorf("搜索结果超过服务器限制")
			}
			return fmt.Errorf("搜索失败: %w", err)
		}

		entriesInPage := len(sr.Entries)
		allEntries = append(allEntries, sr.Entries...)

		fmt.Printf("第 %d 页: 获取到 %d 条记录 (累计: %d)\n",
			pageCount, entriesInPage, len(allEntries))

		// 检查是否还有更多页
		pagingControl = nil
		for _, ctrl := range sr.Controls {
			if pc, ok := ctrl.(*ldap.ControlPaging); ok {
				pagingControl = pc
				break
			}
		}

		// 关键检查：Cookie是否为空
		if pagingControl == nil {
			fmt.Printf("✗ 分页控制为空，无法继续\n")
			fmt.Printf("  这通常意味着服务器不支持分页或返回了错误\n")
			break
		}

		if len(pagingControl.Cookie) == 0 {
			fmt.Printf("✓ 分页Cookie为空，已获取所有数据\n")
			break
		}

		// 准备下一页
		pagingControl.SetCookie(pagingControl.Cookie)
	}

	fmt.Printf("\n总计: %d 个%s，分 %d 页获取\n", len(allEntries), typeName, pageCount)

	// 如果恰好是1000条记录，需要特别检查
	if len(allEntries) == 1000 {
		fmt.Printf("⚠ 警告: 恰好获取到1000条记录，这可能是由于AD服务器的MaxPageSize限制\n")
		fmt.Printf("  建议检查AD服务器的MaxPageSize设置\n")
	}

	return nil
}
