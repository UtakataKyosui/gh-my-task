package install

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const claudeMDSection = `
## gh-my-task — PR 管理ツール

現在のリポジトリで自分が author または review-requested な PR を一覧・操作する ` + "`gh`" + ` 拡張機能。

### コマンド一覧

| コマンド | 説明 |
|---|---|
| ` + "`gh my-task`" + ` | TUI で PR 一覧 |
| ` + "`gh my-task -j`" + ` | JSON で PR 一覧 |
| ` + "`gh my-task -R`" + ` | レビュー状況付きで取得（インジケーター表示） |
| ` + "`gh my-task prompt <N>`" + ` | PR #N のレビュー対応 Markdown プロンプトを stdout に出力 |
| ` + "`gh my-task close <N>`" + ` | PR #N を close（番号入力確認あり） |

### 主なフラグ

- ` + "`-j / --json`" + `: JSON 出力（AI・スクリプト連携用）
- ` + "`-s / --state open|closed|all`" + `: 状態フィルタ
- ` + "`-a / --author-only`" + `: 自分が author のみ
- ` + "`-r / --review-only`" + `: レビュー依頼のみ
- ` + "`-R / --with-reviews`" + `: レビュー状況を同時取得

### TUI のレビューインジケーター（` + "`-R`" + ` 使用時）

` + "`-R`" + ` フラグを付けると、review-requested な PR にレビュー状態インジケーターが表示される。

| バッジ表示 | 意味 |
|---|---|
| ` + "`[R]`" + ` | レビュー未着手（pending） |
| ` + "`[R]✓`" + ` | APPROVED あり（緑） |
| ` + "`[R]!`" + ` | CHANGES_REQUESTED あり（赤） |
| ` + "`[R]~`" + ` | COMMENTED のみ（グレー） |

description 欄にも ` + "`· 2 approved`" + ` / ` + "`· 1 changes`" + ` のサマリーが付く。

### AI Agent での活用例

` + "```" + `bash
gh my-task prompt 123       # レビュー対応指示を Markdown で出力
gh my-task -j -R            # PR 一覧 + レビュー状況を JSON で取得
` + "```" + `
`

const skillMD = `---
name: gh-my-task
description: gh my-task CLI を使い、現在のリポジトリで自分に関係する PR を一覧・操作する
---

# gh-my-task スキル

## 概要

` + "`gh my-task`" + ` は GitHub CLI 拡張機能。現在のリポジトリで自分が author または
review-requested な PR を素早く確認・操作できる。

` + "`-R`" + ` フラグを付けると TUI の一覧にレビュー状態インジケーターが表示され、
レビュー前後の PR を視覚的に区別できる。

## よく使うコマンド

### PR 一覧を取得する

` + "```" + `bash
gh my-task -j              # JSON 出力
gh my-task -j -R           # レビュー状況込み JSON
gh my-task -j -s all       # 全ステート
gh my-task -j -a           # 自分が author のみ
` + "```" + `

### TUI でレビュー状態を確認する（` + "`-R`" + ` 推奨）

` + "```" + `bash
gh my-task -R              # TUI + レビューインジケーター
` + "```" + `

review-requested な PR のバッジに状態が付く:

| バッジ | 意味 |
|---|---|
| ` + "`[R]`" + ` | レビュー未着手 |
| ` + "`[R]✓`" + ` | APPROVED あり |
| ` + "`[R]!`" + ` | CHANGES_REQUESTED あり |
| ` + "`[R]~`" + ` | COMMENTED のみ |

### レビュー対応プロンプトを生成する

` + "```" + `bash
gh my-task prompt <PR番号>
` + "```" + `

出力には以下が含まれる:
- PR タイトル・ブランチ・URL
- レビュワーと承認状態（✅ APPROVED / ❌ CHANGES_REQUESTED / 💬 COMMENTED）
- インラインレビューコメント一覧
- git worktree 作成コマンド

### PR を close する

` + "```" + `bash
gh my-task close <PR番号>   # PR番号の入力確認後に close
` + "```" + `

## 使い方のパターン

1. ` + "`gh my-task -R`" + ` で TUI を起動し、レビュー状態インジケーターで優先度を判断する
2. ` + "`gh my-task prompt <N>`" + ` でレビュー対応プロンプトを生成する
3. 生成されたプロンプトを Claude Code に渡して対応する
`

func Run() {
	r := bufio.NewReader(os.Stdin)

	fmt.Println("gh-my-task の使い方を Claude Code に知識として追加します。\n")
	fmt.Println("追加先を選択してください:")
	fmt.Println("  1. ユーザーレベル  — ~/.claude/CLAUDE.md に追記")
	fmt.Println("  2. プロジェクトレベル — .claude/skills/gh-my-task/SKILL.md を作成")
	fmt.Print("\n番号を入力 [1/2]: ")

	line, _ := r.ReadString('\n')
	choice := strings.TrimSpace(line)

	switch choice {
	case "1":
		if err := appendToUserClaude(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "2":
		if err := writeProjectSkill(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "error: 1 または 2 を入力してください")
		os.Exit(1)
	}
}

func appendToUserClaude() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("ホームディレクトリを取得できません: %w", err)
	}
	path := filepath.Join(home, ".claude", "CLAUDE.md")

	existing, _ := os.ReadFile(path)
	if strings.Contains(string(existing), "## gh-my-task") {
		fmt.Printf("既に追加済みです: %s\n", path)
		return nil
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("%s を開けません: %w", path, err)
	}
	defer f.Close()

	if _, err := f.WriteString(claudeMDSection); err != nil {
		return fmt.Errorf("書き込み失敗: %w", err)
	}
	fmt.Printf("✓ 追記しました: %s\n", path)
	return nil
}

func writeProjectSkill() error {
	dir := filepath.Join(".claude", "skills", "gh-my-task")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ディレクトリ作成失敗: %w", err)
	}
	path := filepath.Join(dir, "SKILL.md")

	if _, err := os.Stat(path); err == nil {
		fmt.Printf("既に存在します: %s\n", path)
		return nil
	}

	if err := os.WriteFile(path, []byte(skillMD), 0644); err != nil {
		return fmt.Errorf("書き込み失敗: %w", err)
	}
	fmt.Printf("✓ 作成しました: %s\n", path)
	return nil
}
