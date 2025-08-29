package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Profile string
	Region  string
	
	// AssumeRole configuration (Phase 1)
	AssumeRole *AssumeRoleCredentials
}

// AssumeRoleCredentials holds AssumeRole-specific configuration
type AssumeRoleCredentials struct {
	RoleARN     string
	SessionName string
	Duration    int32
}

// NewCloudFormationClient creates a new CloudFormation client with the specified configuration
func NewCloudFormationClient(ctx context.Context, authConfig AuthConfig) (*Client, error) {
	// Load base AWS configuration
	awsConfig, err := loadBaseConfig(ctx, authConfig)
	if err != nil {
		return nil, err
	}

	// Apply AssumeRole if specified
	if authConfig.AssumeRole != nil {
		awsConfig, err = applyAssumeRole(ctx, awsConfig, authConfig.AssumeRole)
		if err != nil {
			return nil, err
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

// loadBaseConfig loads the base AWS configuration without AssumeRole
func loadBaseConfig(ctx context.Context, authConfig AuthConfig) (aws.Config, error) {
	var cfg config.LoadOptionsFunc

	// Configure profile if specified
	if authConfig.Profile != "" && authConfig.Profile != "default" {
		cfg = config.WithSharedConfigProfile(authConfig.Profile)
	}

	awsConfig, err := config.LoadDefaultConfig(ctx, cfg, config.WithRegion(authConfig.Region))
	if err != nil {
		return aws.Config{}, &Error{
			Type:    ErrorTypePermission,
			Message: fmt.Sprintf("failed to load AWS config for profile '%s' in region '%s'", authConfig.Profile, authConfig.Region),
			Cause:   err,
		}
	}

	return awsConfig, nil
}

// applyAssumeRole applies AssumeRole configuration to the AWS config
func applyAssumeRole(ctx context.Context, awsConfig aws.Config, roleConfig *AssumeRoleCredentials) (aws.Config, error) {
	// Create STS client for AssumeRole
	stsClient := sts.NewFromConfig(awsConfig)

	// Create AssumeRole provider
	provider := stscreds.NewAssumeRoleProvider(stsClient, roleConfig.RoleARN, func(o *stscreds.AssumeRoleOptions) {
		o.RoleSessionName = roleConfig.SessionName
		o.Duration = time.Duration(roleConfig.Duration) * time.Second
	})

	// Create new config with AssumeRole credentials
	assumedConfig := awsConfig.Copy()
	assumedConfig.Credentials = aws.NewCredentialsCache(provider)

	return assumedConfig, nil
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
