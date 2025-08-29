package detector

import "fmt"

// DetectionError represents errors that occur during stack detection
type DetectionError struct {
	StackName string
	Operation string
	Cause     error
}

func (e *DetectionError) Error() string {
	return fmt.Sprintf("detection error for stack %q during %s: %v", e.StackName, e.Operation, e.Cause)
}

func (e *DetectionError) Unwrap() error {
	return e.Cause
}

// NewDetectionError creates a new detection error
func NewDetectionError(stackName, operation string, cause error) *DetectionError {
	return &DetectionError{
		StackName: stackName,
		Operation: operation,
		Cause:     cause,
	}
}
