# GoReleaser リリースワークフロー実装計画

## 概要

GitHub Release作成をトリガーに、GoReleaserを使用して複数プラットフォーム向けのバイナリを自動ビルド・配布するワークフローを構築する。

## 要件

### トリガー条件
- GitHub Release作成・公開時（手動作成を前提）
- タグ名がsemver形式（例: v1.2.3, v1.0.0-beta.1, v1.0.0-rc.1）であること
- プレリリースとリリースの両方に対応（prerelease: auto で自動判定）

### ワークフローパターン（セミ自動型）
1. **手動作業**: 開発者がタグ作成とGitHub Release作成
2. **自動実行**: Release公開をトリガーにGoReleaserが実行
3. **自動判定**: タグ名からプレリリース/正式リリースを自動判定
4. **自動配布**: バイナリビルドとGitHub Releaseへの添付

### 配布対象
- **OS**: Linux, macOS, Windows
- **アーキテクチャ**: AMD64, ARM64
- **配布形式**: 
  - Linux/macOS: tar.gz
  - Windows: zip
- **チェックサム**: SHA256

## 実装フェーズ

### Phase 1: GoReleaser設定ファイル作成

#### 1.1 `.goreleaser.yml` 基本設定
```yaml
# 主要設定項目
builds:
  - Binary名: find_serverless_stacks
  - メインパッケージ: ./cmd/find_serverless_stacks
  - 対象OS/Arch: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64
  - ビルドフラグ: バージョン情報埋め込み（現在のMakefileのLDFLAGSを踏襲）

archives:
  - 命名規則: find_serverless_stacks_{{ .Version }}_{{ .Os }}_{{ .Arch }}
  - Linux/macOS: tar.gz
  - Windows: zip

release:
  - prerelease: auto  # タグ名から自動判定（v1.0.0-beta.1 → prerelease）

checksum:
  - アルゴリズム: sha256
  - ファイル名: checksums.txt
```

#### 1.2 メタデータ設定
- プロジェクト名
- 説明文
- ホームページURL
- ライセンス情報

### Phase 2: GitHub Actions ワークフロー

#### 2.1 `.github/workflows/release.yml` 作成
```yaml
# トリガー設定
on:
  release:
    types: [published]

# 権限設定
permissions:
  contents: write  # GitHub Releaseへの書き込み権限

# ジョブ設定
jobs:
  goreleaser:
    - Go環境セットアップ
    - GoReleaserアクション実行
    - アーティファクト生成とアップロード
```

#### 2.2 セキュリティ考慮事項
- GITHUB_TOKENの適切な権限設定
- プライベートキーやシークレットは使用しない（パブリックリポジトリのため）

### Phase 3: 補助ツール・設定

#### 3.1 Makefile更新
```makefile
# 新規追加するターゲット
release-check:    # GoReleaser設定の検証
release-snapshot: # ローカルでのスナップショットビルド
release-clean:    # リリース関連のクリーンアップ
```

#### 3.2 .gitignore更新
- GoReleaserの出力ディレクトリ（dist/）を追加

### Phase 4: ドキュメント更新

#### 4.1 README.md更新
- Installation セクションの更新
  - Binary Download セクションを復活
  - 実際のGitHub Releaseページを参照するURLに修正
  - 各プラットフォーム向けのインストール手順

#### 4.2 リリース手順ドキュメント作成
- `docs/chore-release-workflow/release-process.md`
- メンテナー向けのリリース手順（手動でのタグ作成・GitHub Release作成）
- バージョニング規則（alpha/beta/rc パターン）
- セミ自動型ワークフローの運用ガイド

### Phase 5: 検証・テスト

#### 5.1 設定ファイル検証
```bash
# GoReleaser設定の文法チェック
goreleaser check

# ローカルでのビルドテスト（リリースなし）
goreleaser build --snapshot --clean
```

#### 5.2 統合テスト
- プライベートリポジトリでの事前テスト
- または draft release での検証

## 技術的検討事項

### ビルド設定詳細

#### バイナリ名とパス
- バイナリ名: `find_serverless_stacks`
- メインパッケージ: `./cmd/find_serverless_stacks`
- 出力パス: デフォルト（GoReleaserが管理）

#### ビルドフラグ
現在のMakefileで使用されているLDFLAGSを踏襲：
```
-ldflags "-X main.version={{ .Version }} -X main.buildTime={{ .Date }} -X main.gitCommit={{ .ShortCommit }}"
```

#### CGOの扱い
- CGO_ENABLED=0 を設定（静的リンク）
- クロスコンパイルの安定性向上

### アーカイブ命名規則
```
find_serverless_stacks_v1.2.3_linux_amd64.tar.gz
find_serverless_stacks_v1.2.3_darwin_arm64.tar.gz
find_serverless_stacks_v1.2.3_windows_amd64.zip
```

### エラーハンドリング
- ビルド失敗時のワークフロー停止
- 部分的な成功時の処理方針

## リスク・制約事項

### リスク
1. **初回リリース時の設定ミス**: テスト不十分によるリリース失敗
2. **権限設定ミス**: GitHub Actionsの権限不足
3. **プラットフォーム固有の問題**: 特定のOS/アーキテクチャでのビルド失敗

### 制約事項
1. **GitHub Actions使用制限**: パブリックリポジトリのため制限は緩い
2. **GoReleaserの無料版制限**: Pro機能は使用不可
3. **バイナリサイズ**: GitHub Releaseの添付ファイルサイズ制限（2GB）

## 成果物一覧

### 設定ファイル
- `.goreleaser.yml` - GoReleaser設定
- `.github/workflows/release.yml` - GitHub Actionsワークフロー

### ドキュメント
- `docs/chore-release-workflow/release-process.md` - リリース手順
- `README.md` 更新 - インストール方法

### 補助ファイル
- `Makefile` 更新 - リリース関連ターゲット
- `.gitignore` 更新 - GoReleaser出力ディレクトリ

## 実装順序

1. **Phase 1**: GoReleaser設定ファイル作成・検証
2. **Phase 3**: Makefile更新（ローカルテスト用）
3. **Phase 5.1**: ローカルでの設定検証
4. **Phase 2**: GitHub Actionsワークフロー作成
5. **Phase 4**: ドキュメント更新
6. **Phase 5.2**: 統合テスト

この順序により、段階的な検証が可能となり、リスクを最小化できる。