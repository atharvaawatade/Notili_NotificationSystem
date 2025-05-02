package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	kafkamocks "github.com/appointy/notli/microservice2_messageq/internal/kafka/mocks"
	servicemocks "github.com/appointy/notli/microservice2_messageq/internal/service/mocks"
	storemocks "github.com/appointy/notli/microservice2_messageq/internal/store/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// EmailNotificationRequest is a test-specific struct for API testing
type EmailNotificationRequest struct {
	IdempotencyKey string   `json:"idempotency_key"`
	Recipient      string   `json:"recipient"`
	Subject        string   `json:"subject"`
	Body           string   `json:"body"`
	FromName       string   `json:"from_name,omitempty"`
	ReplyTo        []string `json:"reply_to,omitempty"`
}

// setupTestRouter creates a gin engine in test mode and registers the handler with mocks.
func setupTestRouter(authMock *servicemocks.AuthServiceMock, storeMock *storemocks.StoreMock, kafkaMock *kafkamocks.KafkaProducerMock) (*gin.Engine, *EmailHandler) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	emailHandler := NewEmailHandler(authMock, storeMock, kafkaMock)
	RegisterEmailRoutes(router, emailHandler)
	return router, emailHandler
}

func TestEmailNotifyHandler_Success(t *testing.T) {
	// Mocks
	authMock := new(servicemocks.AuthServiceMock)
	storeMock := new(storemocks.StoreMock)
	kafkaMock := new(kafkamocks.KafkaProducerMock)
	router, _ := setupTestRouter(authMock, storeMock, kafkaMock)

	// Mock Expectations
	apiKey := "valid-api-key"
	idempotencyKey := "550e8400-e29b-41d4-a716-446655440000" // Valid UUID format
	authMock.On("ValidateAPIKey", apiKey).Return(nil)
	storeMock.On("Check", mock.Anything, idempotencyKey).Return(false, nil) // Not a duplicate
	kafkaMock.On("Enqueue", mock.AnythingOfType("model.NotificationMessage")).Return(nil)

	// Request
	body := EmailNotificationRequest{
		IdempotencyKey: idempotencyKey,
		Recipient:      "test@example.com",
		Subject:        "Test Subject",
		Body:           "Test Body",
		FromName:       "Test Sender",
		ReplyTo:        []string{"replyto@example.com"},
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/v1/email/notify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("X-Idempotency-Key", idempotencyKey)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusAccepted, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Notification accepted for processing", response["message"])
	authMock.AssertExpectations(t)
	storeMock.AssertExpectations(t)
	kafkaMock.AssertExpectations(t)
}

func TestEmailNotifyHandler_DuplicateRequest(t *testing.T) {
	authMock := new(servicemocks.AuthServiceMock)
	storeMock := new(storemocks.StoreMock)
	kafkaMock := new(kafkamocks.KafkaProducerMock)
	router, _ := setupTestRouter(authMock, storeMock, kafkaMock)

	apiKey := "valid-api-key"
	idempotencyKey := "550e8400-e29b-41d4-a716-446655440001" // Valid UUID format
	authMock.On("ValidateAPIKey", apiKey).Return(nil)
	storeMock.On("Check", mock.Anything, idempotencyKey).Return(true, nil) // Is a duplicate

	body := EmailNotificationRequest{IdempotencyKey: idempotencyKey, Recipient: "dup@example.com", Subject: "Dup", Body: "Dup", FromName: "Sender"}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/v1/email/notify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("X-Idempotency-Key", idempotencyKey)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code) // Conflict for duplicate requests
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Duplicate request", response["error"])
	assert.Equal(t, idempotencyKey, response["idempotency_key"])
	authMock.AssertExpectations(t)
	storeMock.AssertExpectations(t)
	kafkaMock.AssertNotCalled(t, "Enqueue", mock.Anything)
}

func TestEmailNotifyHandler_InvalidAPIKey(t *testing.T) {
	authMock := new(servicemocks.AuthServiceMock)
	storeMock := new(storemocks.StoreMock)
	kafkaMock := new(kafkamocks.KafkaProducerMock)
	router, _ := setupTestRouter(authMock, storeMock, kafkaMock)

	apiKey := "invalid-api-key"
	authMock.On("ValidateAPIKey", apiKey).Return(errors.New("invalid or unauthorized API key"))

	body := EmailNotificationRequest{IdempotencyKey: "550e8400-e29b-41d4-a716-446655440005", Recipient: "invalidkey@example.com", Subject: "Invalid", Body: "Invalid", FromName: "Sender"}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/v1/email/notify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("X-Idempotency-Key", "some-key")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	authMock.AssertExpectations(t)
	storeMock.AssertNotCalled(t, "Check", mock.Anything, mock.Anything)
	kafkaMock.AssertNotCalled(t, "Enqueue", mock.Anything)
}

func TestEmailNotifyHandler_MissingAPIKeyHeader(t *testing.T) {
	authMock := new(servicemocks.AuthServiceMock)
	storeMock := new(storemocks.StoreMock)
	kafkaMock := new(kafkamocks.KafkaProducerMock)
	router, _ := setupTestRouter(authMock, storeMock, kafkaMock)

	body := EmailNotificationRequest{IdempotencyKey: "550e8400-e29b-41d4-a716-446655440006", Recipient: "nokey@example.com", Subject: "No Key", Body: "No Key", FromName: "Sender"}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/v1/email/notify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Idempotency-Key", "some-key")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	authMock.AssertNotCalled(t, "ValidateAPIKey", mock.Anything, mock.Anything)
	storeMock.AssertNotCalled(t, "Check", mock.Anything, mock.Anything)
	kafkaMock.AssertNotCalled(t, "Enqueue", mock.Anything)
}

func TestEmailNotifyHandler_MissingIdempotencyKeyHeader(t *testing.T) {
	authMock := new(servicemocks.AuthServiceMock)
	storeMock := new(storemocks.StoreMock)
	kafkaMock := new(kafkamocks.KafkaProducerMock)
	router, _ := setupTestRouter(authMock, storeMock, kafkaMock)

	apiKey := "valid-api-key"
	authMock.On("ValidateAPIKey", apiKey).Return(nil)

	body := EmailNotificationRequest{Recipient: "noidem@example.com", Subject: "No Idem", Body: "No Idem", FromName: "Sender"}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/v1/email/notify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "X-Idempotency-Key header is required")
	authMock.AssertExpectations(t)
	storeMock.AssertNotCalled(t, "Check", mock.Anything, mock.Anything)
	kafkaMock.AssertNotCalled(t, "Enqueue", mock.Anything)
}

func TestEmailNotifyHandler_InvalidInput(t *testing.T) {
	authMock := new(servicemocks.AuthServiceMock)
	storeMock := new(storemocks.StoreMock)
	kafkaMock := new(kafkamocks.KafkaProducerMock)
	router, _ := setupTestRouter(authMock, storeMock, kafkaMock)

	apiKey := "valid-api-key"
	idempotencyKey := "550e8400-e29b-41d4-a716-446655440002" // Valid UUID format
	authMock.On("ValidateAPIKey", apiKey).Return(nil)
	// No store check or produce should happen

	// Request with missing required fields (e.g., Recipients)
	body := EmailNotificationRequest{
		IdempotencyKey: idempotencyKey,
		Subject:        "Test Subject",
		Body:           "Test Body",
		FromName:       "Sender",
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/v1/email/notify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("X-Idempotency-Key", idempotencyKey)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assertions
	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]interface{} // Use interface{} for more flexible error format
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "validation failed")
	assert.Contains(t, response["details"], "Recipients is a required field")
	authMock.AssertExpectations(t)
	storeMock.AssertNotCalled(t, "Check", mock.Anything, mock.Anything)
	kafkaMock.AssertNotCalled(t, "Enqueue", mock.Anything)
}

func TestEmailNotifyHandler_StoreCheckError(t *testing.T) {
	authMock := new(servicemocks.AuthServiceMock)
	storeMock := new(storemocks.StoreMock)
	kafkaMock := new(kafkamocks.KafkaProducerMock)
	router, _ := setupTestRouter(authMock, storeMock, kafkaMock)

	apiKey := "valid-api-key"
	idempotencyKey := "550e8400-e29b-41d4-a716-446655440003" // Valid UUID format
	storeError := errors.New("database connection lost")
	authMock.On("ValidateAPIKey", apiKey).Return(nil)
	storeMock.On("Check", mock.Anything, idempotencyKey).Return(false, storeError)

	body := EmailNotificationRequest{IdempotencyKey: idempotencyKey, Recipient: "storeerr@example.com", Subject: "StoreErr", Body: "StoreErr", FromName: "Sender"}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/v1/email/notify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("X-Idempotency-Key", idempotencyKey)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Failed to check idempotency", response["error"])
	authMock.AssertExpectations(t)
	storeMock.AssertExpectations(t)
	kafkaMock.AssertNotCalled(t, "Enqueue", mock.Anything)
}

func TestEmailNotifyHandler_KafkaProduceError(t *testing.T) {
	authMock := new(servicemocks.AuthServiceMock)
	storeMock := new(storemocks.StoreMock)
	kafkaMock := new(kafkamocks.KafkaProducerMock)
	router, _ := setupTestRouter(authMock, storeMock, kafkaMock)

	apiKey := "valid-api-key"
	idempotencyKey := "550e8400-e29b-41d4-a716-446655440004" // Valid UUID format
	kafkaError := errors.New("kafka broker unavailable")
	authMock.On("ValidateAPIKey", apiKey).Return(nil)
	storeMock.On("Check", mock.Anything, idempotencyKey).Return(false, nil) // Not duplicate
	kafkaMock.On("Enqueue", mock.AnythingOfType("model.NotificationMessage")).Return(kafkaError)

	body := EmailNotificationRequest{IdempotencyKey: idempotencyKey, Recipient: "kafkaerr@example.com", Subject: "KafkaErr", Body: "KafkaErr", FromName: "Sender"}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/v1/email/notify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("X-Idempotency-Key", idempotencyKey)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Failed to queue notification", response["error"])
	authMock.AssertExpectations(t)
	storeMock.AssertExpectations(t)
	kafkaMock.AssertExpectations(t)
}

func TestEmailNotifyHandler_AuthServiceError(t *testing.T) {
	authMock := new(servicemocks.AuthServiceMock)
	storeMock := new(storemocks.StoreMock)
	kafkaMock := new(kafkamocks.KafkaProducerMock)
	router, _ := setupTestRouter(authMock, storeMock, kafkaMock)

	apiKey := "valid-api-key-auth-err"
	authError := errors.New("auth service timeout")
	authMock.On("ValidateAPIKey", apiKey).Return(authError)

	body := EmailNotificationRequest{IdempotencyKey: "550e8400-e29b-41d4-a716-446655440007", Recipient: "autherr@example.com", Subject: "AuthErr", Body: "AuthErr", FromName: "Sender"}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/v1/email/notify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("X-Idempotency-Key", "some-key-auth-err")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Failed to validate API key", response["error"])
	authMock.AssertExpectations(t)
	storeMock.AssertNotCalled(t, "Check", mock.Anything, mock.Anything)
	kafkaMock.AssertNotCalled(t, "Enqueue", mock.Anything)
}
