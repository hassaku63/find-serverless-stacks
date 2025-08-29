# 実装ガイド

## 開発手法の採用

このプロジェクトでは **t-wada 氏が提唱するテスト駆動開発（TDD）アプローチ**を採用します。以下の理由により、このアプローチが本プロジェクトに適しています：

1. **外部依存の適切な分離**: 外部システム（AWS API 等）との統合において、テストファーストアプローチにより依存関係を適切に分離できる
2. **高品質なエラーハンドリング**: TDD により例外ケースを含む包括的なテストを最初から構築できる
3. **継続的なリファクタリング**: テストがある状態で設計を改善し続けることができる
4. **設計品質の向上**: テスタブルなコードは自然に疎結合で責任が分離された設計になる
5. **仕様の明確化**: テストコードが実行可能な仕様書として機能する

### TDD の基本サイクル

```
Red → Green → Refactor
```

1. **Red**: まず失敗するテストを書く
2. **Green**: テストが通る最小限のコードを書く
3. **Refactor**: テストが通る状態を保ちながら設計を改善する

## 実装ガイドライン

### 1. テスタビリティの確保

#### インターフェース駆動開発
外部依存（AWS SDK 等）との統合部分は必ずインターフェースを通して実装する：

```go
// 良い例: テスト可能な設計
type CloudFormationAPI interface {
    ListStacks(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error)
    DescribeStackResources(ctx context.Context, params *cloudformation.DescribeStackResourcesInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStackResourcesOutput, error)
}

func NewDetector(client CloudFormationAPI) *Detector {
    return &Detector{client: client}
}
```

#### モックの活用
テスト時は `docs/example-testable-impl.md` のパターンを参考にしたモックを使用：

```go
type mockCloudFormationAPI func(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error)

func (m mockCloudFormationAPI) ListStacks(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error) {
    return m(ctx, params, optFns...)
}
```

### 2. TDD による実装フロー

#### Phase 3: コア判定ロジックの実装例

**Step 1: 失敗するテストを書く（Red）**
```go
func TestDetectServerlessStacks_WithServerlessDeploymentBucket(t *testing.T) {
    // Setup
    mockClient := mockCloudFormationAPI(func(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error) {
        return &cloudformation.ListStacksOutput{
            StackSummaries: []types.StackSummary{
                {
                    StackName: aws.String("my-api-dev"),
                    StackId:   aws.String("arn:aws:cloudformation:us-east-1:123456789012:stack/my-api-dev/abc123"),
                },
            },
        }, nil
    })
    
    detector := NewDetector(mockClient)
    
    // Execute
    stacks, err := detector.DetectServerlessStacks(context.Background())
    
    // Assert
    assert.NoError(t, err)
    assert.Len(t, stacks, 1)
    assert.Equal(t, "my-api-dev", stacks[0].StackName)
    assert.Contains(t, stacks[0].Reasons, "Contains resource with logical ID 'ServerlessDeploymentBucket'")
}
```

**Step 2: テストが通る最小限のコードを書く（Green）**
```go
func (d *Detector) DetectServerlessStacks(ctx context.Context) ([]models.Stack, error) {
    // まず最小限の実装でテストを通す
    return []models.Stack{
        {
            StackName: "my-api-dev",
            Reasons:   []string{"Contains resource with logical ID 'ServerlessDeploymentBucket'"},
        },
    }, nil
}
```

**Step 3: リファクタリング（Refactor）**
```go
func (d *Detector) DetectServerlessStacks(ctx context.Context) ([]models.Stack, error) {
    // 実際の AWS API 呼び出しを追加
    summaries, err := d.client.ListActiveStacks(ctx)
    if err != nil {
        return nil, err
    }
    
    var serverlessStacks []models.Stack
    for _, summary := range summaries {
        if d.isServerlessStack(ctx, *summary.StackName) {
            stack := d.convertToModel(summary)
            serverlessStacks = append(serverlessStacks, stack)
        }
    }
    
    return serverlessStacks, nil
}
```

### 3. テスト分類と実装戦略

#### ユニットテスト
- **対象**: ビジネスロジック、判定アルゴリズム
- **実装**: モックを使用して外部依存を排除
- **実行**: `make test`

```go
func TestHasServerlessDeploymentBucket(t *testing.T) {
    tests := []struct {
        name      string
        resources []types.StackResource
        expected  bool
    }{
        {
            name: "ServerlessDeploymentBucket exists",
            resources: []types.StackResource{
                {
                    LogicalResourceId: aws.String("ServerlessDeploymentBucket"),
                    ResourceType:     aws.String("AWS::S3::Bucket"),
                },
            },
            expected: true,
        },
        {
            name: "No ServerlessDeploymentBucket",
            resources: []types.StackResource{
                {
                    LogicalResourceId: aws.String("MyBucket"),
                    ResourceType:     aws.String("AWS::S3::Bucket"),
                },
            },
            expected: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := hasServerlessDeploymentBucket(tt.resources)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

#### インテグレーションテスト
- **対象**: AWS API との統合
- **実装**: 実際の AWS 環境を使用
- **実行**: `make test-integration`

```go
//go:build integration

func TestDetectServerlessStacks_Integration(t *testing.T) {
    // 実際の AWS 認証情報が必要
    ctx := context.Background()
    
    authConfig := aws.AuthConfig{
        Profile: "default",
        Region:  "us-east-1",
    }
    
    client, err := aws.NewCloudFormationClient(ctx, authConfig)
    require.NoError(t, err)
    
    detector := detector.NewDetector(client)
    stacks, err := detector.DetectServerlessStacks(ctx)
    
    assert.NoError(t, err)
    // 実際の環境に依存する検証を行う
}
```

### 4. エラーハンドリング戦略

#### エラー種別の定義
各エラー種別に対してテストケースを作成：

```go
func TestDetectServerlessStacks_PermissionError(t *testing.T) {
    mockClient := mockCloudFormationAPI(func(ctx context.Context, params *cloudformation.ListStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.ListStacksOutput, error) {
        return nil, &aws.Error{
            Type:    aws.ErrorTypePermission,
            Message: "Access denied",
        }
    })
    
    detector := NewDetector(mockClient)
    _, err := detector.DetectServerlessStacks(context.Background())
    
    var awsErr *aws.Error
    assert.True(t, errors.As(err, &awsErr))
    assert.Equal(t, aws.ErrorTypePermission, awsErr.Type)
}
```

### 5. パフォーマンステスト

#### 並行処理のテスト
```go
func TestDetectServerlessStacks_Concurrency(t *testing.T) {
    // 大量のスタックを含むモックレスポンス
    mockClient := createMockWithManyStacks(1000)
    
    detector := NewDetector(mockClient)
    
    start := time.Now()
    stacks, err := detector.DetectServerlessStacks(context.Background())
    duration := time.Since(start)
    
    assert.NoError(t, err)
    assert.NotEmpty(t, stacks)
    assert.Less(t, duration, 30*time.Second, "Detection should complete within 30 seconds")
}
```

### 6. コードカバレッジ目標

- **最低要求**: 80% 以上
- **推奨**: 90% 以上
- **測定**: `make test-coverage`

```bash
# カバレッジレポートの確認
make test-coverage
open coverage.html
```

### 7. CI/CD での品質ゲート

#### 必須チェック項目
1. **全テストの成功**: `make test`
2. **リンター通過**: `make lint`
3. **コードフォーマット**: `make fmt`
4. **コードベット**: `make vet`
5. **カバレッジ閾値**: 80% 以上

#### GitHub Actions での実装例
```yaml
- name: Run tests
  run: make test
  
- name: Check coverage
  run: |
    make test-coverage
    go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//' > coverage.txt
    COVERAGE=$(cat coverage.txt)
    if (( $(echo "$COVERAGE < 80" | bc -l) )); then
      echo "Coverage $COVERAGE% is below 80%"
      exit 1
    fi
```

### 8. 実装の優先順位

#### Phase 3 実装順序
1. **ServerlessDeploymentBucket 判定ロジック** (最重要)
   - テストケース: 存在する場合、存在しない場合
   - エラーケース: リソース取得失敗

2. **スタック情報収集**
   - テストケース: 正常なスタック、削除されたスタック
   - エラーケース: アクセス権限不足

3. **結果の整形と出力**
   - テストケース: JSON、TSV 各形式
   - エラーケース: 出力先への書き込み失敗

4. **並行処理の最適化**
   - パフォーマンステスト
   - メモリ使用量の監視

### 9. リファクタリングガイドライン

#### 継続的改善のポイント
- **小さな変更**: 一度に一つの改善に集中
- **テストファースト**: リファクタリング前後でテストが通ることを確認
- **可読性優先**: パフォーマンスより可読性を重視（ボトルネックが特定されるまで）

#### コードレビューのチェックポイント
- [ ] テストが適切に書かれているか
- [ ] エラーハンドリングが包括的か
- [ ] インターフェースが適切に使用されているか
- [ ] ドキュメントが更新されているか

## まとめ

t-wada 氏の TDD アプローチを採用することで、以下を実現します：

1. **高品質**: テストファーストにより品質を最初から確保
2. **保守性**: リファクタリングを安全に継続
3. **信頼性**: 外部システムとの統合や例外処理の網羅的な検証
4. **開発効率**: 適切な設計により長期的な開発効率を向上
5. **ドキュメント化**: テストコードが生きた仕様書として機能

このガイドラインに従って実装を進めることで、堅牢で保守性の高い CLI ツールを構築できます。
