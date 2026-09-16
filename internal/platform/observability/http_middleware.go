package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const RequestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func HTTPRequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			startedAt := time.Now()

			requestID := strings.TrimSpace(
				r.Header.Get(RequestIDHeader),
			)

			if requestID == "" || len(requestID) > 128 {
				requestID = generateRequestID()
			}

			ctx := context.WithValue(
				r.Context(),
				requestIDContextKey{},
				requestID,
			)

			r = r.WithContext(ctx)
			w.Header().Set(RequestIDHeader, requestID)

			recorder := &responseRecorder{
				ResponseWriter: w,
			}

			next.ServeHTTP(recorder, r)

			statusCode := recorder.statusCode
			if statusCode == 0 {
				statusCode = http.StatusOK
			}

			level := slog.LevelInfo

			switch {
			case statusCode >= http.StatusInternalServerError:
				level = slog.LevelError
			case statusCode >= http.StatusBadRequest:
				level = slog.LevelWarn
			}

			route := r.Pattern
			if route == "" {
				route = r.URL.Path
			}

			logger.LogAttrs(
				ctx,
				level,
				"http_request_completed",
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("route", route),
				slog.Int("status_code", statusCode),
				slog.Int("response_bytes", recorder.bytesWritten),
				slog.Int64(
					"duration_ms",
					time.Since(startedAt).Milliseconds(),
				),
			)
		})
	}
}

func generateRequestID() string {
	buffer := make([]byte, 16)

	if _, err := rand.Read(buffer); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}

	return hex.EncodeToString(buffer)
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	if r.statusCode != 0 {
		return
	}

	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.statusCode == 0 {
		r.WriteHeader(http.StatusOK)
	}

	bytesWritten, err := r.ResponseWriter.Write(data)
	r.bytesWritten += bytesWritten

	return bytesWritten, err
}
