package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hassaku63/find-serverless-stacks/internal/aws"
	"github.com/hassaku63/find-serverless-stacks/internal/config"
	"github.com/hassaku63/find-serverless-stacks/internal/detector"
	"github.com/hassaku63/find-serverless-stacks/internal/models"
	"github.com/hassaku63/find-serverless-stacks/internal/output"
	"github.com/spf13/cobra"
)

var (
	profile      string
	region       string
	outputFormat string

	// AssumeRole parameters
	assumeRole  string
	sessionName string
	duration    int32
	externalID  string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "find_serverless_stacks",
		Short: "Find CloudFormation stacks deployed by Serverless Framework",
		Long: `find_serverless_stacks identifies CloudFormation stacks deployed by Serverless Framework
by detecting the presence of ServerlessDeploymentBucket resources.`,
		RunE: runCommand,
	}

	rootCmd.Flags().StringVarP(&profile, "profile", "p", "default", "AWS profile name")
	rootCmd.Flags().StringVarP(&region, "region", "r", "", "AWS region name (required)")
	rootCmd.Flags().StringVarP(&outputFormat, "output", "o", "json", "Output format (json, tsv)")

	// AssumeRole flags
	rootCmd.Flags().StringVar(&assumeRole, "assume-role", "", "ARN of the IAM role to assume")
	rootCmd.Flags().StringVar(&sessionName, "session-name", "find-serverless-stacks-session", "Session name for the assumed role session")
	rootCmd.Flags().Int32Var(&duration, "duration", 3600, "Session duration in seconds (900-43200)")
	rootCmd.Flags().StringVar(&externalID, "external-id", "", "External ID for AssumeRole (required by some roles for security)")

	rootCmd.MarkFlagRequired("region")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create configuration
	cfg := config.Config{
		Profile:      profile,
		Region:       region,
		OutputFormat: outputFormat,
	}

	// Add AssumeRole configuration if specified
	if assumeRole != "" {
		cfg.AssumeRole = &config.AssumeRoleConfig{
			RoleARN:     assumeRole,
			SessionName: sessionName,
			Duration:    duration,
			ExternalID:  externalID,
		}
	}

	// Validate configuration
	if !config.ValidateOutputFormat(cfg.OutputFormat) {
		return fmt.Errorf("invalid output format '%s'. Supported formats: json, tsv", cfg.OutputFormat)
	}

	if cfg.Region == "" {
		return fmt.Errorf("region is required")
	}

	// Validate AssumeRole configuration if present
	if cfg.AssumeRole != nil {
		if err := cfg.AssumeRole.Validate(); err != nil {
			return fmt.Errorf("AssumeRole configuration invalid: %w", err)
		}
	}

	// Create AWS client
	client, err := createAWSClient(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Run detection
	result, err := runDetection(ctx, client, cfg)
	if err != nil {
		return fmt.Errorf("detection failed: %w", err)
	}

	// Output results
	fmt.Print(result)
	return nil
}

// createAWSClient creates and configures an AWS client
func createAWSClient(ctx context.Context, cfg config.Config) (detector.AWSClient, error) {
	auth := aws.AuthConfig{
		Profile: cfg.Profile,
		Region:  cfg.Region,
	}

	// Add AssumeRole configuration if present
	if cfg.AssumeRole != nil {
		auth.AssumeRole = &aws.AssumeRoleCredentials{
			RoleARN:     cfg.AssumeRole.RoleARN,
			SessionName: cfg.AssumeRole.SessionName,
			Duration:    cfg.AssumeRole.Duration,
			ExternalID:  cfg.AssumeRole.ExternalID,
		}
	}

	// Create AWS client
	client, err := aws.CreateClient(ctx, auth)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Validate credentials by attempting a simple operation
	_, err = client.ListActiveStacks(ctx)
	if err != nil {
		return nil, fmt.Errorf("AWS credentials validation failed: %w", err)
	}

	return client, nil
}

// runDetection executes the serverless stack detection
func runDetection(ctx context.Context, client detector.AWSClient, cfg config.Config) (string, error) {
	// Create detector
	d := detector.NewDetector(client, cfg.Region)

	// Detect serverless stacks
	stacks, err := d.DetectServerlessStacks(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to detect serverless stacks: %w", err)
	}

	// Format output
	return formatOutput(stacks, cfg.OutputFormat)
}

// formatOutput formats the detected stacks using the specified formatter
func formatOutput(stacks []models.Stack, format string) (string, error) {
	formatter, err := output.FormatterFactory(format)
	if err != nil {
		return "", fmt.Errorf("failed to create formatter: %w", err)
	}

	return formatter.Format(stacks)
}
