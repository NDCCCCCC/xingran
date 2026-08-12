package cache

import "errors"

// 缓存相关错误
var (
	ErrNotFound      = errors.New("缓存项不存在")
	ErrInvalidType   = errors.New("无效的类型")
	ErrKeyEmpty      = errors.New("键不能为空")
	ErrValueEmpty    = errors.New("值不能为空")
	ErrExpired       = errors.New("缓存已过期")
	ErrCacheFull     = errors.New("缓存已满")
	ErrSerialization = errors.New("序列化失败")
	ErrConnection    = errors.New("缓存连接失败")
)

// IsNotFound 检查错误是否为未找到错误
func IsNotFound(err error) bool {
	return err == ErrNotFound
}

// IsExpired 检查错误是否为过期错误
func IsExpired(err error) bool {
	return err == ErrExpired
}

// IsKeyEmpty 检查错误是否为空键错误
func IsKeyEmpty(err error) bool {
	return err == ErrKeyEmpty
}
