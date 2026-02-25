package errors

// 常用可翻译错误构造函数

// NewPermissionDeniedTranslatable 权限不足错误
func NewPermissionDeniedTranslatable(action string) *TranslatableError {
	return NewTranslatableError(
		"BIZ_PERMISSION_DENIED",
		403,
		map[string]interface{}{
			"Action": action,
		},
	)
}

// NewInvalidCredentialsTranslatable 登录凭据无效错误
func NewInvalidCredentialsTranslatable() *TranslatableError {
	return NewTranslatableError(
		"BIZ_INVALID_CREDENTIALS",
		401,
		nil,
	)
}

// NewUserDisabledTranslatable 用户已禁用错误
func NewUserDisabledTranslatable() *TranslatableError {
	return NewTranslatableError(
		"BIZ_USER_DISABLED",
		403,
		nil,
	)
}

// NewDuplicateUsernameTranslatable 用户名重复错误
func NewDuplicateUsernameTranslatable(username string) *TranslatableError {
	return NewTranslatableError(
		"DUPLICATE_USERNAME",
		400,
		map[string]interface{}{
			"Username": username,
		},
	)
}

// NewResourceNotFoundTranslatable 资源不存在错误
func NewResourceNotFoundTranslatable(resourceType string, id int64) *TranslatableError {
	return NewTranslatableError(
		"RESOURCE_NOT_FOUND",
		404,
		map[string]interface{}{
			"ResourceType": resourceType,
			"ID":           id,
		},
	)
}

// NewValidationErrorTranslatable 验证错误
func NewValidationErrorTranslatable(field, reason string) *TranslatableError {
	return NewTranslatableError(
		"VALIDATION_ERROR",
		400,
		map[string]interface{}{
			"Field":  field,
			"Reason": reason,
		},
	)
}

// NewSystemErrorTranslatable 系统错误
func NewSystemErrorTranslatable(operation string) *TranslatableError {
	return NewTranslatableError(
		"SYSTEM_ERROR",
		500,
		map[string]interface{}{
			"Operation": operation,
		},
	)
}
