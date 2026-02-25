package errors

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"

	"spark/pkg/clog"
	"spark/pkg/trace"

	log "spark/pkg/clog"
)

// ErrorCode represents predefined error codes
type ErrorCode string

const (
	// Common error codes
	ErrorCodeUnknown            ErrorCode = "UNKNOWN_ERROR"
	ErrorCodeInvalidRequest     ErrorCode = "INVALID_REQUEST"
	ErrorCodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrorCodeForbidden          ErrorCode = "FORBIDDEN"
	ErrorCodeNotFound           ErrorCode = "NOT_FOUND"
	ErrorCodeConflict           ErrorCode = "CONFLICT"
	ErrorCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrorCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrorCodeTimeout            ErrorCode = "TIMEOUT"

	// Business error codes
	ErrorCodeInvalidCredentials ErrorCode = "INVALID_CREDENTIALS"
	ErrorCodeResourceNotFound   ErrorCode = "RESOURCE_NOT_FOUND"
	ErrorCodeResourceConflict   ErrorCode = "RESOURCE_CONFLICT"
	ErrorCodeValidationFailed   ErrorCode = "VALIDATION_FAILED"
)

// ErrorResponse represents the standard error response format
type ErrorResponse struct {
	Success   bool   `json:"success"`
	Error     *Error `json:"error"`
	RequestID string `json:"request_id"`
	Timestamp string `json:"timestamp"`
}

// Error represents the error details
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// AppError represents an application error with context
type AppError struct {
	Code       ErrorCode
	Message    string
	Details    string
	HTTPStatus int
	Cause      error
	TraceID    string
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// WithTraceID adds trace ID to the error
func (e *AppError) WithTraceID(traceID string) *AppError {
	e.TraceID = traceID
	return e
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// WithCause adds the underlying cause to the error
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// NewError creates a new application error
func NewError(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: getHTTPStatusForCode(code),
	}
}

// NewErrorWithCause creates a new application error with cause
func NewErrorWithCause(code ErrorCode, message string, cause error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: getHTTPStatusForCode(code),
		Cause:      cause,
	}
}

// NewErrorFromContext creates a new application error with trace ID from context
func NewErrorFromContext(ctx context.Context, code ErrorCode, message string) *AppError {
	traceID := trace.TraceIDFromContext(ctx)
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: getHTTPStatusForCode(code),
		TraceID:    traceID,
	}
}

// getHTTPStatusForCode maps error codes to HTTP status codes
func getHTTPStatusForCode(code ErrorCode) int {
	switch code {
	case ErrorCodeInvalidRequest, ErrorCodeValidationFailed:
		return http.StatusBadRequest
	case ErrorCodeUnauthorized, ErrorCodeInvalidCredentials:
		return http.StatusUnauthorized
	case ErrorCodeForbidden:
		return http.StatusForbidden
	case ErrorCodeNotFound, ErrorCodeResourceNotFound:
		return http.StatusNotFound
	case ErrorCodeConflict, ErrorCodeResourceConflict:
		return http.StatusConflict
	case ErrorCodeTimeout:
		return http.StatusRequestTimeout
	case ErrorCodeServiceUnavailable:
		return http.StatusServiceUnavailable
	case ErrorCodeInternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// toKratosError converts AppError to Kratos error
func (e *AppError) toKratosError() *errors.Error {
	return errors.New(e.HTTPStatus, string(e.Code), e.Message)
}

// ErrorHandler creates a Kratos HTTP error encoder that logs errors and returns standard format
func ErrorHandler(logger clog.Logger) kratosHttp.EncodeErrorFunc {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		ctx := r.Context()
		traceID := trace.TraceIDFromContext(ctx)

		// Start a new span for error handler while preserving existing trace context
		ctx, span := trace.StartSpan(ctx, "error_handler")
		defer span.End()

		// Get trace ID from the current context (should be consistent)
		currentTraceID := trace.TraceIDFromContext(ctx)
		if currentTraceID != "" {
			traceID = currentTraceID
		}

		// Convert error to our standard format
		appErr := convertToAppError(err, traceID)

		// Extract request ID from response headers
		requestID := w.Header().Get("X-Request-ID")

		// Log the error with context
		fields := []log.Field{
			log.String("request_id", requestID),
			log.String("error_code", string(appErr.Code)),
			log.String("method", r.Method),
			log.String("path", r.URL.Path),
			log.String("user_agent", r.UserAgent()),
			log.String("remote_addr", r.RemoteAddr),
		}

		if appErr.Cause != nil {
			fields = append(fields, log.Err(appErr.Cause))
		}

		if appErr.HTTPStatus >= 500 {
			logger.Error(ctx, "Server error occurred", fields...)
		} else if appErr.HTTPStatus >= 400 {
			logger.Warn(ctx, "Client error occurred", fields...)
		} else {
			logger.Info(ctx, "Request completed with error", fields...)
		}

		// Prepare response
		response := &ErrorResponse{
			Success:   false,
			RequestID: requestID,
			Timestamp: getCurrentTimestamp(),
			Error: &Error{
				Code:    string(appErr.Code),
				Message: appErr.Message,
				Details: appErr.Details,
			},
		}

		// Set response headers
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Trace-ID", traceID)
		w.WriteHeader(appErr.HTTPStatus)

		// Encode and write response
		kratosHttp.DefaultResponseEncoder(w, r, response)
	}
}

// RecoveryHandler creates a panic recovery middleware with error logging
func RecoveryHandler(logger clog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					ctx := r.Context()
					traceID := trace.TraceIDFromContext(ctx)
					requestID := w.Header().Get("X-Request-ID")

					// Log panic with stack trace
					logger.Error(ctx, "Panic recovered",
						log.String("request_id", requestID),
						log.Any("panic", err),
						log.String("method", r.Method),
						log.String("path", r.URL.Path),
						log.Stack("stack"),
					)

					// Create error response
					appErr := NewError(ErrorCodeInternalError, "Internal server error").WithTraceID(traceID)

					response := &ErrorResponse{
						Success:   false,
						RequestID: requestID,
						Timestamp: getCurrentTimestamp(),
						Error: &Error{
							Code:    string(appErr.Code),
							Message: appErr.Message,
						},
					}

					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Trace-ID", traceID)
					w.WriteHeader(http.StatusInternalServerError)

					kratosHttp.DefaultResponseEncoder(w, r, response)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// convertToAppError converts various error types to AppError
func convertToAppError(err error, traceID string) *AppError {
	if err == nil {
		return NewError(ErrorCodeUnknown, "Unknown error").WithTraceID(traceID)
	}

	// Check if it's already an AppError
	if appErr, ok := err.(*AppError); ok {
		if appErr.TraceID == "" {
			appErr.TraceID = traceID
		}
		return appErr
	}

	// Check if it's a Kratos error
	if kratosErr := errors.FromError(err); kratosErr != nil {
		code := ErrorCode(kratosErr.Reason)
		if code == "" {
			code = getErrorCodeForHTTPStatus(int(kratosErr.Code))
		}

		return &AppError{
			Code:       code,
			Message:    kratosErr.Message,
			HTTPStatus: int(kratosErr.Code),
			TraceID:    traceID,
			Cause:      err,
		}
	}

	// Default error handling
	return NewError(ErrorCodeInternalError, "Internal server error").
		WithTraceID(traceID).
		WithCause(err)
}

// getErrorCodeForHTTPStatus maps HTTP status codes to error codes
func getErrorCodeForHTTPStatus(status int) ErrorCode {
	switch status {
	case http.StatusBadRequest:
		return ErrorCodeInvalidRequest
	case http.StatusUnauthorized:
		return ErrorCodeUnauthorized
	case http.StatusForbidden:
		return ErrorCodeForbidden
	case http.StatusNotFound:
		return ErrorCodeNotFound
	case http.StatusConflict:
		return ErrorCodeConflict
	case http.StatusRequestTimeout:
		return ErrorCodeTimeout
	case http.StatusServiceUnavailable:
		return ErrorCodeServiceUnavailable
	default:
		return ErrorCodeInternalError
	}
}

// getCurrentTimestamp returns current timestamp in ISO8601 format
func getCurrentTimestamp() string {
	return time.Now().Format(time.RFC3339)
}
