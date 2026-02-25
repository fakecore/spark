package constants

import (
	"time"

	"spark/pkg/principal"
)

const (
	AUTH_ERROR_MSG string = "授权验证失败,请重新登录"
)

const (
	// 禁用
	CommonStatus_DIABLE int32 = 0
	// 正常
	CommonStatus_ACTIVE int32 = 1
	// 待审核
	CommonStatus_DRAFT int32 = 2
)

const (
	RoleStatus_Ban    int32 = 0
	RoleStatus_Active int32 = 1
)

const (
	PostStatus_Ban    int32 = 0
	PostStatus_Active int32 = 1
)

const (
	// 全部数据权限
	DataScope_All int32 = 1
	// 自定数据权限
	DataScope_Custom int32 = 2
	// 本部门数据权限
	DataScope_Dept int32 = 3
	// 本部门及以下数据权限
	DataScope_DeptAndChildren int32 = 4
	// 兼容旧命名
	DataScope_Person int32 = DataScope_DeptAndChildren
)

func DataScopeFromDB(v int32) principal.DataScope {
	switch v {
	case DataScope_All:
		return principal.ScopeAll
	case DataScope_Custom:
		return principal.ScopeCustom
	case DataScope_Dept:
		return principal.ScopeDept
	case DataScope_DeptAndChildren:
		return principal.ScopeDeptTree
	default:
		return ""
	}
}

const (
	FileStatus_Failed   int32 = 0
	FileStatus_Uploaded int32 = 1
	FileStatus_Used     int32 = 2
	FileStatus_Pending  int32 = 3
)

const (
	DEPT_STATUS_DISABLE int32 = 0
	DEPT_STATUS_ENABLE  int32 = 1
)

// TaskStatus 任务状态
type TaskStatus int32

const (
	TaskStatusPending   TaskStatus = 1 // 等待中
	TaskStatusRunning   TaskStatus = 2 // 运行中
	TaskStatusCompleted TaskStatus = 3 // 已完成
	TaskStatusFailed    TaskStatus = 4 // 失败
	TaskStatusTimeout   TaskStatus = 5 // 超时
)

// GetTaskStatusString 获取任务状态字符串
func GetTaskStatusString(status TaskStatus) string {
	switch status {
	case TaskStatusPending:
		return "pending"
	case TaskStatusRunning:
		return "running"
	case TaskStatusCompleted:
		return "completed"
	case TaskStatusFailed:
		return "failed"
	case TaskStatusTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

const DefaultAvailbleAudioTime = 30 * time.Second // 30秒
