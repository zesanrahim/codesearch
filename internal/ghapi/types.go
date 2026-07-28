package ghapi

import (
	"fmt"
	"strings"
	"time"
)

type User struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

type Ref struct {
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}

type SearchResult struct {
	TotalCount        int           `json:"total_count"`
	IncompleteResults bool          `json:"incomplete_results"`
	Items             []SearchIssue `json:"items"`
}

type SearchIssue struct {
	Number        int       `json:"number"`
	Title         string    `json:"title"`
	State         string    `json:"state"`
	Draft         bool      `json:"draft"`
	User          User      `json:"user"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	RepositoryURL string    `json:"repository_url"`
	HTMLURL       string    `json:"html_url"`
}

func (s SearchIssue) OwnerRepo() (owner, repo string, err error) {
	const marker = "/repos/"
	i := strings.Index(s.RepositoryURL, marker)
	if i < 0 {
		return "", "", fmt.Errorf("unexpected repository_url %q", s.RepositoryURL)
	}

	parts := strings.Split(strings.Trim(s.RepositoryURL[i+len(marker):], "/"), "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("cannot parse owner and repo from %q", s.RepositoryURL)
	}
	return parts[0], parts[1], nil
}

type PullRequest struct {
	Number       int       `json:"number"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	State        string    `json:"state"`
	Draft        bool      `json:"draft"`
	Merged       bool      `json:"merged"`
	User         User      `json:"user"`
	Head         Ref       `json:"head"`
	Base         Ref       `json:"base"`
	Additions    int       `json:"additions"`
	Deletions    int       `json:"deletions"`
	ChangedFiles int       `json:"changed_files"`
	HTMLURL      string    `json:"html_url"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type File struct {
	SHA              string `json:"sha"`
	Filename         string `json:"filename"`
	PreviousFilename string `json:"previous_filename"`
	Status           string `json:"status"`
	Additions        int    `json:"additions"`
	Deletions        int    `json:"deletions"`
	Changes          int    `json:"changes"`
	Patch            string `json:"patch"`
}

type ReviewComment struct {
	ID          int64     `json:"id"`
	Path        string    `json:"path"`
	Body        string    `json:"body"`
	Line        int       `json:"line"`
	StartLine   int       `json:"start_line"`
	Side        string    `json:"side"`
	StartSide   string    `json:"start_side"`
	DiffHunk    string    `json:"diff_hunk"`
	CommitID    string    `json:"commit_id"`
	InReplyToID int64     `json:"in_reply_to_id"`
	User        User      `json:"user"`
	HTMLURL     string    `json:"html_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type IssueComment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	User      User      `json:"user"`
	HTMLURL   string    `json:"html_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

const (
	EventApprove        = "APPROVE"
	EventRequestChanges = "REQUEST_CHANGES"
	EventComment        = "COMMENT"

	SideLeft  = "LEFT"
	SideRight = "RIGHT"
)

type DraftComment struct {
	Path      string `json:"path"`
	Body      string `json:"body"`
	Line      int    `json:"line"`
	Side      string `json:"side,omitempty"`
	StartLine int    `json:"start_line,omitempty"`
	StartSide string `json:"start_side,omitempty"`
}

type CommentRequest struct {
	Body      string `json:"body"`
	CommitID  string `json:"commit_id"`
	Path      string `json:"path"`
	Line      int    `json:"line"`
	Side      string `json:"side,omitempty"`
	StartLine int    `json:"start_line,omitempty"`
	StartSide string `json:"start_side,omitempty"`
}

type ReviewRequest struct {
	CommitID string         `json:"commit_id,omitempty"`
	Body     string         `json:"body,omitempty"`
	Event    string         `json:"event"`
	Comments []DraftComment `json:"comments,omitempty"`
}

type Review struct {
	ID          int64     `json:"id"`
	Body        string    `json:"body"`
	State       string    `json:"state"`
	User        User      `json:"user"`
	HTMLURL     string    `json:"html_url"`
	CommitID    string    `json:"commit_id"`
	SubmittedAt time.Time `json:"submitted_at"`
}
