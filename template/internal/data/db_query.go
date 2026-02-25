package data

import (
	"spark/internal/data/dal/query"

	"gorm.io/gorm"
)

// NewDBQuery provides a gorm-gen query helper bound to the application's *gorm.DB.
func NewDBQuery(db *gorm.DB) *query.Query {
	return query.Use(db)
}
