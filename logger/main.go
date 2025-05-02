package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	logFile    *os.File
	logWriter  *bufio.Writer
	logMutex   sync.Mutex
	serviceMap = map[string]string{
		"ms1": "Authentication Service",
		"ms2": "Message Queue Service",
		"ms3": "Prioritization Service",
		"ms4": "Delivery Service",
	}
)

func main() {
	fmt.Println("Starting NOTLI Unified Logger Service")

	// Set up log file
	var err error
	logFile, err = os.OpenFile("notli_unified.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	logWriter = bufio.NewWriter(logFile)
	defer logWriter.Flush()

	// Log startup
	writeLog("logger", "Unified Logger Service started")

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		writeLog("logger", "Unified Logger Service shutting down")
		logWriter.Flush()
		logFile.Close()
		os.Exit(0)
	}()

	// Set up HTTP server with Gin
	router := gin.Default()

	// Log endpoint
	router.POST("/log", func(c *gin.Context) {
		var logEntry struct {
			Service string `json:"service" binding:"required"`
			Level   string `json:"level" binding:"required"`
			Message string `json:"message" binding:"required"`
			TraceID string `json:"trace_id"`
		}

		if err := c.ShouldBindJSON(&logEntry); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Normalize service ID
		serviceID := strings.ToLower(logEntry.Service)
		serviceName := serviceMap[serviceID]
		if serviceName == "" {
			serviceName = logEntry.Service
		}

		// Format and write log
		message := fmt.Sprintf("[%s] %s", logEntry.Level, logEntry.Message)
		if logEntry.TraceID != "" {
			message = fmt.Sprintf("%s (Trace: %s)", message, logEntry.TraceID)
		}
		writeLog(serviceName, message)

		c.JSON(http.StatusOK, gin.H{"status": "logged"})
	})

	// Process monitoring endpoint for dashboard to get recent logs
	router.GET("/logs", func(c *gin.Context) {
		lines, err := readLastLines("notli_unified.log", 50) // Read last 50 lines
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"logs": lines})
	})

	// Start the server
	if err := router.Run(":3005"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// writeLog writes a formatted log entry to the log file
func writeLog(service, message string) {
	logMutex.Lock()
	defer logMutex.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	logLine := fmt.Sprintf("[%s] [%s] %s\n", timestamp, service, message)

	_, err := logWriter.WriteString(logLine)
	if err != nil {
		log.Printf("Error writing to log file: %v", err)
	}

	// Flush on every write to ensure logs are written even if program crashes
	logWriter.Flush()

	// Also print to console for monitoring
	fmt.Print(logLine)
}

// readLastLines reads the last n lines from a file
func readLastLines(filePath string, n int) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string

	// Read all lines (inefficient for very large files, but simpler)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Return last n lines or all if fewer
	if len(lines) <= n {
		return lines, nil
	}
	return lines[len(lines)-n:], nil
}
