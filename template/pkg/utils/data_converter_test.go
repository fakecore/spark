package utils

import (
	"testing"
	"time"

	"gorm.io/plugin/soft_delete"
)

type TestModel struct {
	ID        int64
	DeletedAt soft_delete.DeletedAt
}

type TestDTO struct {
	ID        int64
	DeletedAt int64
}

func TestConvertModelToDTO_SoftDelete(t *testing.T) {
	// Test case 1: Valid soft delete
	now := time.Now().UnixMilli()
	model := &TestModel{
		ID:        1,
		DeletedAt: soft_delete.DeletedAt(now),
	}
	dto := &TestDTO{}

	err := ConvertModelToDTO(model, dto)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dto.DeletedAt != now {
		t.Errorf("expected DeletedAt to be %v, got %v", now, dto.DeletedAt)
	}

	// Test case 2: Invalid soft delete (zero value)
	model = &TestModel{
		ID:        2,
		DeletedAt: 0,
	}
	dto = &TestDTO{}

	err = ConvertModelToDTO(model, dto)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dto.DeletedAt != 0 {
		t.Errorf("expected DeletedAt to be 0, got %v", dto.DeletedAt)
	}
}

func TestConvertDTOToModel_SoftDelete(t *testing.T) {
	// Test case 1: Valid soft delete timestamp
	now := time.Now().UnixMilli()
	dto := &TestDTO{
		ID:        1,
		DeletedAt: now,
	}
	model := &TestModel{}

	err := ConvertDTOToModel(dto, model)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if model.ID != 1 {
		t.Errorf("expected ID to be 1, got %v", model.ID)
	}
	if model.DeletedAt != soft_delete.DeletedAt(now) {
		t.Errorf("expected DeletedAt to be %v, got %v", soft_delete.DeletedAt(now), model.DeletedAt)
	}

	// Test case 2: Zero timestamp
	dto = &TestDTO{
		ID:        2,
		DeletedAt: 0,
	}
	model = &TestModel{}

	err = ConvertDTOToModel(dto, model)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if model.ID != 2 {
		t.Errorf("expected ID to be 2, got %v", model.ID)
	}
	if model.DeletedAt != soft_delete.DeletedAt(0) {
		t.Errorf("expected DeletedAt to be 0, got %v", model.DeletedAt)
	}
}
