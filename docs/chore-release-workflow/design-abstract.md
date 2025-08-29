branch: docs/chore-release-workflow

> [!NOTE]
> あくまで企画時点での案であり、実際の実装計画には反映されない、または一部が変更される可能性がある。

# design abstract

GitHub Release が作成されたことをトリガーに、自動的に各プラットフォームごとのバイナリがビルドされ、配布されるワークフローを構築する。

## Goals

- GitHub Release の作成をトリガーに、自動的にバイナリがビルドされる
  - Release のタグ名が semver (例: v1.2.3) であることを起動条件とする
- 各プラットフォームごとのバイナリが配布される (Linux/macOS/Windows, AMD64/ARM64)

## refs

- [GoReleaser](https://goreleaser.com/)
