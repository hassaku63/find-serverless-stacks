package detector

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockErrorAWSClient simulates various AWS error conditions
type mockErrorAWSClient struct {
	listStacksError      error
	getResourcesError    error
	getDetailsError      error
	inconsistentBehavior bool
	nilStackNames        bool
}

func (m *mockErrorAWSClient) ListActiveStacks(ctx context.Context) ([]types.StackSummary, error) {
	if m.listStacksError != nil {
		return nil, m.listStacksError
	}

	if m.nilStackNames {
		return []types.StackSummary{
			{
				StackName:   nil, // This should be handled gracefully
				StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack//abc123"),
				StackStatus: types.StackStatusCreateComplete,
			},
		}, nil
	}

	return []types.StackSummary{
		{
			StackName:   aws.String("test-stack"),
			StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123"),
			StackStatus: types.StackStatusCreateComplete,
		},
	}, nil
}

func (m *mockErrorAWSClient) GetStackResources(ctx context.Context, stackName string) ([]types.StackResource, error) {
	if m.getResourcesError != nil {
		return nil, m.getResourcesError
	}

	if m.inconsistentBehavior && stackName == "test-stack" {
		// Simulate intermittent failures
		return nil, errors.New("temporary resource access failure")
	}

	return []types.StackResource{
		{
			LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
			ResourceType:      aws.String("AWS::S3::Bucket"),
		},
	}, nil
}

func (m *mockErrorAWSClient) GetStackDetails(ctx context.Context, stackName string) (*types.Stack, error) {
	if m.getDetailsError != nil {
		return nil, m.getDetailsError
	}

	return &types.Stack{
		StackName:   aws.String(stackName),
		StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/" + stackName + "/abc123"),
		StackStatus: types.StackStatusCreateComplete,
	}, nil
}

func TestDetector_ListStacksError(t *testing.T) {
	// Test error in ListActiveStacks
	mockClient := &mockErrorAWSClient{
		listStacksError: errors.New("access denied to list stacks"),
	}

	detector := NewDetector(mockClient, "us-east-1")
	ctx := context.Background()

	stacks, err := detector.DetectServerlessStacks(ctx)

	assert.Error(t, err)
	assert.Nil(t, stacks)
	assert.Contains(t, err.Error(), "access denied")
}

func TestDetector_NilStackName(t *testing.T) {
	// Test handling of nil stack names
	mockClient := &mockErrorAWSClient{
		nilStackNames: true,
	}

	detector := NewDetector(mockClient, "us-east-1")
	ctx := context.Background()

	stacks, err := detector.DetectServerlessStacks(ctx)

	// Should not error, but should skip stacks with nil names
	require.NoError(t, err)
	assert.Empty(t, stacks)
}

func TestDetector_GetResourcesError(t *testing.T) {
	// Test error in GetStackResources
	mockClient := &mockErrorAWSClient{
		getResourcesError: errors.New("insufficient permissions to describe resources"),
	}

	detector := NewDetector(mockClient, "us-east-1")
	ctx := context.Background()

	stacks, err := detector.DetectServerlessStacks(ctx)

	// Should not fail completely, just skip problematic stacks
	require.NoError(t, err)
	assert.Empty(t, stacks) // No stacks detected due to resource access failure
}

func TestDetector_GetDetailsError(t *testing.T) {
	// Test error in GetStackDetails (should continue processing)
	mockClient := &mockErrorAWSClient{
		getDetailsError: errors.New("stack details not accessible"),
	}

	detector := NewDetector(mockClient, "us-east-1")
	ctx := context.Background()

	stacks, err := detector.DetectServerlessStacks(ctx)

	// Should continue processing even if details can't be retrieved
	require.NoError(t, err)
	assert.Len(t, stacks, 1) // Should still detect the serverless stack

	// Verify that basic information is still populated
	stack := stacks[0]
	assert.Equal(t, "test-stack", stack.StackName)
	assert.Equal(t, "us-east-1", stack.Region)
	assert.Contains(t, stack.Reasons, "Contains resource with logical ID 'ServerlessDeploymentBucket'")
}

func TestDetector_InconsistentBehavior(t *testing.T) {
	// Test handling of intermittent failures
	mockClient := &mockErrorAWSClient{
		inconsistentBehavior: true,
	}

	detector := NewDetector(mockClient, "us-east-1")
	ctx := context.Background()

	stacks, err := detector.DetectServerlessStacks(ctx)

	// Should handle intermittent failures gracefully
	require.NoError(t, err)
	assert.Empty(t, stacks) // No stacks detected due to resource access failure
}

func TestDetector_ContextCancellation(t *testing.T) {
	// Test handling of context cancellation
	mockClient := &mockSlowAWSClient{
		stacks: []types.StackSummary{
			{
				StackName:   aws.String("test-stack"),
				StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123"),
				StackStatus: types.StackStatusCreateComplete,
			},
		},
		resources: map[string][]types.StackResource{
			"test-stack": {
				{
					LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
					ResourceType:      aws.String("AWS::S3::Bucket"),
				},
			},
		},
		details: map[string]*types.Stack{
			"test-stack": {
				StackName:   aws.String("test-stack"),
				StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123"),
				StackStatus: types.StackStatusCreateComplete,
			},
		},
		delay: 100 * time.Millisecond, // Slow enough to allow cancellation
	}

	detector := NewDetector(mockClient, "us-east-1")

	// Create a context that will be cancelled quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	stacks, err := detector.DetectServerlessStacks(ctx)

	// The operation might complete or be cancelled depending on timing
	if err != nil {
		// If cancelled, should contain context error
		assert.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
			(err.Error() != "" && stacks == nil))
	} else {
		// If completed, should have valid results
		assert.NotNil(t, stacks)
	}
}

func TestDetector_EmptyResourceList(t *testing.T) {
	// Test handling of stacks with no resources
	mockClient := &mockAWSClient{
		stacks: []types.StackSummary{
			{
				StackName:   aws.String("empty-stack"),
				StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/empty-stack/abc123"),
				StackStatus: types.StackStatusCreateComplete,
			},
		},
		resources: map[string][]types.StackResource{
			"empty-stack": {}, // Empty resource list
		},
		details: map[string]*types.Stack{
			"empty-stack": {
				StackName:   aws.String("empty-stack"),
				StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/empty-stack/abc123"),
				StackStatus: types.StackStatusCreateComplete,
			},
		},
	}

	detector := NewDetector(mockClient, "us-east-1")
	ctx := context.Background()

	stacks, err := detector.DetectServerlessStacks(ctx)

	require.NoError(t, err)
	assert.Empty(t, stacks) // Should not detect any serverless stacks
}

func TestDetector_MalformedStackResources(t *testing.T) {
	// Test handling of malformed stack resources
	mockClient := &mockAWSClient{
		stacks: []types.StackSummary{
			{
				StackName:   aws.String("malformed-stack"),
				StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/malformed-stack/abc123"),
				StackStatus: types.StackStatusCreateComplete,
			},
		},
		resources: map[string][]types.StackResource{
			"malformed-stack": {
				{
					LogicalResourceId: nil, // nil resource ID
					ResourceType:      aws.String("AWS::S3::Bucket"),
				},
				{
					LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
					ResourceType:      nil, // nil resource type
				},
				{
					LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
					ResourceType:      aws.String("AWS::S3::Bucket"), // This should match
				},
			},
		},
		details: make(map[string]*types.Stack),
	}

	detector := NewDetector(mockClient, "us-east-1")
	ctx := context.Background()

	stacks, err := detector.DetectServerlessStacks(ctx)

	require.NoError(t, err)
	assert.Len(t, stacks, 1) // Should still detect the valid resource

	stack := stacks[0]
	assert.Equal(t, "malformed-stack", stack.StackName)
	assert.Contains(t, stack.Reasons, "Contains resource with logical ID 'ServerlessDeploymentBucket'")
}

func TestDetector_ResilientToPartialFailures(t *testing.T) {
	// Test that detector continues processing other stacks when some fail
	mockClient := &mockSelectiveErrorAWSClient{
		stacks: []types.StackSummary{
			{
				StackName:   aws.String("good-stack"),
				StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/good-stack/abc123"),
				StackStatus: types.StackStatusCreateComplete,
			},
			{
				StackName:   aws.String("failing-stack"),
				StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/failing-stack/def456"),
				StackStatus: types.StackStatusCreateComplete,
			},
		},
		failingStacks: map[string]bool{
			"failing-stack": true,
		},
		resources: map[string][]types.StackResource{
			"good-stack": {
				{
					LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
					ResourceType:      aws.String("AWS::S3::Bucket"),
				},
			},
		},
		details: make(map[string]*types.Stack),
	}

	detector := NewDetector(mockClient, "us-east-1")
	ctx := context.Background()

	stacks, err := detector.DetectServerlessStacks(ctx)

	require.NoError(t, err)
	assert.Len(t, stacks, 1) // Should detect only the successful stack
	assert.Equal(t, "good-stack", stacks[0].StackName)
}

// mockSelectiveErrorAWSClient fails only for specific stacks
type mockSelectiveErrorAWSClient struct {
	stacks        []types.StackSummary
	resources     map[string][]types.StackResource
	details       map[string]*types.Stack
	failingStacks map[string]bool
}

func (m *mockSelectiveErrorAWSClient) ListActiveStacks(ctx context.Context) ([]types.StackSummary, error) {
	return m.stacks, nil
}

func (m *mockSelectiveErrorAWSClient) GetStackResources(ctx context.Context, stackName string) ([]types.StackResource, error) {
	if m.failingStacks[stackName] {
		return nil, errors.New("resource access failed for " + stackName)
	}

	if resources, exists := m.resources[stackName]; exists {
		return resources, nil
	}
	return []types.StackResource{}, nil
}

func (m *mockSelectiveErrorAWSClient) GetStackDetails(ctx context.Context, stackName string) (*types.Stack, error) {
	if details, exists := m.details[stackName]; exists {
		return details, nil
	}
	return nil, nil
}
