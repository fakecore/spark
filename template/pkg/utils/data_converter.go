package utils

import (
	"time"

	"github.com/jinzhu/copier"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

func ObjConvert(src, dst interface{}) error {
	return copier.CopyWithOption(dst, src, copier.Option{
		IgnoreEmpty: true,
		DeepCopy:    true,
		Converters: []copier.TypeConverter{
			{
				SrcType: soft_delete.DeletedAt(0),
				DstType: int64(0),
				Fn:      convertSoftDeleteToInt64,
			},
		},
	})
}

// ConvertModelToDTO 将模型转换为DTO
func ConvertModelToDTO(src, dst interface{}) error {
	return copier.CopyWithOption(dst, src, copier.Option{
		IgnoreEmpty: true,
		DeepCopy:    true,
		Converters: []copier.TypeConverter{
			{
				SrcType: time.Time{},
				DstType: string(""),
				Fn:      convertTimeToString,
			},
			{
				SrcType: &time.Time{},
				DstType: string(""),
				Fn:      convertTimeToString,
			},
			{
				SrcType: time.Time{},
				DstType: (*string)(nil),
				Fn:      convertTimeToStringPtr,
			},
			{
				SrcType: &time.Time{},
				DstType: (*string)(nil),
				Fn:      convertTimeToStringPtr,
			},
			{
				SrcType: gorm.DeletedAt{},
				DstType: string(""),
				Fn:      convertDeletedAtToString,
			},
			{
				SrcType: gorm.DeletedAt{},
				DstType: (*string)(nil),
				Fn:      convertDeletedAtToStringPtr,
			},
			{
				SrcType: soft_delete.DeletedAt(0),
				DstType: int64(0),
				Fn:      convertSoftDeleteToInt64,
			},
		},
	})
}

// ConvertDTOToModel 将DTO转换回模型
func ConvertDTOToModel(src, dst interface{}) error {
	return copier.CopyWithOption(dst, src, copier.Option{
		IgnoreEmpty: true,
		DeepCopy:    true,
		Converters: []copier.TypeConverter{
			{
				SrcType: int64(0),
				DstType: soft_delete.DeletedAt(0),
				Fn:      convertInt64ToSoftDelete,
			},
		},
	})
}

func convertTimeToString(src interface{}) (interface{}, error) {
	var t time.Time
	if pt, ok := src.(*time.Time); ok && pt != nil {
		t = *pt
	} else if tt, ok := src.(time.Time); ok {
		t = tt
	} else {
		return "", nil
	}
	return t.Format(time.RFC3339), nil
}

func convertTimeToStringPtr(src interface{}) (interface{}, error) {
	s, err := convertTimeToString(src)
	if err != nil || s == "" {
		return nil, err
	}
	str := s.(string)
	return &str, nil
}

func convertStringToTime(src interface{}) (interface{}, error) {
	var s string
	if ps, ok := src.(*string); ok && ps != nil {
		s = *ps
	} else if ss, ok := src.(string); ok {
		s = ss
	} else {
		return time.Time{}, nil
	}
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, s)
}

func convertStringToTimePtr(src interface{}) (interface{}, error) {
	t, err := convertStringToTime(src)
	if err != nil {
		return nil, err
	}
	if tt, ok := t.(time.Time); ok && !tt.IsZero() {
		return &tt, nil
	}
	return (*time.Time)(nil), nil
}

func convertDeletedAtToString(src interface{}) (interface{}, error) {
	if d, ok := src.(gorm.DeletedAt); ok && d.Valid {
		return d.Time.Format(time.RFC3339), nil
	}
	return "", nil
}

func convertDeletedAtToStringPtr(src interface{}) (interface{}, error) {
	s, err := convertDeletedAtToString(src)
	if err != nil || s == "" {
		return nil, err
	}
	str := s.(string)
	return &str, nil
}

func convertSoftDeleteToInt64(src interface{}) (interface{}, error) {
	if d, ok := src.(soft_delete.DeletedAt); ok {
		return int64(d), nil
	}
	return int64(0), nil
}

func convertInt64ToSoftDelete(src interface{}) (interface{}, error) {
	if i, ok := src.(int64); ok {
		return soft_delete.DeletedAt(i), nil
	}
	return soft_delete.DeletedAt(0), nil
}

func GetDataOrDefault[T any](data *T, defaultValue T) T {
	if data == nil {
		return defaultValue
	}
	return *data
}
