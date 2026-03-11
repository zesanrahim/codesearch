package tui

import (
	"fmt"
	"strings"

	"codesearch/internal/engine"

	"github.com/charmbracelet/lipgloss"
)

func hyperlink(url, text string) string {
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}

func (m Model) View() string {
	if m.Width == 0 {
		return "Initializing..."
	}

	if m.Fullscreen {
		return m.renderFullscreen()
	}

	const chromeLines = 12

	header := lipgloss.JoinHorizontal(lipgloss.Center,
		titleStyle.Render("CODESEARCH"),
	)

	var addTab, searchTab string
	if m.Mode == ModeAdd {
		addTab = activeTabStyle.Render("Add Repos")
		searchTab = inactiveTabStyle.Render("Search")
	} else {
		addTab = inactiveTabStyle.Render("Add Repos")
		searchTab = activeTabStyle.Render("Search")
	}
	tabs := lipgloss.JoinHorizontal(lipgloss.Top, addTab, "  ", searchTab,
		lipgloss.NewStyle().Foreground(gray).MarginLeft(2).Render("(Tab to switch)"),
	)

	var hint string
	if m.Mode == ModeAdd {
		hint = "Press Enter to add repo • Press Esc to quit"
	} else {
		hint = "Enter to search • ↑/↓ select • ` fullscreen • +/- context • Esc quit"
	}
	searchSection := lipgloss.JoinVertical(lipgloss.Left,
		inputStyle.Width(m.Width-10).Render(m.TextInput.View()),
		lipgloss.NewStyle().Italic(true).Foreground(gray).Render(hint),
	)

	status := lipgloss.NewStyle().Foreground(cyan).Render(m.Status)

	var progressBar string
	if m.Indexing {
		progressBar = m.Progress.ViewAs(m.IndexPercent)
	}

	bodyHeight := m.Height - chromeLines
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	var body string
	if m.Mode == ModeAdd {
		body = m.renderAddMode()
	} else {
		body = m.renderSearchMode()
	}
	body = lipgloss.NewStyle().
		MaxHeight(bodyHeight).
		Width(m.Width - 4).
		Render(body)

	repoCount := fmt.Sprintf("%d repo(s) indexed", len(m.Repos))
	footer := lipgloss.NewStyle().
		Width(m.Width - 4).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(gray).
		Foreground(gray).
		Render(repoCount)

	full := lipgloss.JoinVertical(lipgloss.Left,
		header, "", tabs, "", searchSection, "", status, progressBar, "", body, footer,
	)

	return pageStyle.
		Width(m.Width).
		MaxHeight(m.Height).
		Render(full)
}

func (m Model) renderAddMode() string {
	if len(m.Results) == 0 {
		return lipgloss.NewStyle().Italic(true).Foreground(gray).Render("No repositories added yet.")
	}

	var s string
	for i, result := range m.Results {
		s += resultStyle.Render(fmt.Sprintf("%d. %s", i+1, result)) + "\n"
	}
	return s
}

func (m Model) renderSearchMode() string {
	if len(m.SearchResults) == 0 {
		if m.SearchQuery != "" {
			return lipgloss.NewStyle().Italic(true).Foreground(gray).
				Render(fmt.Sprintf("No results for '%s'.", m.SearchQuery))
		}
		if len(m.Repos) == 0 {
			return lipgloss.NewStyle().Italic(true).Foreground(gray).
				Render("No repos indexed. Switch to Add Repos tab first.")
		}
		return lipgloss.NewStyle().Italic(true).Foreground(gray).
			Render("Enter a search query above.")
	}

	maxVisible := m.maxSearchResults()
	end := m.ScrollOffset + maxVisible
	if end > len(m.SearchResults) {
		end = len(m.SearchResults)
	}
	visible := m.SearchResults[m.ScrollOffset:end]

	var s string
	s += searchDividerStyle.Render(fmt.Sprintf(
		"Showing %d–%d of %d results",
		m.ScrollOffset+1, end, len(m.SearchResults),
	)) + "\n\n"

	for i, r := range visible {
		num := m.ScrollOffset + i + 1
		blobURL := r.GetBlobURL()
		fileLine := fmt.Sprintf("%s:%d", r.FilePath, r.Line)

		var styledFileLine string
		if blobURL != "" {
			styledFileLine = hyperlink(blobURL, filePathStyle.Render(fileLine))
		} else {
			styledFileLine = filePathStyle.Render(fileLine)
		}

		selected := ""
		if m.Mode == ModeSearch && m.SelectedResult == num-1 {
			selected = " > "
		} else {
			selected = "   "
		}

		line := fmt.Sprintf("%s%s  %s",
			selected,
			styledFileLine,
			lineNumStyle.Render(fmt.Sprintf("(line %d)", r.Line)),
		)

		s += resultStyle.Render(fmt.Sprintf("%d.", num)) + line + "\n"

		if m.ContextLines > 0 && r.Context != "" {
			snippet := renderCodeSnippet(r, m.ContextLines, m.Width-8)
			s += snippet + "\n"
		}

		s += "\n"
	}

	if m.ScrollOffset > 0 || end < len(m.SearchResults) {
		s += searchDividerStyle.Render("  ↑/↓ to scroll") + "\n"
	}

	ctxInfo := searchDividerStyle.Render(
		fmt.Sprintf("  Context: %d lines  (+/- to adjust)", m.ContextLines),
	)
	s += ctxInfo + "\n"

	return s
}

func renderCodeSnippet(r engine.SearchResult, contextLines int, maxWidth int) string {
	if r.Context == "" {
		return ""
	}

	allLines := strings.Split(r.Context, "\n")

	const window = 10
	contextStartLine := r.Line - window
	if contextStartLine < 1 {
		contextStartLine = 1
	}

	matchIdx := r.Line - contextStartLine
	if matchIdx < 0 {
		matchIdx = 0
	}
	if matchIdx >= len(allLines) {
		matchIdx = len(allLines) - 1
	}

	start := matchIdx - contextLines
	if start < 0 {
		start = 0
	}
	end := matchIdx + contextLines + 1
	if end > len(allLines) {
		end = len(allLines)
	}
	displayLines := allLines[start:end]

	var rows []string
	for i, line := range displayLines {
		fileLineNum := contextStartLine + start + i
		numStr := codeLineNumStyle.Render(fmt.Sprintf("%d", fileLineNum))

		codeWidth := maxWidth - 8
		if codeWidth < 20 {
			codeWidth = 20
		}
		if len(line) > codeWidth {
			line = line[:codeWidth-1] + "…"
		}

		if fileLineNum == r.Line {
			rows = append(rows, numStr+" "+codeMatchLineStyle.Render(" "+line+" "))
		} else {
			rows = append(rows, numStr+"  "+line)
		}
	}

	block := strings.Join(rows, "\n")
	return codeBlockStyle.Width(maxWidth).Render(block)
}

func (m Model) renderFullscreen() string {
	if m.SelectedResult >= len(m.SearchResults) {
		return "No result selected."
	}

	r := m.SearchResults[m.SelectedResult]

	blobURL := r.GetBlobURL()
	fileLine := fmt.Sprintf("%s:%d", r.FilePath, r.Line)
	var headerText string
	if blobURL != "" {
		headerText = hyperlink(blobURL, filePathStyle.Render(fileLine))
	} else {
		headerText = filePathStyle.Render(fileLine)
	}

	header := lipgloss.JoinHorizontal(lipgloss.Center,
		titleStyle.Render("FILE PREVIEW"),
		lipgloss.NewStyle().MarginLeft(2).Render(headerText),
	)

	hint := lipgloss.NewStyle().Italic(true).Foreground(gray).
		Render("` or Esc to go back • ↑/↓ to scroll")

	if r.Context == "" {
		body := lipgloss.NewStyle().Italic(true).Foreground(gray).
			Render("No file context available for this result.")
		full := lipgloss.JoinVertical(lipgloss.Left, header, "", hint, "", body)
		return pageStyle.Width(m.Width).MaxHeight(m.Height).Render(full)
	}

	allLines := strings.Split(r.Context, "\n")

	const window = 10
	contextStartLine := r.Line - window
	if contextStartLine < 1 {
		contextStartLine = 1
	}

	codeHeight := m.Height - 6
	if codeHeight < 1 {
		codeHeight = 1
	}

	scrollEnd := m.FullscreenScroll + codeHeight
	if scrollEnd > len(allLines) {
		scrollEnd = len(allLines)
	}
	if m.FullscreenScroll >= len(allLines) {
		m.FullscreenScroll = len(allLines) - 1
	}
	visibleLines := allLines[m.FullscreenScroll:scrollEnd]

	codeWidth := m.Width - 14
	if codeWidth < 20 {
		codeWidth = 20
	}

	var rows []string
	for i, line := range visibleLines {
		fileLineNum := contextStartLine + m.FullscreenScroll + i

		numStr := codeLineNumStyle.Render(fmt.Sprintf("%d", fileLineNum))

		if len(line) > codeWidth {
			line = line[:codeWidth-1] + "…"
		}

		if fileLineNum == r.Line {
			rows = append(rows, numStr+" "+codeMatchLineStyle.Render(" "+line+" "))
		} else {
			rows = append(rows, numStr+"  "+line)
		}
	}

	block := strings.Join(rows, "\n")
	codeView := codeBlockStyle.Width(m.Width - 6).Render(block)

	scrollInfo := searchDividerStyle.Render(fmt.Sprintf(
		"Line %d-%d of %d total context lines",
		contextStartLine+m.FullscreenScroll,
		contextStartLine+scrollEnd-1,
		len(allLines),
	))

	full := lipgloss.JoinVertical(lipgloss.Left,
		header, "", hint, "", codeView, scrollInfo,
	)

	return pageStyle.Width(m.Width).MaxHeight(m.Height).Render(full)
}
