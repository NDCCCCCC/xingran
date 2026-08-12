//go:build windows
// +build windows

package system

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemTimes = kernel32.NewProc("GetSystemTimes")
)

type filetime struct {
	dwLowDateTime  uint32
	dwHighDateTime uint32
}

var (
	lastIdleTime   uint64
	lastKernelTime uint64
	lastUserTime   uint64
	firstCall      = true
)

// GetCPUUsage 获取Windows系统的CPU使用率
func GetCPUUsage() (float64, error) {
	var idleTime, kernelTime, userTime filetime

	// 调用Windows API获取系统时间
	ret, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idleTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)

	if ret == 0 {
		// 如果GetSystemTimes失败，返回错误
		return 0, fmt.Errorf("GetSystemTimes API调用失败")
	}

	// 转换为uint64
	currentIdle := filetimeToUint64(idleTime)
	currentKernel := filetimeToUint64(kernelTime)
	currentUser := filetimeToUint64(userTime)

	if firstCall {
		// 第一次调用，初始化时间戳
		lastIdleTime = currentIdle
		lastKernelTime = currentKernel
		lastUserTime = currentUser
		firstCall = false
		time.Sleep(500 * time.Millisecond) // 等待500ms确保有足够的时间差
		return GetCPUUsage()               // 递归调用获取真实使用率
	}

	// 计算差值
	idleDiff := currentIdle - lastIdleTime
	kernelDiff := currentKernel - lastKernelTime
	userDiff := currentUser - lastUserTime

	// 更新上次时间
	lastIdleTime = currentIdle
	lastKernelTime = currentKernel
	lastUserTime = currentUser

	// 总时间 = 内核时间 + 用户时间 (注意：内核时间包含空闲时间)
	total := kernelDiff + userDiff

	if total == 0 {
		// 如果总时间为0，可能是因为系统负载太低或测量间隔太短
		// 使用备用方法
		return getCPUUsageByRuntime(), nil
	}

	// CPU使用率 = (总时间 - 空闲时间) / 总时间 * 100
	usage := float64(total-idleDiff) / float64(total) * 100

	if usage < 0 {
		usage = 0
	}
	if usage > 100 {
		usage = 100
	}

	return usage, nil
}

// filetimeToUint64 将FILETIME转换为uint64
func filetimeToUint64(ft filetime) uint64 {
	return uint64(ft.dwHighDateTime)<<32 | uint64(ft.dwLowDateTime)
}

// getCPUUsageByRuntime 备用的CPU使用率计算方法
func getCPUUsageByRuntime() float64 {
	// 使用Go运行时信息作为备用方案
	// 这是一个简化的估算
	numCPU := float64(runtime.NumCPU())
	numGoroutine := float64(runtime.NumGoroutine())

	// 基于协程数和CPU核心数的简单估算
	usage := (numGoroutine / 10.0) / numCPU * 100.0

	if usage > 95.0 {
		usage = 95.0
	}
	if usage < 2.0 {
		usage = 2.0
	}

	return usage
}
