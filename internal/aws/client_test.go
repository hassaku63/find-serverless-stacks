package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCloudFormationAPI implements CloudFormationAPI interface for testing
type mockCloudFormationAPI struct {
	listStacksFunc             func(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error)
	describeStacksFunc         func(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
	describeStackResourcesFunc func(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error)
}

func (m *mockCloudFormationAPI) ListStacks(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error) {
	if m.listStacksFunc != nil {
		return m.listStacksFunc(ctx, params, optFns...)
	}
	return &cloudformation.ListStacksOutput{}, nil
}

func (m *mockCloudFormationAPI) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	if m.describeStacksFunc != nil {
		return m.describeStacksFunc(ctx, params, optFns...)
	}
	return &cloudformation.DescribeStacksOutput{}, nil
}

func (m *mockCloudFormationAPI) DescribeStackResources(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
	if m.describeStackResourcesFunc != nil {
		return m.describeStackResourcesFunc(ctx, params, optFns...)
	}
	return &cloudformation.DescribeStackResourcesOutput{}, nil
}

func TestClient_ListActiveStacks(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   *cloudformation.ListStacksOutput
		mockError      error
		expectedStacks int
		expectError    bool
	}{
		{
			name: "successful stack listing",
			mockResponse: &cloudformation.ListStacksOutput{
				StackSummaries: []types.StackSummary{
					{
						StackName:   aws.String("stack-1"),
						StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/stack-1/abc123"),
						StackStatus: types.StackStatusCreateComplete,
					},
					{
						StackName:   aws.String("stack-2"),
						StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/stack-2/def456"),
						StackStatus: types.StackStatusUpdateComplete,
					},
				},
			},
			expectedStacks: 2,
			expectError:    false,
		},
		{
			name:           "empty stack list",
			mockResponse:   &cloudformation.ListStacksOutput{StackSummaries: []types.StackSummary{}},
			expectedStacks: 0,
			expectError:    false,
		},
		{
			name:        "API error",
			mockError:   assert.AnError,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCloudFormationAPI{
				listStacksFunc: func(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			client := NewClient(mock, "us-east-1")
			ctx := context.Background()

			stacks, err := client.ListActiveStacks(ctx)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, stacks, tt.expectedStacks)

			if tt.expectedStacks > 0 {
				assert.NotNil(t, stacks[0].StackName)
				assert.NotNil(t, stacks[0].StackId)
			}
		})
	}
}

func TestClient_GetStackResources(t *testing.T) {
	tests := []struct {
		name              string
		stackName         string
		mockResponse      *cloudformation.DescribeStackResourcesOutput
		mockError         error
		expectedResources int
		expectError       bool
	}{
		{
			name:      "successful resource listing",
			stackName: "test-stack",
			mockResponse: &cloudformation.DescribeStackResourcesOutput{
				StackResources: []types.StackResource{
					{
						LogicalResourceId:  aws.String("ServerlessDeploymentBucket"),
						PhysicalResourceId: aws.String("test-stack-serverlessdeploymentbucket-abc123"),
						ResourceType:       aws.String("AWS::S3::Bucket"),
						ResourceStatus:     types.ResourceStatusCreateComplete,
					},
					{
						LogicalResourceId:  aws.String("MyFunction"),
						PhysicalResourceId: aws.String("test-stack-MyFunction-def456"),
						ResourceType:       aws.String("AWS::Lambda::Function"),
						ResourceStatus:     types.ResourceStatusCreateComplete,
					},
				},
			},
			expectedResources: 2,
			expectError:       false,
		},
		{
			name:      "empty resource list",
			stackName: "empty-stack",
			mockResponse: &cloudformation.DescribeStackResourcesOutput{
				StackResources: []types.StackResource{},
			},
			expectedResources: 0,
			expectError:       false,
		},
		{
			name:        "stack not found",
			stackName:   "nonexistent-stack",
			mockError:   assert.AnError,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCloudFormationAPI{
				describeStackResourcesFunc: func(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
					assert.Equal(t, tt.stackName, *params.StackName)
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			client := NewClient(mock, "us-east-1")
			ctx := context.Background()

			resources, err := client.GetStackResources(ctx, tt.stackName)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, resources, tt.expectedResources)
		})
	}
}

func TestClient_GetStackDetails(t *testing.T) {
	tests := []struct {
		name         string
		stackName    string
		mockResponse *cloudformation.DescribeStacksOutput
		mockError    error
		expectStack  bool
		expectError  bool
	}{
		{
			name:      "successful stack details",
			stackName: "test-stack",
			mockResponse: &cloudformation.DescribeStacksOutput{
				Stacks: []types.Stack{
					{
						StackName:   aws.String("test-stack"),
						StackId:     aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123"),
						StackStatus: types.StackStatusCreateComplete,
						Description: aws.String("Test stack description"),
					},
				},
			},
			expectStack: true,
			expectError: false,
		},
		{
			name:      "stack not found",
			stackName: "nonexistent-stack",
			mockResponse: &cloudformation.DescribeStacksOutput{
				Stacks: []types.Stack{},
			},
			expectStack: false,
			expectError: false,
		},
		{
			name:        "API error",
			stackName:   "error-stack",
			mockError:   assert.AnError,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockCloudFormationAPI{
				describeStacksFunc: func(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
					assert.Equal(t, tt.stackName, *params.StackName)
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockResponse, nil
				},
			}

			client := NewClient(mock, "us-east-1")
			ctx := context.Background()

			stack, err := client.GetStackDetails(ctx, tt.stackName)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.expectStack {
				require.NotNil(t, stack)
				assert.Equal(t, tt.stackName, *stack.StackName)
			} else {
				assert.Nil(t, stack)
			}
		})
	}
}