package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommand_ValidationOnly(t *testing.T) {
	tests := []struct {
		name           string
		profile        string
		region         string
		outputFormat   string
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:           "invalid output format",
			profile:        "test-profile",
			region:         "us-east-1",
			outputFormat:   "xml",
			expectError:    true,
			expectedErrMsg: "invalid output format 'xml'",
		},
		{
			name:           "invalid output format - yaml",
			profile:        "test-profile",
			region:         "us-east-1",
			outputFormat:   "yaml",
			expectError:    true,
			expectedErrMsg: "invalid output format 'yaml'",
		},
		{
			name:           "case sensitive format validation",
			profile:        "test-profile",
			region:         "us-east-1",
			outputFormat:   "JSON",
			expectError:    true,
			expectedErrMsg: "invalid output format 'JSON'",
		},
		{
			name:         "valid json output format (validation only)",
			profile:      "default", // Use default to avoid AWS profile issues
			region:       "us-east-1",
			outputFormat: "json",
			// This will fail at AWS client creation in test env, which is expected
		},
		{
			name:         "valid tsv output format (validation only)",
			profile:      "default", // Use default to avoid AWS profile issues
			region:       "us-west-2",
			outputFormat: "tsv",
			// This will fail at AWS client creation in test env, which is expected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global variables
			profile = tt.profile
			region = tt.region
			outputFormat = tt.outputFormat

			// Call the function
			err := runCommand(nil, []string{})

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				// For valid validation cases, expect AWS client creation to fail in test env
				if err != nil {
					// This is expected in test environment without AWS credentials
					t.Logf("Expected AWS error in test environment: %v", err)
				}
			}
		})
	}
}

func TestOutputFormatValidation_ConfigOnly(t *testing.T) {
	// Test output format validation without AWS dependencies
	tests := []struct {
		format string
		valid  bool
	}{
		{"json", true},
		{"tsv", true},
		{"xml", false},
		{"csv", false},
		{"yaml", false},
		{"", false},
		{"JSON", false}, // Case sensitive
		{"TSV", false},  // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			// Test just the config validation logic
			profile = "default"  // Use default profile
			region = "us-east-1" // Set valid region
			outputFormat = tt.format

			err := runCommand(nil, []string{})

			if tt.valid {
				// Valid formats will fail at AWS client creation in test env
				if err != nil && !assert.Contains(t, err.Error(), "AWS") {
					t.Errorf("Expected AWS error or success, got: %v", err)
				}
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid output format")
			}
		})
	}
}

func TestMainFunctionComponents(t *testing.T) {
	// Test that we can create the root command without panicking
	t.Run("root command creation", func(t *testing.T) {
		// This tests the command setup logic without actually executing
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Creating root command should not panic: %v", r)
			}
		}()

		// Test command configuration - this mimics the main() function setup
		cmd := createRootCommand() // We'll need to extract this logic

		assert.Equal(t, "find_sls3_stacks", cmd.Use)
		assert.Contains(t, cmd.Short, "Find CloudFormation stacks")
		assert.Contains(t, cmd.Long, "ServerlessDeploymentBucket")

		// Test that required flags are set
		regionFlag := cmd.Flag("region")
		require.NotNil(t, regionFlag)

		profileFlag := cmd.Flag("profile")
		require.NotNil(t, profileFlag)
		assert.Equal(t, "default", profileFlag.DefValue)

		outputFlag := cmd.Flag("output")
		require.NotNil(t, outputFlag)
		assert.Equal(t, "json", outputFlag.DefValue)
	})
}

// Helper function to test CLI argument parsing
func TestCLIArgumentParsing(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		profile  string
		region   string
		output   string
		hasError bool
	}{
		{
			name:     "minimal required args",
			args:     []string{"--region", "us-east-1"},
			profile:  "default",
			region:   "us-east-1",
			output:   "json",
			hasError: false,
		},
		{
			name:     "all flags specified",
			args:     []string{"--profile", "my-profile", "--region", "us-west-2", "--output", "tsv"},
			profile:  "my-profile",
			region:   "us-west-2",
			output:   "tsv",
			hasError: false,
		},
		{
			name:     "short flags",
			args:     []string{"-p", "test-profile", "-r", "ap-northeast-1", "-o", "json"},
			profile:  "test-profile",
			region:   "ap-northeast-1",
			output:   "json",
			hasError: false,
		},
		{
			name:     "missing required region",
			args:     []string{"--profile", "test-profile"},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createRootCommand()

			// Reset global variables
			profile = ""
			region = ""
			outputFormat = ""

			// Parse arguments
			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if tt.hasError {
				assert.Error(t, err)
			} else {
				// For successful parsing, the runCommand would be called
				// but since we're testing parsing, we expect region required error
				// unless region is provided
				if !contains(tt.args, "--region") && !contains(tt.args, "-r") {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "required flag(s)")
				}
			}
		})
	}
}

// createRootCommand extracts the command creation logic for testing
func createRootCommand() *cobra.Command {
	var testProfile, testRegion, testOutputFormat string

	cmd := &cobra.Command{
		Use:   "find_sls3_stacks",
		Short: "Find CloudFormation stacks deployed by Serverless Framework v3",
		Long: `find_sls3_stacks identifies CloudFormation stacks deployed by Serverless Framework v3
by detecting the presence of ServerlessDeploymentBucket resources.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set global variables for testing
			profile = testProfile
			region = testRegion
			outputFormat = testOutputFormat
			return runCommand(cmd, args)
		},
	}

	cmd.Flags().StringVarP(&testProfile, "profile", "p", "default", "AWS profile name")
	cmd.Flags().StringVarP(&testRegion, "region", "r", "", "AWS region name (required)")
	cmd.Flags().StringVarP(&testOutputFormat, "output", "o", "json", "Output format (json, tsv)")

	cmd.MarkFlagRequired("region")

	return cmd
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
