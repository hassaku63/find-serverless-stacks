# Serverless Framework で構築されたスタックの判定方法

## 概要

Serverless Framework でデプロイされた CloudFormation スタックを推定するための判定ロジックについて説明する。

## 判定方法

確実な判定方法は存在しないが、確率の高い判定方法として存在する。

### ServerlessDeploymentBucket リソースの存在

#### 概要

Stack テンプレートに論理 ID `ServerlessDeploymentBucket` リソースを持つスタックは、Serverless Framework によってデプロイされた可能性が高い。

将来的に他のロジックも追加する可能性もあるが、当面はこの方法のみを採用する。

#### ServerlessDeploymentBucket とは
ServerlessDeploymentBucket は Serverless Framework が自動的に作成する S3 バケットで、以下の用途で使用される：
- Lambda 関数のデプロイメントパッケージ（ZIP ファイル）の保存
- CloudFormation テンプレートファイルの保存
- その他のデプロイメントアーティファクトの保存

#### 物理リソース名の特徴
この論理 ID に対応する物理リソース名は以下のパターンを持つ：
```
{service}-{stage}-serverlessdeploymentbucket-{randomString}
```

例：
- `my-api-dev-serverlessdeploymentbucket-abc123def456`
- `user-service-prod-serverlessdeploymentbucket-xyz789ghj012`

#### 実装方法
CloudFormation の DescribeStackResources API を使用して、各スタックのリソース一覧を取得し、論理 ID が `ServerlessDeploymentBucket` のリソースが存在するかチェックする。

```go
func hasServerlessDeploymentBucket(resources []types.StackResource) bool {
    for _, resource := range resources {
        if resource.LogicalResourceId != nil && 
           *resource.LogicalResourceId == "ServerlessDeploymentBucket" &&
           resource.ResourceType != nil &&
           *resource.ResourceType == "AWS::S3::Bucket" {
            return true
        }
    }
    return false
}
```

#### API 呼び出しパターン
1. `ListStacks` API でリージョン内のすべてのスタックを取得
2. 各スタックに対して `DescribeStackResources` API を呼び出し
3. 論理 ID `ServerlessDeploymentBucket` のリソースが存在するスタックを抽出

#### 判定精度
この方法は非常に高い精度で Serverless Framework のスタックを判定できる理由：
- `ServerlessDeploymentBucket` は Serverless Framework 特有のリソース名
- 他のツールやフレームワークがこの論理 ID を使用する可能性は極めて低い
- Serverless Framework でデプロイされたほぼすべてのスタックにこのリソースが含まれる

## 判定結果の提示方法

`$.stacks[].reasons` に、判定理由を配列で格納する。
