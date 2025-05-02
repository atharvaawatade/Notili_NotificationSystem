package mocks

import (
	"github.com/stretchr/testify/mock"
	"github.com/appointy/notli/microservice2_messageq/internal/model"
)

// KafkaProducerMock is a mock type for the MessageProducer interface
type KafkaProducerMock struct {
	mock.Mock
}

// Enqueue mocks the Enqueue method
func (_m *KafkaProducerMock) Enqueue(msg model.NotificationMessage) error {
	ret := _m.Called(msg)
	return ret.Error(0)
}

// Close mocks the Close method
func (_m *KafkaProducerMock) Close() error {
	ret := _m.Called()
	return ret.Error(0)
}
