package tui

import (
	"codesearch/internal/engine"
	"codesearch/internal/github"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Mode int

const (
	ModeAdd    Mode = iota
	ModeSearch
)

type Model struct {
	TextInput      textinput.Model
	Width          int
	Height         int
	Results        []string
	Status         string
	Repos          []*github.Repo
	Mode           Mode
	SearchResults  []engine.SearchResult
	SearchQuery    string
	ScrollOffset   int
	ContextLines   int
	Progress       progress.Model
	Indexing       bool
	IndexPercent   float64
	SelectedResult int
	Fullscreen     bool
	FullscreenScroll int
}

func InitialModel() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter a GitHub repo URL (e.g., https://github.com/org/repo)..."
	ti.Focus()
	ti.Prompt = " > "

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
	)

	return Model{
		TextInput:     ti,
		Results:       []string{},
		Status:        "Enter a GitHub repository URL to clone and index.",
		Repos:         []*github.Repo{},
		Mode:          ModeAdd,
		SearchResults: []engine.SearchResult{},
		ContextLines:  3,
		Progress:      p,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, LoadCachedRepos())
}
