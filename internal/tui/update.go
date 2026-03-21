package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.TextInput.Width = m.Width - 20
		m.Progress.Width = m.Width - 16
		return m, nil

	case IndexResultMsg:
		m.Indexing = false
		m.IndexPercent = 0
		if msg.Err != nil {
			m.Status = fmt.Sprintf("[error] Failed: %s: %v", msg.RepoName, msg.Err)
		} else {
			m.Status = fmt.Sprintf("[ok] Successfully cloned and indexed %s", msg.RepoName)
			m.Results = append(m.Results, msg.Result)
			if msg.Repo != nil {
				m.Repos = append(m.Repos, msg.Repo)
			}
		}
		return m, nil

	case IndexProgressMsg:
		m.Indexing = true
		m.IndexPercent = msg.Percent
		if msg.Total > 0 {
			m.Status = fmt.Sprintf("Indexing... %d/%d files", msg.Processed, msg.Total)
		}
		return m, WaitForProgress()

	case SearchResultMsg:
		if msg.Err != nil {
			m.Status = fmt.Sprintf("[error] Search failed: %v", msg.Err)
		} else if len(msg.Results) == 0 {
			m.Status = fmt.Sprintf("No matches found for '%s'", msg.Query)
			m.SearchResults = nil
		} else {
			m.Status = fmt.Sprintf("[ok] Found %d matches for '%s'", len(msg.Results), msg.Query)
			m.SearchResults = msg.Results
			m.SearchQuery = msg.Query
			m.ScrollOffset = 0
		}
		return m, nil

	case SearchMultiLineResultMsg:
		if msg.Err != nil {
			m.Status = fmt.Sprintf("[error] Search failed: %v", msg.Err)
		} else if len(msg.Results) == 0 {
			m.Status = fmt.Sprintf("No matches found for %d-line query", msg.QueryLineCount)
			m.SearchResults = nil
		} else {
			m.Status = fmt.Sprintf("[ok] Found %d matches (%d-line query)", len(msg.Results), msg.QueryLineCount)
			m.SearchResults = msg.Results
			m.MultiLineMode = true
			m.ScrollOffset = 0
			m.SelectedResult = 0
		}
		return m, nil

	case CachedReposMsg:
		if len(msg.Repos) > 0 {
			m.Repos = msg.Repos
			m.Results = msg.Results
			m.Status = fmt.Sprintf("Loaded %d cached repo(s).", len(msg.Repos))
		}
		return m, nil

	case tea.KeyMsg:
		if m.Fullscreen {
			switch msg.Type {
			case tea.KeyUp:
				if m.FullscreenScroll > 0 {
					m.FullscreenScroll--
				}
				return m, nil
			case tea.KeyDown:
				m.FullscreenScroll++
				return m, nil
			case tea.KeyEsc:
				m.Fullscreen = false
				m.FullscreenScroll = 0
				return m, nil
			}
			if msg.Type == tea.KeyRunes && string(msg.Runes) == "`" {
				m.Fullscreen = false
				m.FullscreenScroll = 0
				return m, nil
			}
			return m, nil
		}

		switch msg.Type {
		case tea.KeyTab:
			if m.Mode == ModeAdd {
				m.Mode = ModeSearch
				m.TextInput.Placeholder = "Enter search query..."
				m.TextInput.SetValue("")
				m.Status = "Search across indexed repositories."
			} else {
				m.Mode = ModeAdd
				m.TextInput.Placeholder = "Enter a GitHub repo URL (e.g., https://github.com/org/repo)..."
				m.TextInput.SetValue("")
				m.SearchResults = nil
				m.Status = "Enter a GitHub repository URL to clone and index."
			}
			return m, nil

		case tea.KeyEnter:
			value := strings.TrimSpace(m.TextInput.Value())
			if value == "" {
				return m, nil
			}
			m.TextInput.SetValue("")

			if m.Mode == ModeSearch {
				if len(m.Repos) == 0 {
					m.Status = "[error] No repos indexed yet. Add a repo first (Tab to switch)."
					return m, nil
				}
				
				lines := strings.Split(value, "\n")
				var validLines []string
				for _, line := range lines {
					if strings.TrimSpace(line) != "" {
						validLines = append(validLines, line)
					}
				}
				
				if len(validLines) > 1 {
					m.Status = fmt.Sprintf("Searching for %d-line pattern...", len(validLines))
					m.SearchResults = nil
					m.MultiLineMode = true
					m.QueryLines = validLines
					return m, SearchReposMultiLine(validLines, m.Repos)
				} else {
					m.Status = fmt.Sprintf("Searching for '%s'...", value)
					m.SearchResults = nil
					m.MultiLineMode = false
					return m, SearchRepos(value, m.Repos)
				}
			}

			m.Status = fmt.Sprintf("Cloning and indexing: %s", value)
			m.Indexing = true
			m.IndexPercent = 0
			return m, tea.Batch(CloneAndIndex(value), WaitForProgress())

		case tea.KeyUp:
			if m.Mode == ModeSearch && len(m.SearchResults) > 0 {
				if m.SelectedResult > 0 {
					m.SelectedResult--
				}
				if m.SelectedResult < m.ScrollOffset {
					m.ScrollOffset = m.SelectedResult
				}
			}
			return m, nil

		case tea.KeyDown:
			if m.Mode == ModeSearch && len(m.SearchResults) > 0 {
				if m.SelectedResult < len(m.SearchResults)-1 {
					m.SelectedResult++
				}
				maxVisible := m.maxSearchResults()
				if m.SelectedResult >= m.ScrollOffset+maxVisible {
					m.ScrollOffset = m.SelectedResult - maxVisible + 1
				}
			}
			return m, nil

		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

		if msg.Type == tea.KeyRunes {
			switch string(msg.Runes) {
			case "`":
				if m.Mode == ModeSearch && len(m.SearchResults) > 0 && m.SelectedResult < len(m.SearchResults) {
					m.Fullscreen = true
					m.FullscreenScroll = 0
				}
				return m, nil
			case "+", "=":
				if m.Mode == ModeSearch && m.ContextLines < 10 {
					m.ContextLines++
					m.Status = fmt.Sprintf("Context: %d lines above/below match", m.ContextLines)
				}
				return m, nil
			case "-":
				if m.Mode == ModeSearch && m.ContextLines > 0 {
					m.ContextLines--
					m.Status = fmt.Sprintf("Context: %d lines above/below match", m.ContextLines)
				}
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.TextInput, cmd = m.TextInput.Update(msg)
	return m, cmd
}

func (m Model) maxSearchResults() int {
	const chromeLines = 12
	const searchChrome = 4

	bodyHeight := m.Height - chromeLines - searchChrome
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	linesPerResult := 2
	if m.ContextLines > 0 {
		linesPerResult += 2 + (m.ContextLines*2 + 1) + 1
	}
	if linesPerResult < 1 {
		linesPerResult = 1
	}

	count := bodyHeight / linesPerResult
	if count < 1 {
		return 1
	}
	return count
}
