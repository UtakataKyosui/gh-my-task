package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("170")).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	badgeAuthor = lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Bold(true).
			SetString("[A]")

	badgeReview = lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Bold(true).
			SetString("[R]")

	badgeBoth = lipgloss.NewStyle().
			Foreground(lipgloss.Color("213")).
			Bold(true).
			SetString("[AR]")

	draftStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			SetString("draft")

	reviewApprovedIndicator = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82")).
				Bold(true).
				SetString("✓")

	reviewChangesIndicator = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true).
				SetString("!")

	reviewCommentedIndicator = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245")).
				SetString("~")

	reviewApprovedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82"))

	reviewChangesStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196"))

	reviewCommentedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	previewTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205"))

	previewLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))

	previewValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255"))

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("237"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	confirmPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)
