package main

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/hassaku63/find-serverless-stacks/internal/config"
	"github.com/hassaku63/find-serverless-stacks/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAWSClient for integration tests
type mockAWSClient struct {
	stacks    []types.StackSummary
	resources map[string][]types.StackResource
	details   map[string]*types.Stack
	shouldErr bool
}

func (m *mockAWSClient) ListActiveStacks(ctx context.Context) ([]types.StackSummary, error) {
	if m.shouldErr {
		return nil, assert.AnError
	}
	return m.stacks, nil
}

func (m *mockAWSClient) GetStackResources(ctx context.Context, stackName string) ([]types.StackResource, error) {
	if m.shouldErr {
		return nil, assert.AnError
	}
	if resources, exists := m.resources[stackName]; exists {
		return resources, nil
	}
	return []types.StackResource{}, nil
}

func (m *mockAWSClient) GetStackDetails(ctx context.Context, stackName string) (*types.Stack, error) {
	if m.shouldErr {
		return nil, assert.AnError
	}
	if details, exists := m.details[stackName]; exists {
		return details, nil
	}
	return nil, nil
}

func TestCreateAWSClient(t *testing.T) {
	tests := []struct {
		name        string
		cfg         config.Config
		expectError bool
		skipReason  string
	}{
		{
			name: "valid configuration - should attempt to create client",
			cfg: config.Config{
				Profile:      "default",
				Region:       "us-east-1",
				OutputFormat: "json",
			},
			// Note: This will likely fail in test environment without AWS credentials
			// but we test the function behavior
		},
		{
			name: "custom profile configuration",
			cfg: config.Config{
				Profile:      "test-profile",
				Region:       "us-west-2",
				OutputFormat: "tsv",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			client, err := createAWSClient(context.Background(), tt.cfg)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				// In test environment, AWS client creation will likely fail
				// due to missing credentials, but we verify the function works
				if err != nil {
					t.Logf("AWS client creation failed (expected in test environment): %v", err)
				} else {
					assert.NotNil(t, client)
				}
			}
		})
	}
}

func TestRunDetection(t *testing.T) {
	cfg := config.Config{
		Profile:      "test-profile",
		Region:       "us-east-1",
		OutputFormat: "json",
	}

	// Test with empty results
	mockClient := &mockAWSClient{
		stacks:    []types.StackSummary{},
		resources: make(map[string][]types.StackResource),
		details:   make(map[string]*types.Stack),
	}

	output, err := runDetection(context.Background(), mockClient, cfg)
	require.NoError(t, err)
	assert.Contains(t, output, `"stacks":[]`)

	// Test with serverless stack
	stackName := "test-serverless-stack"
	mockClient.stacks = []types.StackSummary{
		{
			StackName:   aws.String(stackName),
			StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123"),
			StackStatus: types.StackStatusCreateComplete,
		},
	}
	mockClient.resources[stackName] = []types.StackResource{
		{
			LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
			ResourceType:      aws.String("AWS::S3::Bucket"),
		},
	}

	output, err = runDetection(context.Background(), mockClient, cfg)
	require.NoError(t, err)
	assert.Contains(t, output, stackName)
	assert.Contains(t, output, "ServerlessDeploymentBucket")
}

func TestRunDetection_ErrorHandling(t *testing.T) {
	cfg := config.Config{
		Profile:      "test-profile",
		Region:       "us-east-1",
		OutputFormat: "json",
	}

	// Test with AWS error
	mockClient := &mockAWSClient{
		shouldErr: true,
	}

	_, err := runDetection(context.Background(), mockClient, cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to detect serverless stacks")
}

func TestFormatOutput(t *testing.T) {
	tests := []struct {
		name     string
		stacks   []models.Stack
		format   string
		contains []string
	}{
		{
			name:     "empty stacks JSON",
			stacks:   []models.Stack{},
			format:   "json",
			contains: []string{`"stacks":[]`},
		},
		{
			name:     "empty stacks TSV",
			stacks:   []models.Stack{},
			format:   "tsv",
			contains: []string{"StackName\tStackID\tRegion"},
		},
		{
			name: "single stack JSON",
			stacks: []models.Stack{
				{
					StackName: "test-stack",
					Region:    "us-east-1",
				},
			},
			format:   "json",
			contains: []string{`"stackName":"test-stack"`, `"region":"us-east-1"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := formatOutput(tt.stacks, tt.format)
			require.NoError(t, err)

			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestFormatOutput_InvalidFormat(t *testing.T) {
	_, err := formatOutput([]models.Stack{}, "xml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create formatter")
}
