package diff

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Kind uint8

const (
	Context Kind = iota
	Add
	Delete
)

const (
	SideLeft  = "LEFT"
	SideRight = "RIGHT"
)

type Line struct {
	Kind    Kind
	Content string
	OldLine int
	NewLine int
}

func (l Line) Side() string {
	if l.Kind == Delete {
		return SideLeft
	}
	return SideRight
}

func (l Line) AnchorLine() int {
	if l.Kind == Delete {
		return l.OldLine
	}
	return l.NewLine
}

type Hunk struct {
	Header   string
	Section  string
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []Line
}

var hunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(?: ?(.*))?$`)

func Parse(patch string) ([]Hunk, error) {
	if strings.TrimSpace(patch) == "" {
		return nil, nil
	}

	var (
		hunks   []Hunk
		current *Hunk
		oldLine int
		newLine int
	)

	for _, raw := range strings.Split(patch, "\n") {
		line := strings.TrimSuffix(raw, "\r")

		if strings.HasPrefix(line, "@@") {
			m := hunkHeader.FindStringSubmatch(line)
			if m == nil {
				return nil, fmt.Errorf("malformed hunk header: %q", line)
			}

			h := Hunk{
				Header:   line,
				Section:  m[5],
				OldStart: atoi(m[1]),
				OldCount: countOrOne(m[2]),
				NewStart: atoi(m[3]),
				NewCount: countOrOne(m[4]),
			}
			hunks = append(hunks, h)
			current = &hunks[len(hunks)-1]

			oldLine = current.OldStart
			newLine = current.NewStart
			continue
		}

		if current == nil {
			continue
		}

		if strings.HasPrefix(line, `\`) {
			continue
		}

		switch {
		case line == "":
			current.Lines = append(current.Lines, Line{
				Kind:    Context,
				Content: "",
				OldLine: oldLine,
				NewLine: newLine,
			})
			oldLine++
			newLine++

		case line[0] == '+':
			current.Lines = append(current.Lines, Line{
				Kind:    Add,
				Content: line[1:],
				NewLine: newLine,
			})
			newLine++

		case line[0] == '-':
			current.Lines = append(current.Lines, Line{
				Kind:    Delete,
				Content: line[1:],
				OldLine: oldLine,
			})
			oldLine++

		case line[0] == ' ':
			current.Lines = append(current.Lines, Line{
				Kind:    Context,
				Content: line[1:],
				OldLine: oldLine,
				NewLine: newLine,
			})
			oldLine++
			newLine++

		default:
			return nil, fmt.Errorf("unexpected line prefix in patch: %q", line)
		}
	}

	return hunks, nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func countOrOne(s string) int {
	if s == "" {
		return 1
	}
	return atoi(s)
}

func Stats(hunks []Hunk) (additions, deletions int) {
	for _, h := range hunks {
		for _, l := range h.Lines {
			switch l.Kind {
			case Add:
				additions++
			case Delete:
				deletions++
			}
		}
	}
	return additions, deletions
}
