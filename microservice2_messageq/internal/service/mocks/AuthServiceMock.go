package mocks

import (
	"github.com/stretchr/testify/mock"
)

// AuthServiceMock is a mock type for the AuthClient interface
type AuthServiceMock struct {
	mock.Mock
}

// ValidateAPIKey mocks the ValidateAPIKey method
func (_m *AuthServiceMock) ValidateAPIKey(apiKey string) error {
	ret := _m.Called(apiKey)
	return ret.Error(0)
}
