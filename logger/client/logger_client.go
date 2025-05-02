package loggerclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// LogClient provides functions to send logs to the central logger
type LogClient struct {
	serviceID    string
	logServerURL string
	httpClient   *http.Client
}

// LogLevel defines log severity levels
type LogLevel string

const (
	// Log levels
	DEBUG   LogLevel = "DEBUG"
	INFO    LogLevel = "INFO"
	WARNING LogLevel = "WARN"
	ERROR   LogLevel = "ERROR"
	FATAL   LogLevel = "FATAL"
)

// LogEntry represents a log entry to be sent to the server
type LogEntry struct {
	Service string   `json:"service"`
	Level   LogLevel `json:"level"`
	Message string   `json:"message"`
	TraceID string   `json:"trace_id,omitempty"`
}

// NewLogClient creates a new log client
func NewLogClient(serviceID, logServerURL string) *LogClient {
	return &LogClient{
		serviceID:    serviceID,
		logServerURL: logServerURL,
		httpClient: &http.Client{
			Timeout: 3 * time.Second, // Short timeout to avoid blocking application
		},
	}
}

// Log sends a log entry to the logging server
func (c *LogClient) Log(level LogLevel, message string, traceID string) error {
	entry := LogEntry{
		Service: c.serviceID,
		Level:   level,
		Message: message,
		TraceID: traceID,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("error marshaling log entry: %w", err)
	}

	// Send to log server
	resp, err := c.httpClient.Post(
		c.logServerURL+"/log",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		// If sending to log server fails, log locally to stderr so it's not lost
		fmt.Fprintf(os.Stderr, "[%s] [%s] [%s] %s (Trace: %s)\n",
			time.Now().Format("2006-01-02 15:04:05.000"), 
			c.serviceID, level, message, traceID)
		return fmt.Errorf("error sending log to server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("log server returned non-OK status: %d", resp.StatusCode)
	}

	return nil
}

// Debug logs a debug message
func (c *LogClient) Debug(message string, traceID string) {
	c.Log(DEBUG, message, traceID)
}

// Info logs an info message
func (c *LogClient) Info(message string, traceID string) {
	c.Log(INFO, message, traceID)
}

// Warning logs a warning message
func (c *LogClient) Warning(message string, traceID string) {
	c.Log(WARNING, message, traceID)
}

// Error logs an error message
func (c *LogClient) Error(message string, traceID string) {
	c.Log(ERROR, message, traceID)
}

// Fatal logs a fatal message
func (c *LogClient) Fatal(message string, traceID string) {
	c.Log(FATAL, message, traceID)
}
