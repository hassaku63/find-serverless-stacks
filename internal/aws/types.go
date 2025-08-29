package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// StackInfo represents comprehensive stack information
type StackInfo struct {
	Summary   types.StackSummary
	Details   *types.Stack
	Resources []types.StackResource
}

// Error types for better error handling
type Error struct {
	Type    ErrorType
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

type ErrorType string

const (
	ErrorTypePermission    ErrorType = "PERMISSION_DENIED"
	ErrorTypeInvalidRegion ErrorType = "INVALID_REGION"
	ErrorTypeRateLimit     ErrorType = "RATE_LIMIT"
	ErrorTypeNetwork       ErrorType = "NETWORK_ERROR"
	ErrorTypeUnknown       ErrorType = "UNKNOWN_ERROR"
)
