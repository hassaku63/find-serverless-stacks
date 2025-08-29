# MFA 入力設計分析

## 概要

この文書では、AssumeRole 機能における MFA トークン入力に関する2つの異なるアプローチを分析します：
1. **明示的トークンアプローチ** (現在の設計)
2. **インタラクティブ入力アプローチ** (代替設計)

## アプローチ 1: 明示的トークン (現在の設計)

### CLI インターフェース
```bash
# MFA トークンを明示的に提供
find_sls3_stacks --region us-east-1 \
  --assume-role arn:aws:iam::123456789012:role/MFARole \
  --mfa-serial arn:aws:iam::111122223333:mfa/user@example.com \
  --mfa-token 123456
```

### 実装
- `--mfa-serial` が提供されている場合は `--mfa-token` が必須
- 両方のパラメータを一緒に提供する必要がある
- インタラクティブ入力は不要

## アプローチ 2: インタラクティブ入力 (代替設計)

### CLI インターフェース
```bash
# インタラクティブ MFA トークンプロンプト
find_sls3_stacks --region us-east-1 \
  --assume-role arn:aws:iam::123456789012:role/MFARole \
  --mfa-serial arn:aws:iam::111122223333:mfa/user@example.com

# CLI プロンプト:
# MFA トークンを入力してください (arn:aws:iam::111122223333:mfa/user@example.com): [非表示入力]
```

### 実装オプション
```go
// オプション A: mfa-serial が提供されている場合は常にプロンプト
func promptForMFAToken(serial string) (string, error) {
    fmt.Printf("MFA トークンを入力してください (%s): ", serial)
    token, err := term.ReadPassword(int(syscall.Stdin))
    return string(token), err
}

// オプション B: --mfa-token が提供されていない場合のみプロンプト
func getMFAToken(providedToken, serial string) (string, error) {
    if providedToken != "" {
        return providedToken, nil
    }
    return promptForMFAToken(serial)
}

// オプション C: 確認付きプロンプト
func promptForMFATokenWithRetry(serial string) (string, error) {
    for attempts := 0; attempts < 3; attempts++ {
        token, err := promptForMFAToken(serial)
        if err != nil {
            return "", err
        }
        if validateMFATokenFormat(token) {
            return token, nil
        }
        fmt.Println("無効なトークン形式です。MFA トークンは通常6桁です。")
    }
    return "", errors.New("最大試行回数を超えました")
}
```

## 詳細比較

### 1. セキュリティ考慮事項

#### 明示的トークンアプローチ
**利点:**
- ✅ 環境変数を使用すればトークンがコマンド履歴に残らない
- ✅ スクリプト化と自動化が可能
- ✅ MFA 使用の明確な監査証跡
- ✅ 非インタラクティブ環境（CI/CD）で動作

**欠点:**
- ❌ 直接提供した場合はトークンがコマンド履歴に表示される
- ❌ コマンド実行中にトークンがプロセスリストに表示される
- ❌ コマンド実行前にトークンを事前生成する必要がある

#### インタラクティブ入力アプローチ
**利点:**
- ✅ トークンがコマンド履歴に表示されない
- ✅ トークンがプロセスリストに表示されない
- ✅ ジャストインタイム トークン入力（新しいトークン）
- ✅ 非表示入力（パスワード形式）

**欠点:**
- ❌ スクリプト化や自動化ができない
- ❌ 非インタラクティブ環境では動作しない
- ❌ ターミナルでの対話が必要

### 2. ユーザーエクスペリエンス

#### 明示的トークンアプローチ
**利点:**
- ✅ 予測可能で、驚きがない
- ✅ 高速実行（ユーザー入力待ちなし）
- ✅ 不足パラメータに関する明確なエラーメッセージ
- ✅ 他の CLI ツール（AWS CLI、kubectl）と一貫性

**欠点:**
- ❌ 2ステップのプロセスが必要（トークン取得、コマンド実行）
- ❌ トークンが生成と使用の間に期限切れになる可能性
- ❌ より複雑なコマンドライン

#### インタラクティブ入力アプローチ
**利点:**
- ✅ 簡素化されたコマンドラインインターフェース
- ✅ 新しいトークン（期限切れの心配なし）
- ✅ ガイド付きユーザーエクスペリエンス
- ✅ MFA付き `aws sts assume-role` と似た操作感

**欠点:**
- ❌ 予期しないプロンプトでユーザーが混乱する可能性
- ❌ 自動化ワークフローを阻害
- ❌ スクリプト内でのトラブルシューティングが困難

### 3. 実装の複雑さ

#### 明示的トークンアプローチ
**複雑さ:** 低
- シンプルなパラメータ解析
- 標準的なバリデーションロジック
- ターミナル処理が不要

```go
type AssumeRoleConfig struct {
    RoleARN     string
    MFASerial   string
    MFAToken    string  // MFASerial が提供されている場合は常に必須
}

func (c *AssumeRoleConfig) Validate() error {
    if c.MFASerial != "" && c.MFAToken == "" {
        return errors.New("--mfa-serial が指定されている場合は --mfa-token が必要です")
    }
    return nil
}
```

#### インタラクティブ入力アプローチ
**複雑さ:** 中-高
- ターミナル処理（クロスプラットフォーム）
- パスワード形式入力
- 非インタラクティブ環境のエラーハンドリング
- フォールバック機構

```go
import (
    "golang.org/x/term"
    "syscall"
)

type AssumeRoleConfig struct {
    RoleARN     string
    MFASerial   string
    MFAToken    string  // インタラクティブモードが有効な場合はオプション
}

func (c *AssumeRoleConfig) ResolveMFAToken() error {
    if c.MFASerial != "" && c.MFAToken == "" {
        if !isInteractiveTerminal() {
            return errors.New("実行モードがインタラクティブターミナルではない場合、 --mfa-token を指定する必要があります")
        }
        token, err := promptForMFAToken(c.MFASerial)
        if err != nil {
            return err
        }
        c.MFAToken = token
    }
    return nil
}
```

### 4. 互換性と標準

#### 業界標準分析

**AWS CLI アプローチ:**
```bash
# AWS CLI は明示的トークンを使用
aws sts assume-role \
  --role-arn arn:aws:iam::123456789012:role/Role \
  --role-session-name session \
  --serial-number arn:aws:iam::111122223333:mfa/user \
  --token-code 123456
```

**kubectl アプローチ:**
```bash
# kubectl は一部の認証方式でインタラクティブプロンプトを使用
kubectl get pods
# トークンを入力: [インタラクティブプロンプト]
```

**HashiCorp Vault:**
```bash
# Vault は両方のアプローチをサポート
vault write auth/aws/login role=dev-role  # インタラクティブ
vault write auth/aws/login role=dev-role token=123456  # 明示的
```

## ハイブリッドアプローチ推奨

### 推奨設計: **両方の世界のベスト**

```bash
# 明示的トークン（現在の動作）
find_sls3_stacks --assume-role ROLE --mfa-serial SERIAL --mfa-token 123456

# トークンが提供されていない場合はインタラクティブプロンプト
find_sls3_stacks --assume-role ROLE --mfa-serial SERIAL
# プロンプト: MFA トークンを入力: [非表示入力]

# 非インタラクティブモードを強制
find_sls3_stacks --assume-role ROLE --mfa-serial SERIAL --non-interactive
# エラー: MFA トークンが必要ですが提供されておらず、非インタラクティブモードで実行中
```

### 実装戦略

```go
type AssumeRoleConfig struct {
    RoleARN        string
    MFASerial      string
    MFAToken       string
    NonInteractive bool  // 新しいフラグ
}

func (c *AssumeRoleConfig) ResolveMFAToken() error {
    if c.MFASerial == "" {
        return nil // MFA 不要
    }
    
    if c.MFAToken != "" {
        return nil // トークンは既に提供済み
    }
    
    if c.NonInteractive || !isInteractiveTerminal() {
        return errors.New("MFA トークンが必要ですが提供されていません。--mfa-token を使用するかインタラクティブに実行してください")
    }
    
    // インタラクティブプロンプト
    fmt.Printf("MFA トークンを入力してください (%s): ", c.MFASerial)
    token, err := readHiddenInput()
    if err != nil {
        return fmt.Errorf("MFA トークンの読み取りに失敗: %w", err)
    }
    
    c.MFAToken = strings.TrimSpace(token)
    if c.MFAToken == "" {
        return errors.New("空の MFA トークンが提供されました")
    }
    
    return nil
}

func isInteractiveTerminal() bool {
    return term.IsTerminal(int(syscall.Stdin)) && term.IsTerminal(int(syscall.Stdout))
}

func readHiddenInput() (string, error) {
    bytes, err := term.ReadPassword(int(syscall.Stdin))
    fmt.Println() // 非表示入力後に改行
    return string(bytes), err
}
```

## 評価サマリー

### 現在の設計（明示的のみ）
- **セキュリティ**: 7/10（良好だが CLI で可視）
- **UX**: 6/10（機能的だが2ステップ必要）
- **自動化**: 10/10（スクリプトに完璧）
- **実装**: 10/10（シンプル）
- **標準**: 8/10（AWS CLI パターンに従う）

### インタラクティブのみ
- **セキュリティ**: 9/10（優秀、決して可視にならない）
- **UX**: 8/10（人間にとってスムーズ）
- **自動化**: 2/10（自動化を阻害）
- **実装**: 6/10（より複雑）
- **標準**: 5/10（AWS CLI と一致しない）

### ハイブリッドアプローチ（推奨）
- **セキュリティ**: 9/10（優秀、ベストプラクティス）
- **UX**: 9/10（人間と自動化の両方にスムーズ）
- **自動化**: 9/10（明示的または CI フラグで動作）
- **実装**: 7/10（適度な複雑さ）
- **標準**: 9/10（両方のパターンをサポート）

## 推奨事項

### 1. ハイブリッドアプローチの実装
ハイブリッドアプローチが最高の全体的エクスペリエンスを提供します：
- 後方互換性を維持
- 人間と自動化の両方の使用をサポート
- セキュリティベストプラクティスに従う
- 明確なフォールバック機構を提供

### 2. 実装優先度
```
フェーズ 1: 明示的トークンサポート（現在の設計）
フェーズ 2: インタラクティブ入力機能を追加
フェーズ 3: CI/CD用の --non-interactive フラグを追加
```

### 3. 設定オプション
```go
// CLI フラグ
--mfa-serial string       # MFA デバイスシリアル（MFA 要件をトリガー）
--mfa-token string        # MFA トークン（インタラクティブの場合はオプション）
--non-interactive         # インタラクティブプロンプトを無効化
```

### 4. 環境変数サポート
```bash
# 自動化用の環境変数
export FIND_SLS3_MFA_TOKEN=123456
find_sls3_stacks --assume-role ROLE --mfa-serial SERIAL
```

このハイブリッドアプローチは、異なる使用例においてセキュリティと使いやすさを維持しながら最大の柔軟性を提供します。
