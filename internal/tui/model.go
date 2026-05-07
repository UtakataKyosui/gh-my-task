package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/UtakataKyosui/gh-my-task/internal/ghclient"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cli/go-gh/v2/pkg/browser"
)

type prItem struct {
	pr ghclient.PR
}

func (p prItem) FilterValue() string { return p.pr.Title }
func (p prItem) Title() string       { return badge(p.pr) + " " + p.pr.Title }
func (p prItem) Description() string {
	ago := humanTime(p.pr.UpdatedAt)
	draft := ""
	if p.pr.IsDraft {
		draft = " " + draftStyle.String()
	}
	return fmt.Sprintf("#%d · %s · %s%s", p.pr.Number, p.pr.Author, ago, draft)
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

type model struct {
	repo     string
	list     list.Model
	preview  viewport.Model
	prs      []ghclient.PR
	ready    bool
	width    int
	height   int
	quitting bool
}

func Run(owner, name string, prs []ghclient.PR) error {
	items := make([]list.Item, len(prs))
	for i, pr := range prs {
		items[i] = prItem{pr}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	l := list.New(items, delegate, 0, 0)
	l.Title = fmt.Sprintf("My Tasks in %s/%s", owner, name)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.KeyMap.Quit = keys.Quit

	vp := viewport.New(0, 0)

	m := model{
		repo:    fmt.Sprintf("%s/%s", owner, name),
		list:    l,
		preview: vp,
		prs:     prs,
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
		// reserve 2 lines for help bar + title
		contentH := msg.Height - 3
		m.list.SetSize(listW, contentH)
		m.preview.Width = previewW
		m.preview.Height = contentH
		m.ready = true
		m.preview.SetContent(m.buildPreview())

	case tea.KeyMsg:
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

func (m model) buildPreview() string {
	if len(m.prs) == 0 {
		return "No PRs found."
	}
	sel, ok := m.list.SelectedItem().(prItem)
	if !ok {
		return ""
	}
	pr := sel.pr

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
	line("Category", strings.Join(catLabels, ", "))
	line("Author", pr.Author)
	line("State", formatState(pr))
	line("Updated", pr.UpdatedAt.Local().Format("2006-01-02 15:04"))
	if len(pr.Labels) > 0 {
		line("Labels", strings.Join(pr.Labels, ", "))
	}
	line("URL", pr.URL)

	if pr.Body != "" {
		sb.WriteString("\n")
		sb.WriteString(dividerStyle.Render(strings.Repeat("─", 40)))
		sb.WriteString("\n")
		// truncate body to avoid huge previews
		body := pr.Body
		if len(body) > 2000 {
			body = body[:2000] + "\n…"
		}
		sb.WriteString(body)
	}

	return sb.String()
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
	// trim divider to content height
	divLines := strings.Split(divider, "\n")
	contentH := m.height - 3
	if contentH < len(divLines) {
		divLines = divLines[:contentH]
	}
	divider = strings.Join(divLines, "\n")

	top := lipgloss.JoinHorizontal(lipgloss.Top, listPane, divider, previewPane)

	help := helpStyle.Render("↑/↓ navigate · enter open in browser · / filter · q quit")

	return top + "\n" + help
}
