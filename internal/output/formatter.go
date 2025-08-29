package output

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hassaku63/find-sls3-stacks/internal/models"
)

// Formatter defines the interface for output formatters
type Formatter interface {
	Format(stacks []models.Stack) (string, error)
}

// JSONFormatter formats output as JSON
type JSONFormatter struct{}

// Format implements the Formatter interface for JSON output
func (f *JSONFormatter) Format(stacks []models.Stack) (string, error) {
	// Ensure we have a non-nil slice for proper JSON serialization
	if stacks == nil {
		stacks = []models.Stack{}
	}
	
	output := models.StacksOutput{
		Stacks: stacks,
	}
	
	jsonData, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	
	return string(jsonData), nil
}

// TSVFormatter formats output as Tab-Separated Values
type TSVFormatter struct{}

// Format implements the Formatter interface for TSV output
func (f *TSVFormatter) Format(stacks []models.Stack) (string, error) {
	var result strings.Builder
	
	// Write header
	header := []string{
		"StackName",
		"StackID", 
		"Region",
		"Description",
		"CreatedAt",
		"UpdatedAt",
		"Tags",
		"Reasons",
	}
	result.WriteString(strings.Join(header, "\t"))
	result.WriteString("\n")
	
	// Write data rows
	for _, stack := range stacks {
		row := []string{
			f.escapeValue(stack.StackName),
			f.escapeValue(stack.StackID),
			f.escapeValue(stack.Region),
			f.escapeValue(stack.Description),
			f.formatTime(stack.CreatedAt),
			f.formatTime(stack.UpdatedAt),
			f.formatTags(stack.StackTags),
			f.formatReasons(stack.Reasons),
		}
		result.WriteString(strings.Join(row, "\t"))
		result.WriteString("\n")
	}
	
	// Remove trailing newline if present
	output := result.String()
	if strings.HasSuffix(output, "\n") {
		output = output[:len(output)-1]
	}
	
	return output, nil
}

// escapeValue escapes tabs and newlines in TSV values
func (f *TSVFormatter) escapeValue(value string) string {
	value = strings.ReplaceAll(value, "\t", "\\t")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, "\r", "\\r")
	return value
}

// formatTime formats time to RFC3339 string
func (f *TSVFormatter) formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// formatTags formats tags map as key=value pairs separated by semicolons
func (f *TSVFormatter) formatTags(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}
	
	var pairs []string
	
	// Sort keys for consistent output
	var keys []string
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	
	for _, key := range keys {
		value := tags[key]
		pair := fmt.Sprintf("%s=%s", f.escapeValue(key), f.escapeValue(value))
		pairs = append(pairs, pair)
	}
	
	return strings.Join(pairs, ";")
}

// formatReasons formats reasons slice as semicolon-separated values
func (f *TSVFormatter) formatReasons(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	
	var escapedReasons []string
	for _, reason := range reasons {
		escapedReasons = append(escapedReasons, f.escapeValue(reason))
	}
	
	return strings.Join(escapedReasons, ";")
}

// FormatterFactory creates a formatter based on the specified format
func FormatterFactory(format string) (Formatter, error) {
	switch format {
	case "json":
		return &JSONFormatter{}, nil
	case "tsv":
		return &TSVFormatter{}, nil
	default:
		return nil, fmt.Errorf("unsupported output format: %s (supported formats: json, tsv)", format)
	}
}