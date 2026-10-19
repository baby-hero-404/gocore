// Package lockmock provides test doubles for github.com/baby-hero-404/gocore/lock.
package lockmock

import (
	"context"
	"github.com/baby-hero-404/gocore/lock"
	"time"

	"github.com/stretchr/testify/mock"
)

// Locker provides a mock implementation of Locker for unit testing.
type Locker struct {
	mock.Mock
}

// NewLocker creates a new instance of Locker.
func NewLocker(t mock.TestingT) *Locker {
	m := &Locker{}
	m.Test(t)
	return m
}

func (m *Locker) Acquire(ctx context.Context, key string, ttl time.Duration, opts ...lock.Option) (lock.Lock, error) {
	args := m.Called(ctx, key, ttl)
	if l, ok := args.Get(0).(lock.Lock); ok {
		return l, args.Error(1)
	}
	return nil, args.Error(1)
}

// Lock provides a mock implementation of Lock for unit testing.
type Lock struct {
	mock.Mock
}

// NewLock creates a new instance of Lock.
func NewLock(t mock.TestingT) *Lock {
	m := &Lock{}
	m.Test(t)
	return m
}

func (m *Lock) Release(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *Lock) Extend(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *Lock) Key() string {
	args := m.Called()
	return args.String(0)
}
