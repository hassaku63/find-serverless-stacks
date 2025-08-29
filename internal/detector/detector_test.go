package detector

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock AWS client for testing
type mockAWSClient struct {
	stacks    []types.StackSummary
	resources map[string][]types.StackResource
	details   map[string]*types.Stack
}

func (m *mockAWSClient) ListActiveStacks(ctx context.Context) ([]types.StackSummary, error) {
	return m.stacks, nil
}

func (m *mockAWSClient) GetStackResources(ctx context.Context, stackName string) ([]types.StackResource, error) {
	if resources, exists := m.resources[stackName]; exists {
		return resources, nil
	}
	return []types.StackResource{}, nil
}

func (m *mockAWSClient) GetStackDetails(ctx context.Context, stackName string) (*types.Stack, error) {
	if details, exists := m.details[stackName]; exists {
		return details, nil
	}
	return nil, nil
}

func TestDetector_DetectServerlessStacks_WithServerlessDeploymentBucket(t *testing.T) {
	mockClient := &mockAWSClient{
		stacks: []types.StackSummary{
			{
				StackName:   aws.String("my-api-dev"),
				StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/my-api-dev/abc123"),
				StackStatus: types.StackStatusCreateComplete,
			},
		},
		resources: map[string][]types.StackResource{
			"my-api-dev": {
				{
					LogicalResourceId:  aws.String("ServerlessDeploymentBucket"),
					PhysicalResourceId: aws.String("my-api-dev-serverlessdeploymentbucket-abc123"),
					ResourceType:       aws.String("AWS::S3::Bucket"),
					ResourceStatus:     types.ResourceStatusCreateComplete,
				},
				{
					LogicalResourceId:  aws.String("MyFunction"),
					PhysicalResourceId: aws.String("my-api-dev-MyFunction-def456"),
					ResourceType:       aws.String("AWS::Lambda::Function"),
					ResourceStatus:     types.ResourceStatusCreateComplete,
				},
			},
		},
		details: map[string]*types.Stack{
			"my-api-dev": {
				StackName:       aws.String("my-api-dev"),
				StackId:         aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/my-api-dev/abc123"),
				StackStatus:     types.StackStatusCreateComplete,
				Description:     aws.String("My Serverless Framework v3 stack"),
				CreationTime:    aws.Time(mustParseTime("2023-10-01T12:34:56Z")),
				LastUpdatedTime: aws.Time(mustParseTime("2023-10-02T12:34:56Z")),
				Tags: []types.Tag{
					{Key: aws.String("Owner"), Value: aws.String("team-a")},
					{Key: aws.String("Environment"), Value: aws.String("development")},
				},
			},
		},
	}

	detector := NewDetector(mockClient, "us-east-1")
	ctx := context.Background()

	stacks, err := detector.DetectServerlessStacks(ctx)

	require.NoError(t, err)
	assert.Len(t, stacks, 1)

	stack := stacks[0]
	assert.Equal(t, "my-api-dev", stack.StackName)
	assert.Equal(t, "arn:aws:cloudformation:us-east-1:123456789012:stack/my-api-dev/abc123", stack.StackID)
	assert.Equal(t, "us-east-1", stack.Region)
	assert.Contains(t, stack.Reasons, "Contains resource with logical ID 'ServerlessDeploymentBucket'")
	assert.Equal(t, "My Serverless Framework v3 stack", stack.Description)
	assert.Contains(t, stack.StackTags, "Owner")
	assert.Equal(t, "team-a", stack.StackTags["Owner"])
}

func TestDetector_DetectServerlessStacks_WithoutServerlessDeploymentBucket(t *testing.T) {
	mockClient := &mockAWSClient{
		stacks: []types.StackSummary{
			{
				StackName:   aws.String("regular-stack"),
				StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/regular-stack/xyz789"),
				StackStatus: types.StackStatusCreateComplete,
			},
		},
		resources: map[string][]types.StackResource{
			"regular-stack": {
				{
					LogicalResourceId:  aws.String("MyBucket"),
					PhysicalResourceId: aws.String("regular-stack-mybucket-xyz789"),
					ResourceType:       aws.String("AWS::S3::Bucket"),
					ResourceStatus:     types.ResourceStatusCreateComplete,
				},
			},
		},
		details: map[string]*types.Stack{
			"regular-stack": {
				StackName:   aws.String("regular-stack"),
				StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/regular-stack/xyz789"),
				StackStatus: types.StackStatusCreateComplete,
				Description: aws.String("Regular CloudFormation stack"),
			},
		},
	}

	detector := NewDetector(mockClient, "us-east-1")
	ctx := context.Background()

	stacks, err := detector.DetectServerlessStacks(ctx)

	require.NoError(t, err)
	assert.Empty(t, stacks, "Should not detect non-serverless stacks")
}

func TestDetector_DetectServerlessStacks_MultipleStacks(t *testing.T) {
	mockClient := &mockAWSClient{
		stacks: []types.StackSummary{
			{
				StackName:   aws.String("sls-stack-1"),
				StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/sls-stack-1/abc123"),
				StackStatus: types.StackStatusCreateComplete,
			},
			{
				StackName:   aws.String("regular-stack"),
				StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/regular-stack/def456"),
				StackStatus: types.StackStatusCreateComplete,
			},
			{
				StackName:   aws.String("sls-stack-2"),
				StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/sls-stack-2/ghi789"),
				StackStatus: types.StackStatusCreateComplete,
			},
		},
		resources: map[string][]types.StackResource{
			"sls-stack-1": {
				{
					LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
					ResourceType:      aws.String("AWS::S3::Bucket"),
				},
			},
			"regular-stack": {
				{
					LogicalResourceId: aws.String("MyBucket"),
					ResourceType:      aws.String("AWS::S3::Bucket"),
				},
			},
			"sls-stack-2": {
				{
					LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
					ResourceType:      aws.String("AWS::S3::Bucket"),
				},
			},
		},
		details: map[string]*types.Stack{
			"sls-stack-1": {
				StackName:   aws.String("sls-stack-1"),
				StackStatus: types.StackStatusCreateComplete,
			},
			"regular-stack": {
				StackName:   aws.String("regular-stack"),
				StackStatus: types.StackStatusCreateComplete,
			},
			"sls-stack-2": {
				StackName:   aws.String("sls-stack-2"),
				StackStatus: types.StackStatusCreateComplete,
			},
		},
	}

	detector := NewDetector(mockClient, "us-east-1")
	ctx := context.Background()

	stacks, err := detector.DetectServerlessStacks(ctx)

	require.NoError(t, err)
	assert.Len(t, stacks, 2, "Should detect exactly 2 serverless stacks")

	stackNames := make([]string, len(stacks))
	for i, stack := range stacks {
		stackNames[i] = stack.StackName
		assert.Contains(t, stack.Reasons, "Contains resource with logical ID 'ServerlessDeploymentBucket'")
	}

	assert.Contains(t, stackNames, "sls-stack-1")
	assert.Contains(t, stackNames, "sls-stack-2")
	assert.NotContains(t, stackNames, "regular-stack")
}

func TestHasServerlessDeploymentBucket(t *testing.T) {
	tests := []struct {
		name      string
		resources []types.StackResource
		expected  bool
	}{
		{
			name: "has ServerlessDeploymentBucket",
			resources: []types.StackResource{
				{
					LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
					ResourceType:      aws.String("AWS::S3::Bucket"),
				},
			},
			expected: true,
		},
		{
			name: "has multiple resources including ServerlessDeploymentBucket",
			resources: []types.StackResource{
				{
					LogicalResourceId: aws.String("MyFunction"),
					ResourceType:      aws.String("AWS::Lambda::Function"),
				},
				{
					LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
					ResourceType:      aws.String("AWS::S3::Bucket"),
				},
				{
					LogicalResourceId: aws.String("MyRole"),
					ResourceType:      aws.String("AWS::IAM::Role"),
				},
			},
			expected: true,
		},
		{
			name: "does not have ServerlessDeploymentBucket",
			resources: []types.StackResource{
				{
					LogicalResourceId: aws.String("MyBucket"),
					ResourceType:      aws.String("AWS::S3::Bucket"),
				},
			},
			expected: false,
		},
		{
			name: "has bucket with similar name but not exact match",
			resources: []types.StackResource{
				{
					LogicalResourceId: aws.String("ServerlessDeploymentBucket123"),
					ResourceType:      aws.String("AWS::S3::Bucket"),
				},
			},
			expected: false,
		},
		{
			name: "has ServerlessDeploymentBucket but not S3 bucket",
			resources: []types.StackResource{
				{
					LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
					ResourceType:      aws.String("AWS::Lambda::Function"),
				},
			},
			expected: false,
		},
		{
			name:      "empty resources",
			resources: []types.StackResource{},
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasServerlessDeploymentBucket(tt.resources)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to parse time for tests
func mustParseTime(s string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", s)
	if err != nil {
		panic(err)
	}
	return t
}
