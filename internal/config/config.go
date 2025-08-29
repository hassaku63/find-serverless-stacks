package config

import "fmt"

// Config holds the application configuration
type Config struct {
	Profile      string
	Region       string
	OutputFormat string

	// AssumeRole configuration
	AssumeRole *AssumeRoleConfig
}

// AssumeRoleConfig holds AssumeRole-specific configuration
type AssumeRoleConfig struct {
	RoleARN     string `json:"roleArn"`
	SessionName string `json:"sessionName"`
	Duration    int32  `json:"duration"`

	ExternalID string `json:"externalId,omitempty"`
}

// ValidateOutputFormat checks if the output format is supported
func ValidateOutputFormat(format string) bool {
	switch format {
	case "json", "tsv":
		return true
	default:
		return false
	}
}

// Validate validates the AssumeRole configuration
func (arc *AssumeRoleConfig) Validate() error {
	if arc.RoleARN == "" {
		return fmt.Errorf("role ARN cannot be empty when using AssumeRole")
	}

	if arc.Duration < 900 || arc.Duration > 43200 {
		return fmt.Errorf("session duration must be between 900 and 43200 seconds, got %d", arc.Duration)
	}

	if arc.SessionName == "" {
		return fmt.Errorf("session name cannot be empty")
	}

	return nil
}
