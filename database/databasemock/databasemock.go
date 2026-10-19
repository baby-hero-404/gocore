// Package databasemock provides test doubles for github.com/baby-hero-404/gocore/database.
package databasemock

import (
	"context"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// Database is a mock implementation of Database for unit testing downstream microservices.
type Database struct {
	mock.Mock
}

// NewDatabase creates a new instance of Database.
func NewDatabase(t mock.TestingT) *Database {
	m := &Database{}
	m.Test(t)
	return m
}

// DB returns mock *gorm.DB.
func (m *Database) DB() *gorm.DB {
	args := m.Called()
	if db, ok := args.Get(0).(*gorm.DB); ok {
		return db
	}
	return nil
}

// NewDb returns mock *gorm.DB for session.
func (m *Database) NewDb(ctx context.Context) *gorm.DB {
	args := m.Called(ctx)
	if db, ok := args.Get(0).(*gorm.DB); ok {
		return db
	}
	return nil
}

// WithContext returns mock *gorm.DB with context.
func (m *Database) WithContext(ctx context.Context) *gorm.DB {
	args := m.Called(ctx)
	if db, ok := args.Get(0).(*gorm.DB); ok {
		return db
	}
	return nil
}

// Ping mocks connectivity check.
func (m *Database) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Close mocks closing the database.
func (m *Database) Close() error {
	args := m.Called()
	return args.Error(0)
}
