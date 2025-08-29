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

// CreateClient creates a real AWS CloudFormation client with authentication
func CreateClient(ctx context.Context, auth AuthConfig) (*Client, error) {
	// Load base AWS configuration
	cfg, err := loadBaseAWSConfig(ctx, auth)
	if err != nil {
		return nil, err
	}
	
	// Apply AssumeRole if specified
	if auth.AssumeRole != nil {
		cfg, err = applyAssumeRoleToConfig(ctx, cfg, auth.AssumeRole)
		if err != nil {
			return nil, err
		}
	}
	
	// Create CloudFormation service client
	cfClient := cloudformation.NewFromConfig(cfg)
	
	// Create our client wrapper
	client := NewClient(cfClient, auth.Region)
	
	return client, nil
}

// loadBaseAWSConfig loads the base AWS configuration without AssumeRole
func loadBaseAWSConfig(ctx context.Context, auth AuthConfig) (aws.Config, error) {
	var opts []func(*config.LoadOptions) error
	
	// Set profile if specified
	if auth.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(auth.Profile))
	}
	
	// Set region
	if auth.Region != "" {
		opts = append(opts, config.WithRegion(auth.Region))
	}
	
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS configuration: %w", err)
	}
	
	return cfg, nil
}

// applyAssumeRoleToConfig applies AssumeRole configuration to the AWS config
func applyAssumeRoleToConfig(ctx context.Context, cfg aws.Config, roleConfig *AssumeRoleCredentials) (aws.Config, error) {
	// Create STS client for AssumeRole
	stsClient := sts.NewFromConfig(cfg)

	// Create AssumeRole provider
	provider := stscreds.NewAssumeRoleProvider(stsClient, roleConfig.RoleARN, func(o *stscreds.AssumeRoleOptions) {
		o.RoleSessionName = roleConfig.SessionName
		o.Duration = time.Duration(roleConfig.Duration) * time.Second
	})

	// Create new config with AssumeRole credentials
	assumedConfig := cfg.Copy()
	assumedConfig.Credentials = aws.NewCredentialsCache(provider)

	return assumedConfig, nil
}