package mocks

import (
	"context"
	"github.com/stretchr/testify/mock"
)

// StoreMock is a mock type for the IdempotencyStore interface
type StoreMock struct {
	mock.Mock
}

// Check mocks the Check method of IdempotencyStore
func (_m *StoreMock) Check(ctx context.Context, key string) (bool, error) {
	ret := _m.Called(ctx, key)
	return ret.Get(0).(bool), ret.Error(1)
}

// Disconnect mocks the Disconnect method of IdempotencyStore
func (_m *StoreMock) Disconnect(ctx context.Context) error {
	ret := _m.Called(ctx)
	return ret.Error(0)
}
