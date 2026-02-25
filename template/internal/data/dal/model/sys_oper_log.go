package model

const TableNameSysOperLog = "sys_oper_log"

// SysOperLog 操作日志记录
type SysOperLog struct {
	ID            int64   `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedBy     *int64  `gorm:"comment:创建者" json:"created_by"`
	Title         *string `gorm:"size:128;comment:标题" json:"title"`
	BusinessType  int32   `gorm:"type:smallint;not null;comment:业务类型 0其他 1新增 2修改 3删除" json:"business_type"`
	Method        *string `gorm:"size:256;comment:方法" json:"method"`
	RequestMethod *string `gorm:"size:16;comment:请求方式" json:"request_method"`
	OperatorType  int32   `gorm:"type:smallint;not null;comment:操作类型 0其他 1后台 2手机" json:"operator_type"`
	OperName      *string `gorm:"size:64;comment:操作人" json:"oper_name"`
	DeptName      *string `gorm:"size:64;comment:部门" json:"dept_name"`
	OperURL       *string `gorm:"size:512;comment:请求URL" json:"oper_url"`
	OperIP        *string `gorm:"size:64;comment:操作IP" json:"oper_ip"`
	OperLocation  *string `gorm:"size:256;comment:操作地点" json:"oper_location"`
	OperParam     *string `gorm:"type:text;comment:请求参数" json:"oper_param"`
	ErrorMsg      *string `gorm:"type:text;comment:错误消息" json:"error_msg"`
	CreatedAt     int64   `gorm:"autoCreateTime:milli" json:"created_at"`
}

func (SysOperLog) TableName() string {
	return TableNameSysOperLog
}
