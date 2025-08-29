# find_sls3_stacks CLI 実装計画

## プロジェクト概要

Serverless Framework v3 でデプロイされた CloudFormation スタックを特定する Go CLI ツール `find_sls3_stacks` の実装計画。

## アーキテクチャ設計

### ディレクトリ構造
```
find-sls3-stacks/
├── cmd/
│   └── find_sls3_stacks/
│       └── main.go              # CLI エントリーポイント
├── internal/
│   ├── config/
│   │   └── config.go            # 設定管理
│   ├── aws/
│   │   ├── client.go            # AWS クライアント抽象化
│   │   ├── cloudformation.go    # CloudFormation API ラッパー
│   │   └── types.go             # AWS 関連の型定義
│   ├── detector/
│   │   ├── detector.go          # スタック判定ロジック
│   │   └── rules.go             # 判定ルール実装
│   ├── output/
│   │   ├── formatter.go         # 出力フォーマット抽象化
│   │   ├── json.go              # JSON 出力
│   │   └── tsv.go               # TSV 出力
│   └── models/
│       └── stack.go             # データモデル定義
├── pkg/
│   └── cli/
│       └── flags.go             # CLI フラグ定義
├── go.mod
├── go.sum
└── README.md
```

## 実装フェーズ

### Phase 1: プロジェクト基盤構築
- [ ] Go モジュールの初期化
- [ ] 基本的なディレクトリ構造の作成
- [ ] 依存関係の設定 (AWS SDK for Go v2, CLI ライブラリ)
- [ ] 基本的な CI/CD パイプラインの設定

### Phase 2: AWS 統合層の実装
- [ ] AWS クライアントの抽象化インタフェースの定義
- [ ] CloudFormation API ラッパーの実装
- [ ] AWS 認証情報とプロファイル管理
- [ ] レート制限とエラーハンドリング

### Phase 3: コア判定ロジックの実装
- [ ] ServerlessDeploymentBucket リソース検出
- [ ] スタック情報取得とフィルタリング
- [ ] 判定結果の理由記録
- [ ] 並行処理による性能最適化

### Phase 4: CLI インターフェースの実装
- [ ] コマンドライン引数の解析
- [ ] 出力フォーマット選択機能
- [ ] エラーハンドリングとユーザーフレンドリーなメッセージ
- [ ] ヘルプとドキュメント

### Phase 5: 出力機能の実装
- [ ] JSON 出力フォーマッター
- [ ] TSV 出力フォーマッター  
- [ ] 出力結果の検証

### Phase 6: テストとドキュメント
- [ ] ユニットテストの実装
- [ ] インテグレーションテストの実装
- [ ] パフォーマンステスト
- [ ] README とユーザーマニュアルの作成

## 技術仕様詳細

### データモデル

#### Stack 構造体
```go
type Stack struct {
    StackName    string            `json:"stackName"`
    StackID      string            `json:"stackId"`
    Region       string            `json:"region"`
    CreatedAt    time.Time         `json:"createdAt"`
    UpdatedAt    time.Time         `json:"updatedAt"`
    Description  string            `json:"description"`
    StackTags    map[string]string `json:"stackTags"`
    Reasons      []string          `json:"reasons"`
}
```

#### CLI 設定
```go
type Config struct {
    Profile    string
    Region     string
    OutputFormat string // json, tsv
}
```

### AWS API 使用方法

#### 必要な API 権限
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

#### API 呼び出しシーケンス
1. **ListStacks** - アクティブなスタック一覧を取得
2. **DescribeStackResources** - 各スタックのリソース詳細を取得
3. **判定ロジック実行** - ServerlessDeploymentBucket の存在確認
4. **結果の整形と出力**

### 判定ロジック実装

#### 主要判定ルール
```go
type DetectionRule interface {
    Check(stack *cloudformation.Stack, resources []cloudformation.StackResource) (bool, string)
}

type ServerlessDeploymentBucketRule struct{}

func (r *ServerlessDeploymentBucketRule) Check(stack *cloudformation.Stack, resources []cloudformation.StackResource) (bool, string) {
    for _, resource := range resources {
        if aws.StringValue(resource.LogicalResourceId) == "ServerlessDeploymentBucket" &&
           aws.StringValue(resource.ResourceType) == "AWS::S3::Bucket" {
            return true, "Contains resource with logical ID 'ServerlessDeploymentBucket'"
        }
    }
    return false, ""
}
```

### エラーハンドリング戦略

#### API エラー処理
- **アクセス拒否**: 権限不足の明確なメッセージ
- **レート制限**: 指数バックオフによるリトライ
- **ネットワークエラー**: 接続問題の診断情報
- **不正なリージョン**: 利用可能なリージョン一覧の表示

#### データ検証
- スタック状態の確認（CREATE_COMPLETE, UPDATE_COMPLETE のみ対象）
- リソース情報の欠損チェック
- タイムアウト設定による長時間実行の防止

### 性能考慮事項

#### 並行処理
- スタック単位での並行処理（Goroutine プール使用）
- API レート制限を考慮した同時実行数制御
- メモリ使用量の監視と制限

#### キャッシング
- 同一セッション内でのスタック情報キャッシュ
- リージョンごとの結果キャッシュ

## テスト戦略

### ユニットテスト
- 各判定ルールの単体テスト
- AWS API モックを使用したテスト（docs/example-testable-impl.md のパターンを適用）
- 出力フォーマッターのテスト
- エラーハンドリングのテスト

### インテグレーションテスト
- 実際の AWS 環境での E2E テスト
- 複数リージョンでのテスト
- 大量スタック環境でのパフォーマンステスト

### テストデータ
```go
type MockCloudFormationAPI struct {
    stacks    []types.Stack
    resources map[string][]types.StackResource
}
```

## セキュリティ考慮事項

- AWS 認証情報の安全な取り扱い
- 最小権限の原則に基づく IAM ポリシー
- ログ出力時の機密情報マスキング
- 入力値の検証（リージョン名、プロファイル名等）

## デプロイメント計画

### リリース戦略
1. **アルファ版**: 基本機能のみ、内部テスト用
2. **ベータ版**: 全機能実装、限定ユーザーでのテスト
3. **正式版**: 本格運用開始

### 配布方法
- GitHub Releases でのバイナリ配布
- 主要 OS（Linux、macOS、Windows）向けクロスコンパイル
- Homebrew Formula の提供（macOS）
- Docker イメージの提供

## 運用・保守計画

### 監視とロギング
- 実行時間の計測とログ出力
- API エラー率の監視
- パフォーマンスメトリクスの収集

### アップデート戦略
- セマンティックバージョニングの採用
- 後方互換性の維持
- アップデート通知機能
