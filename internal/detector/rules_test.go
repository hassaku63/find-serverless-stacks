package detector

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/stretchr/testify/assert"
)

func TestServerlessDeploymentBucketRule_Check(t *testing.T) {
	rule := &ServerlessDeploymentBucketRule{}

	tests := []struct {
		name      string
		resources []types.StackResource
		details   *types.Stack
		expected  bool
		reason    string
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
			reason:   "Contains resource with logical ID 'ServerlessDeploymentBucket'",
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
			reason:   "",
		},
		{
			name:      "empty resources",
			resources: []types.StackResource{},
			expected:  false,
			reason:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, reason := rule.Check(tt.resources, tt.details)
			assert.Equal(t, tt.expected, matches)
			assert.Equal(t, tt.reason, reason)
		})
	}
}

func TestServerlessDeploymentBucketRule_Name(t *testing.T) {
	rule := &ServerlessDeploymentBucketRule{}
	assert.Equal(t, "ServerlessDeploymentBucket", rule.Name())
}

func TestRuleEngine_NewRuleEngine(t *testing.T) {
	engine := NewRuleEngine()
	assert.NotNil(t, engine)
	assert.Len(t, engine.rules, 1) // Should have default ServerlessDeploymentBucket rule
}

func TestRuleEngine_AddRule(t *testing.T) {
	engine := NewRuleEngine()
	initialCount := len(engine.rules)

	// Create a mock rule
	mockRule := &mockDetectionRule{
		name:    "MockRule",
		matches: true,
		reason:  "Mock reason",
	}

	engine.AddRule(mockRule)
	assert.Len(t, engine.rules, initialCount+1)
}

func TestRuleEngine_Evaluate(t *testing.T) {
	tests := []struct {
		name            string
		resources       []types.StackResource
		details         *types.Stack
		expectedMatch   bool
		expectedReasons []string
	}{
		{
			name: "matches ServerlessDeploymentBucket rule",
			resources: []types.StackResource{
				{
					LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
					ResourceType:      aws.String("AWS::S3::Bucket"),
				},
			},
			expectedMatch:   true,
			expectedReasons: []string{"Contains resource with logical ID 'ServerlessDeploymentBucket'"},
		},
		{
			name: "no matching rules",
			resources: []types.StackResource{
				{
					LogicalResourceId: aws.String("RegularBucket"),
					ResourceType:      aws.String("AWS::S3::Bucket"),
				},
			},
			expectedMatch:   false,
			expectedReasons: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewRuleEngine()
			matches, reasons := engine.Evaluate(tt.resources, tt.details)

			assert.Equal(t, tt.expectedMatch, matches)
			assert.Equal(t, tt.expectedReasons, reasons)
		})
	}
}

func TestRuleEngine_EvaluateWithMultipleRules(t *testing.T) {
	engine := NewRuleEngine()

	// Add a custom rule that always matches
	mockRule := &mockDetectionRule{
		name:    "AlwaysMatch",
		matches: true,
		reason:  "Always matches for testing",
	}
	engine.AddRule(mockRule)

	resources := []types.StackResource{
		{
			LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
			ResourceType:      aws.String("AWS::S3::Bucket"),
		},
	}

	matches, reasons := engine.Evaluate(resources, nil)

	assert.True(t, matches)
	assert.Len(t, reasons, 2) // Should have reasons from both rules
	assert.Contains(t, reasons, "Contains resource with logical ID 'ServerlessDeploymentBucket'")
	assert.Contains(t, reasons, "Always matches for testing")
}

// mockDetectionRule for testing
type mockDetectionRule struct {
	name    string
	matches bool
	reason  string
}

func (m *mockDetectionRule) Name() string {
	return m.name
}

func (m *mockDetectionRule) Check(resources []types.StackResource, details *types.Stack) (bool, string) {
	if m.matches {
		return true, m.reason
	}
	return false, ""
}
