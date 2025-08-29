package aws

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/smithy-go"
	"golang.org/x/time/rate"
)

// RateLimitedClient wraps the AWS client with rate limiting and error handling
type RateLimitedClient struct {
	client  CloudFormationAPI
	limiter *rate.Limiter
	region  string
}

// NewRateLimitedClient creates a new rate-limited client
// AWS CloudFormation has default limits of ~10 requests per second
func NewRateLimitedClient(client CloudFormationAPI, region string) *RateLimitedClient {
	// Conservative rate limiting: 5 requests per second with burst of 10
	limiter := rate.NewLimiter(rate.Limit(5), 10)

	return &RateLimitedClient{
		client:  client,
		limiter: limiter,
		region:  region,
	}
}

// ListStacks implements CloudFormationAPI with rate limiting
func (r *RateLimitedClient) ListStacks(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error) {
	if err := r.limiter.Wait(ctx); err != nil {
		return nil, &Error{
			Type:    ErrorTypeRateLimit,
			Message: "rate limit context cancelled",
			Cause:   err,
		}
	}

	output, err := r.client.ListStacks(ctx, params, optFns...)
	return output, r.handleError(err)
}

// DescribeStacks implements CloudFormationAPI with rate limiting
func (r *RateLimitedClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	if err := r.limiter.Wait(ctx); err != nil {
		return nil, &Error{
			Type:    ErrorTypeRateLimit,
			Message: "rate limit context cancelled",
			Cause:   err,
		}
	}

	output, err := r.client.DescribeStacks(ctx, params, optFns...)
	return output, r.handleError(err)
}

// DescribeStackResources implements CloudFormationAPI with rate limiting
func (r *RateLimitedClient) DescribeStackResources(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
	if err := r.limiter.Wait(ctx); err != nil {
		return nil, &Error{
			Type:    ErrorTypeRateLimit,
			Message: "rate limit context cancelled",
			Cause:   err,
		}
	}

	output, err := r.client.DescribeStackResources(ctx, params, optFns...)
	return output, r.handleError(err)
}

// handleError converts AWS errors to our custom error types
func (r *RateLimitedClient) handleError(err error) error {
	if err == nil {
		return nil
	}

	// Handle AWS service errors
	var awsErr smithy.APIError
	if errors.As(err, &awsErr) {
		switch awsErr.ErrorCode() {
		case "AccessDenied", "UnauthorizedOperation":
			return &Error{
				Type:    ErrorTypePermission,
				Message: "insufficient AWS permissions",
				Cause:   err,
			}
		case "Throttling", "RequestLimitExceeded", "TooManyRequestsException":
			return &Error{
				Type:    ErrorTypeRateLimit,
				Message: "AWS API rate limit exceeded",
				Cause:   err,
			}
		case "InvalidParameterValue":
			if strings.Contains(awsErr.ErrorMessage(), "region") {
				return &Error{
					Type:    ErrorTypeInvalidRegion,
					Message: "invalid AWS region: " + r.region,
					Cause:   err,
				}
			}
		}
	}

	// Handle context errors
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &Error{
			Type:    ErrorTypeNetwork,
			Message: "request timeout or cancelled",
			Cause:   err,
		}
	}

	// Handle network-related errors
	errMsg := err.Error()
	if strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "timeout") {
		return &Error{
			Type:    ErrorTypeNetwork,
			Message: "network connectivity issue",
			Cause:   err,
		}
	}

	// Check if it's already our custom error type
	var customErr *Error
	if errors.As(err, &customErr) {
		return err
	}

	// Default to unknown error
	return &Error{
		Type:    ErrorTypeUnknown,
		Message: "unexpected AWS API error",
		Cause:   err,
	}
}

// RetryableClient wraps RateLimitedClient with exponential backoff retry logic
type RetryableClient struct {
	client     *RateLimitedClient
	maxRetries int
}

// NewRetryableClient creates a new client with retry logic
func NewRetryableClient(client CloudFormationAPI, region string, maxRetries int) *RetryableClient {
	if maxRetries <= 0 {
		maxRetries = 3
	}

	return &RetryableClient{
		client:     NewRateLimitedClient(client, region),
		maxRetries: maxRetries,
	}
}

// ListStacks implements CloudFormationAPI with retry logic
func (r *RetryableClient) ListStacks(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error) {
	return r.retryListStacks(func() (*cloudformation.ListStacksOutput, error) {
		return r.client.ListStacks(ctx, params, optFns...)
	})
}

// DescribeStacks implements CloudFormationAPI with retry logic
func (r *RetryableClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	return r.retryDescribeStacks(func() (*cloudformation.DescribeStacksOutput, error) {
		return r.client.DescribeStacks(ctx, params, optFns...)
	})
}

// DescribeStackResources implements CloudFormationAPI with retry logic
func (r *RetryableClient) DescribeStackResources(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error) {
	return r.retryDescribeStackResources(func() (*cloudformation.DescribeStackResourcesOutput, error) {
		return r.client.DescribeStackResources(ctx, params, optFns...)
	})
}

// retryListStacks performs exponential backoff retry for ListStacks operations
func (r *RetryableClient) retryListStacks(operation func() (*cloudformation.ListStacksOutput, error)) (*cloudformation.ListStacksOutput, error) {
	var result *cloudformation.ListStacksOutput
	var lastErr error

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		result, lastErr = operation()
		if lastErr == nil {
			return result, nil
		}

		// Check if error is retryable
		var customErr *Error
		if errors.As(lastErr, &customErr) {
			if !isRetryableError(customErr.Type) {
				return result, lastErr
			}
		}

		// Don't sleep after the last attempt
		if attempt < r.maxRetries {
			backoffDuration := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(backoffDuration)
		}
	}

	return result, lastErr
}

// retryDescribeStacks performs exponential backoff retry for DescribeStacks operations
func (r *RetryableClient) retryDescribeStacks(operation func() (*cloudformation.DescribeStacksOutput, error)) (*cloudformation.DescribeStacksOutput, error) {
	var result *cloudformation.DescribeStacksOutput
	var lastErr error

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		result, lastErr = operation()
		if lastErr == nil {
			return result, nil
		}

		// Check if error is retryable
		var customErr *Error
		if errors.As(lastErr, &customErr) {
			if !isRetryableError(customErr.Type) {
				return result, lastErr
			}
		}

		// Don't sleep after the last attempt
		if attempt < r.maxRetries {
			backoffDuration := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(backoffDuration)
		}
	}

	return result, lastErr
}

// retryDescribeStackResources performs exponential backoff retry for DescribeStackResources operations
func (r *RetryableClient) retryDescribeStackResources(operation func() (*cloudformation.DescribeStackResourcesOutput, error)) (*cloudformation.DescribeStackResourcesOutput, error) {
	var result *cloudformation.DescribeStackResourcesOutput
	var lastErr error

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		result, lastErr = operation()
		if lastErr == nil {
			return result, nil
		}

		// Check if error is retryable
		var customErr *Error
		if errors.As(lastErr, &customErr) {
			if !isRetryableError(customErr.Type) {
				return result, lastErr
			}
		}

		// Don't sleep after the last attempt
		if attempt < r.maxRetries {
			backoffDuration := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(backoffDuration)
		}
	}

	return result, lastErr
}

// isRetryableError determines if an error type should be retried
func isRetryableError(errorType ErrorType) bool {
	switch errorType {
	case ErrorTypeRateLimit, ErrorTypeNetwork:
		return true
	default:
		return false
	}
}