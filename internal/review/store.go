package review

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"codesearch/internal/paths"
)

type Draft struct {
	Path      string    `json:"path"`
	Line      int       `json:"line"`
	Side      string    `json:"side"`
	Body      string    `json:"body"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type State struct {
	Summary string  `json:"summary"`
	Drafts  []Draft `json:"drafts"`
}

type Key struct {
	Owner  string
	Repo   string
	Number int
}

func (k Key) valid() error {
	if k.Owner == "" || k.Repo == "" || k.Number <= 0 {
		return errors.New("owner, repo and a positive number are required")
	}
	if strings.ContainsAny(k.Owner+k.Repo, `/\.`) {
		return fmt.Errorf("unsafe owner or repo: %q/%q", k.Owner, k.Repo)
	}
	return nil
}

func (k Key) path() string {
	return filepath.Join(paths.Root(), "review", k.Owner, k.Repo, fmt.Sprintf("%d.json", k.Number))
}

var mu sync.Mutex

func Load(k Key) (State, error) {
	if err := k.valid(); err != nil {
		return State{}, err
	}

	mu.Lock()
	defer mu.Unlock()
	return load(k)
}

func load(k Key) (State, error) {
	data, err := os.ReadFile(k.path())
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, err
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, fmt.Errorf("draft file %s is corrupt: %w", k.path(), err)
	}
	return s, nil
}

func store(k Key, s State) error {
	target := k.path()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(target), ".draft-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), target)
}

func Upsert(k Key, d Draft) (State, error) {
	if err := k.valid(); err != nil {
		return State{}, err
	}
	if d.Path == "" || d.Line <= 0 {
		return State{}, errors.New("draft needs a path and a positive line")
	}
	if strings.TrimSpace(d.Body) == "" {
		return State{}, errors.New("draft body is empty")
	}
	if d.Side != "LEFT" && d.Side != "RIGHT" {
		d.Side = "RIGHT"
	}
	d.UpdatedAt = time.Now().UTC()

	mu.Lock()
	defer mu.Unlock()

	s, err := load(k)
	if err != nil {
		return State{}, err
	}

	replaced := false
	for i := range s.Drafts {
		if sameAnchor(s.Drafts[i], d) {
			s.Drafts[i] = d
			replaced = true
			break
		}
	}
	if !replaced {
		s.Drafts = append(s.Drafts, d)
	}

	if err := store(k, s); err != nil {
		return State{}, err
	}
	return s, nil
}

func Delete(k Key, path string, line int, side string) (State, error) {
	if err := k.valid(); err != nil {
		return State{}, err
	}
	if side != "LEFT" && side != "RIGHT" {
		side = "RIGHT"
	}

	mu.Lock()
	defer mu.Unlock()

	s, err := load(k)
	if err != nil {
		return State{}, err
	}

	target := Draft{Path: path, Line: line, Side: side}
	kept := s.Drafts[:0]
	for _, d := range s.Drafts {
		if !sameAnchor(d, target) {
			kept = append(kept, d)
		}
	}
	s.Drafts = kept

	if err := store(k, s); err != nil {
		return State{}, err
	}
	return s, nil
}

func SetSummary(k Key, summary string) (State, error) {
	if err := k.valid(); err != nil {
		return State{}, err
	}

	mu.Lock()
	defer mu.Unlock()

	s, err := load(k)
	if err != nil {
		return State{}, err
	}
	s.Summary = summary

	if err := store(k, s); err != nil {
		return State{}, err
	}
	return s, nil
}

func Clear(k Key) error {
	if err := k.valid(); err != nil {
		return err
	}

	mu.Lock()
	defer mu.Unlock()

	if err := os.Remove(k.path()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func sameAnchor(a, b Draft) bool {
	return a.Path == b.Path && a.Line == b.Line && a.Side == b.Side
}
