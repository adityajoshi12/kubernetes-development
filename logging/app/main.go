package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	logger := setupZapLogger()
	defer logger.Sync()

	logger.Info("Starting HTTP application",
		zap.String("version", "1.0.0"),
		zap.String("environment", getEnv("ENVIRONMENT", "production")),
		zap.Int("port", 8080),
	)

	// Create HTTP server with routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler(logger))
	mux.HandleFunc("/health", healthHandler(logger))
	mux.HandleFunc("/api/users", usersHandler(logger))
	mux.HandleFunc("/api/process", processHandler(logger))
	mux.HandleFunc("/api/error", errorHandler(logger))

	// Wrap with logging middleware
	handler := loggingMiddleware(logger, mux)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		logger.Info("HTTP server listening",
			zap.String("address", server.Addr),
			zap.String("protocol", "HTTP/1.1"),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}

	logger.Info("Server stopped gracefully")
}

func setupZapLogger() *zap.Logger {
	config := zap.NewProductionConfig()
	
	// Configure encoder for OpenObserve compatibility
	config.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	
	config.Encoding = "json"
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	
	logger, err := config.Build(zap.AddCaller())
	if err != nil {
		panic(err)
	}
	
	return logger
}

// loggingMiddleware logs all HTTP requests
func loggingMiddleware(logger *zap.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := generateRequestID()
		
		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		// Log incoming request
		logger.Info("incoming_request",
			zap.String("request_id", requestID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
		)
		
		// Serve the request
		next.ServeHTTP(wrapped, r)
		
		// Log response
		duration := time.Since(start)
		logger.Info("request_completed",
			zap.String("request_id", requestID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status_code", wrapped.statusCode),
			zap.Duration("duration", duration),
			zap.Int64("duration_ms", duration.Milliseconds()),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Handlers
func homeHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("home_endpoint_accessed")
		
		response := map[string]interface{}{
			"message": "Welcome to Logging API",
			"version": "1.0.0",
			"endpoints": []string{
				"/health",
				"/api/users",
				"/api/process",
				"/api/error",
			},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func healthHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("health_check_requested")
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

func usersHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := generateRequestID()
		reqLogger := logger.With(
			zap.String("request_id", requestID),
			zap.String("handler", "users"),
		)
		
		reqLogger.Info("fetching_users",
			zap.String("method", r.Method),
		)
		
		// Simulate database query
		time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
		
		users := []map[string]interface{}{
			{"id": 1, "name": "Alice", "email": "alice@example.com"},
			{"id": 2, "name": "Bob", "email": "bob@example.com"},
			{"id": 3, "name": "Charlie", "email": "charlie@example.com"},
		}
		
		reqLogger.Info("users_fetched_successfully",
			zap.Int("count", len(users)),
		)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"users": users,
			"count": len(users),
		})
	}
}

func processHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := generateRequestID()
		reqLogger := logger.With(
			zap.String("request_id", requestID),
			zap.String("handler", "process"),
		)
		
		reqLogger.Info("starting_data_processing")
		
		// Simulate processing
		processingTime := time.Duration(rand.Intn(200)+100) * time.Millisecond
		itemsProcessed := rand.Intn(100) + 50
		
		time.Sleep(processingTime)
		
		reqLogger.Info("processing_completed",
			zap.Duration("processing_time", processingTime),
			zap.Int("items_processed", itemsProcessed),
			zap.Float64("items_per_second", float64(itemsProcessed)/processingTime.Seconds()),
		)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"request_id": requestID,
			"items_processed": itemsProcessed,
			"duration_ms": processingTime.Milliseconds(),
		})
	}
}

func errorHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := generateRequestID()
		reqLogger := logger.With(
			zap.String("request_id", requestID),
			zap.String("handler", "error"),
		)
		
		reqLogger.Warn("error_endpoint_called")
		
		// Simulate different types of errors
		errorTypes := []string{"database_error", "timeout_error", "validation_error", "network_error"}
		errorType := errorTypes[rand.Intn(len(errorTypes))]
		
		reqLogger.Error("request_failed",
			zap.String("error_type", errorType),
			zap.String("error_message", fmt.Sprintf("Simulated %s occurred", errorType)),
			zap.Int("retry_count", 3),
			zap.Bool("recoverable", errorType != "database_error"),
		)
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"request_id": requestID,
			"error": errorType,
			"message": "Internal server error",
		})
	}
}

func generateRequestID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randString(8))
}

func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
	


