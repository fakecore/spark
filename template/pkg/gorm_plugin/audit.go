package gorm_plugin

import (
	"reflect"
	"time"

	"spark/pkg/clog"
	"spark/pkg/viewer"

	"gorm.io/gorm"
)

// AuditPlugin GORM审计插件，用于自动记录操作的用户ID
type AuditPlugin struct {
	log clog.Logger
}

// NewAuditPlugin 创建新的审计插件
func NewAuditPlugin(log clog.Logger) *AuditPlugin {
	return &AuditPlugin{
		log: log,
	}
}

// Name 返回插件名称
func (p *AuditPlugin) Name() string {
	return "audit_plugin"
}

// Initialize 初始化插件，注册回调函数
func (p *AuditPlugin) Initialize(db *gorm.DB) error {
	ctx := db.Statement.Context
	p.log.Info(ctx, "Registering audit plugin callbacks")

	// 注册创建前的回调
	db.Callback().Create().Before("gorm:create").Register("audit:before_create", p.beforeCreate)

	// 注册更新前的回调
	db.Callback().Update().Before("gorm:update").Register("audit:before_update", p.beforeUpdate)

	// 注册删除前的回调
	db.Callback().Delete().Before("gorm:delete").Register("audit:before_delete", p.beforeDelete)

	p.log.Info(ctx, "Audit plugin callbacks registered successfully")
	return nil
}

// getUserID 从上下文中获取用户ID
func (p *AuditPlugin) getUserID(db *gorm.DB) (int64, error) {
	v := viewer.FromContext(db.Statement.Context)
	if v == nil {
		return 0, nil // 如果没有viewer上下文，不设置用户ID（系统操作）
	}

	// 获取用户信息
	if userViewer, ok := viewer.UserViewFromContext(db.Statement.Context); ok {
		if user := userViewer.GetUser(); user != nil {
			return user.ID, nil
		}
	}

	return 0, nil // 如果没有用户信息，不设置用户ID（系统操作）
}

// getCurrentTimestamp 获取当前时间戳（毫秒）
func (p *AuditPlugin) getCurrentTimestamp() int64 {
	return time.Now().UnixMilli()
}

// processAuditFields 处理审计字段
func (p *AuditPlugin) processAuditFields(db *gorm.DB, operation string) error {
	if db.Statement.Schema == nil {
		return nil
	}

	userID, err := p.getUserID(db)
	if err != nil {
		return err
	}

	currentTime := p.getCurrentTimestamp()

	// 处理单个实体或切片
	reflectValue := db.Statement.ReflectValue
	if reflectValue.Kind() == reflect.Slice {
		// 处理切片中的每个元素
		for i := 0; i < reflectValue.Len(); i++ {
			item := reflectValue.Index(i)
			if item.Kind() == reflect.Ptr && !item.IsNil() {
				item = item.Elem()
			}
			if err := p.processAuditFieldsForValue(db, item, operation, userID, currentTime); err != nil {
				return err
			}
		}
	} else {
		// 处理单个实体
		if err := p.processAuditFieldsForValue(db, reflectValue, operation, userID, currentTime); err != nil {
			return err
		}
	}

	return nil
}

// processAuditFieldsForValue 为单个值处理审计字段
func (p *AuditPlugin) processAuditFieldsForValue(db *gorm.DB, reflectValue reflect.Value, operation string, userID int64, currentTime int64) error {
	// 确保我们处理的是结构体
	if reflectValue.Kind() == reflect.Ptr && !reflectValue.IsNil() {
		reflectValue = reflectValue.Elem()
	}

	if reflectValue.Kind() != reflect.Struct {
		return nil
	}

	// 根据操作类型设置不同的字段
	switch operation {
	case "create":
		return p.setCreateFields(db, reflectValue, userID, currentTime)
	case "update":
		return p.setUpdateFields(db, reflectValue, userID, currentTime)
	case "delete":
		return p.setDeleteFields(db, reflectValue, userID, currentTime)
	}

	return nil
}

// setCreateFields 设置创建时的审计字段
func (p *AuditPlugin) setCreateFields(db *gorm.DB, reflectValue reflect.Value, userID int64, currentTime int64) error {
	schema := db.Statement.Schema

	// 设置创建时间
	if field := schema.LookUpField("created_at"); field != nil {
		currentValue := field.ReflectValueOf(db.Statement.Context, reflectValue)
		if currentValue.IsZero() && currentValue.CanSet() {
			currentValue.SetInt(currentTime)
		}
	}

	// 设置创建人ID
	if field := schema.LookUpField("created_by"); field != nil {
		currentValue := field.ReflectValueOf(db.Statement.Context, reflectValue)
		if currentValue.IsZero() && userID > 0 && currentValue.CanSet() {
			// 处理指针类型字段
			if currentValue.Type().Kind() == reflect.Ptr {
				if currentValue.IsNil() {
					currentValue.Set(reflect.New(currentValue.Type().Elem()))
				}
				currentValue.Elem().SetInt(userID)
			} else {
				currentValue.SetInt(userID)
			}
		}
	}

	// 创建时也设置更新时间（如果字段存在）
	if field := schema.LookUpField("updated_at"); field != nil {
		currentValue := field.ReflectValueOf(db.Statement.Context, reflectValue)
		if currentValue.IsZero() && currentValue.CanSet() {
			currentValue.SetInt(currentTime)
		}
	}

	// 创建时也设置更新人ID（如果字段存在）
	if field := schema.LookUpField("updated_by"); field != nil {
		currentValue := field.ReflectValueOf(db.Statement.Context, reflectValue)
		if currentValue.IsZero() && userID > 0 && currentValue.CanSet() {
			// 处理指针类型字段
			if currentValue.Type().Kind() == reflect.Ptr {
				if currentValue.IsNil() {
					currentValue.Set(reflect.New(currentValue.Type().Elem()))
				}
				currentValue.Elem().SetInt(userID)
			} else {
				currentValue.SetInt(userID)
			}
		}
	}

	return nil
}

// setUpdateFields 设置更新时的审计字段
func (p *AuditPlugin) setUpdateFields(db *gorm.DB, reflectValue reflect.Value, userID int64, currentTime int64) error {
	schema := db.Statement.Schema

	// 设置更新时间
	if field := schema.LookUpField("updated_at"); field != nil {
		currentValue := field.ReflectValueOf(db.Statement.Context, reflectValue)
		if currentValue.IsValid() && currentValue.CanSet() {
			currentValue.SetInt(currentTime)
		}
	}

	// 设置更新人ID
	if field := schema.LookUpField("updated_by"); field != nil {
		if userID > 0 {
			currentValue := field.ReflectValueOf(db.Statement.Context, reflectValue)
			if currentValue.IsValid() && currentValue.CanSet() {
				// 处理指针类型字段
				if currentValue.Type().Kind() == reflect.Ptr {
					if currentValue.IsNil() {
						currentValue.Set(reflect.New(currentValue.Type().Elem()))
					}
					currentValue.Elem().SetInt(userID)
				} else {
					currentValue.SetInt(userID)
				}
			}
		}
	}

	return nil
}

// setDeleteFields 设置删除时的审计字段（软删除）
func (p *AuditPlugin) setDeleteFields(db *gorm.DB, reflectValue reflect.Value, userID int64, currentTime int64) error {
	//fixme 该代码不生效
	// schema := db.Statement.Schema

	// // 检查是否为软删除
	// if field := schema.LookUpField("deleted_at"); field != nil {
	// 	// 这是软删除，设置删除时间
	// 	currentValue := field.ReflectValueOf(db.Statement.Context, reflectValue)
	// 	if currentValue.IsValid() && currentValue.CanSet() {
	// 		currentValue.SetInt(currentTime)
	// 	}
	// }

	// // 设置删除人ID（如果有该字段）
	// if field := schema.LookUpField("deleted_by"); field != nil {
	// 	if userID > 0 {
	// 		currentValue := field.ReflectValueOf(db.Statement.Context, reflectValue)
	// 		if currentValue.IsValid() && currentValue.CanSet() {
	// 			// 处理指针类型字段
	// 			if currentValue.Type().Kind() == reflect.Ptr {
	// 				if currentValue.IsNil() {
	// 					currentValue.Set(reflect.New(currentValue.Type().Elem()))
	// 				}
	// 				currentValue.Elem().SetInt(userID)
	// 			} else {
	// 				currentValue.SetInt(userID)
	// 			}
	// 		}
	// 	}
	// }

	return nil
}

// beforeCreate 创建前的回调
func (p *AuditPlugin) beforeCreate(db *gorm.DB) {
	if err := p.processAuditFields(db, "create"); err != nil {
		db.AddError(err)
		return
	}
}

// beforeUpdate 更新前的回调
func (p *AuditPlugin) beforeUpdate(db *gorm.DB) {
	if err := p.processAuditFields(db, "update"); err != nil {
		db.AddError(err)
		return
	}
}

// beforeDelete 删除前的回调
func (p *AuditPlugin) beforeDelete(db *gorm.DB) {
	if err := p.processAuditFields(db, "delete"); err != nil {
		db.AddError(err)
		return
	}
}
