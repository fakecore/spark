package do

// UserProfileAnalysis 用户画像分析 DO
type UserProfileAnalysis struct {
	ID              int64    `json:"id,omitempty"`
	AudioID         int64    `json:"audio_id,omitempty"`
	UserID          *int64   `json:"user_id,omitempty"`
	ProjectID       *int64   `json:"project_id,omitempty"`
	CategoryName    *string  `json:"category_name,omitempty"`
	ProfileName     *string  `json:"profile_name,omitempty"`
	ProfileValue    *string  `json:"profile_value,omitempty"`
	AnalysisType    *string  `json:"analysis_type,omitempty"`
	ProfileType     *int32   `json:"profile_type,omitempty"`
	AnalysisReason  *string  `json:"analysis_reason,omitempty"`
	ConfidenceScore *float64 `json:"confidence_score,omitempty"`
	Version         int32    `json:"version,omitempty"`
	AnalysisStatus  *int32   `json:"analysis_status,omitempty"`
	CreatedAt       int64    `json:"created_at,omitempty"`
	CreatedBy       *int64   `json:"created_by,omitempty"`
	UpdatedAt       int64    `json:"updated_at,omitempty"`
	UpdatedBy       *int64   `json:"updated_by,omitempty"`
	DeletedAt       *int64   `json:"deleted_at,omitempty"`
	DeletedBy       *int64   `json:"deleted_by,omitempty"`
}

// ListUserProfileAnalysisRequest 查询用户画像分析列表请求
type ListUserProfileAnalysisRequest struct {
	PageSize       int32   `json:"page_size,omitempty"`
	Current        int32   `json:"current,omitempty"`
	AudioID        *int64  `json:"audio_id,omitempty"`
	CustomerID     *int64  `json:"customer_id,omitempty"`
	ProjectID      *int64  `json:"project_id,omitempty"`
	CategoryName   *string `json:"category_name,omitempty"`
	ProfileName    *string `json:"profile_name,omitempty"`
	ProfileType    *int32  `json:"profile_type,omitempty"`
	AnalysisType   *string `json:"analysis_type,omitempty"`
	AnalysisStatus *int32  `json:"analysis_status,omitempty"`
	Version        *int32  `json:"version,omitempty"`
	CreatedAtStart *int64  `json:"created_at_start,omitempty"`
	CreatedAtEnd   *int64  `json:"created_at_end,omitempty"`
}

// GetAnalysisStatsRequest 统计分析请求
type GetAnalysisStatsRequest struct {
	ProjectID      *int64 `json:"project_id,omitempty"`
	CustomerID     *int64 `json:"customer_id,omitempty"`
	AudioID        *int64 `json:"audio_id,omitempty"`
	CreatedAtStart *int64 `json:"created_at_start,omitempty"`
	CreatedAtEnd   *int64 `json:"created_at_end,omitempty"`
}

// AnalysisStats 统计分析结果
type AnalysisStats struct {
	TotalAnalyses    int64            `json:"total_analyses,omitempty"`
	TotalAudios      int64            `json:"total_audios,omitempty"`
	CategoryStats    map[string]int64 `json:"category_stats,omitempty"`
	ProfileTypeStats map[string]int64 `json:"profile_type_stats,omitempty"`
	VersionStats     []VersionStats   `json:"version_stats,omitempty"`
}

// VersionStats 版本统计
type VersionStats struct {
	Version   int32 `json:"version,omitempty"`
	Count     int64 `json:"count,omitempty"`
	CreatedAt int64 `json:"created_at,omitempty"`
}

// CreateUserProfileAnalysisRequest 创建用户画像分析请求
type CreateUserProfileAnalysisRequest struct {
	AudioID        int64  `json:"audio_id,omitempty"`
	CustomerID     *int64 `json:"customer_id,omitempty"`
	VirtCustomerID *int64 `json:"virt_customer_id,omitempty"` // 虚拟客户ID，用于临时客户标识
	ProjectID      *int64 `json:"project_id,omitempty"`
	CategoryName   string `json:"category_name,omitempty"`
	ProfileName    string `json:"profile_name,omitempty"`
	ProfileValue   string `json:"profile_value,omitempty"`
	AnalysisType   string `json:"analysis_type,omitempty"`
	ProfileType    int32  `json:"profile_type,omitempty"`
	AnalysisReason string `json:"analysis_reason,omitempty"`
}

// BatchCreateUserProfileAnalysisRequest 批量创建用户画像分析请求
type BatchCreateUserProfileAnalysisRequest struct {
	Analyses []*CreateUserProfileAnalysisRequest `json:"analyses,omitempty"`
}

// UpdateUserProfileAnalysisRequest 更新用户画像分析请求
type UpdateUserProfileAnalysisRequest struct {
	ID             int64   `json:"id,omitempty"`
	ProfileValue   *string `json:"profile_value,omitempty"`
	AnalysisReason *string `json:"analysis_reason,omitempty"`
	AnalysisStatus *int32  `json:"analysis_status,omitempty"`
}

// OfflinePortraitAnalysisTag 表示单个画像标签
type OfflinePortraitAnalysisTag struct {
	CategoryName string `json:"category_name"` // 类别名称
	ProfileName  string `json:"profile_name"`  // 画像名称
	ProfileValue string `json:"profile_value"` // 画像值
	ObtainType   string `json:"obtain_type"`   // 获取方式
	Reason       string `json:"reason"`        // 分析理由
}

// OfflinePortraitAnalysisCategory 表示单个分类的分析结果
type OfflinePortraitAnalysisCategory struct {
	ProjectID   string                       `json:"project_id"`   // 项目ID (字符串格式)
	CustomerID  string                       `json:"customer_id"`  // 客户ID (字符串格式)
	ProfileType string                       `json:"profile_type"` // 画像类型 (字符串格式)
	Tags        []OfflinePortraitAnalysisTag `json:"tags"`         // 画像标签列表
}

// OfflinePortraitAnalysis 表示离线画像分析结构化输出
type OfflinePortraitAnalysis struct {
	CustomerOverview    *OfflinePortraitAnalysisCategory `json:"customer_overview,omitempty"`    // 客户描摹
	CustomerFeedback    *OfflinePortraitAnalysisCategory `json:"customer_feedback,omitempty"`    // 客户反馈
	CustomerLevel       *OfflinePortraitAnalysisCategory `json:"customer_level,omitempty"`       // 客户等级
	ProductValue        *OfflinePortraitAnalysisCategory `json:"product_value,omitempty"`        // 产品价值
	CustomerPersonality *OfflinePortraitAnalysisCategory `json:"customer_personality,omitempty"` // 客户性格
	CustomerStrategy    *OfflinePortraitAnalysisCategory `json:"customer_strategy,omitempty"`    // 跟进策略
	CustomerProfile     *OfflinePortraitAnalysisCategory `json:"customer_profile,omitempty"`     // 用户画像
}

type PortraitAnalysisRequest struct {
	Content    string `json:"content"`
	CustomerID int64  `json:"customer_id"`
	ProjectID  int64  `json:"project_id"`
}
