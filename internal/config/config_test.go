package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateOutputFormat(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		expected bool
	}{
		{
			name:     "json format is valid",
			format:   "json",
			expected: true,
		},
		{
			name:     "tsv format is valid",
			format:   "tsv",
			expected: true,
		},
		{
			name:     "xml format is invalid",
			format:   "xml",
			expected: false,
		},
		{
			name:     "yaml format is invalid",
			format:   "yaml",
			expected: false,
		},
		{
			name:     "empty string is invalid",
			format:   "",
			expected: false,
		},
		{
			name:     "uppercase JSON is invalid",
			format:   "JSON",
			expected: false,
		},
		{
			name:     "mixed case is invalid",
			format:   "Json",
			expected: false,
		},
		{
			name:     "csv format is invalid",
			format:   "csv",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateOutputFormat(tt.format)
			assert.Equal(t, tt.expected, result, "ValidateOutputFormat(%q) should return %v", tt.format, tt.expected)
		})
	}
}

func TestConfig_Struct(t *testing.T) {
	t.Run("config struct initialization", func(t *testing.T) {
		config := Config{
			Profile:      "test-profile",
			Region:       "us-west-2",
			OutputFormat: "json",
		}

		assert.Equal(t, "test-profile", config.Profile)
		assert.Equal(t, "us-west-2", config.Region)
		assert.Equal(t, "json", config.OutputFormat)
	})

	t.Run("empty config struct", func(t *testing.T) {
		config := Config{}

		assert.Empty(t, config.Profile)
		assert.Empty(t, config.Region)
		assert.Empty(t, config.OutputFormat)
	})
}

// Test that demonstrates the expected usage pattern
func TestConfig_UsagePattern(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		config := Config{
			Profile:      "my-aws-profile",
			Region:       "us-east-1",
			OutputFormat: "json",
		}

		// Validate the output format
		isValidFormat := ValidateOutputFormat(config.OutputFormat)
		assert.True(t, isValidFormat, "Output format should be valid")

		// Ensure required fields are present
		assert.NotEmpty(t, config.Region, "Region should not be empty")
	})

	t.Run("invalid output format", func(t *testing.T) {
		config := Config{
			Profile:      "my-aws-profile",
			Region:       "us-east-1",
			OutputFormat: "xml", // Invalid format
		}

		isValidFormat := ValidateOutputFormat(config.OutputFormat)
		assert.False(t, isValidFormat, "Invalid output format should be rejected")
	})
}

func TestAssumeRoleConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      AssumeRoleConfig
		expectError bool
		errorText   string
	}{
		{
			name: "valid configuration",
			config: AssumeRoleConfig{
				RoleARN:     "arn:aws:iam::123456789012:role/TestRole",
				SessionName: "test-session",
				Duration:    3600,
			},
			expectError: false,
		},
		{
			name: "empty role ARN",
			config: AssumeRoleConfig{
				RoleARN:     "",
				SessionName: "test-session",
				Duration:    3600,
			},
			expectError: true,
			errorText:   "role ARN cannot be empty",
		},
		{
			name: "empty session name",
			config: AssumeRoleConfig{
				RoleARN:     "arn:aws:iam::123456789012:role/TestRole",
				SessionName: "",
				Duration:    3600,
			},
			expectError: true,
			errorText:   "session name cannot be empty",
		},
		{
			name: "duration too short",
			config: AssumeRoleConfig{
				RoleARN:     "arn:aws:iam::123456789012:role/TestRole",
				SessionName: "test-session",
				Duration:    800, // Less than 900
			},
			expectError: true,
			errorText:   "session duration must be between 900 and 43200",
		},
		{
			name: "duration too long",
			config: AssumeRoleConfig{
				RoleARN:     "arn:aws:iam::123456789012:role/TestRole",
				SessionName: "test-session",
				Duration:    50000, // More than 43200
			},
			expectError: true,
			errorText:   "session duration must be between 900 and 43200",
		},
		{
			name: "minimum valid duration",
			config: AssumeRoleConfig{
				RoleARN:     "arn:aws:iam::123456789012:role/TestRole",
				SessionName: "test-session",
				Duration:    900,
			},
			expectError: false,
		},
		{
			name: "maximum valid duration",
			config: AssumeRoleConfig{
				RoleARN:     "arn:aws:iam::123456789012:role/TestRole",
				SessionName: "test-session",
				Duration:    43200,
			},
			expectError: false,
		},
		{
			name: "valid configuration with External ID",
			config: AssumeRoleConfig{
				RoleARN:     "arn:aws:iam::123456789012:role/TestRole",
				SessionName: "test-session",
				Duration:    3600,
				ExternalID:  "unique-external-id",
			},
			expectError: false,
		},
		{
			name: "valid configuration with empty External ID",
			config: AssumeRoleConfig{
				RoleARN:     "arn:aws:iam::123456789012:role/TestRole",
				SessionName: "test-session",
				Duration:    3600,
				ExternalID:  "",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorText)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_WithAssumeRole(t *testing.T) {
	t.Run("config with valid AssumeRole", func(t *testing.T) {
		config := Config{
			Profile:      "test-profile",
			Region:       "us-east-1",
			OutputFormat: "json",
			AssumeRole: &AssumeRoleConfig{
				RoleARN:     "arn:aws:iam::123456789012:role/TestRole",
				SessionName: "test-session",
				Duration:    3600,
			},
		}

		// Validate base config
		assert.True(t, ValidateOutputFormat(config.OutputFormat))
		assert.NotEmpty(t, config.Region)

		// Validate AssumeRole config
		assert.NotNil(t, config.AssumeRole)
		assert.NoError(t, config.AssumeRole.Validate())
	})

	t.Run("config with AssumeRole and External ID", func(t *testing.T) {
		config := Config{
			Profile:      "test-profile",
			Region:       "us-east-1",
			OutputFormat: "json",
			AssumeRole: &AssumeRoleConfig{
				RoleARN:     "arn:aws:iam::123456789012:role/ThirdPartyRole",
				SessionName: "test-session",
				Duration:    3600,
				ExternalID:  "company-unique-id-2023",
			},
		}

		// Validate base config
		assert.True(t, ValidateOutputFormat(config.OutputFormat))
		assert.NotEmpty(t, config.Region)

		// Validate AssumeRole config
		assert.NotNil(t, config.AssumeRole)
		assert.NoError(t, config.AssumeRole.Validate())
		assert.Equal(t, "company-unique-id-2023", config.AssumeRole.ExternalID)
	})

	t.Run("config without AssumeRole", func(t *testing.T) {
		config := Config{
			Profile:      "test-profile",
			Region:       "us-east-1",
			OutputFormat: "json",
			AssumeRole:   nil,
		}

		assert.Nil(t, config.AssumeRole)
		assert.True(t, ValidateOutputFormat(config.OutputFormat))
	})
}
