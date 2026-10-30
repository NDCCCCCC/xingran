package rpa

// RPA 模型包 - 包含所有 RPA 相关的数据模型
//
// 本包提供了 RPA（机器人流程自动化）系统的核心数据模型：
// - Task: RPA 任务定义
// - Worker: Worker 节点
// - Execution: 执行记录
// - Schedule: 定时调度
// - Variable: 全局变量
// - Notification: 通知配置
// - AuditLog: 审计日志
// - Template: 脚本模板

// 所有模型都遵循项目规范：
// - 使用 UUID 作为主键
// - 包含 created_at, updated_at, deleted_at, created_by, updated_by, version 字段
// - 支持软删除（deleted_at）
// - JSONB 字段用于存储复杂结构

// 使用示例：
//
//	import "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
//
//	// 创建任务
//	task := &rpa.Task{
//	    TaskName: "示例任务",
//	    Status: rpa.TaskStatusEnabled,
//	}
//	task.SetActions([]rpa.ScriptAction{...})
