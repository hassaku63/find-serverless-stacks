package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
)

// CreateClient creates a real AWS CloudFormation client with authentication
func CreateClient(ctx context.Context, auth AuthConfig) (*Client, error) {
	// Load AWS configuration
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
		return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
	}
	
	// Create CloudFormation service client
	cfClient := cloudformation.NewFromConfig(cfg)
	
	// Create our client wrapper
	client := NewClient(cfClient, auth.Region)
	
	return client, nil
}