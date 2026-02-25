package utils

import (
	"reflect"
	"strings"
)

type Option func(*options)

type options struct {
	ignoredFields map[string]bool // 存储小写无下划线的字段名（如 userid）
}

func IgnoreFields(fields ...string) Option {
	return func(o *options) {
		for _, field := range fields {
			o.ignoredFields[strings.ToLower(field)] = true
		}
	}
}

// IgnoreFieldsWithDefault 忽略默认字段（ID/CreatedAt等）和自定义字段
func IgnoreFieldsWithDefault(fields ...string) Option {
	return func(o *options) {
		if o.ignoredFields == nil {
			o.ignoredFields = make(map[string]bool)
		}
		defaultFields := []string{"id", "createdat", "updatedat", "deletedat", "created_at", "updated_at", "deleted_at"}
		for _, field := range append(defaultFields, fields...) {
			o.ignoredFields[removeSnakeCase(strings.ToLower(field))] = true
		}
	}
}

// StructToMap 生成适配 GORM 的 map（字段名下划线格式）
// 默认忽略 utils.IgnoreFieldsWithDefault()
// 支持 override 机制：如果 opts 里有 IgnoreFieldsWithDefaultOverride()，则不自动加默认忽略
func StructToMap(obj interface{}, opts ...Option) map[string]interface{} {
	opt := &options{
		ignoredFields: make(map[string]bool),
	}
	// 检查是否有 override
	override := false
	for _, o := range opts {
		if isIgnoreFieldsWithDefaultOverride(o) {
			override = true
			break
		}
	}
	// 没有 override 时，默认加上 IgnoreFieldsWithDefault
	if !override {
		IgnoreFieldsWithDefault()(opt)
	}
	// 再应用用户自定义
	for _, o := range opts {
		o(opt)
	}

	result := make(map[string]interface{})
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return result
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		structField := t.Field(i)

		fieldName := strings.ToLower(structField.Name)
		// 1. 获取字段名（优先用 JSON 标签）
		fieldDBName := getFieldNameFromJsonTag(structField)
		if fieldDBName == "" {
			continue
		}

		// 2. 检查是否忽略字段
		if opt.ignoredFields[fieldName] {
			continue
		}

		// 3. 处理 omitempty
		if strings.Contains(structField.Tag.Get("json"), "omitempty") {
			if isEmptyValue(field) {
				continue
			}
		}

		// 4. 添加字段到结果
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			result[fieldDBName] = field.Elem().Interface()
		} else if field.IsValid() && field.CanInterface() {
			result[fieldDBName] = field.Interface()
		}
	}
	return result
}

// IgnoreFieldsWithDefaultOverride 用于覆盖默认的 IgnoreFieldsWithDefault 行为
func IgnoreFieldsWithDefaultOverride() Option {
	return func(o *options) {
		// 什么都不做，仅用于标记
	}
}

// isIgnoreFieldsWithDefaultOverride 判断 Option 是否为 IgnoreFieldsWithDefaultOverride
func isIgnoreFieldsWithDefaultOverride(opt Option) bool {
	// 通过反射判断函数名
	return getFuncName(opt) == getFuncName(IgnoreFieldsWithDefaultOverride())
}

// getFuncName 获取函数名
func getFuncName(i interface{}) string {
	return strings.TrimPrefix(strings.TrimSuffix(reflect.TypeOf(i).String(), "-fm"), "func(")
}

func getFieldNameFromJsonTag(field reflect.StructField) string {
	name := field.Name
	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if parts[0] == "-" {
			return "" // 明确跳过
		}
		if parts[0] != "" {
			name = parts[0] // 使用 JSON 标签名
		}
	}
	return name
}

// removeSnakeCase 去除下划线（user_id -> userid）
func removeSnakeCase(s string) string {
	return strings.ReplaceAll(s, "_", "")
}

// isEmptyValue 检查零值
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}
