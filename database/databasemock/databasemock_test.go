package databasemock

import (
	"context"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
	"testing"
)

func TestMockDatabase(t *testing.T) {
	mockDB := NewDatabase(t)
	mockDB.On("Ping", mock.Anything).Return(nil)
	mockDB.On("Close").Return(nil)
	mockDB.On("DB").Return((*gorm.DB)(nil))
	mockDB.On("NewDb", mock.Anything).Return((*gorm.DB)(nil))
	mockDB.On("WithContext", mock.Anything).Return((*gorm.DB)(nil))

	ctx := context.Background()
	if err := mockDB.Ping(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := mockDB.Close(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if db := mockDB.DB(); db != nil {
		t.Errorf("expected nil db, got %v", db)
	}
	if session := mockDB.NewDb(ctx); session != nil {
		t.Errorf("expected nil session, got %v", session)
	}
	if ctxDB := mockDB.WithContext(ctx); ctxDB != nil {
		t.Errorf("expected nil ctxDB, got %v", ctxDB)
	}
}
