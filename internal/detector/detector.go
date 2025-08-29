package detector

import (
	"context"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/hassaku63/find-serverless-stacks/internal/models"
)

// AWSClient defines the interface for AWS operations needed by the detector
type AWSClient interface {
	ListActiveStacks(ctx context.Context) ([]types.StackSummary, error)
	GetStackResources(ctx context.Context, stackName string) ([]types.StackResource, error)
	GetStackDetails(ctx context.Context, stackName string) (*types.Stack, error)
}

// Detector identifies Serverless Framework v3 stacks
type Detector struct {
	client     AWSClient
	region     string
	ruleEngine *RuleEngine
	maxWorkers int
}

// NewDetector creates a new stack detector
func NewDetector(client AWSClient, region string) *Detector {
	return &Detector{
		client:     client,
		region:     region,
		ruleEngine: NewRuleEngine(),
		maxWorkers: 10, // Default to 10 concurrent workers
	}
}

// DetectServerlessStacks identifies all stacks deployed by Serverless Framework v3
func (d *Detector) DetectServerlessStacks(ctx context.Context) ([]models.Stack, error) {
	// Get all active stacks
	summaries, err := d.client.ListActiveStacks(ctx)
	if err != nil {
		return nil, err
	}

	return d.processStacksConcurrently(ctx, summaries)
}

// processStacksConcurrently processes stacks using worker pools for better performance
func (d *Detector) processStacksConcurrently(ctx context.Context, summaries []types.StackSummary) ([]models.Stack, error) {
	// Create channels for communication
	jobs := make(chan types.StackSummary, len(summaries))
	results := make(chan *models.Stack, len(summaries))

	// Start workers
	var wg sync.WaitGroup
	numWorkers := d.maxWorkers
	if len(summaries) < numWorkers {
		numWorkers = len(summaries)
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go d.worker(ctx, jobs, results, &wg)
	}

	// Send jobs to workers
	for _, summary := range summaries {
		jobs <- summary
	}
	close(jobs)

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var serverlessStacks []models.Stack
	for result := range results {
		if result != nil {
			serverlessStacks = append(serverlessStacks, *result)
		}
	}

	return serverlessStacks, nil
}

// worker processes individual stacks
func (d *Detector) worker(ctx context.Context, jobs <-chan types.StackSummary, results chan<- *models.Stack, wg *sync.WaitGroup) {
	defer wg.Done()

	for summary := range jobs {
		result := d.processStack(ctx, summary)
		results <- result
	}
}

// processStack processes a single stack
func (d *Detector) processStack(ctx context.Context, summary types.StackSummary) *models.Stack {
	if summary.StackName == nil {
		return nil
	}

	stackName := *summary.StackName

	// Get stack resources
	resources, err := d.client.GetStackResources(ctx, stackName)
	if err != nil {
		// Log error but continue processing
		return nil
	}

	// Get detailed stack information
	details, err := d.client.GetStackDetails(ctx, stackName)
	if err != nil {
		// Continue with basic information if details cannot be retrieved
		details = nil
	}

	// Check if this is a serverless stack using rule engine
	isServerless, reasons := d.ruleEngine.Evaluate(resources, details)
	if isServerless {
		stack := d.convertToModel(summary, details, reasons)
		return &stack
	}

	return nil
}

// hasServerlessDeploymentBucket checks if the stack contains the ServerlessDeploymentBucket resource
func hasServerlessDeploymentBucket(resources []types.StackResource) bool {
	for _, resource := range resources {
		if resource.LogicalResourceId != nil &&
			*resource.LogicalResourceId == "ServerlessDeploymentBucket" &&
			resource.ResourceType != nil &&
			*resource.ResourceType == "AWS::S3::Bucket" {
			return true
		}
	}
	return false
}

// convertToModel converts AWS types to our internal model
func (d *Detector) convertToModel(summary types.StackSummary, details *types.Stack, reasons []string) models.Stack {
	stack := models.Stack{
		Region:  d.region,
		Reasons: reasons,
	}

	// Set basic information from summary
	if summary.StackName != nil {
		stack.StackName = *summary.StackName
	}
	if summary.StackId != nil {
		stack.StackID = *summary.StackId
	}

	// Add detailed information if available
	if details != nil {
		if details.Description != nil {
			stack.Description = *details.Description
		}
		if details.CreationTime != nil {
			stack.CreatedAt = *details.CreationTime
		}
		if details.LastUpdatedTime != nil {
			stack.UpdatedAt = *details.LastUpdatedTime
		}

		// Convert tags
		stack.StackTags = make(map[string]string)
		for _, tag := range details.Tags {
			if tag.Key != nil && tag.Value != nil {
				stack.StackTags[*tag.Key] = *tag.Value
			}
		}
	} else {
		// Use summary information if details are not available
		if summary.CreationTime != nil {
			stack.CreatedAt = *summary.CreationTime
		}
		if summary.LastUpdatedTime != nil {
			stack.UpdatedAt = *summary.LastUpdatedTime
		}
		stack.StackTags = make(map[string]string)
	}

	// Ensure timestamps are not zero if not set
	if stack.CreatedAt.IsZero() {
		stack.CreatedAt = time.Now().UTC()
	}
	if stack.UpdatedAt.IsZero() {
		stack.UpdatedAt = stack.CreatedAt
	}

	return stack
}
