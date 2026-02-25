package errors

import (
	"errors"
	"fmt"
)

// 定义具体的错误类型
var (
	// 数据不存在错误
	ErrNotFound      = errors.New("resource not found")
	ErrUserNotFound  = errors.New("user not found")
	ErrConfigNotFound = errors.New("config not found")
	ErrRoleNotFound  = errors.New("role not found")
	ErrMenuNotFound  = errors.New("menu not found")
	ErrDictNotFound  = errors.New("dictionary not found")
	ErrDeptNotFound  = errors.New("department not found")
	ErrPostNotFound  = errors.New("post not found")

	// 数据已存在错误
	ErrAlreadyExists     = errors.New("resource already exists")
	ErrDuplicateUsername = errors.New("username already exists")
	ErrDuplicateConfigKey = errors.New("config key already exists")
	ErrDuplicateRole     = errors.New("role already exists")
	ErrDuplicateDictCode = errors.New("dictionary code already exists")
	ErrDuplicatePostCode = errors.New("post code already exists")
	ErrDuplicateDeptName = errors.New("department name already exists")
	ErrDuplicateMenuName = errors.New("menu name already exists")

	// 关联错误
	ErrRelationNotFound  = errors.New("relation not found")
	ErrDataInconsistency = errors.New("data inconsistency")
	ErrInvalidRelation   = errors.New("invalid relation")
	ErrCircularReference = errors.New("circular reference")

	// 业务规则错误
	ErrBusinessRule       = errors.New("business rule violation")
	ErrInvalidStatus      = errors.New("invalid status")
	ErrInvalidOperation   = errors.New("invalid operation")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserDisabled       = errors.New("user account disabled")
	ErrNoRoleAssigned     = errors.New("no role assigned to user")
	ErrAllRolesDisabled   = errors.New("all user roles are disabled")

	// 验证错误
	ErrValidation    = errors.New("validation error")
	ErrRequiredField = errors.New("required field missing")
	ErrInvalidFormat = errors.New("invalid format")
	ErrOutOfRange    = errors.New("value out of range")
	ErrInvalidParam  = errors.New("invalid parameter")

	// 系统错误
	ErrDatabase      = errors.New("database error")
	ErrNetwork       = errors.New("network error")
	ErrTimeout       = errors.New("timeout error")
	ErrInternal      = errors.New("internal error")
	ErrTransaction   = errors.New("transaction error")
	ErrHashPassword  = errors.New("password hash error")
	ErrCreateToken   = errors.New("token creation error")
	ErrRedis         = errors.New("redis error")
	ErrJSONMarshal   = errors.New("json marshal error")
	ErrJSONUnmarshal = errors.New("json unmarshal error")
)

// ErrorWithContext 包含上下文信息的错误
type ErrorWithContext struct {
	Type    error
	Message string
	Context map[string]interface{}
	Cause   error
}

func (e *ErrorWithContext) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Type.Error()
}

func (e *ErrorWithContext) Unwrap() error {
	return e.Cause
}

func (e *ErrorWithContext) Is(target error) bool {
	return errors.Is(e.Type, target)
}

// NewError 创建带上下文的错误
func NewError(errType error, message string, context map[string]interface{}) *ErrorWithContext {
	return &ErrorWithContext{
		Type:    errType,
		Message: message,
		Context: context,
	}
}

// WrapError 包装错误并添加上下文
func WrapError(errType error, cause error, message string, context map[string]interface{}) *ErrorWithContext {
	return &ErrorWithContext{
		Type:    errType,
		Message: message,
		Context: context,
		Cause:   cause,
	}
}

// 具体的错误构造函数 - Resource Not Found
func NewUserNotFound(id int64) *ErrorWithContext {
	return NewError(ErrUserNotFound, fmt.Sprintf("user %d not found", id), map[string]interface{}{
		"resource_type": "user",
		"resource_id":   id,
	})
}

func NewConfigNotFound(key string) *ErrorWithContext {
	return NewError(ErrConfigNotFound, fmt.Sprintf("config not found for key: %s", key), map[string]interface{}{
		"resource_type": "config",
		"config_key":    key,
	})
}

func NewDictNotFound(id int64) *ErrorWithContext {
	return NewError(ErrDictNotFound, fmt.Sprintf("dictionary %d not found", id), map[string]interface{}{
		"resource_type": "dictionary",
		"resource_id":   id,
	})
}

func NewRoleNotFound(id int64) *ErrorWithContext {
	return NewError(ErrRoleNotFound, fmt.Sprintf("role %d not found", id), map[string]interface{}{
		"resource_type": "role",
		"resource_id":   id,
	})
}

func NewPostNotFound(id int64) *ErrorWithContext {
	return NewError(ErrPostNotFound, fmt.Sprintf("post %d not found", id), map[string]interface{}{
		"resource_type": "post",
		"resource_id":   id,
	})
}

func NewMenuNotFound(id int64) *ErrorWithContext {
	return NewError(ErrMenuNotFound, fmt.Sprintf("menu %d not found", id), map[string]interface{}{
		"resource_type": "menu",
		"resource_id":   id,
	})
}

func NewDeptNotFound(id int64) *ErrorWithContext {
	return NewError(ErrDeptNotFound, fmt.Sprintf("department %d not found", id), map[string]interface{}{
		"resource_type": "department",
		"resource_id":   id,
	})
}

// Duplicate Resource Errors
func NewDuplicateUsername(username string) *ErrorWithContext {
	return NewError(ErrDuplicateUsername, fmt.Sprintf("username '%s' already exists", username), map[string]interface{}{
		"resource_type": "user",
		"username":      username,
	})
}

func NewDuplicateConfigKey(key string) *ErrorWithContext {
	return NewError(ErrDuplicateConfigKey, fmt.Sprintf("config key '%s' already exists", key), map[string]interface{}{
		"resource_type": "config",
		"config_key":    key,
	})
}

func NewDuplicateDictCode(dictCode string) *ErrorWithContext {
	return NewError(ErrDuplicateDictCode, fmt.Sprintf("dictionary code '%s' already exists", dictCode), map[string]interface{}{
		"resource_type": "dictionary",
		"dict_code":     dictCode,
	})
}

func NewDuplicatePostCode(postCode string) *ErrorWithContext {
	return NewError(ErrDuplicatePostCode, fmt.Sprintf("post code '%s' already exists", postCode), map[string]interface{}{
		"resource_type": "post",
		"post_code":     postCode,
	})
}

func NewDuplicateDeptName(deptName string) *ErrorWithContext {
	return NewError(ErrDuplicateDeptName, fmt.Sprintf("department name '%s' already exists", deptName), map[string]interface{}{
		"resource_type": "department",
		"dept_name":     deptName,
	})
}

func NewDuplicateMenuName(menuName string) *ErrorWithContext {
	return NewError(ErrDuplicateMenuName, fmt.Sprintf("menu name '%s' already exists", menuName), map[string]interface{}{
		"resource_type": "menu",
		"menu_name":     menuName,
	})
}

// Relation Errors
func NewRelationNotFound(sourceType string, sourceID int64, targetType string, targetID int64) *ErrorWithContext {
	return NewError(ErrRelationNotFound, fmt.Sprintf("%s %d has no relation to %s %d", sourceType, sourceID, targetType, targetID), map[string]interface{}{
		"source_type": sourceType,
		"source_id":   sourceID,
		"target_type": targetType,
		"target_id":   targetID,
	})
}

func NewDataInconsistency(description string, context map[string]interface{}) *ErrorWithContext {
	return NewError(ErrDataInconsistency, description, context)
}

// Validation Errors
func NewValidationError(field string, reason string) *ErrorWithContext {
	return NewError(ErrValidation, fmt.Sprintf("validation failed for field '%s': %s", field, reason), map[string]interface{}{
		"field":  field,
		"reason": reason,
	})
}

// Business Rule Errors
func NewBusinessRuleError(rule string, context map[string]interface{}) *ErrorWithContext {
	return NewError(ErrBusinessRule, fmt.Sprintf("business rule violation: %s", rule), context)
}

func NewInvalidCredentials(username string) *ErrorWithContext {
	return NewError(ErrInvalidCredentials, "invalid username or password", map[string]interface{}{
		"username": username,
	})
}

// System Errors
func NewDatabaseError(operation string, cause error) *ErrorWithContext {
	return WrapError(ErrDatabase, cause, fmt.Sprintf("database error during %s", operation), map[string]interface{}{
		"operation": operation,
	})
}

func NewTransactionError(operation string, cause error) *ErrorWithContext {
	return WrapError(ErrTransaction, cause, fmt.Sprintf("transaction error during %s", operation), map[string]interface{}{
		"operation": operation,
	})
}

func NewRedisError(operation string, cause error) *ErrorWithContext {
	return WrapError(ErrRedis, cause, fmt.Sprintf("redis error during %s", operation), map[string]interface{}{
		"operation": operation,
	})
}

// 错误类型检查函数
func IsNotFound(err error) bool {
	var e *ErrorWithContext
	if errors.As(err, &e) {
		return errors.Is(e.Type, ErrNotFound) ||
			errors.Is(e.Type, ErrUserNotFound) ||
			errors.Is(e.Type, ErrConfigNotFound) ||
			errors.Is(e.Type, ErrRoleNotFound) ||
			errors.Is(e.Type, ErrMenuNotFound) ||
			errors.Is(e.Type, ErrDictNotFound) ||
			errors.Is(e.Type, ErrDeptNotFound) ||
			errors.Is(e.Type, ErrPostNotFound)
	}
	return false
}

func IsDuplicateError(err error) bool {
	var e *ErrorWithContext
	if errors.As(err, &e) {
		return errors.Is(e.Type, ErrAlreadyExists) ||
			errors.Is(e.Type, ErrDuplicateUsername) ||
			errors.Is(e.Type, ErrDuplicateConfigKey) ||
			errors.Is(e.Type, ErrDuplicateRole) ||
			errors.Is(e.Type, ErrDuplicateDictCode) ||
			errors.Is(e.Type, ErrDuplicatePostCode) ||
			errors.Is(e.Type, ErrDuplicateDeptName) ||
			errors.Is(e.Type, ErrDuplicateMenuName)
	}
	return false
}

func IsRelationError(err error) bool {
	var e *ErrorWithContext
	if errors.As(err, &e) {
		return errors.Is(e.Type, ErrRelationNotFound) ||
			errors.Is(e.Type, ErrDataInconsistency) ||
			errors.Is(e.Type, ErrInvalidRelation) ||
			errors.Is(e.Type, ErrCircularReference)
	}
	return false
}

func IsValidationError(err error) bool {
	var e *ErrorWithContext
	if errors.As(err, &e) {
		return errors.Is(e.Type, ErrValidation) ||
			errors.Is(e.Type, ErrRequiredField) ||
			errors.Is(e.Type, ErrInvalidFormat) ||
			errors.Is(e.Type, ErrOutOfRange) ||
			errors.Is(e.Type, ErrInvalidParam)
	}
	return false
}

func IsBusinessRuleError(err error) bool {
	var e *ErrorWithContext
	if errors.As(err, &e) {
		return errors.Is(e.Type, ErrBusinessRule) ||
			errors.Is(e.Type, ErrInvalidStatus) ||
			errors.Is(e.Type, ErrInvalidOperation) ||
			errors.Is(e.Type, ErrInvalidCredentials) ||
			errors.Is(e.Type, ErrUserDisabled) ||
			errors.Is(e.Type, ErrNoRoleAssigned) ||
			errors.Is(e.Type, ErrAllRolesDisabled)
	}
	return false
}

func IsSystemError(err error) bool {
	var e *ErrorWithContext
	if errors.As(err, &e) {
		return errors.Is(e.Type, ErrDatabase) ||
			errors.Is(e.Type, ErrNetwork) ||
			errors.Is(e.Type, ErrTimeout) ||
			errors.Is(e.Type, ErrInternal) ||
			errors.Is(e.Type, ErrTransaction) ||
			errors.Is(e.Type, ErrHashPassword) ||
			errors.Is(e.Type, ErrCreateToken) ||
			errors.Is(e.Type, ErrRedis) ||
			errors.Is(e.Type, ErrJSONMarshal) ||
			errors.Is(e.Type, ErrJSONUnmarshal)
	}
	return false
}

// GetErrorContext 获取错误的上下文信息
func GetErrorContext(err error) map[string]interface{} {
	var e *ErrorWithContext
	if errors.As(err, &e) {
		return e.Context
	}
	return nil
}

// TranslatableError 是可翻译的错误类型，包含翻译键和元数据
type TranslatableError struct {
	Reason       string                 // 错误原因码
	TemplateData map[string]interface{} // 模板数据
	HTTPCode     int32                  // HTTP 状态码
	Cause        error                  // 原始错误
}

func (e *TranslatableError) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return e.Reason
}

func (e *TranslatableError) Unwrap() error {
	return e.Cause
}

// NewTranslatableError 创建一个可翻译的错误
func NewTranslatableError(reason string, httpCode int32, templateData map[string]interface{}) *TranslatableError {
	return &TranslatableError{
		Reason:       reason,
		TemplateData: templateData,
		HTTPCode:     httpCode,
	}
}

// NewTranslatableErrorWithCause 创建一个带原始错误的可翻译错误
func NewTranslatableErrorWithCause(reason string, httpCode int32, cause error, templateData map[string]interface{}) *TranslatableError {
	return &TranslatableError{
		Reason:       reason,
		TemplateData: templateData,
		HTTPCode:     httpCode,
		Cause:        cause,
	}
}

// IsTranslatableError 检查错误是否为可翻译错误
func IsTranslatableError(err error) (*TranslatableError, bool) {
	var e *TranslatableError
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}
