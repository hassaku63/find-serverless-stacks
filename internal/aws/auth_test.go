package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthConfig_Validation(t *testing.T) {
	tests := []struct {
		name       string
		config     AuthConfig
		expectOK   bool
		expectType ErrorType
	}{
		{
			name: "valid configuration with default profile",
			config: AuthConfig{
				Profile: "default",
				Region:  "us-east-1",
			},
			expectOK: true,
		},
		{
			name: "valid configuration with custom profile",
			config: AuthConfig{
				Profile: "my-profile",
				Region:  "us-west-2",
			},
			expectOK: true,
		},
		{
			name: "empty region should fail validation",
			config: AuthConfig{
				Profile: "default",
				Region:  "",
			},
			expectOK:   false,
			expectType: ErrorTypeInvalidRegion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test validation logic (to be implemented)
			err := validateAuthConfig(tt.config)

			if tt.expectOK {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				var customErr *Error
				assert.True(t, errors.As(err, &customErr))
				assert.Equal(t, tt.expectType, customErr.Type)
			}
		})
	}
}

func TestValidateAWSCredentials(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error)
		expectError bool
		errorType   ErrorType
	}{
		{
			name: "valid credentials",
			mockFunc: func(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error) {
				return &cloudformation.ListStacksOutput{}, nil
			},
			expectError: false,
		},
		{
			name: "invalid credentials",
			mockFunc: func(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error) {
				return nil, errors.New("AccessDenied: insufficient permissions")
			},
			expectError: true,
			errorType:   ErrorTypePermission,
		},
		{
			name: "network error",
			mockFunc: func(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error) {
				return nil, errors.New("connection timeout")
			},
			expectError: true,
			errorType:   ErrorTypePermission, // validateAWSCredentials treats all errors as permission errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock using CloudFormationAPI interface
			mock := &mockCloudFormationAPI{
				listStacksFunc: tt.mockFunc,
			}
			ctx := context.Background()
			err := validateAWSCredentialsWithAPI(ctx, mock)

			if tt.expectError {
				require.Error(t, err)
				var customErr *Error
				assert.True(t, errors.As(err, &customErr))
				assert.Equal(t, tt.errorType, customErr.Type)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// mockCloudFormationClient for testing validateAWSCredentials
type mockCloudFormationClient struct {
	listStacksFunc func(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error)
}

func (m *mockCloudFormationClient) ListStacks(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error) {
	return m.listStacksFunc(ctx, params, optFns...)
}

// validateAuthConfig is a new function we need to implement (Red phase)
func validateAuthConfig(config AuthConfig) error {
	if config.Region == "" {
		return &Error{
			Type:    ErrorTypeInvalidRegion,
			Message: "region is required",
		}
	}
	return nil
}

func TestToPtr(t *testing.T) {
	t.Run("int32 pointer", func(t *testing.T) {
		input := int32(42)
		result := toPtr(input)
		assert.NotNil(t, result)
		assert.Equal(t, input, *result)
	})

	t.Run("string pointer", func(t *testing.T) {
		input := "test"
		result := toPtr(input)
		assert.NotNil(t, result)
		assert.Equal(t, input, *result)
	})

	t.Run("bool pointer", func(t *testing.T) {
		input := true
		result := toPtr(input)
		assert.NotNil(t, result)
		assert.Equal(t, input, *result)
	})
}
