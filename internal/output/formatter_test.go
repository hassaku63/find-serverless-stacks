package output

import (
	"strings"
	"testing"
	"time"

	"github.com/hassaku63/find-sls3-stacks/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONFormatter_Format(t *testing.T) {
	formatter := &JSONFormatter{}

	// Test data
	stacks := []models.Stack{
		{
			StackName:   "test-stack-1",
			StackID:     "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack-1/abc123",
			Region:      "us-east-1",
			Description: "Test stack 1",
			CreatedAt:   time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2023, 1, 16, 11, 45, 0, 0, time.UTC),
			StackTags: map[string]string{
				"Environment": "test",
				"Service":     "serverless",
			},
			Reasons: []string{"Contains resource with logical ID 'ServerlessDeploymentBucket'"},
		},
		{
			StackName: "test-stack-2",
			StackID:   "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack-2/def456",
			Region:    "us-east-1",
			CreatedAt: time.Date(2023, 2, 1, 9, 15, 0, 0, time.UTC),
			UpdatedAt: time.Date(2023, 2, 1, 9, 15, 0, 0, time.UTC),
			StackTags: map[string]string{},
			Reasons:   []string{"Contains resource with logical ID 'ServerlessDeploymentBucket'"},
		},
	}

	tests := []struct {
		name     string
		stacks   []models.Stack
		validate func(t *testing.T, output string)
	}{
		{
			name:   "format multiple stacks",
			stacks: stacks,
			validate: func(t *testing.T, output string) {
				// Should contain valid JSON structure
				assert.Contains(t, output, `"stacks":[`)
				assert.Contains(t, output, `"stackName":"test-stack-1"`)
				assert.Contains(t, output, `"stackName":"test-stack-2"`)
				assert.Contains(t, output, `"region":"us-east-1"`)
				assert.Contains(t, output, `"Environment":"test"`)

				// Should be properly formatted JSON (no extra whitespace)
				assert.False(t, strings.Contains(output, "\n"))
			},
		},
		{
			name:   "format empty stack list",
			stacks: []models.Stack{},
			validate: func(t *testing.T, output string) {
				expected := `{"stacks":[]}`
				assert.Equal(t, expected, output)
			},
		},
		{
			name:   "format single stack",
			stacks: stacks[:1],
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, `"stacks":[{`)
				assert.Contains(t, output, `"stackName":"test-stack-1"`)
				// Should only contain one stack
				stackCount := strings.Count(output, `"stackName":`)
				assert.Equal(t, 1, stackCount)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := formatter.Format(tt.stacks)
			require.NoError(t, err)
			tt.validate(t, output)
		})
	}
}

func TestTSVFormatter_Format(t *testing.T) {
	formatter := &TSVFormatter{}

	// Test data
	stacks := []models.Stack{
		{
			StackName:   "test-stack-1",
			StackID:     "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack-1/abc123",
			Region:      "us-east-1",
			Description: "Test stack 1",
			CreatedAt:   time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2023, 1, 16, 11, 45, 0, 0, time.UTC),
			StackTags: map[string]string{
				"Environment": "test",
				"Service":     "serverless",
			},
			Reasons: []string{"Contains resource with logical ID 'ServerlessDeploymentBucket'"},
		},
		{
			StackName: "test-stack-2",
			StackID:   "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack-2/def456",
			Region:    "us-east-1",
			CreatedAt: time.Date(2023, 2, 1, 9, 15, 0, 0, time.UTC),
			UpdatedAt: time.Date(2023, 2, 1, 9, 15, 0, 0, time.UTC),
			StackTags: map[string]string{},
			Reasons:   []string{"Contains resource with logical ID 'ServerlessDeploymentBucket'"},
		},
	}

	tests := []struct {
		name     string
		stacks   []models.Stack
		validate func(t *testing.T, output string)
	}{
		{
			name:   "format multiple stacks",
			stacks: stacks,
			validate: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")

				// Should have header + 2 data rows
				assert.Len(t, lines, 3)

				// Check header
				expectedHeader := "StackName\tStackID\tRegion\tDescription\tCreatedAt\tUpdatedAt\tTags\tReasons"
				assert.Equal(t, expectedHeader, lines[0])

				// Check first data row
				assert.Contains(t, lines[1], "test-stack-1")
				assert.Contains(t, lines[1], "us-east-1")
				assert.Contains(t, lines[1], "Test stack 1")

				// Check second data row
				assert.Contains(t, lines[2], "test-stack-2")
				assert.Contains(t, lines[2], "us-east-1")
			},
		},
		{
			name:   "format empty stack list",
			stacks: []models.Stack{},
			validate: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")

				// Should only have header
				assert.Len(t, lines, 1)
				expectedHeader := "StackName\tStackID\tRegion\tDescription\tCreatedAt\tUpdatedAt\tTags\tReasons"
				assert.Equal(t, expectedHeader, lines[0])
			},
		},
		{
			name:   "format single stack",
			stacks: stacks[:1],
			validate: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")

				// Should have header + 1 data row
				assert.Len(t, lines, 2)
				assert.Contains(t, lines[1], "test-stack-1")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := formatter.Format(tt.stacks)
			require.NoError(t, err)
			tt.validate(t, output)
		})
	}
}

func TestTSVFormatter_EscapeTabsAndNewlines(t *testing.T) {
	formatter := &TSVFormatter{}

	// Stack with tabs and newlines in description
	stacks := []models.Stack{
		{
			StackName:   "test-stack",
			StackID:     "arn:aws:cloudformation:us-east-1:123456789012:stack/test-stack/abc123",
			Region:      "us-east-1",
			Description: "Description with\ttabs and\nnewlines",
			CreatedAt:   time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2023, 1, 15, 10, 30, 0, 0, time.UTC),
			StackTags:   map[string]string{},
			Reasons:     []string{"Test reason"},
		},
	}

	output, err := formatter.Format(stacks)
	require.NoError(t, err)

	// Should escape tabs and newlines
	assert.Contains(t, output, "Description with\\ttabs and\\nnewlines")

	// Should not contain actual tabs or newlines in the data
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 2) // Only header and one data row
}

func TestFormatterFactory_Create(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		expectType  interface{}
		expectError bool
	}{
		{
			name:       "create JSON formatter",
			format:     "json",
			expectType: &JSONFormatter{},
		},
		{
			name:       "create TSV formatter",
			format:     "tsv",
			expectType: &TSVFormatter{},
		},
		{
			name:        "invalid format",
			format:      "xml",
			expectError: true,
		},
		{
			name:        "empty format",
			format:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter, err := FormatterFactory(tt.format)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, formatter)
			} else {
				assert.NoError(t, err)
				assert.IsType(t, tt.expectType, formatter)
			}
		})
	}
}
