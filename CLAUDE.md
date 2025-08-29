# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Serverless Framework v3 でデプロイされた CloudFormation Stack を推測する。

- CLI ツール `find_sls3_stacks` として実行可能
- AWS プロファイルと、リージョンを指定して実行可能。一度の実行で特定の AWS アカウントの1つのリージョンを対象とする
- 結果は stdout に出力。json, tsv 形式をサポート

### Example Usage

```bash
$ find_sls3_stacks --profile my-aws-profile --region us-east-1 --output json
{
    "stacks": [
        {
            "stackName": "my-stack",
            "stackId": "arn:aws:cloudformation:us-east-1:123456789012:stack/my-stack/abcd1234-efgh-5678-ijkl-mnopqrstuv",
            "region": "us-east-1",
            "createdAt": "2023-10-01T12:34:56Z",
            "updatedAt": "2023-10-02T12:34:56Z",
            "description": "My Serverless Framework v3 stack",
            "stackTags": {
                "Owner": "team-a",
            },
            "reasons": [
                "Contains resource with logical ID 'ServerlessDeploymentBucket'"
            ]
        },
        {
            "stackName": "another-stack",
            "stackId": "arn:aws:cloudformation:us-east-1:123456789012:stack/another-stack/wxyz9876-vuts-5432-rqpo-nmlkjihgfe",
            "region": "us-east-1",
            "createdAt": "2023-10-03T12:34:56Z",
            "updatedAt": "2023-10-04T12:34:56Z",
            "description": "Another Serverless Framework v3 stack",
            "stackTags": {
                "Owner": "team-b",
            },
            "reasons": [
                "Contains resource with logical ID 'ServerlessDeploymentBucket'"
            ]
        }
    ]
}
```

## Development Notes

This project is in early development stage. The primary goal is to create a Go CLI tool that identifies CloudFormation stacks deployed by Serverless Framework v3.

### Architecture Guidelines

When implementing this CLI tool, follow the testable architecture patterns documented in `docs/example-testable-impl.md`:
- Use interface abstractions for AWS service clients to enable mocking in tests
- Implement dependency injection patterns for better testability
- Follow the AWS SDK for Go v2 testing patterns for mocking client operations and paginators

### CLI Requirements

The tool should support:
- `--profile` flag for AWS profile specification
- `--region` flag for AWS region targeting
- `--output` flag supporting `json` and `tsv` formats
- Single region processing per execution

### Output Format

JSON output should include:
- `stacks` array with `stackName`, `stackId`, and `region` fields
- Stack identification should focus on Serverless Framework v3 deployed resources
