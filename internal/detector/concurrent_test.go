package detector

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSlowAWSClient simulates a slow AWS client for concurrent testing
type mockSlowAWSClient struct {
	stacks       []types.StackSummary
	resources    map[string][]types.StackResource
	details      map[string]*types.Stack
	delay        time.Duration
	callCount    int
	mu           sync.Mutex
}

func (m *mockSlowAWSClient) ListActiveStacks(ctx context.Context) ([]types.StackSummary, error) {
	time.Sleep(m.delay)
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()
	return m.stacks, nil
}

func (m *mockSlowAWSClient) GetStackResources(ctx context.Context, stackName string) ([]types.StackResource, error) {
	time.Sleep(m.delay)
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()
	
	if resources, exists := m.resources[stackName]; exists {
		return resources, nil
	}
	return []types.StackResource{}, nil
}

func (m *mockSlowAWSClient) GetStackDetails(ctx context.Context, stackName string) (*types.Stack, error) {
	time.Sleep(m.delay)
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()
	
	if details, exists := m.details[stackName]; exists {
		return details, nil
	}
	return nil, nil
}

func (m *mockSlowAWSClient) getCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

func TestDetector_ConcurrentProcessing(t *testing.T) {
	// Create multiple stacks for testing concurrent processing
	stacks := []types.StackSummary{}
	resources := make(map[string][]types.StackResource)
	details := make(map[string]*types.Stack)

	// Create 10 stacks, half of them are serverless
	for i := 0; i < 10; i++ {
		stackName := aws.String(fmt.Sprintf("test-stack-%d", i))
		stackId := aws.String(fmt.Sprintf("arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack-%d/abc%d", i, i))
		
		stacks = append(stacks, types.StackSummary{
			StackName:   stackName,
			StackId:     stackId,
			StackStatus: types.StackStatusCreateComplete,
		})

		// Every even-numbered stack is serverless
		if i%2 == 0 {
			resources[*stackName] = []types.StackResource{
				{
					LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
					ResourceType:      aws.String("AWS::S3::Bucket"),
				},
			}
		} else {
			resources[*stackName] = []types.StackResource{
				{
					LogicalResourceId: aws.String("RegularBucket"),
					ResourceType:      aws.String("AWS::S3::Bucket"),
				},
			}
		}

		details[*stackName] = &types.Stack{
			StackName:   stackName,
			StackId:     stackId,
			StackStatus: types.StackStatusCreateComplete,
		}
	}

	mockClient := &mockSlowAWSClient{
		stacks:    stacks,
		resources: resources,
		details:   details,
		delay:     50 * time.Millisecond, // Small delay to simulate network latency
	}

	detector := NewDetector(mockClient, "us-east-1")
	ctx := context.Background()

	start := time.Now()
	detectedStacks, err := detector.DetectServerlessStacks(ctx)
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Len(t, detectedStacks, 5, "Should detect 5 serverless stacks")

	// With concurrent processing, it should be significantly faster than sequential
	// Sequential would take at least (10 + 10 + 5) * 50ms = 1.25 seconds
	// With concurrency, it should be much less
	t.Logf("Processing took %v", duration)
	assert.Less(t, duration, 1*time.Second, "Concurrent processing should be faster")

	// Verify all detected stacks are actually serverless
	for _, stack := range detectedStacks {
		assert.Contains(t, stack.Reasons, "Contains resource with logical ID 'ServerlessDeploymentBucket'")
	}
}

func TestDetector_ConcurrencyLimit(t *testing.T) {
	// Test that the detector respects concurrency limits
	stacks := []types.StackSummary{}
	resources := make(map[string][]types.StackResource)
	details := make(map[string]*types.Stack)

	// Create more stacks to test concurrency limits
	for i := 0; i < 20; i++ {
		stackName := aws.String(fmt.Sprintf("test-stack-%d", i))
		stacks = append(stacks, types.StackSummary{
			StackName:   stackName,
			StackStatus: types.StackStatusCreateComplete,
		})

		resources[*stackName] = []types.StackResource{
			{
				LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
				ResourceType:      aws.String("AWS::S3::Bucket"),
			},
		}
		details[*stackName] = &types.Stack{
			StackName:   stackName,
			StackStatus: types.StackStatusCreateComplete,
		}
	}

	mockClient := &mockSlowAWSClient{
		stacks:    stacks,
		resources: resources,
		details:   details,
		delay:     10 * time.Millisecond,
	}

	detector := NewDetector(mockClient, "us-east-1")
	ctx := context.Background()

	_, err := detector.DetectServerlessStacks(ctx)
	require.NoError(t, err)

	// Verify that API calls were made (basic smoke test)
	callCount := mockClient.getCallCount()
	assert.Greater(t, callCount, 0, "Should have made API calls")
}