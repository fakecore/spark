package utils

import (
	"gorm.io/gen"
)

type PaginateOption struct {
	EnableLargeQuery bool
}

type PaginateOptionFunc func(*PaginateOption)

func EnableLargeQuery() PaginateOptionFunc {
	return func(option *PaginateOption) {
		option.EnableLargeQuery = true
	}
}

// Paginate 通用分页函数，支持任意包含 CurrentPage 和 PageSize 字段的请求结构体
func Paginate(currentPage, pageSize int32, options ...PaginateOptionFunc) func(gen.Dao) gen.Dao {
	return func(db gen.Dao) gen.Dao {
		maxQuerySize := int32(100)
		option := &PaginateOption{}
		for _, optionFunc := range options {
			optionFunc(option)
		}

		if currentPage <= 0 {
			currentPage = 1
		}

		if option.EnableLargeQuery {
			maxQuerySize = 10000
		}

		switch {
		case pageSize > maxQuerySize:
			pageSize = maxQuerySize
		case pageSize <= 0:
			pageSize = 10
		}

		offset := (currentPage - 1) * pageSize
		return db.Offset(int(offset)).Limit(int(pageSize))
	}
}
