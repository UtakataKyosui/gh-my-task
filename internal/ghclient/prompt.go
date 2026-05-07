package ghclient

import (
	"fmt"
	"strings"
)

func BuildPrompt(owner, name string, pr PR) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# レビュー対応タスク: PR #%d %s\n\n", pr.Number, pr.Title))

	sb.WriteString("## PR 情報\n\n")
	sb.WriteString(fmt.Sprintf("- **リポジトリ**: %s/%s\n", owner, name))
	if pr.HeadRef != "" {
		sb.WriteString(fmt.Sprintf("- **ブランチ**: `%s`\n", pr.HeadRef))
	}
	sb.WriteString(fmt.Sprintf("- **作成者**: %s\n", pr.Author))
	sb.WriteString(fmt.Sprintf("- **状態**: %s\n", pr.State))
	sb.WriteString(fmt.Sprintf("- **URL**: %s\n\n", pr.URL))

	if len(pr.Reviews) > 0 {
		sb.WriteString("## レビュー状況\n\n")
		sb.WriteString("| レビュワー | 状態 |\n|---|---|\n")
		for _, r := range pr.Reviews {
			icon := reviewIcon(r.State)
			sb.WriteString(fmt.Sprintf("| %s | %s %s |\n", r.Reviewer, icon, r.State))
		}
		sb.WriteString("\n")
	}

	if len(pr.ReviewComments) > 0 {
		sb.WriteString(fmt.Sprintf("## レビューコメント (%d件)\n\n", len(pr.ReviewComments)))
		for _, c := range pr.ReviewComments {
			loc := c.Path
			if c.Line > 0 {
				loc = fmt.Sprintf("%s:%d", c.Path, c.Line)
			}
			sb.WriteString(fmt.Sprintf("### %s — `%s`\n\n", c.Reviewer, loc))
			sb.WriteString(c.Body)
			sb.WriteString("\n\n")
		}
		sb.WriteString("---\n\n")
	}

	if pr.Body != "" {
		sb.WriteString("## PR 概要\n\n")
		body := pr.Body
		if len(body) > 3000 {
			body = body[:3000] + "\n…(省略)"
		}
		sb.WriteString(body)
		sb.WriteString("\n\n---\n\n")
	}

	sb.WriteString("## AI Agent への指示\n\n")
	if pr.HeadRef != "" {
		sb.WriteString(fmt.Sprintf("以下のコマンドで git worktree を作成し、上記のレビューコメントに対応してください。\n\n"))
		sb.WriteString("```bash\n")
		sb.WriteString(fmt.Sprintf("git worktree add ../pr-%d %s\n", pr.Number, pr.HeadRef))
		sb.WriteString(fmt.Sprintf("cd ../pr-%d\n", pr.Number))
		sb.WriteString("```\n\n")
	}
	if len(pr.ReviewComments) > 0 {
		sb.WriteString("上記のレビューコメントをすべて確認し、必要な修正を行ってください。\n")
		sb.WriteString("対応が完了したらコミットしてプッシュしてください。\n")
	} else {
		sb.WriteString("レビューコメントはありません。PR の概要を確認し、必要であれば対応してください。\n")
	}

	return sb.String()
}

func reviewIcon(state string) string {
	switch state {
	case "APPROVED":
		return "✅"
	case "CHANGES_REQUESTED":
		return "❌"
	default:
		return "💬"
	}
}
