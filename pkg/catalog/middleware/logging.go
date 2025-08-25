package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// RequestIDKey is the context key for request ID
type RequestIDKey struct{}

// LoggerKey is the context key for logger
type LoggerKey struct{}

// RequestLogger is a simple logger interface
type RequestLogger interface {
	Log(level, message string, fields map[string]interface{})
}

// ZeroLogger implements RequestLogger using zerolog
type ZeroLogger struct {
	requestID string
	logger    zerolog.Logger
}

func (zl *ZeroLogger) Log(level, message string, fields map[string]interface{}) {
	// Create event with request ID and timestamp
	event := zl.logger.With().
		Str("request_id", zl.requestID).
		Time("timestamp", time.Now().UTC()).
		Str("level", level)

	// Add custom fields
	for k, v := range fields {
		switch val := v.(type) {
		case string:
			event = event.Str(k, val)
		case int:
			event = event.Int(k, val)
		case int64:
			event = event.Int64(k, val)
		case float64:
			event = event.Float64(k, val)
		case bool:
			event = event.Bool(k, val)
		default:
			event = event.Interface(k, val)
		}
	}

	// Create logger from context and log based on level
	logger := event.Logger()
	switch level {
	case "INFO":
		logger.Info().Msg(message)
	case "ERROR":
		logger.Error().Msg(message)
	case "WARN":
		logger.Warn().Msg(message)
	case "DEBUG":
		logger.Debug().Msg(message)
	default:
		logger.Info().Msg(message)
	}
}

// LoggingMiddleware adds a logger to the request context
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Get request ID from context
		requestID := ""
		if ctx := r.Context(); ctx != nil {
			if id, ok := ctx.Value(RequestIDKey{}).(string); ok {
				requestID = id
			}
		}

		// Create logger for this request
		logger := log.With().Str("request_id", requestID).Logger()
		requestLogger := &ZeroLogger{requestID: requestID, logger: logger}

		// Add logger to context
		ctx := context.WithValue(r.Context(), LoggerKey{}, requestLogger)

		// Log request start
		requestLogger.Log("INFO", "Request started", map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"query":      r.URL.RawQuery,
			"user_agent": r.UserAgent(),
			"remote_ip":  r.RemoteAddr,
		})

		// Create response writer wrapper to capture status code
		wrappedWriter := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Call next handler
		next.ServeHTTP(wrappedWriter, r.WithContext(ctx))

		// Log request completion
		duration := time.Since(start)
		requestLogger.Log("INFO", "Request completed", map[string]interface{}{
			"method":      r.Method,
			"path":        r.URL.Path,
			"status":      wrappedWriter.statusCode,
			"duration":    duration.String(),
			"duration_ms": duration.Milliseconds(),
		})
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// GetLogger extracts logger from context
func GetLogger(ctx context.Context) RequestLogger {
	if logger, ok := ctx.Value(LoggerKey{}).(RequestLogger); ok {
		return logger
	}
	// Return a default logger if none is found
	defaultLogger := log.With().Str("request_id", "unknown").Logger()
	return &ZeroLogger{requestID: "unknown", logger: defaultLogger}
}
