package utils

import (
	"context"

	"spark/internal/data/dal/query"

	"gorm.io/gorm"
)

// txKey 用于在context中存储事务的key
type txKey struct{}

// WithTx 将事务放入context
func WithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// GetTx 从context中获取事务
func GetTx(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(txKey{}).(*gorm.DB)
	return tx, ok
}

// GetDB 从context中获取事务，如果没有则返回默认DB
func GetDB(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := GetTx(ctx); ok && tx != nil {
		return tx
	}
	return db.WithContext(ctx)
}

// GetQuery 从context中获取Query对象（支持gorm gen）
// 如果context中有事务，返回使用该事务的Query；否则返回使用默认DB的Query
func GetQuery(ctx context.Context, db *gorm.DB) *query.Query {
	if tx, ok := GetTx(ctx); ok && tx != nil {
		return query.Use(tx)
	}
	return query.Use(db)
}
