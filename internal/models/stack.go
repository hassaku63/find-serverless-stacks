package models

import (
	"time"
)

// Stack represents a CloudFormation stack with detection information
type Stack struct {
	StackName   string            `json:"stackName"`
	StackID     string            `json:"stackId"`
	Region      string            `json:"region"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
	Description string            `json:"description"`
	StackTags   map[string]string `json:"stackTags"`
	Reasons     []string          `json:"reasons"`
}

// StacksOutput represents the output structure for multiple stacks
type StacksOutput struct {
	Stacks []Stack `json:"stacks"`
}
