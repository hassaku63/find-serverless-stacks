package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// CloudFormationAPI defines the interface for CloudFormation operations
// This interface enables mocking for testing
type CloudFormationAPI interface {
	ListStacks(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error)
	DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
	DescribeStackResources(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error)
}

// Client wraps AWS CloudFormation client with additional functionality
type Client struct {
	cf     CloudFormationAPI
	region string
}

// NewClient creates a new AWS client wrapper
func NewClient(cf CloudFormationAPI, region string) *Client {
	return &Client{
		cf:     cf,
		region: region,
	}
}

// ListActiveStacks returns all stacks in CREATE_COMPLETE, UPDATE_COMPLETE, or UPDATE_ROLLBACK_COMPLETE state
func (c *Client) ListActiveStacks(ctx context.Context) ([]types.StackSummary, error) {
	input := &cloudformation.ListStacksInput{
		StackStatusFilter: []types.StackStatus{
			types.StackStatusCreateComplete,
			types.StackStatusUpdateComplete,
			types.StackStatusUpdateRollbackComplete,
		},
	}

	var allStacks []types.StackSummary
	paginator := cloudformation.NewListStacksPaginator(c.cf, input)

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		allStacks = append(allStacks, output.StackSummaries...)
	}

	return allStacks, nil
}

// GetStackResources returns all resources for a given stack
func (c *Client) GetStackResources(ctx context.Context, stackName string) ([]types.StackResource, error) {
	input := &cloudformation.DescribeStackResourcesInput{
		StackName: &stackName,
	}

	output, err := c.cf.DescribeStackResources(ctx, input)
	if err != nil {
		return nil, err
	}

	return output.StackResources, nil
}

// GetStackDetails returns detailed information about a stack
func (c *Client) GetStackDetails(ctx context.Context, stackName string) (*types.Stack, error) {
	input := &cloudformation.DescribeStacksInput{
		StackName: &stackName,
	}

	output, err := c.cf.DescribeStacks(ctx, input)
	if err != nil {
		return nil, err
	}

	if len(output.Stacks) == 0 {
		return nil, nil
	}

	return &output.Stacks[0], nil
}
