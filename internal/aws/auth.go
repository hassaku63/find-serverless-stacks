package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Profile string
	Region  string
}

// NewCloudFormationClient creates a new CloudFormation client with the specified configuration
func NewCloudFormationClient(ctx context.Context, authConfig AuthConfig) (*Client, error) {
	// Load AWS configuration
	var cfg config.LoadOptionsFunc

	// Configure profile if specified
	if authConfig.Profile != "" && authConfig.Profile != "default" {
		cfg = config.WithSharedConfigProfile(authConfig.Profile)
	}

	awsConfig, err := config.LoadDefaultConfig(ctx, cfg, config.WithRegion(authConfig.Region))
	if err != nil {
		return nil, &Error{
			Type:    ErrorTypePermission,
			Message: fmt.Sprintf("failed to load AWS config for profile '%s' in region '%s'", authConfig.Profile, authConfig.Region),
			Cause:   err,
		}
	}

	// Create CloudFormation client
	cfClient := cloudformation.NewFromConfig(awsConfig)

	// Validate that we can make API calls
	if err := validateAWSCredentials(ctx, cfClient); err != nil {
		return nil, err
	}

	return NewClient(cfClient, authConfig.Region), nil
}

// validateAWSCredentials performs a minimal API call to validate credentials
func validateAWSCredentials(ctx context.Context, client *cloudformation.Client) error {
	return validateAWSCredentialsWithAPI(ctx, client)
}

// validateAWSCredentialsWithAPI validates credentials using the CloudFormationAPI interface
func validateAWSCredentialsWithAPI(ctx context.Context, client CloudFormationAPI) error {
	// Try to list stacks with a minimal request to validate credentials
	_, err := client.ListStacks(ctx, &cloudformation.ListStacksInput{})

	if err != nil {
		return &Error{
			Type:    ErrorTypePermission,
			Message: "failed to validate AWS credentials",
			Cause:   err,
		}
	}

	return nil
}

// Helper function to convert value to pointer
func toPtr[T any](v T) *T {
	return &v
}
