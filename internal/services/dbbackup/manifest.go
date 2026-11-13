package dbbackup

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 本文件实现备份容器（.zip）的封装与校验。
//
// ⚠️ 安全告警：备份包含全库敏感数据（sys_user 口令散列、个人信息、业务数据、
// 配置参数等），任何持有该 .zip 即等同于持有整库明文。因此：
//   - manifest.json 绝不含任何凭据字段（见下方结构注释）；
//   - 产物文件权限、远程存储桶策略、下载端点审计与前端二次提示必须按「最高敏感级」管控；
//   - 严禁把备份容器提交进仓库、放进静态资源目录或日志里打印文件内容。
//
// 容器结构：
//
//	<dump_file>     数据库转储产物（PG: pg_dump custom 格式；SQLite: VACUUM INTO 产物）
//	manifest.json   元数据清单（零凭据：绝不包含数据库口令、OSS/FTP/SMB 密钥）
//
// 设计约束：
//   - GB 级备份不允许整体读入内存：dump 条目用 io.Copy 流式写入 zip、
//     流式解压落盘，manifest.json 单独限制读取上限防 zip bomb。
//   - 三层 sha256 校验（"双校验"的落地）：
//     ① sha256          = manifest 规范内容（剔除 sha256 字段）的签名，防清单被篡改；
//     ② dump_sha256     = dump 条目解压后字节流的哈希；
//     ③ container_sha256= 容器数据条目按 "文件名 + \x00 + 内容" 规范化拼接的哈希，
//     代表容器数据完整性（与 ① 独立，避免只信任清单自述）。
//   - 整个 .zip 文件的 sha256 由 writeContainer 返回并写入 sys_db_backup.sha256，
//     供下载/恢复时校验文件传输完整性；它不进 manifest（自引用会形成鸡生蛋问题）。
const (
	manifestEntryName = "manifest.json"

	// manifestMaxBytes 清单读取上限，防恶意 zip 用超大 manifest 打爆内存
	manifestMaxBytes = 4 << 20 // 4 MiB
	// maxDumpEntryBytes 单个 dump 条目解压后的合理上限（流式落盘不进内存，
	// 仅用于拦截明显的 zip bomb；真实全库备份通常远小于此值）
	maxDumpEntryBytes = 64 << 30 // 64 GiB

	checksumAlgorithm = "sha256"
	// appVersion 写入清单便于跨版本兼容性判断；不依赖构建期注入
	appVersion = "xingran-next-dbbackup/1.0"
)

// manifest 备份清单。字段命名与 CONTRACT/需求一致，全部 snake_case。
// 注意：本结构绝不允许出现任何凭据字段（数据库口令 / accessKey / secretKey /
// ftp password 等），否则会随备份文件外泄。
type manifest struct {
	SchemaVersion     string    `json:"schema_version"`
	CreatedAt         time.Time `json:"created_at"`
	DBType            string    `json:"db_type"`
	TableCount        int       `json:"table_count"`
	Tables            []string  `json:"tables"`
	DumpFile          string    `json:"dump_file"`
	DumpSha256        string    `json:"dump_sha256"`
	ContainerSha256   string    `json:"container_sha256"`
	AppVersion        string    `json:"app_version"`
	ChecksumAlgorithm string    `json:"checksum_algorithm"`
	Sha256            string    `json:"sha256"`
	ExcludedTables    []string  `json:"excluded_tables"`
}

// sha256Hex 返回哈希的十六进制字符串
func sha256Hex(h interface{ Sum([]byte) []byte }) string {
	return hex.EncodeToString(h.Sum(nil))
}

// fileSha256 流式计算文件 sha256，不把文件读入内存
func fileSha256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("打开待哈希文件: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("计算文件 sha256: %w", err)
	}
	return sha256Hex(h), nil
}

// manifestSignature 计算清单自签名：把 Sha256 字段清空后按结构体字段顺序
// 序列化（Go 的 struct json.Marshal 顺序稳定，无 map 参与），再取 sha256。
func manifestSignature(m *manifest) (string, error) {
	saved := m.Sha256
	m.Sha256 = ""
	defer func() { m.Sha256 = saved }()

	b, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("序列化清单: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// sanitizeEntryName 清洗 zip 条目名，防 zip-slip 路径逃逸。
// 只允许文件名（去目录），并剔除分隔符与通配符等危险字符。
func sanitizeEntryName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	clean := b.String()
	if clean == "" || clean == "." || clean == ".." {
		return "dump.bin"
	}
	return clean
}

// dumpEntryName 按数据库类型给出容器内 dump 条目名
func dumpEntryName(dbType string) string {
	if strings.EqualFold(dbType, dbTypeSQLite) {
		return "dump.db"
	}
	return "dump.sql"
}

// writeContainer 把 dump 文件与 manifest.json 流式写入 zipPath。
//
// meta 由调用方填入业务字段（schema_version/db_type/tables/...），本函数补全
// dump_sha256、container_sha256、sha256 三个校验字段。返回整个 zip 文件的
// sha256（供写入备份记录），并保证写入失败时清理半成品文件。
func writeContainer(ctx context.Context, zipPath, dumpPath string, meta *manifest) (string, error) {
	// 前置校验：dump 必须是真实存在的普通文件
	info, err := os.Stat(dumpPath)
	if err != nil {
		return "", fmt.Errorf("检查 dump 文件: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("dump 路径是目录而非文件: %s", dumpPath)
	}

	if err := os.MkdirAll(filepath.Dir(zipPath), 0o755); err != nil {
		return "", fmt.Errorf("创建备份目录: %w", err)
	}

	out, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("创建备份容器: %w", err)
	}
	// 边写边算整个容器文件的哈希（含中央目录），关闭 zw 后即为最终值
	zipFileHash := sha256.New()
	zw := zip.NewWriter(io.MultiWriter(out, zipFileHash))

	ok := false
	defer func() {
		if !ok {
			_ = zw.Close()
			_ = out.Close()
			_ = os.Remove(zipPath)
		}
	}()

	meta.DumpFile = sanitizeEntryName(dumpEntryName(meta.DBType))
	meta.ChecksumAlgorithm = checksumAlgorithm
	meta.AppVersion = appVersion
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now().UTC()
	}
	if meta.Tables == nil {
		meta.Tables = []string{}
	}
	if meta.ExcludedTables == nil {
		meta.ExcludedTables = []string{}
	}
	meta.TableCount = len(meta.Tables)

	// ① 写 dump 条目：一次读取同时喂给 zip 条目、dump 哈希、容器哈希。
	//    io.TeeReader 保证喂给 MultiWriter 的字节与写入 zip 条目的字节完全一致
	//    （zip 内部压缩发生在 entry 写入之后，哈希取的是压缩前的原始内容）。
	dumpFile, err := os.Open(dumpPath)
	if err != nil {
		return "", fmt.Errorf("打开 dump 文件: %w", err)
	}
	hdr := &zip.FileHeader{
		Name:     meta.DumpFile,
		Method:   zip.Deflate,
		Modified: meta.CreatedAt,
	}
	entry, err := zw.CreateHeader(hdr)
	if err != nil {
		_ = dumpFile.Close()
		return "", fmt.Errorf("创建 dump 条目: %w", err)
	}
	hDump := sha256.New()
	hContainer := sha256.New()
	// 容器哈希的规范化前缀：条目名 + \x00，避免不同文件名产生相同容器哈希
	_, _ = hContainer.Write([]byte(meta.DumpFile))
	_, _ = hContainer.Write([]byte{0})

	src := io.TeeReader(dumpFile, io.MultiWriter(hDump, hContainer))
	if _, err := ioCopyWithContext(ctx, entry, src); err != nil {
		_ = dumpFile.Close()
		return "", fmt.Errorf("流式写入 dump 条目: %w", err)
	}
	if err := dumpFile.Close(); err != nil {
		return "", fmt.Errorf("关闭 dump 文件: %w", err)
	}
	meta.DumpSha256 = sha256Hex(hDump)
	meta.ContainerSha256 = sha256Hex(hContainer)

	// ② 计算清单自签名并写 manifest.json
	sig, err := manifestSignature(meta)
	if err != nil {
		return "", err
	}
	meta.Sha256 = sig
	manifestBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return "", fmt.Errorf("序列化 manifest.json: %w", err)
	}
	mfWriter, err := zw.Create(manifestEntryName)
	if err != nil {
		return "", fmt.Errorf("创建 manifest 条目: %w", err)
	}
	if _, err := mfWriter.Write(manifestBytes); err != nil {
		return "", fmt.Errorf("写入 manifest 条目: %w", err)
	}

	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("关闭 zip writer: %w", err)
	}
	if err := out.Close(); err != nil {
		return "", fmt.Errorf("关闭备份容器文件: %w", err)
	}
	ok = true
	return sha256Hex(zipFileHash), nil
}

// verifyManifest 校验清单自身的合法性（不含 dump 内容校验）
func verifyManifest(m *manifest) error {
	if m.ChecksumAlgorithm != checksumAlgorithm {
		return fmt.Errorf("不支持的校验算法: %q", m.ChecksumAlgorithm)
	}
	if m.SchemaVersion != SchemaVersionV1 {
		return fmt.Errorf("不兼容的清单版本: %q", m.SchemaVersion)
	}
	if m.DumpFile == "" || m.DumpFile == manifestEntryName {
		return fmt.Errorf("清单 dump_file 非法: %q", m.DumpFile)
	}
	// 防 zip-slip：dump_file 必须是纯文件名
	if m.DumpFile != sanitizeEntryName(m.DumpFile) {
		return fmt.Errorf("清单 dump_file 含路径分隔符，拒绝解压: %q", m.DumpFile)
	}
	if m.TableCount != len(m.Tables) {
		return fmt.Errorf("清单 table_count=%d 与 tables 长度=%d 不一致", m.TableCount, len(m.Tables))
	}
	if m.DumpSha256 == "" || m.ContainerSha256 == "" || m.Sha256 == "" {
		return fmt.Errorf("清单缺少 sha256 校验字段")
	}

	sig, err := manifestSignature(m)
	if err != nil {
		return err
	}
	if sig != m.Sha256 {
		return fmt.Errorf("清单 sha256 不匹配（manifest.json 可能被篡改）")
	}
	return nil
}

// verifyAndExtract 解析 zip 容器：校验 manifest sha256（自签名 + dump/容器双哈希），
// 并把 dump 条目流式解压到 destDir，返回清单与解压后的 dump 文件绝对路径。
//
// 恢复流程必须先调用本函数再执行恢复前自动备份，确保上传的是合法备份。
func verifyAndExtract(ctx context.Context, zipPath, destDir string) (*manifest, string, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, "", fmt.Errorf("打开备份容器: %w", err)
	}
	defer zr.Close()

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, "", fmt.Errorf("创建解压目录: %w", err)
	}

	var mf *zip.File
	for _, f := range zr.File {
		if f.Name == manifestEntryName {
			mf = f
			break
		}
	}
	if mf == nil {
		return nil, "", fmt.Errorf("备份容器缺少 %s", manifestEntryName)
	}

	mfReader, err := mf.Open()
	if err != nil {
		return nil, "", fmt.Errorf("打开 manifest 条目: %w", err)
	}
	manifestBytes, err := io.ReadAll(io.LimitReader(mfReader, manifestMaxBytes+1))
	_ = mfReader.Close()
	if err != nil {
		return nil, "", fmt.Errorf("读取 manifest 条目: %w", err)
	}
	if len(manifestBytes) > manifestMaxBytes {
		return nil, "", fmt.Errorf("manifest.json 超过 %d 字节上限", manifestMaxBytes)
	}
	var m manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return nil, "", fmt.Errorf("解析 manifest.json: %w", err)
	}
	if err := verifyManifest(&m); err != nil {
		return nil, "", err
	}

	// 定位 dump 条目
	var df *zip.File
	for _, f := range zr.File {
		if f.Name == m.DumpFile {
			df = f
			break
		}
	}
	if df == nil {
		return nil, "", fmt.Errorf("备份容器缺少清单声明的 dump 条目 %q", m.DumpFile)
	}
	if df.UncompressedSize64 > maxDumpEntryBytes {
		return nil, "", fmt.Errorf("dump 条目解压后大小 %d 超过上限 %d", df.UncompressedSize64, maxDumpEntryBytes)
	}

	// 流式解压到临时文件，同时复算 dump 哈希与容器哈希
	destPath := filepath.Join(destDir, sanitizeEntryName(m.DumpFile))
	out, err := os.Create(destPath)
	if err != nil {
		return nil, "", fmt.Errorf("创建解压目标文件: %w", err)
	}
	cleanup := func() { _ = out.Close(); _ = os.Remove(destPath) }

	rc, err := df.Open()
	if err != nil {
		cleanup()
		return nil, "", fmt.Errorf("打开 dump 条目: %w", err)
	}
	hDump := sha256.New()
	hContainer := sha256.New()
	_, _ = hContainer.Write([]byte(m.DumpFile))
	_, _ = hContainer.Write([]byte{0})
	src := io.TeeReader(rc, io.MultiWriter(hDump, hContainer))
	// 解压期间用传入 ctx 控制取消，并累计实际写出字节数兜底 zip bomb：
	// UncompressedSize64 是 zip 头声明值，可被伪造；这里以真实写出量为硬上限。
	written, copyErr := ioCopyWithContextLimit(ctx, out, src, maxDumpEntryBytes)
	_ = rc.Close()
	if copyErr != nil {
		cleanup()
		return nil, "", fmt.Errorf("流式解压 dump 条目: %w", copyErr)
	}
	if written > maxDumpEntryBytes {
		cleanup()
		return nil, "", fmt.Errorf("dump 条目实际解压 %d 字节超过上限 %d", written, maxDumpEntryBytes)
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(destPath)
		return nil, "", fmt.Errorf("关闭解压文件: %w", err)
	}

	if got := sha256Hex(hDump); got != m.DumpSha256 {
		_ = os.Remove(destPath)
		return nil, "", fmt.Errorf("dump_sha256 不匹配: 期望 %s 实得 %s", m.DumpSha256, got)
	}
	if got := sha256Hex(hContainer); got != m.ContainerSha256 {
		_ = os.Remove(destPath)
		return nil, "", fmt.Errorf("container_sha256 不匹配: 期望 %s 实得 %s", m.ContainerSha256, got)
	}

	abs, err := filepath.Abs(destPath)
	if err != nil {
		_ = os.Remove(destPath)
		return nil, "", fmt.Errorf("解析解压文件绝对路径: %w", err)
	}
	return &m, abs, nil
}

// verifyContainerOnly 只校验容器合法性、不解压 dump，用于恢复前自动备份之前的
// 快速门禁（避免为非法上传文件先做一次昂贵的全量备份）。
func verifyContainerOnly(zipPath string) (*manifest, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("打开备份容器: %w", err)
	}
	defer zr.Close()

	var mf *zip.File
	for _, f := range zr.File {
		if f.Name == manifestEntryName {
			mf = f
			break
		}
	}
	if mf == nil {
		return nil, fmt.Errorf("备份容器缺少 %s", manifestEntryName)
	}
	r, err := mf.Open()
	if err != nil {
		return nil, fmt.Errorf("打开 manifest 条目: %w", err)
	}
	b, err := io.ReadAll(io.LimitReader(r, manifestMaxBytes+1))
	_ = r.Close()
	if err != nil {
		return nil, fmt.Errorf("读取 manifest 条目: %w", err)
	}
	if len(b) > manifestMaxBytes {
		return nil, fmt.Errorf("manifest.json 超过 %d 字节上限", manifestMaxBytes)
	}
	var m manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("解析 manifest.json: %w", err)
	}
	if err := verifyManifest(&m); err != nil {
		return nil, err
	}
	// 同时确认声明的 dump 条目确实存在
	found := false
	for _, f := range zr.File {
		if f.Name == m.DumpFile {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("备份容器缺少清单声明的 dump 条目 %q", m.DumpFile)
	}
	return &m, nil
}
