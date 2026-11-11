package dbbackup

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// 本文件实现 Storage 的四个适配器：local / oss / ftp / smb。
//
// 远程路径（remotePath）约定统一为「相对于各自后端根的 POSIX 风格路径」，
// 例如 2026/11/20261111_020000_postgres.zip。Delete/Put/Get 都不接受绝对路径，
// 防路径逃逸（cleanRemoteName 统一清洗）。
//
// 凭据只来自 Plan（由 config.go 从 sys_config 读取），绝不写进 manifest / 日志。

// newStorage 按计划构造存储适配器。
//   - local / smb：本地文件系统路径（smb 用 SMBConfig.Path，即 OS 已挂载的共享目录）
//   - oss：minio-go/v7 v7.0.98（S3 兼容）
//   - ftp：jlaffaye/ftp v0.2.4
func newStorage(p *Plan) (Storage, error) {
	if p == nil {
		return nil, fmt.Errorf("存储计划不能为 nil")
	}
	switch p.Storage {
	case StorageLocal, "":
		return &localStorage{root: filepath.Join("data", "backups")}, nil
	case StorageSMB:
		root := strings.TrimSpace(p.SMB.Path)
		if root == "" {
			return nil, fmt.Errorf("SMB 存储需要已挂载的本地目录绝对路径（sys.database.backup.smb.path）")
		}
		if !filepath.IsAbs(root) {
			return nil, fmt.Errorf("SMB 路径必须是绝对路径: %s", root)
		}
		return &localStorage{root: root, isSMB: true}, nil
	case StorageOSS:
		if strings.TrimSpace(p.OSS.Endpoint) == "" || strings.TrimSpace(p.OSS.Bucket) == "" {
			return nil, fmt.Errorf("OSS 存储需要 endpoint 与 bucket")
		}
		cli, err := minio.New(p.OSS.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(p.OSS.AccessKey, p.OSS.SecretKey, ""),
			Secure: p.OSS.UseSSL,
		})
		if err != nil {
			return nil, fmt.Errorf("初始化 OSS 客户端失败: %w", err)
		}
		return &ossStorage{client: cli, bucket: p.OSS.Bucket}, nil
	case StorageFTP:
		c := p.FTP
		if strings.TrimSpace(c.Host) == "" {
			return nil, fmt.Errorf("FTP 存储需要 host")
		}
		port := c.Port
		if port <= 0 {
			port = 21
		}
		return &ftpStorage{
			addr:      net.JoinHostPort(c.Host, strconv.Itoa(port)),
			user:      c.User,
			password:  c.Password,
			remoteDir: normalizeFTPDir(c.RemoteDir),
		}, nil
	default:
		return nil, fmt.Errorf("不支持的存储类型: %s", p.Storage)
	}
}

// sameFile 判断两个路径是否指向同一个文件：先比绝对路径字符串，
// 再比 os.SameFile（Windows 下大小写/短名差异也能识别）。
// 用于 local 适配器避免「把文件复制到它自己」导致源文件被 os.Create 截断。
func sameFile(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA == nil && errB == nil && absA == absB {
		return true
	}
	fiA, errA := os.Stat(a)
	fiB, errB := os.Stat(b)
	if errA != nil || errB != nil {
		return false
	}
	return os.SameFile(fiA, fiB)
}

// cleanRemoteName 清洗远程名：去目录、去盘符、防 ".." 与绝对路径逃逸。
func cleanRemoteName(name string) (string, error) {
	n := strings.ReplaceAll(name, "\\", "/")
	parts := strings.Split(n, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			continue
		}
		clean = append(clean, part)
	}
	if len(clean) == 0 {
		return "", fmt.Errorf("远程文件名非法: %q", name)
	}
	joined := strings.Join(clean, "/")
	// 二次防御：不允许任何绝对路径或 Windows 盘符残留
	if strings.HasPrefix(joined, "/") || strings.Contains(joined, ":") {
		return "", fmt.Errorf("远程文件名含路径逃逸字符: %q", name)
	}
	return joined, nil
}

// localStorage 本地/SMB 适配器：两者都是本地文件系统路径，共享同一实现。
// Put 把文件复制进 root/remoteName，保持流式（GB 级不进内存）。
type localStorage struct {
	root  string
	isSMB bool
}

func (s *localStorage) Type() StorageType {
	if s.isSMB {
		return StorageSMB
	}
	return StorageLocal
}

func (s *localStorage) resolve(remotePath string) (string, error) {
	clean, err := cleanRemoteName(remotePath)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(s.root)
	if err != nil {
		return "", fmt.Errorf("解析存储根目录: %w", err)
	}
	target := filepath.Join(rootAbs, filepath.FromSlash(clean))
	// 逃逸校验：target 必须仍在 root 之下
	if !strings.HasPrefix(target, rootAbs+string(os.PathSeparator)) && target != rootAbs {
		return "", fmt.Errorf("远程路径越出存储根目录: %q", remotePath)
	}
	return target, nil
}

func (s *localStorage) Put(ctx context.Context, localPath, remoteName string) (string, error) {
	remote, err := cleanRemoteName(remoteName)
	if err != nil {
		return "", err
	}
	target, err := s.resolve(remote)
	if err != nil {
		return "", err
	}
	// local 适配器根目录默认就是 data/backups，与备份产物目录重合。
	// 若 src 与 target 是同一个文件，绝不能 os.Create(target)（会截断源文件），
	// 直接返回 remoteName 即可 —— 本地文件本身已是存储产物。
	if sameFile(localPath, target) {
		return remote, nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", fmt.Errorf("创建本地存储目录: %w", err)
	}
	if err := copyFile(ctx, localPath, target); err != nil {
		return "", fmt.Errorf("写入本地存储失败: %w", err)
	}
	return remote, nil
}

func (s *localStorage) Get(ctx context.Context, remotePath, localPath string) error {
	src, err := s.resolve(remotePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return fmt.Errorf("创建下载目标目录: %w", err)
	}
	if err := copyFile(ctx, src, localPath); err != nil {
		return fmt.Errorf("读取本地存储失败: %w", err)
	}
	return nil
}

func (s *localStorage) Delete(ctx context.Context, remotePath string) error {
	target, err := s.resolve(remotePath)
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除本地存储文件失败: %w", err)
	}
	return nil
}

// ossStorage S3 兼容对象存储适配器（minio-go/v7）。
type ossStorage struct {
	client *minio.Client
	bucket string
}

func (s *ossStorage) Type() StorageType { return StorageOSS }

func (s *ossStorage) Put(ctx context.Context, localPath, remoteName string) (string, error) {
	remote, err := cleanRemoteName(remoteName)
	if err != nil {
		return "", err
	}
	f, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("打开待上传文件: %w", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("读取待上传文件信息: %w", err)
	}
	if _, err := s.client.PutObject(ctx, s.bucket, remote, f, info.Size(),
		minio.PutObjectOptions{ContentType: "application/zip"}); err != nil {
		return "", fmt.Errorf("上传 OSS 失败: %w", err)
	}
	return remote, nil
}

func (s *ossStorage) Get(ctx context.Context, remotePath, localPath string) error {
	remote, err := cleanRemoteName(remotePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return fmt.Errorf("创建下载目标目录: %w", err)
	}
	if err := s.client.FGetObject(ctx, s.bucket, remote, localPath, minio.GetObjectOptions{}); err != nil {
		return fmt.Errorf("下载 OSS 对象失败: %w", err)
	}
	return nil
}

func (s *ossStorage) Delete(ctx context.Context, remotePath string) error {
	remote, err := cleanRemoteName(remotePath)
	if err != nil {
		return err
	}
	if err := s.client.RemoveObject(ctx, s.bucket, remote, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("删除 OSS 对象失败: %w", err)
	}
	return nil
}

// ftpStorage FTP 适配器（jlaffaye/ftp）。
// 每次操作独立建连并 Login，用完 Quit，避免长连接半开占据服务端资源。
type ftpStorage struct {
	addr      string
	user      string
	password  string
	remoteDir string
}

func (s *ftpStorage) Type() StorageType { return StorageFTP }

// normalizeFTPDir 把 FTP 远端目录归一为以 "/" 开头、不以 "/" 结尾的形式
func normalizeFTPDir(dir string) string {
	dir = strings.TrimSpace(strings.ReplaceAll(dir, "\\", "/"))
	if dir == "" || dir == "/" {
		return "/"
	}
	if !strings.HasPrefix(dir, "/") {
		dir = "/" + dir
	}
	return strings.TrimSuffix(dir, "/")
}

func (s *ftpStorage) dial(ctx context.Context) (*ftp.ServerConn, error) {
	conn, err := ftp.Dial(s.addr,
		ftp.DialWithContext(ctx),
		ftp.DialWithTimeout(30*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("连接 FTP 服务器失败: %w", err)
	}
	if err := conn.Login(s.user, s.password); err != nil {
		_ = conn.Quit()
		return nil, fmt.Errorf("FTP 登录失败: %w", err)
	}
	if err := conn.ChangeDir(s.remoteDir); err != nil {
		_ = conn.Quit()
		return nil, fmt.Errorf("切换 FTP 远端目录失败: %w", err)
	}
	return conn, nil
}

// ensureFTPPath 逐级创建远端子目录（Stor 不会自动建多级目录）
func (s *ftpStorage) ensureFTPPath(conn *ftp.ServerConn, remote string) error {
	parts := strings.Split(remote, "/")
	if len(parts) <= 1 {
		return nil // 纯文件名，无需建目录
	}
	for _, part := range parts[:len(parts)-1] {
		// MakeDir 对已存在目录会报错，忽略即可；其它错误同样先尝试继续
		_ = conn.MakeDir(part)
		if err := conn.ChangeDir(part); err != nil {
			return fmt.Errorf("进入 FTP 子目录 %q 失败: %w", part, err)
		}
	}
	return nil
}

func (s *ftpStorage) Put(ctx context.Context, localPath, remoteName string) (string, error) {
	remote, err := cleanRemoteName(remoteName)
	if err != nil {
		return "", err
	}
	f, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("打开待上传文件: %w", err)
	}
	defer f.Close()

	conn, err := s.dial(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Quit() }()
	if err := s.ensureFTPPath(conn, remote); err != nil {
		return "", err
	}
	if err := conn.Stor(filepath.Base(remote), f); err != nil {
		return "", fmt.Errorf("上传 FTP 文件失败: %w", err)
	}
	return remote, nil
}

func (s *ftpStorage) Get(ctx context.Context, remotePath, localPath string) error {
	remote, err := cleanRemoteName(remotePath)
	if err != nil {
		return err
	}
	conn, err := s.dial(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Quit() }()

	// Retr 只接受相对当前目录的路径；逐段切换后取文件名
	parts := strings.Split(remote, "/")
	for _, part := range parts[:len(parts)-1] {
		if err := conn.ChangeDir(part); err != nil {
			return fmt.Errorf("进入 FTP 子目录 %q 失败: %w", part, err)
		}
	}
	resp, err := conn.Retr(parts[len(parts)-1])
	if err != nil {
		return fmt.Errorf("下载 FTP 文件失败: %w", err)
	}
	defer resp.Close()

	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return fmt.Errorf("创建下载目标目录: %w", err)
	}
	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("创建下载目标文件: %w", err)
	}
	_, copyErr := ioCopyWithContext(ctx, out, resp)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(localPath)
		return fmt.Errorf("写出 FTP 下载内容失败: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(localPath)
		return fmt.Errorf("关闭下载目标文件失败: %w", closeErr)
	}
	return nil
}

func (s *ftpStorage) Delete(ctx context.Context, remotePath string) error {
	remote, err := cleanRemoteName(remotePath)
	if err != nil {
		return err
	}
	conn, err := s.dial(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Quit() }()
	parts := strings.Split(remote, "/")
	for _, part := range parts[:len(parts)-1] {
		if err := conn.ChangeDir(part); err != nil {
			return fmt.Errorf("进入 FTP 子目录 %q 失败: %w", part, err)
		}
	}
	if err := conn.Delete(parts[len(parts)-1]); err != nil {
		return fmt.Errorf("删除 FTP 文件失败: %w", err)
	}
	return nil
}
