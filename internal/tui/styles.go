package tui

import "github.com/charmbracelet/lipgloss"

var (
	purple = lipgloss.Color("#7D56F4")
	cyan   = lipgloss.Color("#00ADD8")
	gray   = lipgloss.Color("#3C3C3C")
	green  = lipgloss.Color("#04B575")
	yellow = lipgloss.Color("#FDFF90")

	pageStyle = lipgloss.NewStyle().
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF")).
			Background(purple).
			Padding(0, 1).
			Bold(true)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple).
			Padding(0, 1)

	resultStyle = lipgloss.NewStyle().
			Foreground(cyan).
			Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF")).
			Background(purple).
			Padding(0, 1).
			Bold(true)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(gray).
				Padding(0, 1)

	filePathStyle = lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true)

	lineNumStyle = lipgloss.NewStyle().
			Foreground(yellow)

	linkStyle = lipgloss.NewStyle().
			Foreground(green).
			Italic(true)

	searchDividerStyle = lipgloss.NewStyle().
				Foreground(gray)

	codeBlockStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#555")).
			Padding(0, 1).
			Foreground(lipgloss.Color("#CCC"))

	codeLineNumStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666")).
				Width(5).
				Align(lipgloss.Right)

	codeMatchLineStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFF")).
				Background(lipgloss.Color("#3D3D6B")).
				Bold(true)
)
