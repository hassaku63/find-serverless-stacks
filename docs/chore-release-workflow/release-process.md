# リリースプロセス

このドキュメントは、メンテナー向けのリリース手順を説明します。

## リリースワークフロー概要

**セミ自動型ワークフロー**を採用しています：

1. **手動作業**: タグ作成とGitHub Release作成
2. **自動実行**: Release公開をトリガーにGoReleaserが実行
3. **自動判定**: タグ名からプレリリース/正式リリースを自動判定
4. **自動配布**: バイナリビルドとGitHub Releaseへの添付

## バージョニング規則

### 正式リリース
```
v1.0.0
v1.2.3
v2.0.0
```

### プレリリース（alpha/beta/rc パターン）
```
v1.0.0-alpha.1    # アルファ版
v1.0.0-beta.1     # ベータ版
v1.0.0-rc.1       # リリース候補版
```

## リリース手順

### 事前準備

1. **ローカル環境での検証**
   ```bash
   # GoReleaser設定の検証
   make release-check
   
   # ローカルでのスナップショットビルドテスト
   make release-snapshot
   ```

2. **変更内容の確認**
   - コードの動作確認
   - テストの実行: `make test`
   - ドキュメントの更新確認

### リリース実行

#### 1. タグ作成
```bash
# 正式リリースの場合
git tag v1.0.0

# プレリリースの場合
git tag v1.0.0-beta.1

# タグをプッシュ
git push origin <tag-name>
```

#### 2. GitHub Release作成

**GitHub UI から:**
1. [Releases](https://github.com/hassaku63/find-serverless-stacks/releases) ページにアクセス
2. "Create a new release" をクリック
3. 作成したタグを選択
4. リリースタイトルとリリースノートを記入
5. プレリリースの場合は "This is a pre-release" をチェック
6. "Publish release" をクリック

**GitHub CLI から:**
```bash
# 正式リリースの場合
gh release create v1.0.0 --title "v1.0.0" --notes "リリースノート"

# プレリリースの場合
gh release create v1.0.0-beta.1 --title "v1.0.0-beta.1" --notes "ベータリリース" --prerelease
```

#### 3. 自動ビルドの確認

GitHub Release公開後、以下が自動実行されます：

1. **GitHub Actions トリガー**: `.github/workflows/release.yml`
2. **GoReleaser実行**: バイナリビルドとアーカイブ作成
3. **プレリリース判定**: タグ名から自動判定（`prerelease: auto`）
4. **アーティファクト添付**: GitHub Releaseにバイナリを自動添付

#### 4. リリース完了確認

- [Actions](https://github.com/hassaku63/find-serverless-stacks/actions) ページでワークフローの成功を確認
- GitHub Releaseページでバイナリが添付されていることを確認
- 各プラットフォーム向けのバイナリ（6種類）とチェックサムファイル

## 生成されるアーティファクト

### バイナリアーカイブ
- `find_serverless_stacks_v1.0.0_linux_amd64.tar.gz`
- `find_serverless_stacks_v1.0.0_linux_arm64.tar.gz`
- `find_serverless_stacks_v1.0.0_darwin_amd64.tar.gz`
- `find_serverless_stacks_v1.0.0_darwin_arm64.tar.gz`
- `find_serverless_stacks_v1.0.0_windows_amd64.tar.gz`
- `find_serverless_stacks_v1.0.0_windows_arm64.tar.gz`

### その他
- `checksums.txt` - SHA256チェックサム

## トラブルシューティング

### リリースワークフローが失敗した場合

1. **GitHub Actions ログの確認**
   - [Actions](https://github.com/hassaku63/find-serverless-stacks/actions) ページでエラー内容を確認

2. **GoReleaser設定の検証**
   ```bash
   make release-check
   ```

3. **権限の確認**
   - `GITHUB_TOKEN` の権限設定
   - リポジトリの Settings > Actions > General で権限確認

### よくある問題

**問題**: ビルドが特定のプラットフォームで失敗する
**解決**: `.goreleaser.yml` の `builds` セクションで該当プラットフォームを一時的に除外

**問題**: プレリリース判定が正しく動作しない
**解決**: タグ名の形式を確認（例: `v1.0.0-beta.1`）

## ベストプラクティス

1. **段階的リリース**: alpha → beta → rc → 正式リリース
2. **テスト**: 各段階でのフィードバック収集
3. **ドキュメント**: リリースノートの充実
4. **バックアップ**: 重要なリリース前のブランチ保護

## 緊急時の対応

### リリースの取り下げ

1. **GitHub Release の削除**
   ```bash
   gh release delete v1.0.0
   ```

2. **タグの削除**
   ```bash
   git tag -d v1.0.0
   git push origin :refs/tags/v1.0.0
   ```

### ホットフィックス

1. パッチバージョンを上げてリリース（例: v1.0.0 → v1.0.1）
2. 必要に応じてブランチから cherry-pick