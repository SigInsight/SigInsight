// Package litequery defines the storage-independent query model used by the
// lightweight telemetry query engine.
package litequery

import "fmt"

type ErrorCode string

const (
	ErrorInvalidRequest     ErrorCode = "invalid_request"
	ErrorUnsupported        ErrorCode = "unsupported"
	ErrorInvalidFilter      ErrorCode = "invalid_filter"
	ErrorInvalidAggregation ErrorCode = "invalid_aggregation"
	ErrorInvalidFormula     ErrorCode = "invalid_formula"
	ErrorBudgetExceeded     ErrorCode = "budget_exceeded"
)

// Error describes a query-domain failure. HTTP status codes are deliberately
// assigned outside this package so planning can be tested without a handler.
type Error struct {
	Code    ErrorCode
	Message string
	Field   string
}

func (e *Error) Error() string {
	if e.Field == "" {
		return string(e.Code) + ": " + e.Message
	}
	return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Field)
}

func newError(code ErrorCode, field, format string, args ...any) error {
	return &Error{Code: code, Field: field, Message: fmt.Sprintf(format, args...)}
}
