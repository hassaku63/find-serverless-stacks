package detector

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// BenchmarkDetector_ConcurrentProcessing benchmarks the concurrent stack processing
func BenchmarkDetector_ConcurrentProcessing(b *testing.B) {
	testCases := []struct {
		name       string
		numStacks  int
		maxWorkers int
		delay      time.Duration
	}{
		{"10_stacks_1_worker", 10, 1, 10 * time.Millisecond},
		{"10_stacks_5_workers", 10, 5, 10 * time.Millisecond},
		{"10_stacks_10_workers", 10, 10, 10 * time.Millisecond},
		{"50_stacks_10_workers", 50, 10, 5 * time.Millisecond},
		{"100_stacks_20_workers", 100, 20, 2 * time.Millisecond},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Create test data
			stacks, resources, details := createLargeTestDataset(tc.numStacks)

			mockClient := &mockSlowAWSClient{
				stacks:    stacks,
				resources: resources,
				details:   details,
				delay:     tc.delay,
			}

			detector := NewDetector(mockClient, "us-east-1")
			detector.maxWorkers = tc.maxWorkers
			ctx := context.Background()

			// Reset timer after setup
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := detector.DetectServerlessStacks(ctx)
				if err != nil {
					b.Fatalf("DetectServerlessStacks failed: %v", err)
				}
			}
		})
	}
}

// TestDetector_LargeScale tests detection with large numbers of stacks
func TestDetector_LargeScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large scale test in short mode")
	}

	testCases := []struct {
		name               string
		numStacks          int
		serverlessPercent  float64
		expectedServerless int
		maxExecutionTime   time.Duration
	}{
		{
			name:               "100_stacks_50_percent",
			numStacks:          100,
			serverlessPercent:  0.5,
			expectedServerless: 50,
			maxExecutionTime:   5 * time.Second,
		},
		{
			name:               "500_stacks_30_percent",
			numStacks:          500,
			serverlessPercent:  0.3,
			expectedServerless: 150,
			maxExecutionTime:   15 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create large test dataset
			stacks, resources, details := createLargeTestDatasetWithRatio(tc.numStacks, tc.serverlessPercent)

			mockClient := &mockSlowAWSClient{
				stacks:    stacks,
				resources: resources,
				details:   details,
				delay:     2 * time.Millisecond, // Simulate network latency
			}

			detector := NewDetector(mockClient, "us-east-1")
			ctx := context.Background()

			start := time.Now()
			detectedStacks, err := detector.DetectServerlessStacks(ctx)
			elapsed := time.Since(start)

			require.NoError(t, err)
			assert.Len(t, detectedStacks, tc.expectedServerless)
			assert.Less(t, elapsed, tc.maxExecutionTime)

			t.Logf("Processed %d stacks in %v, detected %d serverless stacks",
				tc.numStacks, elapsed, len(detectedStacks))
		})
	}
}

// TestDetector_ConcurrencyScaling tests how performance scales with worker count
func TestDetector_ConcurrencyScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency scaling test in short mode")
	}

	const numStacks = 50
	const baseDelay = 20 * time.Millisecond

	workerCounts := []int{1, 2, 5, 10, 20}
	results := make(map[int]time.Duration)

	// Create consistent test data
	stacks, resources, details := createLargeTestDataset(numStacks)

	for _, workers := range workerCounts {
		t.Run(fmt.Sprintf("%d_workers", workers), func(t *testing.T) {
			mockClient := &mockSlowAWSClient{
				stacks:    stacks,
				resources: resources,
				details:   details,
				delay:     baseDelay,
			}

			detector := NewDetector(mockClient, "us-east-1")
			detector.maxWorkers = workers
			ctx := context.Background()

			start := time.Now()
			detectedStacks, err := detector.DetectServerlessStacks(ctx)
			elapsed := time.Since(start)

			require.NoError(t, err)
			assert.Greater(t, len(detectedStacks), 0)

			results[workers] = elapsed
			t.Logf("Workers: %d, Time: %v, Rate: %.2f stacks/sec",
				workers, elapsed, float64(numStacks)/elapsed.Seconds())
		})
	}

	// Verify that concurrency improves performance
	assert.Greater(t, results[1], results[5], "5 workers should be faster than 1 worker")
	assert.Greater(t, results[5], results[10], "10 workers should be faster than 5 workers")
}

// TestDetector_MemoryUsage tests memory efficiency
func TestDetector_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	const numStacks = 100 // Reduce number for more reliable memory measurement
	stacks, resources, details := createLargeTestDataset(numStacks)

	mockClient := &mockSlowAWSClient{
		stacks:    stacks,
		resources: resources,
		details:   details,
		delay:     time.Millisecond,
	}

	detector := NewDetector(mockClient, "us-east-1")
	ctx := context.Background()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	detectedStacks, err := detector.DetectServerlessStacks(ctx)
	require.NoError(t, err)

	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Use TotalAlloc for more accurate measurement of total allocated memory
	memUsed := int64(m2.TotalAlloc - m1.TotalAlloc)

	// Avoid negative memory usage (can happen with GC timing)
	if memUsed < 0 {
		memUsed = int64(m2.Alloc) // Fallback to current allocation
	}

	memPerStack := float64(memUsed) / float64(numStacks)

	t.Logf("Processed %d stacks, total allocated %d bytes (%.2f bytes/stack), detected %d serverless",
		numStacks, memUsed, memPerStack, len(detectedStacks))

	// Reasonable memory usage per stack (less than 10KB per stack for this test)
	assert.Less(t, memPerStack, 10240.0, "Memory usage per stack should be reasonable")
	assert.Greater(t, len(detectedStacks), 0, "Should detect some serverless stacks")
}

// createLargeTestDataset creates test data for performance testing
func createLargeTestDataset(numStacks int) ([]types.StackSummary, map[string][]types.StackResource, map[string]*types.Stack) {
	return createLargeTestDatasetWithRatio(numStacks, 0.5) // 50% serverless by default
}

// createLargeTestDatasetWithRatio creates test data with specified serverless ratio
func createLargeTestDatasetWithRatio(numStacks int, serverlessRatio float64) ([]types.StackSummary, map[string][]types.StackResource, map[string]*types.Stack) {
	stacks := make([]types.StackSummary, numStacks)
	resources := make(map[string][]types.StackResource)
	details := make(map[string]*types.Stack)

	numServerless := int(float64(numStacks) * serverlessRatio)

	for i := 0; i < numStacks; i++ {
		stackName := aws.String(fmt.Sprintf("test-stack-%d", i))
		stackId := aws.String(fmt.Sprintf("arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack-%d/abc%d", i, i))

		stacks[i] = types.StackSummary{
			StackName:    stackName,
			StackId:      stackId,
			StackStatus:  types.StackStatusCreateComplete,
			CreationTime: aws.Time(time.Now().Add(-time.Duration(i) * time.Hour)),
		}

		// Make first numServerless stacks serverless
		if i < numServerless {
			resources[*stackName] = []types.StackResource{
				{
					LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
					ResourceType:      aws.String("AWS::S3::Bucket"),
				},
				{
					LogicalResourceId: aws.String("MyFunction"),
					ResourceType:      aws.String("AWS::Lambda::Function"),
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
			Description: aws.String(fmt.Sprintf("Test stack %d", i)),
		}
	}

	return stacks, resources, details
}
