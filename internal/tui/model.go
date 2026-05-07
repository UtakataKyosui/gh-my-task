package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/UtakataKyosui/gh-my-task/internal/ghclient"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cli/go-gh/v2/pkg/browser"
)

func updatedAt(item list.Item) time.Time {
	switch it := item.(type) {
	case prItem:
		return it.pr.UpdatedAt
	case issueItem:
		return it.issue.UpdatedAt
	}
	return time.Time{}
}

type prItem struct {
	pr ghclient.PR
}

func (p prItem) FilterValue() string { return p.pr.Title }
func (p prItem) Title() string       { return badge(p.pr) + reviewIndicator(p.pr) + " " + p.pr.Title }
func (p prItem) Description() string {
	ago := humanTime(p.pr.UpdatedAt)
	draft := ""
	if p.pr.IsDraft {
		draft = " " + draftStyle.String()
	}
	reviewSummary := ""
	if len(p.pr.Reviews) > 0 {
		reviewSummary = " · " + formatReviewSummary(p.pr.Reviews)
	}
	return fmt.Sprintf("#%d · %s · %s%s%s", p.pr.Number, p.pr.Author, ago, draft, reviewSummary)
}

type issueItem struct {
	issue ghclient.Issue
}

func (i issueItem) FilterValue() string { return i.issue.Title }
func (i issueItem) Title() string       { return badgeIssue.String() + " " + i.issue.Title }
func (i issueItem) Description() string {
	ago := humanTime(i.issue.UpdatedAt)
	return fmt.Sprintf("#%d · by %s · %s", i.issue.Number, i.issue.Author, ago)
}

func badge(pr ghclient.PR) string {
	hasAuthor := contains(pr.Categories, "author")
	hasReview := contains(pr.Categories, "review-requested")
	switch {
	case hasAuthor && hasReview:
		return badgeBoth.String()
	case hasReview:
		return badgeReview.String()
	default:
		return badgeAuthor.String()
	}
}

func reviewIndicator(pr ghclient.PR) string {
	if !contains(pr.Categories, "review-requested") || len(pr.Reviews) == 0 {
		return ""
	}
	switch dominantReviewState(pr.Reviews) {
	case "APPROVED":
		return reviewApprovedIndicator.String()
	case "CHANGES_REQUESTED":
		return reviewChangesIndicator.String()
	default:
		return reviewCommentedIndicator.String()
	}
}

func dominantReviewState(reviews []ghclient.Review) string {
	for _, r := range reviews {
		if r.State == "CHANGES_REQUESTED" {
			return "CHANGES_REQUESTED"
		}
	}
	for _, r := range reviews {
		if r.State == "APPROVED" {
			return "APPROVED"
		}
	}
	return "COMMENTED"
}

func formatReviewSummary(reviews []ghclient.Review) string {
	approved, changes, commented := 0, 0, 0
	for _, r := range reviews {
		switch r.State {
		case "APPROVED":
			approved++
		case "CHANGES_REQUESTED":
			changes++
		default:
			commented++
		}
	}
	if changes > 0 {
		return reviewChangesStyle.Render(fmt.Sprintf("%d changes", changes))
	}
	if approved > 0 {
		return reviewApprovedStyle.Render(fmt.Sprintf("%d approved", approved))
	}
	return reviewCommentedStyle.Render(fmt.Sprintf("%d commented", commented))
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func humanTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

type closeResultMsg struct {
	number  int
	isIssue bool
	err     error
}

type model struct {
	owner      string
	name       string
	repo       string
	list       list.Model
	preview    viewport.Model
	prs        []ghclient.PR
	issues     []ghclient.Issue
	ready      bool
	width      int
	height     int
	quitting   bool
	closeMode  bool
	closeInput textinput.Model
	closeErr   string
}

func Run(owner, name string, prs []ghclient.PR, issues []ghclient.Issue) error {
	items := make([]list.Item, 0, len(prs)+len(issues))
	for _, pr := range prs {
		items = append(items, prItem{pr})
	}
	for _, issue := range issues {
		items = append(items, issueItem{issue})
	}
	sort.Slice(items, func(i, j int) bool {
		return updatedAt(items[i]).After(updatedAt(items[j]))
	})

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	l := list.New(items, delegate, 0, 0)
	l.Title = fmt.Sprintf("My Tasks in %s/%s", owner, name)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.KeyMap.Quit = keys.Quit

	vp := viewport.New(0, 0)

	ti := textinput.New()
	ti.Placeholder = "PR番号"
	ti.CharLimit = 10

	m := model{
		owner:      owner,
		name:       name,
		repo:       fmt.Sprintf("%s/%s", owner, name),
		list:       l,
		preview:    vp,
		prs:        prs,
		issues:     issues,
		closeInput: ti,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		listW := msg.Width / 2
		previewW := msg.Width - listW - 1
		contentH := msg.Height - 3
		m.list.SetSize(listW, contentH)
		m.preview.Width = previewW
		m.preview.Height = contentH
		m.ready = true
		m.preview.SetContent(m.buildPreview())

	case closeResultMsg:
		m.closeMode = false
		m.closeInput.Reset()
		if msg.err != nil {
			m.closeErr = msg.err.Error()
			return m, nil
		}
		m.closeErr = ""
		items := m.list.Items()
		newItems := make([]list.Item, 0, len(items))
		for _, item := range items {
			switch it := item.(type) {
			case prItem:
				if msg.isIssue || it.pr.Number != msg.number {
					newItems = append(newItems, item)
				}
			case issueItem:
				if !msg.isIssue || it.issue.Number != msg.number {
					newItems = append(newItems, item)
				}
			}
		}
		m.list.SetItems(newItems)
		if msg.isIssue {
			newIssues := make([]ghclient.Issue, 0, len(m.issues))
			for _, issue := range m.issues {
				if issue.Number != msg.number {
					newIssues = append(newIssues, issue)
				}
			}
			m.issues = newIssues
		} else {
			newPRs := make([]ghclient.PR, 0, len(m.prs))
			for _, pr := range m.prs {
				if pr.Number != msg.number {
					newPRs = append(newPRs, pr)
				}
			}
			m.prs = newPRs
		}
		m.preview.SetContent(m.buildPreview())
		return m, nil

	case tea.KeyMsg:
		if m.closeMode {
			switch msg.Type {
			case tea.KeyEsc:
				m.closeMode = false
				m.closeErr = ""
				m.closeInput.Reset()
				return m, nil
			case tea.KeyEnter:
				return m, m.doClose()
			}
			var cmd tea.Cmd
			m.closeInput, cmd = m.closeInput.Update(msg)
			return m, cmd
		}

		if m.list.FilterState() == list.Filtering {
			break
		}
		switch {
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, keys.Enter):
			if sel, ok := m.list.SelectedItem().(prItem); ok {
				b := browser.New("", nil, nil)
				_ = b.Browse(sel.pr.URL)
			}
			return m, nil
		case key.Matches(msg, keys.Close):
			switch sel := m.list.SelectedItem().(type) {
			case prItem:
				m.closeMode = true
				m.closeErr = ""
				m.closeInput.Reset()
				m.closeInput.Placeholder = strconv.Itoa(sel.pr.Number)
				return m, m.closeInput.Focus()
			case issueItem:
				m.closeMode = true
				m.closeErr = ""
				m.closeInput.Reset()
				m.closeInput.Placeholder = strconv.Itoa(sel.issue.Number)
				return m, m.closeInput.Focus()
			}
			return m, nil
		}
	}

	prevIdx := m.list.Index()
	var listCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)
	cmds = append(cmds, listCmd)

	if m.list.Index() != prevIdx {
		m.preview.SetContent(m.buildPreview())
		m.preview.GotoTop()
	}

	var vpCmd tea.Cmd
	m.preview, vpCmd = m.preview.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m model) doClose() tea.Cmd {
	input := strings.TrimSpace(m.closeInput.Value())
	owner := m.owner
	name := m.name

	if isi, ok := m.list.SelectedItem().(issueItem); ok {
		expected := strconv.Itoa(isi.issue.Number)
		if input != expected {
			return func() tea.Msg {
				return closeResultMsg{err: fmt.Errorf("番号が一致しません（期待: %s）", expected)}
			}
		}
		number := isi.issue.Number
		return func() tea.Msg {
			err := ghclient.CloseIssue(owner, name, number)
			return closeResultMsg{number: number, isIssue: true, err: err}
		}
	}

	sel, ok := m.list.SelectedItem().(prItem)
	if !ok {
		return nil
	}
	expected := strconv.Itoa(sel.pr.Number)
	if input != expected {
		return func() tea.Msg {
			return closeResultMsg{err: fmt.Errorf("番号が一致しません（期待: %s）", expected)}
		}
	}
	number := sel.pr.Number
	return func() tea.Msg {
		err := ghclient.ClosePR(owner, name, number)
		return closeResultMsg{number: number, isIssue: false, err: err}
	}
}

func (m model) buildPreview() string {
	if len(m.list.Items()) == 0 {
		return "No tasks found."
	}
	switch sel := m.list.SelectedItem().(type) {
	case prItem:
		return m.buildPRPreview(sel.pr)
	case issueItem:
		return m.buildIssuePreview(sel.issue)
	}
	return ""
}

func (m model) buildPRPreview(pr ghclient.PR) string {
	var sb strings.Builder

	sb.WriteString(previewTitleStyle.Render(fmt.Sprintf("#%d %s", pr.Number, pr.Title)))
	sb.WriteString("\n\n")

	line := func(label, value string) {
		sb.WriteString(previewLabelStyle.Render(label+": "))
		sb.WriteString(previewValueStyle.Render(value))
		sb.WriteString("\n")
	}

	catLabels := make([]string, 0, len(pr.Categories))
	for _, c := range pr.Categories {
		catLabels = append(catLabels, c)
	}
	line("Type", "PR")
	line("Category", strings.Join(catLabels, ", "))
	line("Author", pr.Author)
	line("State", formatState(pr))
	line("Updated", pr.UpdatedAt.Local().Format("2006-01-02 15:04"))
	if len(pr.Labels) > 0 {
		line("Labels", strings.Join(pr.Labels, ", "))
	}
	line("URL", pr.URL)

	if len(pr.Reviews) > 0 {
		sb.WriteString("\n")
		sb.WriteString(dividerStyle.Render(strings.Repeat("─", 40)))
		sb.WriteString("\n")
		sb.WriteString(previewLabelStyle.Render("Reviews:"))
		sb.WriteString("\n")
		for _, r := range pr.Reviews {
			icon := reviewStateIcon(r.State)
			sb.WriteString(fmt.Sprintf("  %s %-20s %s\n", icon, r.Reviewer, r.State))
		}
		if len(pr.ReviewComments) > 0 {
			sb.WriteString(previewLabelStyle.Render(fmt.Sprintf("Review Comments: %d件\n", len(pr.ReviewComments))))
		}
	}

	if pr.Body != "" {
		sb.WriteString("\n")
		sb.WriteString(dividerStyle.Render(strings.Repeat("─", 40)))
		sb.WriteString("\n")
		body := pr.Body
		if len(body) > 2000 {
			body = body[:2000] + "\n…"
		}
		sb.WriteString(body)
	}

	return sb.String()
}

func (m model) buildIssuePreview(issue ghclient.Issue) string {
	var sb strings.Builder

	sb.WriteString(previewTitleStyle.Render(fmt.Sprintf("#%d %s", issue.Number, issue.Title)))
	sb.WriteString("\n\n")

	line := func(label, value string) {
		sb.WriteString(previewLabelStyle.Render(label+": "))
		sb.WriteString(previewValueStyle.Render(value))
		sb.WriteString("\n")
	}

	line("Type", "Issue")
	line("Author", issue.Author)
	line("State", issue.State)
	line("Updated", issue.UpdatedAt.Local().Format("2006-01-02 15:04"))
	if len(issue.Labels) > 0 {
		line("Labels", strings.Join(issue.Labels, ", "))
	}
	line("URL", issue.URL)

	if issue.Body != "" {
		sb.WriteString("\n")
		sb.WriteString(dividerStyle.Render(strings.Repeat("─", 40)))
		sb.WriteString("\n")
		body := issue.Body
		if len(body) > 2000 {
			body = body[:2000] + "\n…"
		}
		sb.WriteString(body)
	}

	return sb.String()
}

func reviewStateIcon(state string) string {
	switch state {
	case "APPROVED":
		return "✅"
	case "CHANGES_REQUESTED":
		return "❌"
	default:
		return "💬"
	}
}

func formatState(pr ghclient.PR) string {
	if pr.IsDraft {
		return pr.State + " (draft)"
	}
	return pr.State
}

func (m model) View() string {
	if !m.ready {
		return "Loading…"
	}
	if m.quitting {
		return ""
	}

	listPane := m.list.View()
	previewPane := m.preview.View()

	divider := dividerStyle.Render(strings.Repeat("│\n", m.height))
	divLines := strings.Split(divider, "\n")
	contentH := m.height - 3
	if contentH < len(divLines) {
		divLines = divLines[:contentH]
	}
	divider = strings.Join(divLines, "\n")

	top := lipgloss.JoinHorizontal(lipgloss.Top, listPane, divider, previewPane)

	var bottom string
	if m.closeMode {
		var itemType string
		var itemNum int
		switch sel := m.list.SelectedItem().(type) {
		case prItem:
			itemType = "PR"
			itemNum = sel.pr.Number
		case issueItem:
			itemType = "Issue"
			itemNum = sel.issue.Number
		}
		prompt := confirmPromptStyle.Render(fmt.Sprintf("%s #%d を close します。番号を入力 (Esc でキャンセル): ", itemType, itemNum))
		bottom = prompt + m.closeInput.View()
		if m.closeErr != "" {
			bottom += "  " + errorStyle.Render(m.closeErr)
		}
	} else if m.closeErr != "" {
		bottom = errorStyle.Render(m.closeErr)
	} else {
		bottom = helpStyle.Render("↑/↓ navigate · enter open in browser · c close PR/Issue · / filter · q quit")
	}

	return top + "\n" + bottom
}
