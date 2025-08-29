# 機能: AssumeRole サポート

## 概要

この文書では、`find_sls3_stacks` に AWS AssumeRole サポートを追加するための設計と実装計画を説明します。この機能により、クロスアカウント スタック検出と、企業 AWS 環境で一般的に使用されるロールベースのアクセスパターンをサポートできます。

## 使用例

### 1. クロスアカウント スタック検出
```bash
# AssumeRole を使用して異なるアカウントのスタックを検出
find_sls3_stacks --region us-east-1 \
  --assume-role arn:aws:iam::123456789012:role/CrossAccountReadRole \
  --output json
```

### 2. マルチアカウント組織スキャン
```bash
# 組織レベルロールで複数アカウントをスキャン
find_sls3_stacks --region us-east-1 \
  --assume-role arn:aws:iam::123456789012:role/OrganizationReaderRole \
  --external-id "unique-external-id" \
  --session-name "sls3-discovery-session"
```

### 3. 一時的な昇格権限
```bash
# 包括的スキャンのために一時的な昇格権限を使用
find_sls3_stacks --region us-east-1 \
  --profile base-profile \
  --assume-role arn:aws:iam::123456789012:role/ElevatedReadRole \
  --duration 3600
```

## CLI インターフェース設計

### 新しいコマンドライン引数

```bash
find_sls3_stacks [フラグ]

既存のフラグ:
  -h, --help             find_sls3_stacks のヘルプ
  -o, --output string    出力フォーマット (json, tsv) (default "json")
  -p, --profile string   AWS プロファイル名 (default "default")
  -r, --region string    AWS リージョン名 (必須)

新しい AssumeRole フラグ:
      --assume-role string        引き受ける IAM ロールの ARN
      --external-id string        AssumeRole 用の External ID (ロールで必要な場合はオプション)
      --session-name string       引き受けたロールセッション用のセッション名 (default "find-sls3-stacks-session")
      --duration int              セッション期間（秒）(900-43200, default 3600)
      --mfa-serial string         MFA デバイスのシリアル番号 (ロールで MFA が必要な場合はオプション)
      --mfa-token string          MFA トークンコード (オプション、提供されない場合はインタラクティブに入力要求)
      --non-interactive           インタラクティブプロンプトを無効化 (自動化用)
```

### CLI 使用例

#### 基本的な AssumeRole
```bash
find_sls3_stacks --region us-east-1 \
  --assume-role arn:aws:iam::123456789012:role/ReadOnlyRole
```

#### External ID を使った AssumeRole
```bash
find_sls3_stacks --region us-east-1 \
  --assume-role arn:aws:iam::123456789012:role/ThirdPartyRole \
  --external-id "company-unique-id-2023"
```

#### MFA を使った AssumeRole - 明示的トークン (自動化対応)
```bash
find_sls3_stacks --region us-east-1 \
  --assume-role arn:aws:iam::123456789012:role/MFAProtectedRole \
  --mfa-serial arn:aws:iam::111122223333:mfa/user@example.com \
  --mfa-token 123456
```

#### MFA を使った AssumeRole - インタラクティブ入力 (人間が使いやすい)
```bash
find_sls3_stacks --region us-east-1 \
  --assume-role arn:aws:iam::123456789012:role/MFAProtectedRole \
  --mfa-serial arn:aws:iam::111122223333:mfa/user@example.com

# CLI プロンプト:
# MFA トークンを入力してください (arn:aws:iam::111122223333:mfa/user@example.com): [非表示入力]
```

#### MFA なしの AssumeRole (最も一般的なケース)
```bash
find_sls3_stacks --region us-east-1 \
  --assume-role arn:aws:iam::123456789012:role/StandardReadRole
```

#### CI/CD での AssumeRole - 非インタラクティブモード
```bash
find_sls3_stacks --region us-east-1 \
  --assume-role arn:aws:iam::123456789012:role/MFAProtectedRole \
  --mfa-serial arn:aws:iam::111122223333:mfa/user@example.com \
  --non-interactive

# エラー: MFA トークンが必要ですが提供されておらず、非インタラクティブモードで実行中です
# --mfa-token を使用するか、FIND_SLS3_MFA_TOKEN 環境変数を設定してください
```

#### プロファイルと AssumeRole の組み合わせ
```bash
find_sls3_stacks --region us-east-1 \
  --profile source-account-profile \
  --assume-role arn:aws:iam::123456789012:role/CrossAccountRole \
  --session-name "audit-session" \
  --duration 7200
```

#### 自動化用の環境変数サポート
```bash
# 環境変数で MFA トークンを設定
export FIND_SLS3_MFA_TOKEN=123456
find_sls3_stacks --region us-east-1 \
  --assume-role arn:aws:iam::123456789012:role/MFAProtectedRole \
  --mfa-serial arn:aws:iam::111122223333:mfa/user@example.com

# トークンは環境変数から読み取られ、インタラクティブプロンプトはありません
```

## 内部インターフェース設計

### 1. 強化された設定構造

```go
// internal/config/config.go
package config

type Config struct {
    // 既存のフィールド
    Profile      string
    Region       string
    OutputFormat string
    
    // 新しい AssumeRole フィールド
    AssumeRole   *AssumeRoleConfig
}

type AssumeRoleConfig struct {
    RoleARN        string        `json:"roleArn"`
    ExternalID     string        `json:"externalId,omitempty"`
    SessionName    string        `json:"sessionName"`
    Duration       int32         `json:"duration"`
    MFASerial      string        `json:"mfaSerial,omitempty"`
    MFAToken       string        `json:"mfaToken,omitempty"`
    NonInteractive bool          `json:"nonInteractive,omitempty"`
}

// バリデーションとインタラクティブ入力メソッド
func (c *Config) ValidateAssumeRole() error
func (arc *AssumeRoleConfig) Validate() error
func (arc *AssumeRoleConfig) RequiresMFA() bool
func (arc *AssumeRoleConfig) ResolveMFAToken() error  // 新規: インタラクティブ入力を処理
func (arc *AssumeRoleConfig) ValidateMFARequirements() error
```

### 2. 強化された AWS 認証

```go
// internal/aws/auth.go
package aws

import (
    "github.com/aws/aws-sdk-go-v2/service/sts"
)

type AuthConfig struct {
    // 既存のフィールド
    Profile string
    Region  string
    
    // 新しい AssumeRole フィールド
    AssumeRole *AssumeRoleCredentials
}

type AssumeRoleCredentials struct {
    RoleARN        string
    ExternalID     string
    SessionName    string
    Duration       int32
    MFASerial      string
    MFAToken       string
    NonInteractive bool
}

// 強化されたクライアントファクトリー
func CreateClientWithAssumeRole(ctx context.Context, auth AuthConfig) (*Client, error)

// AssumeRole 固有の関数
func (a *AuthConfig) AssumeRole(ctx context.Context, stsClient *sts.Client) (*types.Credentials, error)
func buildAssumeRoleInput(roleConfig *AssumeRoleCredentials) *sts.AssumeRoleInput
func validateAssumeRolePermissions(ctx context.Context, client CloudFormationAPI) error

// インタラクティブ MFA 入力関数
func (arc *AssumeRoleCredentials) ResolveMFAToken() error
func isInteractiveTerminal() bool
func promptForMFAToken(serialNumber string) (string, error)
func readHiddenInput() (string, error)
```

### 3. 強化された AWS クライアント ファクトリー

```go
// internal/aws/factory.go
package aws

func CreateClient(ctx context.Context, auth AuthConfig) (*Client, error) {
    // ベース設定を読み込み
    cfg, err := loadBaseConfig(ctx, auth)
    if err != nil {
        return nil, err
    }
    
    // AssumeRole が指定されている場合は適用
    if auth.AssumeRole != nil {
        cfg, err = applyAssumeRole(ctx, cfg, auth.AssumeRole)
        if err != nil {
            return nil, err
        }
    }
    
    // CloudFormation クライアントを作成
    cfClient := cloudformation.NewFromConfig(cfg)
    return NewClient(cfClient, auth.Region), nil
}

func loadBaseConfig(ctx context.Context, auth AuthConfig) (aws.Config, error)
func applyAssumeRole(ctx context.Context, cfg aws.Config, roleConfig *AssumeRoleCredentials) (aws.Config, error)

// インタラクティブ MFA トークン解決の実装例
func (arc *AssumeRoleCredentials) ResolveMFAToken() error {
    if arc.MFASerial == "" {
        return nil // MFA は不要
    }
    
    if arc.MFAToken != "" {
        return nil // トークンは既に提供済み
    }
    
    // 環境変数をチェック
    if envToken := os.Getenv("FIND_SLS3_MFA_TOKEN"); envToken != "" {
        arc.MFAToken = envToken
        return nil
    }
    
    if arc.NonInteractive || !isInteractiveTerminal() {
        return errors.New("MFA トークンが必要ですが提供されていません。--mfa-token を使用するか FIND_SLS3_MFA_TOKEN 環境変数を設定してください")
    }
    
    // インタラクティブプロンプト
    token, err := promptForMFAToken(arc.MFASerial)
    if err != nil {
        return fmt.Errorf("MFA トークンの読み取りに失敗しました: %w", err)
    }
    
    arc.MFAToken = strings.TrimSpace(token)
    if arc.MFAToken == "" {
        return errors.New("空の MFA トークンが提供されました")
    }
    
    return nil
}
```

### 4. 強化されたエラーハンドリング

```go
// internal/aws/errors.go
package aws

type AssumeRoleError struct {
    RoleARN   string
    Reason    string
    Cause     error
}

func (e *AssumeRoleError) Error() string
func (e *AssumeRoleError) Unwrap() error

// 一般的な AssumeRole エラーパターン
func IsAssumeRoleError(err error) bool
func IsAccessDeniedError(err error) bool
func IsMFARequiredError(err error) bool
func IsExternalIDError(err error) bool
```

## 実装計画

### Phase 1: 基本 AssumeRole サポート (1-2 日)

**目標**: MFA・External ID なしの基本 AssumeRole を完全サポート

#### 1.1 CLI インターフェース
- [ ] 基本 AssumeRole フラグ (`--assume-role`) の追加
- [ ] セッション管理フラグ (`--session-name`, `--duration`) の追加
- [ ] CLI フラグ解析とバリデーション
- [ ] ヘルプテキストの更新

**完了時に利用可能な機能:**
```bash
find_sls3_stacks --region us-east-1 \
  --assume-role arn:aws:iam::123456789012:role/ReadRole \
  --session-name "sls3-discovery" \
  --duration 7200
```

#### 1.2 内部アーキテクチャ
- [ ] `Config` 構造体を基本 `AssumeRoleConfig` で拡張
- [ ] `AuthConfig` に AssumeRole 認証情報を追加
- [ ] STS AssumeRole 統合を実装
- [ ] `CreateClient` を AssumeRole フローに対応

#### 1.3 テストとバリデーション
- [ ] AssumeRole 設定のユニットテスト
- [ ] STS 統合のモックベーステスト
- [ ] 基本エラーハンドリング（アクセス拒否、無効ロール）
- [ ] CLI 引数解析テスト

### Phase 2: External ID サポート (0.5 日)

**目標**: サードパーティアクセスパターンをサポート

#### 2.1 External ID 機能追加
- [ ] `--external-id` フラグの追加
- [ ] STS AssumeRole リクエストに External ID パラメータを追加
- [ ] External ID 関連のエラーハンドリング
- [ ] バリデーションロジック

**完了時に利用可能な機能:**
```bash
find_sls3_stacks --region us-east-1 \
  --assume-role arn:aws:iam::123456789012:role/ThirdPartyRole \
  --external-id "company-unique-id-2023"
```

#### 2.2 テスト
- [ ] External ID 使用パターンのテスト
- [ ] Invalid External ID エラーハンドリングテスト

### Phase 3: MFA サポート (1.5-2 日)

**目標**: MFA 保護ロールへの完全対応

#### 3.1 MFA フラグとロジック
- [ ] MFA 関連フラグ (`--mfa-serial`, `--mfa-token`, `--non-interactive`) の追加
- [ ] ハイブリッド MFA トークン解決ロジック:
  - 明示的 `--mfa-token` 指定
  - インタラクティブプロンプト（デフォルト）
  - 環境変数サポート (`FIND_SLS3_MFA_TOKEN`)
- [ ] `--non-interactive` フラグ動作の実装

**完了時に利用可能な機能:**
```bash
# 明示的トークン
find_sls3_stacks --assume-role ROLE --mfa-serial SERIAL --mfa-token 123456

# インタラクティブ入力
find_sls3_stacks --assume-role ROLE --mfa-serial SERIAL

# 非インタラクティブモード
find_sls3_stacks --assume-role ROLE --mfa-serial SERIAL --non-interactive
```

#### 3.2 インタラクティブ入力実装
- [ ] ターミナル検出 (`isInteractiveTerminal()`)
- [ ] `golang.org/x/term` を使用した非表示入力
- [ ] クロスプラットフォーム対応
- [ ] MFA トークン形式バリデーション

#### 3.3 包括的エラーハンドリング
- [ ] MFA 関連エラータイプ
- [ ] ユーザーフレンドリーなエラーメッセージ
- [ ] トラブルシューティングガイダンス

#### 3.4 テスト
- [ ] MFA ワークフローのユニットテスト
- [ ] インタラクティブ入力のモックテスト
- [ ] 統合テスト（実際の AWS MFA ロール）

### Phase 4: 最終仕上げとドキュメント (0.5 日)

#### 4.1 統合テストと品質保証
- [ ] エンドツーエンド AssumeRole ワークフローテスト
- [ ] 全パターンの組み合わせテスト
- [ ] パフォーマンス影響評価
- [ ] セキュリティレビュー

#### 4.2 ドキュメント完成
- [ ] README に使用例を追加
- [ ] トラブルシューティングガイドの完成
- [ ] CLI ヘルプの最終チェック

## 段階的な機能追加

| Phase | 基本AssumeRole | External ID | MFA |
|-------|---------------|------------|-----|
| 1     | ✅            | ❌          | ❌   |
| 2     | ✅            | ✅          | ❌   |
| 3     | ✅            | ✅          | ✅   |

## 必要な IAM 権限

### ベースロール権限 (クロスアカウントロール)
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

### AssumeRole 用の信頼ポリシー
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::SOURCE-ACCOUNT:user/USERNAME"
            },
            "Action": "sts:AssumeRole",
            "Condition": {
                "StringEquals": {
                    "sts:ExternalId": "unique-external-id"
                },
                "Bool": {
                    "aws:MultiFactorAuthPresent": "true"
                }
            }
        }
    ]
}
```

### ソースアカウント権限 (ユーザー/ロール)
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "sts:AssumeRole",
            "Resource": "arn:aws:iam::TARGET-ACCOUNT:role/ROLE-NAME"
        }
    ]
}
```

## セキュリティ考慮事項

### 1. 認証情報セキュリティ
- **認証情報の保存なし**: MFA トークンと一時的認証情報は保存されません
- **セッション期間制限**: 合理的なセッション期間制限を強制
- **安全な認証情報受け渡し**: 可能な場合は安全な環境変数を使用

### 2. アクセス制御
- **最小権限の原則**: 必要な権限のみを要求
- **External ID バリデーション**: サードパーティアクセス用の適切な External ID 処理
- **MFA 強制**: MFA 保護ロールのサポート

### 3. 監査証跡
- **セッション命名**: CloudTrail 監査用の明確なセッション名
- **エラーログ**: 認証試行のログ（認証情報なし）
- **操作追跡**: どのアカウント/ロールが操作を実行したかの明確な追跡

## テスト戦略

### 1. ユニットテスト
- 設定解析とバリデーション
- STS API 呼び出しモック
- エラーハンドリングシナリオ
- CLI 引数解析

### 2. 統合テスト
- 実際の AWS AssumeRole ワークフロー
- クロスアカウントスタック検出
- MFA 保護ロールテスト
- エラーシナリオバリデーション

### 3. セキュリティテスト
- 無効な External ID 処理
- 期限切れセッション処理
- 不十分な権限シナリオ
- MFA トークンバリデーション

## エラーメッセージとユーザーエクスペリエンス

### MFA 要件

**MFA はオプション**であり、以下の場合にのみ必要です:
1. ロールの信頼ポリシーが明確に MFA を要求する場合: `"aws:MultiFactorAuthPresent": "true"`
2. `--mfa-serial` が指定された場合、MFA トークンが必要になります
3. MFA トークンは以下の方法で提供できます:
   - `--mfa-token` での明示的指定
   - インタラクティブプロンプトでの入力（デフォルト動作）
   - `FIND_SLS3_MFA_TOKEN` 環境変数

### `--non-interactive` フラグの動作

**適用範囲**: このフラグは MFA 使用時のインタラクティブプロンプトを制御します

#### MFA 使用時の動作
- **デフォルト（`--non-interactive` 未指定）**: インタラクティブモード
  - `--mfa-serial` 指定時、`--mfa-token` がなければ自動的にプロンプト表示
  
- **`--non-interactive` 指定時**: 非インタラクティブモード  
  - プロンプト表示を無効化
  - `--mfa-token` または環境変数 `FIND_SLS3_MFA_TOKEN` が必要

#### MFA 不使用時の動作
- **影響なし**: MFA を使用しない AssumeRole では、`--non-interactive` の有無に関係なく同じ動作
- 将来的に他のインタラクティブ機能が追加された場合にも適用される汎用フラグ

```bash
# MFA 不使用時 - 両方とも同じ結果
find_sls3_stacks --assume-role ROLE
find_sls3_stacks --assume-role ROLE --non-interactive
```

### 役立つエラーメッセージ

#### アクセス拒否
```
エラー: arn:aws:iam::123456789012:role/ReadRole の AssumeRole に失敗しました
理由: アクセス拒否 - ロールが存在しないか、引き受ける権限がない可能性があります

トラブルシューティング手順:
1. ロール ARN が正しいことを確認してください
2. 現在の認証情報に sts:AssumeRole 権限があることを確認してください
3. ロールの信頼ポリシーがあなたのアカウント/ユーザーを許可していることを確認してください
4. External ID を使用している場合、ロールの要件と一致することを確認してください

IAM ポリシー例:
{
  "Effect": "Allow",
  "Action": "sts:AssumeRole", 
  "Resource": "arn:aws:iam::123456789012:role/ReadRole"
}
```

#### MFA 必要
```
エラー: AssumeRole に失敗しました - MFA が必要です
理由: ロールの信頼ポリシーが多要素認証を要求しています

ロールは MFA を要求しますが、MFA 認証情報が提供されていません。
--mfa-serial と --mfa-token の両方を提供してください:

例:
  find_sls3_stacks --region us-east-1 \
    --assume-role arn:aws:iam::123456789012:role/MFAProtectedRole \
    --mfa-serial arn:aws:iam::111122223333:mfa/user@example.com \
    --mfa-token 123456
```

#### 無効な MFA トークン
```
エラー: AssumeRole に失敗しました - 無効な MFA トークンです
理由: 提供された MFA トークンが無効または期限切れです

以下をチェックしてください:
1. MFA トークンが最新であること（6桁コード）
2. MFA デバイスのシリアル番号が正しいこと
3. トークンが期限切れでないこと（トークンは約30秒間有効）
```

#### MFA トークン不足
```
エラー: 無効な引数 - MFA シリアルがトークンなしで提供されました
使用法: --mfa-serial を使用する場合、--mfa-token が必要です

例:
  find_sls3_stacks --region us-east-1 \
    --assume-role arn:aws:iam::123456789012:role/Role \
    --mfa-serial arn:aws:iam::111122223333:mfa/user@example.com \
    --mfa-token 123456
```

### 進行状況インジケーター
```
✓ AssumeRole 設定を検証中...
MFA トークンを入力してください (arn:aws:iam::111122223333:mfa/user@example.com): ●●●●●●
✓ ロール arn:aws:iam::123456789012:role/ReadRole を引き受け中...
✓ CloudFormation 権限を検証中...
✓ us-east-1 のスタックを検出中...
3つの Serverless Framework v3 スタックが見つかりました
```

### MFA 入力エクスペリエンス
```
# インタラクティブプロンプト（非表示入力）
find_sls3_stacks --region us-east-1 --assume-role ROLE --mfa-serial SERIAL
MFA トークンを入力してください (arn:aws:iam::111122223333:mfa/user@example.com): 
[ユーザーが 123456 を入力、●●●●●● として表示]
✓ MFA 認証成功
```

## 後方互換性

- **既存機能**: 現在のすべての機能は変更されません
- **オプションパラメータ**: AssumeRole パラメータはオプションです
- **設定形式**: 既存の設定は引き続き動作します
- **API 互換性**: 内部インターフェースは後方互換性を維持します

## パフォーマンスへの影響

- **最小オーバーヘッド**: AssumeRole は認証情報交換のため約100-200ms追加
- **キャッシュ機会**: 一時的認証情報をセッション期間中キャッシュ可能
- **ネットワーク呼び出し**: 実行ごとに1つの追加 STS API 呼び出し
- **メモリ使用量**: 認証情報保存のための無視できる追加メモリ

## ハイブリッド MFA アプローチの利点

### セキュリティの利点
- ✅ **コマンド履歴露出なし**: インタラクティブトークンはシェル履歴に表示されません
- ✅ **プロセスリスト可視性なし**: 非表示入力により `ps` 出力でのトークン露出を防止
- ✅ **ジャストインタイムトークン**: 新しいトークンが脆弱性のウィンドウを減らします
- ✅ **環境変数フォールバック**: `FIND_SLS3_MFA_TOKEN` による安全な自動化

### ユーザーエクスペリエンスの向上
- ✅ **簡素化されたコマンド**: インタラクティブ使用でトークンを別途生成する必要なし
- ✅ **ガイド付きエクスペリエンス**: デバイスシリアル情報付きの明確なプロンプト
- ✅ **自動化対応**: スクリプト用の明示的トークンと環境変数
- ✅ **CI/CD 互換**: 自動化環境でのハングを防ぐ `--non-interactive` フラグ

### 実装の柔軟性
- ✅ **段階的向上**: 明示的トークンから開始し、インタラクティブ入力を追加
- ✅ **後方互換性**: 既存のすべてのコマンドパターンが引き続き動作
- ✅ **クロスプラットフォーム**: Windows、macOS、Linux ターミナルで動作
- ✅ **標準準拠**: AWS CLI やその他のエンタープライズツールのパターンに従う

## 今後の拡張

### 1. 認証情報キャッシュ
- セッション期間中の一時的認証情報キャッシュ
- 期限切れ前の自動更新
- 安全な認証情報保存

### 2. ロールチェーン
- マルチホップ AssumeRole チェーンのサポート
- 複雑なクロスアカウント権限シナリオ
- 組織単位ベースのロール引き受け

### 3. インタラクティブモード
- インタラクティブ MFA トークンプロンプト
- 利用可能なロールからのロール選択
- セッション管理 UI