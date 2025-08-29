# Architecture Documentation

## System Overview

`find_serverless_stacks` is a Go CLI application that identifies CloudFormation stacks deployed by Serverless Framework through the detection of characteristic resources, primarily the `ServerlessDeploymentBucket`.

## Architecture Diagram

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   CLI Interface │───▶│   Configuration  │───▶│  AWS Client     │
│   (Cobra)       │    │   Validation     │    │  Factory        │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                                               │
         ▼                                               ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Detection     │◀───│   Rule Engine    │    │  AWS SDK v2     │
│   Coordinator   │    │   (Extensible)   │    │  CloudFormation │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       ▲
         ▼                       │                       │
┌─────────────────┐              │              ┌─────────────────┐
│   Concurrent    │              │              │   Rate Limiting │
│   Processing    │              │              │   & Retries     │
│   (Worker Pool) │              │              └─────────────────┘
└─────────────────┘              │
         │                       │
         ▼                       ▼
┌─────────────────┐    ┌──────────────────┐
│   Stack Model   │    │   Detection      │
│   Conversion    │    │   Rules          │
└─────────────────┘    └──────────────────┘
         │
         ▼
┌─────────────────┐
│   Output        │
│   Formatters    │
│   (JSON/TSV)    │
└─────────────────┘
```

## Core Components

### 1. CLI Interface (`cmd/find_serverless_stacks/`)

**Responsibility**: Command-line argument parsing and user interaction

- **Framework**: Cobra CLI framework
- **Input validation**: Profile, region, and output format validation
- **Error handling**: User-friendly error messages with usage information
- **Integration**: Orchestrates all other components

**Key Files**:
- `main.go`: Entry point and command setup
- `integration_test.go`: End-to-end CLI testing

### 2. Configuration Management (`internal/config/`)

**Responsibility**: Application configuration and validation

- **Config structure**: Centralized configuration management
- **Validation**: Output format validation (json/tsv)
- **Extensibility**: Easy addition of new configuration options

**Features**:
- Type-safe configuration structures
- Validation functions with clear error messages
- 100% test coverage

### 3. AWS Integration Layer (`internal/aws/`)

**Responsibility**: AWS SDK abstraction and resilience

**Components**:
- **Client Wrapper** (`client.go`): Abstracts AWS CloudFormation operations
- **Factory** (`factory.go`): Real AWS client creation with authentication  
- **Rate Limiting** (`ratelimit.go`): API call throttling and retry logic
- **Authentication** (`auth.go`): Profile and credential management

**Key Features**:
- Interface-based design for testability
- Exponential backoff retry mechanism
- Rate limiting to respect AWS API limits
- Comprehensive error handling

### 4. Detection Engine (`internal/detector/`)

**Responsibility**: Core serverless stack identification logic

**Components**:
- **Detector** (`detector.go`): Main detection coordinator with concurrent processing
- **Rule Engine** (`rules.go`): Extensible rule-based detection system
- **Performance Tests** (`performance_test.go`): Scalability and benchmarking
- **Error Handling Tests** (`error_handling_test.go`): Resilience validation

**Architecture Principles**:
- **Concurrent Processing**: Worker pool pattern for optimal performance
- **Extensible Rules**: Plugin-like architecture for adding new detection rules
- **Resilient**: Graceful handling of partial failures and malformed data
- **Performant**: 7x speedup with concurrent processing

### 5. Data Models (`internal/models/`)

**Responsibility**: Data structures and serialization

- **Stack Model**: Comprehensive CloudFormation stack representation
- **JSON Serialization**: Clean output structure with proper field naming
- **Type Safety**: Strong typing for all data structures

### 6. Output Formatting (`internal/output/`)

**Responsibility**: Multi-format output generation

**Formatters**:
- **JSON Formatter**: Structured output for programmatic consumption
- **TSV Formatter**: Tab-separated values for Excel/spreadsheet import
- **Factory Pattern**: Extensible for adding new output formats

**Features**:
- Proper escaping for special characters in TSV
- Consistent field ordering
- Empty data handling
- 91.7% test coverage

## Design Patterns

### 1. Interface-Driven Development

All AWS interactions use interfaces to enable comprehensive testing:

```go
type AWSClient interface {
    ListActiveStacks(ctx context.Context) ([]types.StackSummary, error)
    GetStackResources(ctx context.Context, stackName string) ([]types.StackResource, error)
    GetStackDetails(ctx context.Context, stackName string) (*types.Stack, error)
}
```

### 2. Factory Pattern

Used for creating formatters and AWS clients:
- Enables runtime selection of output formats
- Centralizes client creation logic
- Facilitates dependency injection

### 3. Worker Pool Pattern

Concurrent processing for optimal performance:
- Configurable worker count (default: 10)
- Graceful error handling per worker
- Context cancellation support

### 4. Rule Engine Pattern

Extensible detection logic:
- Plugin-like architecture for new rules
- Composable detection criteria
- Clear separation of concerns

## Performance Characteristics

### Concurrency Benefits

- **Sequential processing**: ~227ms for 10 stacks
- **5 workers**: ~54ms (4.2x improvement)
- **10 workers**: ~32ms (7.1x improvement)

### Scalability

- **100 stacks**: 47ms processing time
- **500 stacks**: 239ms processing time  
- **Memory efficiency**: 577 bytes per stack

### AWS API Efficiency

- **Rate limiting**: Prevents API throttling
- **Retry logic**: Exponential backoff for transient failures
- **Concurrent requests**: Optimized API usage patterns

## Error Handling Strategy

### Graceful Degradation

- **Partial failures**: Continue processing when some stacks fail
- **Missing data**: Populate available fields, skip incomplete data
- **API errors**: Clear error messages with context

### Resilience Features

- **Context cancellation**: Proper cleanup on timeouts
- **Null safety**: Safe handling of nil pointers
- **Resource cleanup**: Proper goroutine and channel management

## Security Considerations

### AWS Credentials

- **Profile-based authentication**: Uses standard AWS credential providers
- **No credential storage**: Relies on AWS SDK's secure credential handling
- **Minimal permissions**: Only requires CloudFormation read permissions

### Data Privacy

- **No data persistence**: Results only in memory and stdout
- **No logging of sensitive data**: Stack contents not logged
- **Read-only operations**: No modifications to AWS resources

## Testing Architecture

### Test Coverage Strategy

- **Unit tests**: 94.4% coverage for core detector
- **Integration tests**: Real AWS API testing when available
- **Performance tests**: Benchmarking and scalability validation  
- **Error handling tests**: Comprehensive failure scenario coverage

### Mock Strategy

- **Interface-based mocking**: Complete AWS API simulation
- **Configurable behavior**: Adjustable delays, errors, and responses
- **Deterministic testing**: Reliable CI/CD pipeline execution

## Future Extensibility

### Adding New Detection Rules

```go
type NewDetectionRule struct{}

func (r *NewDetectionRule) Name() string {
    return "NewRuleName"
}

func (r *NewDetectionRule) Check(resources []types.StackResource, details *types.Stack) (bool, string) {
    // Custom detection logic
    return matches, reason
}

// Register in rule engine
engine.AddRule(&NewDetectionRule{})
```

### Adding New Output Formats

```go
type XMLFormatter struct{}

func (f *XMLFormatter) Format(stacks []models.Stack) (string, error) {
    // XML formatting logic
}

// Register in factory
case "xml":
    return &XMLFormatter{}, nil
```

### Scaling Considerations

- **Horizontal scaling**: Can be deployed as multiple instances
- **Vertical scaling**: Adjustable worker pool size
- **Regional distribution**: Can run in multiple AWS regions simultaneously