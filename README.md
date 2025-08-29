# find_serverless_stacks

A CLI tool to identify CloudFormation stacks deployed by Serverless Framework.

## Overview

`find_serverless_stacks` detects CloudFormation stacks deployed by Serverless Framework in a specified AWS account and region, outputting results in JSON or TSV format.

## Features

- **High-precision detection**: Identifies Serverless Framework stacks by detecting `ServerlessDeploymentBucket` resources
- **Multiple output formats**: Supports JSON and TSV output formats
- **AWS profile support**: Works with multiple AWS accounts
- **Region targeting**: Search within specific regions
- **Detection reasoning**: Shows why each stack was identified as SLS3

## Installation

### Binary Download
```bash
# Download the latest release (Linux/macOS/Windows)
curl -L https://github.com/username/find-serverless-stacks/releases/latest/download/find_serverless_stacks_$(uname -s)_$(uname -m).tar.gz | tar xz
sudo mv find_serverless_stacks /usr/local/bin/
```

### Go Install
```bash
go install github.com/username/find-serverless-stacks/cmd/find_serverless_stacks@latest
```

### Homebrew (macOS)
```bash
brew install username/tap/find_serverless_stacks
```

## Usage

### Basic Usage
```bash
# Search in us-east-1 with default profile
find_serverless_stacks --region us-east-1

# Use specific AWS profile
find_serverless_stacks --profile my-aws-profile --region us-west-2

# Output in TSV format
find_serverless_stacks --profile prod --region ap-northeast-1 --output tsv
```

### Command Line Options

| Option | Short | Required | Description | 
|--------|-------|----------|-------------|
| `--profile` | `-p` | No | AWS profile name (default: default) |
| `--region` | `-r` | Yes | AWS region name |
| `--output` | `-o` | No | Output format: json, tsv (default: json) |
| `--help` | `-h` | No | Show help |
| `--version` | `-v` | No | Show version information |

## Output Format

### JSON Output Example
```json
{
    "stacks": [
        {
            "stackName": "my-api-dev",
            "stackId": "arn:aws:cloudformation:us-east-1:123456789012:stack/my-api-dev/abcd1234-efgh-5678-ijkl-mnopqrstuv",
            "region": "us-east-1",
            "createdAt": "2023-10-01T12:34:56Z",
            "updatedAt": "2023-10-02T12:34:56Z",
            "description": "My Serverless Framework stack",
            "stackTags": {
                "Owner": "team-a",
                "Environment": "development"
            },
            "reasons": [
                "Contains resource with logical ID 'ServerlessDeploymentBucket'"
            ]
        }
    ]
}
```

### TSV Output Example
```
stackName	stackId	region	createdAt	updatedAt	description	reasons
my-api-dev	arn:aws:cloudformation:us-east-1:123456789012:stack/my-api-dev/abcd1234	us-east-1	2023-10-01T12:34:56Z	2023-10-02T12:34:56Z	My Serverless Framework stack	Contains resource with logical ID 'ServerlessDeploymentBucket'
```

## Detection Logic

This tool identifies stacks deployed by Serverless Framework using the following method:

### ServerlessDeploymentBucket Resource Detection

When a stack contains an S3 bucket resource with the logical ID `ServerlessDeploymentBucket`, the stack is identified as deployed by Serverless Framework.

This bucket is automatically created by Serverless Framework for:
- Storing Lambda function deployment packages (ZIP files)
- Storing CloudFormation template files  
- Storing other deployment artifacts

**Detection Accuracy**: This method provides very high accuracy for identifying Serverless Framework stacks. `ServerlessDeploymentBucket` is a resource name specific to Serverless Framework, and the probability of other tools using the same logical ID is extremely low.

## Required AWS Permissions

To run this tool, the following IAM permissions are required:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "cloudformation:ListStacks",
                "cloudformation:DescribeStacks",
                "cloudformation:DescribeStackResources"
            ],
            "Resource": "*"
        }
    ]
}
```

## Troubleshooting

### Common Issues

#### Permission Error
```
Error: Access Denied - insufficient permissions
```
**Solution**: Ensure the required IAM permissions listed above are granted.

#### Invalid Region Error
```
Error: Invalid region 'invalid-region'
```
**Solution**: Specify a valid AWS region name. Available regions can be checked with `aws ec2 describe-regions`.

#### Profile Not Found Error
```
Error: Profile 'nonexistent' not found
```
**Solution**: Check available profiles with `aws configure list-profiles` and specify a valid profile name.

### Debug Mode
For detailed logging output:
```bash
export AWS_SDK_LOAD_CONFIG=1
export AWS_LOG_LEVEL=debug
find_serverless_stacks --region us-east-1
```

## Development

### Prerequisites
- Go 1.19 or later
- AWS CLI configured

### Quick Start
```bash
git clone https://github.com/username/find-serverless-stacks.git
cd find-serverless-stacks
make build
```

### Available Make Targets

#### Build and Clean
```bash
make build         # Build the binary
make build-cross   # Cross-compile for multiple platforms
make clean         # Clean build artifacts
```

#### Testing
```bash
make test              # Run unit tests
make test-verbose      # Run unit tests with verbose output
make test-coverage     # Run tests with coverage report
make test-integration  # Run integration tests (requires AWS credentials)
```

#### Development Tools
```bash
make fmt           # Format code
make lint          # Run linter (requires golangci-lint)
make vet           # Run go vet
make run-dev       # Run with sample development parameters
```

#### Dependencies and Installation
```bash
make deps-update   # Update and tidy dependencies
make deps-list     # List all dependencies
make install       # Install binary to GOPATH/bin
make uninstall     # Remove binary from GOPATH/bin
```

#### Other Tasks
```bash
make docker-build  # Build Docker image
make help          # Show all available targets
```

### Manual Build (Alternative)
If you prefer not to use Makefile:
```bash
go build -o find_serverless_stacks ./cmd/find_serverless_stacks
```

## License

MIT License - see [LICENSE](LICENSE) file for details

## Contributing

Pull requests and issue reports are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## Related Documentation

- [Detection Logic Details](docs/how-to-checking-serverless.md) (Japanese)
- [Implementation Plan](docs/implementation-plan.md) (Japanese)
- [Testable Implementation Examples](docs/example-testable-impl.md) (Japanese)
