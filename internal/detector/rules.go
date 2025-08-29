package detector

import "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"

// DetectionRule defines a rule for identifying serverless stacks
type DetectionRule interface {
	Check(resources []types.StackResource, details *types.Stack) (bool, string)
	Name() string
}

// ServerlessDeploymentBucketRule checks for the presence of ServerlessDeploymentBucket
type ServerlessDeploymentBucketRule struct{}

func (r *ServerlessDeploymentBucketRule) Name() string {
	return "ServerlessDeploymentBucket"
}

func (r *ServerlessDeploymentBucketRule) Check(resources []types.StackResource, details *types.Stack) (bool, string) {
	if hasServerlessDeploymentBucket(resources) {
		return true, "Contains resource with logical ID 'ServerlessDeploymentBucket'"
	}
	return false, ""
}

// RuleEngine manages and executes detection rules
type RuleEngine struct {
	rules []DetectionRule
}

// NewRuleEngine creates a new rule engine with default rules
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules: []DetectionRule{
			&ServerlessDeploymentBucketRule{},
		},
	}
}

// AddRule adds a custom detection rule
func (re *RuleEngine) AddRule(rule DetectionRule) {
	re.rules = append(re.rules, rule)
}

// Evaluate runs all rules against the given stack data
func (re *RuleEngine) Evaluate(resources []types.StackResource, details *types.Stack) (bool, []string) {
	var reasons []string
	isServerless := false

	for _, rule := range re.rules {
		if matches, reason := rule.Check(resources, details); matches {
			isServerless = true
			reasons = append(reasons, reason)
		}
	}

	return isServerless, reasons
}