package models

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStack_JSONSerialization(t *testing.T) {
	createdAt := time.Date(2023, 10, 1, 12, 34, 56, 0, time.UTC)
	updatedAt := time.Date(2023, 10, 2, 12, 34, 56, 0, time.UTC)

	stack := Stack{
		StackName:   "my-api-dev",
		StackID:     "arn:aws:cloudformation:us-east-1:123456789012:stack/my-api-dev/abc123",
		Region:      "us-east-1",
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		Description: "My Serverless Framework v3 stack",
		StackTags: map[string]string{
			"Owner":       "team-a",
			"Environment": "development",
		},
		Reasons: []string{
			"Contains resource with logical ID 'ServerlessDeploymentBucket'",
		},
	}

	// Test marshaling to JSON
	jsonData, err := json.Marshal(stack)
	require.NoError(t, err)

	expectedFields := []string{
		"stackName", "stackId", "region", "createdAt", "updatedAt",
		"description", "stackTags", "reasons",
	}

	jsonStr := string(jsonData)
	for _, field := range expectedFields {
		assert.Contains(t, jsonStr, field, "JSON should contain field %s", field)
	}

	// Test unmarshaling from JSON
	var unmarshaled Stack
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, stack.StackName, unmarshaled.StackName)
	assert.Equal(t, stack.StackID, unmarshaled.StackID)
	assert.Equal(t, stack.Region, unmarshaled.Region)
	assert.Equal(t, stack.CreatedAt.Unix(), unmarshaled.CreatedAt.Unix()) // Compare Unix timestamps to avoid timezone issues
	assert.Equal(t, stack.UpdatedAt.Unix(), unmarshaled.UpdatedAt.Unix())
	assert.Equal(t, stack.Description, unmarshaled.Description)
	assert.Equal(t, stack.StackTags, unmarshaled.StackTags)
	assert.Equal(t, stack.Reasons, unmarshaled.Reasons)
}

func TestStacksOutput_JSONSerialization(t *testing.T) {
	createdAt := time.Date(2023, 10, 1, 12, 34, 56, 0, time.UTC)
	updatedAt := time.Date(2023, 10, 2, 12, 34, 56, 0, time.UTC)

	output := StacksOutput{
		Stacks: []Stack{
			{
				StackName:   "stack-1",
				StackID:     "arn:aws:cloudformation:us-east-1:123456789012:stack/stack-1/abc123",
				Region:      "us-east-1",
				CreatedAt:   createdAt,
				UpdatedAt:   updatedAt,
				Description: "First stack",
				StackTags: map[string]string{
					"Owner": "team-a",
				},
				Reasons: []string{"reason-1"},
			},
			{
				StackName:   "stack-2",
				StackID:     "arn:aws:cloudformation:us-east-1:123456789012:stack/stack-2/def456",
				Region:      "us-east-1",
				CreatedAt:   createdAt,
				UpdatedAt:   updatedAt,
				Description: "Second stack",
				StackTags: map[string]string{
					"Owner": "team-b",
				},
				Reasons: []string{"reason-2"},
			},
		},
	}

	// Test marshaling
	jsonData, err := json.Marshal(output)
	require.NoError(t, err)

	jsonStr := string(jsonData)
	assert.Contains(t, jsonStr, "stacks")
	assert.Contains(t, jsonStr, "stack-1")
	assert.Contains(t, jsonStr, "stack-2")

	// Test unmarshaling
	var unmarshaled StacksOutput
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Len(t, unmarshaled.Stacks, 2)
	assert.Equal(t, "stack-1", unmarshaled.Stacks[0].StackName)
	assert.Equal(t, "stack-2", unmarshaled.Stacks[1].StackName)
}

func TestStack_EmptyValues(t *testing.T) {
	t.Run("empty stack", func(t *testing.T) {
		stack := Stack{}

		// Test that empty values are serialized correctly
		jsonData, err := json.Marshal(stack)
		require.NoError(t, err)

		var unmarshaled Stack
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Empty(t, unmarshaled.StackName)
		assert.Empty(t, unmarshaled.StackID)
		assert.Empty(t, unmarshaled.Region)
		assert.True(t, unmarshaled.CreatedAt.IsZero())
		assert.True(t, unmarshaled.UpdatedAt.IsZero())
		assert.Empty(t, unmarshaled.Description)
		assert.Nil(t, unmarshaled.StackTags) // nil map becomes nil after JSON roundtrip
		assert.Nil(t, unmarshaled.Reasons)   // nil slice becomes nil after JSON roundtrip
	})

	t.Run("empty collections", func(t *testing.T) {
		stack := Stack{
			StackName: "test-stack",
			StackTags: make(map[string]string), // Empty map
			Reasons:   make([]string, 0),       // Empty slice
		}

		jsonData, err := json.Marshal(stack)
		require.NoError(t, err)

		var unmarshaled Stack
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, "test-stack", unmarshaled.StackName)
		assert.Empty(t, unmarshaled.StackTags) // Empty map
		assert.Empty(t, unmarshaled.Reasons)   // Empty slice
	})
}

func TestStacksOutput_EmptyStacks(t *testing.T) {
	output := StacksOutput{
		Stacks: []Stack{},
	}

	jsonData, err := json.Marshal(output)
	require.NoError(t, err)

	jsonStr := string(jsonData)
	assert.Contains(t, jsonStr, `"stacks":[]`)

	var unmarshaled StacksOutput
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Empty(t, unmarshaled.Stacks)
}

func TestStack_LargeDataset(t *testing.T) {
	// Test with many tags and reasons
	largeTagMap := make(map[string]string)
	largeReasons := make([]string, 0)

	for i := 0; i < 100; i++ {
		largeTagMap[fmt.Sprintf("tag-%d", i)] = fmt.Sprintf("value-%d", i)
		largeReasons = append(largeReasons, fmt.Sprintf("reason-%d", i))
	}

	stack := Stack{
		StackName: "large-stack",
		StackTags: largeTagMap,
		Reasons:   largeReasons,
	}

	// Should handle large datasets without issues
	jsonData, err := json.Marshal(stack)
	require.NoError(t, err)

	var unmarshaled Stack
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, stack.StackName, unmarshaled.StackName)
	assert.Equal(t, len(stack.StackTags), len(unmarshaled.StackTags))
	assert.Equal(t, len(stack.Reasons), len(unmarshaled.Reasons))
}