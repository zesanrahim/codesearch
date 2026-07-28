package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"codesearch/internal/diff"
	"codesearch/internal/ghapi"
	"codesearch/internal/github"
)

type Config struct {
	Addr      string
	Org       string
	Token     string
	StaticDir string
}

type Server struct {
	cfg   Config
	gh    *ghapi.Client
	cache *cache

	viewerOnce sync.Once
	viewer     string
}

const (
	inboxTTL    = 30 * time.Second
	prTTL       = 45 * time.Second
	loadTimeout = 25 * time.Second
)

func detached() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), loadTimeout)
}

func (s *Server) viewerLogin(ctx context.Context) string {
	s.viewerOnce.Do(func() {
		user, err := s.gh.CurrentUser(ctx)
		if err != nil {
			log.Printf("current user: %v", err)
			return
		}
		s.viewer = user.Login
	})
	return s.viewer
}

func NewServer(cfg Config) (*Server, error) {
	if cfg.Token == "" {
		return nil, errors.New("no GitHub token: set GITHUB_TOKEN")
	}
	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}
	return &Server{cfg: cfg, gh: ghapi.New(cfg.Token), cache: newCache()}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/inbox", s.handleInbox)
	mux.HandleFunc("GET /api/pr/{owner}/{repo}/{number}", s.handlePullRequest)

	if s.cfg.StaticDir != "" {
		mux.Handle("/", spaHandler(s.cfg.StaticDir))
	}

	return withCORS(withLogging(mux))
}

func (s *Server) ListenAndServe() error {
	srv := &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	fmt.Fprintf(os.Stderr, "codesearch serving on http://localhost%s\n", s.cfg.Addr)
	if s.cfg.Org != "" {
		fmt.Fprintf(os.Stderr, "inbox org: %s\n", s.cfg.Org)
	}
	return srv.ListenAndServe()
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	rate := s.gh.Rate()
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"org":  s.cfg.Org,
		"rate": rate,
	})
}

type inboxItem struct {
	Owner     string    `json:"owner"`
	Repo      string    `json:"repo"`
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Author    string    `json:"author"`
	AvatarURL string    `json:"avatarUrl"`
	Draft     bool      `json:"draft"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Indexed   bool      `json:"indexed"`
}

func (s *Server) handleInbox(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	repo := r.URL.Query().Get("repo")
	org := r.URL.Query().Get("org")
	if org == "" {
		org = s.cfg.Org
	}

	var key string
	switch {
	case owner != "" && repo != "":
		key = "inbox:repo:" + owner + "/" + repo
	case org != "":
		key = "inbox:org:" + org
	default:
		writeError(w, http.StatusBadRequest, "provide ?org= or ?owner=&repo=")
		return
	}

	payload, err := s.cache.load(key, inboxTTL, func() (any, error) {
		ctx, cancel := detached()
		defer cancel()
		return s.buildInbox(ctx, org, owner, repo)
	})
	if err != nil {
		writeUpstreamError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) buildInbox(ctx context.Context, org, owner, repo string) (any, error) {
	var (
		issues []ghapi.SearchIssue
		err    error
	)

	if owner != "" && repo != "" {
		issues, err = s.gh.RepoPullRequests(ctx, owner, repo)
	} else {
		issues, err = s.gh.OrgPullRequests(ctx, org)
	}
	if err != nil {
		return nil, err
	}

	indexed := indexedRepos()

	items := make([]inboxItem, 0, len(issues))
	for _, issue := range issues {
		o, rp, err := issue.OwnerRepo()
		if err != nil {
			continue
		}

		items = append(items, inboxItem{
			Owner:     o,
			Repo:      rp,
			Number:    issue.Number,
			Title:     issue.Title,
			Author:    issue.User.Login,
			AvatarURL: issue.User.AvatarURL,
			Draft:     issue.Draft,
			URL:       issue.HTMLURL,
			CreatedAt: issue.CreatedAt,
			UpdatedAt: issue.UpdatedAt,
			Indexed:   indexed[o+"/"+rp],
		})
	}

	return map[string]any{
		"items":  items,
		"viewer": s.viewerLogin(ctx),
		"rate":   s.gh.Rate(),
	}, nil
}

func indexedRepos() map[string]bool {
	names, err := github.IndexedRepoNames()
	if err != nil {
		return nil
	}
	return names
}

type prLine struct {
	Kind    string `json:"kind"`
	Content string `json:"content"`
	OldLine int    `json:"oldLine"`
	NewLine int    `json:"newLine"`
	Side    string `json:"side"`
	Anchor  int    `json:"anchor"`
}

type prHunk struct {
	Header   string   `json:"header"`
	Section  string   `json:"section"`
	OldStart int      `json:"oldStart"`
	NewStart int      `json:"newStart"`
	Lines    []prLine `json:"lines"`
}

type prFile struct {
	Path         string   `json:"path"`
	PreviousPath string   `json:"previousPath,omitempty"`
	Status       string   `json:"status"`
	Additions    int      `json:"additions"`
	Deletions    int      `json:"deletions"`
	Binary       bool     `json:"binary"`
	Hunks        []prHunk `json:"hunks"`
}

type prComment struct {
	ID        int64     `json:"id"`
	Path      string    `json:"path"`
	Body      string    `json:"body"`
	Line      int       `json:"line"`
	Side      string    `json:"side"`
	Author    string    `json:"author"`
	AvatarURL string    `json:"avatarUrl"`
	URL       string    `json:"url"`
	InReplyTo int64     `json:"inReplyTo,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

func (s *Server) handlePullRequest(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "pull request number must be an integer")
		return
	}

	key := fmt.Sprintf("pr:%s/%s/%d", owner, repo, number)
	payload, err := s.cache.load(key, prTTL, func() (any, error) {
		ctx, cancel := detached()
		defer cancel()
		return s.buildPullRequest(ctx, owner, repo, number)
	})
	if err != nil {
		writeUpstreamError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) buildPullRequest(ctx context.Context, owner, repo string, number int) (any, error) {
	var (
		pr          *ghapi.PullRequest
		files       []ghapi.File
		comments    []ghapi.ReviewComment
		errPR       error
		errFiles    error
		errComments error
		wg          sync.WaitGroup
	)

	wg.Add(3)
	go func() {
		defer wg.Done()
		pr, errPR = s.gh.PullRequest(ctx, owner, repo, number)
	}()
	go func() {
		defer wg.Done()
		files, errFiles = s.gh.PullRequestFiles(ctx, owner, repo, number)
	}()
	go func() {
		defer wg.Done()
		comments, errComments = s.gh.ReviewComments(ctx, owner, repo, number)
	}()
	wg.Wait()

	if err := errors.Join(errPR, errFiles, errComments); err != nil {
		return nil, err
	}

	outFiles := make([]prFile, 0, len(files))
	for _, f := range files {
		hunks, err := diff.Parse(f.Patch)
		if err != nil {
			log.Printf("diff parse %s/%s#%d %s: %v", owner, repo, number, f.Filename, err)
		}

		outFiles = append(outFiles, prFile{
			Path:         f.Filename,
			PreviousPath: f.PreviousFilename,
			Status:       f.Status,
			Additions:    f.Additions,
			Deletions:    f.Deletions,
			Binary:       f.Patch == "" && f.Changes > 0,
			Hunks:        toHunks(hunks),
		})
	}

	outComments := make([]prComment, 0, len(comments))
	for _, c := range comments {
		outComments = append(outComments, prComment{
			ID:        c.ID,
			Path:      c.Path,
			Body:      c.Body,
			Line:      c.Line,
			Side:      c.Side,
			Author:    c.User.Login,
			AvatarURL: c.User.AvatarURL,
			URL:       c.HTMLURL,
			InReplyTo: c.InReplyToID,
			CreatedAt: c.CreatedAt,
		})
	}

	return map[string]any{
		"owner":        owner,
		"repo":         repo,
		"number":       pr.Number,
		"title":        pr.Title,
		"body":         pr.Body,
		"author":       pr.User.Login,
		"avatarUrl":    pr.User.AvatarURL,
		"state":        pr.State,
		"draft":        pr.Draft,
		"head":         map[string]string{"ref": pr.Head.Ref, "sha": pr.Head.SHA},
		"base":         map[string]string{"ref": pr.Base.Ref, "sha": pr.Base.SHA},
		"additions":    pr.Additions,
		"deletions":    pr.Deletions,
		"changedFiles": pr.ChangedFiles,
		"url":          pr.HTMLURL,
		"createdAt":    pr.CreatedAt,
		"files":        outFiles,
		"comments":     outComments,
		"rate":         s.gh.Rate(),
	}, nil
}

func toHunks(hunks []diff.Hunk) []prHunk {
	out := make([]prHunk, 0, len(hunks))
	for _, h := range hunks {
		lines := make([]prLine, 0, len(h.Lines))
		for _, l := range h.Lines {
			lines = append(lines, prLine{
				Kind:    kindName(l.Kind),
				Content: l.Content,
				OldLine: l.OldLine,
				NewLine: l.NewLine,
				Side:    l.Side(),
				Anchor:  l.AnchorLine(),
			})
		}

		out = append(out, prHunk{
			Header:   h.Header,
			Section:  h.Section,
			OldStart: h.OldStart,
			NewStart: h.NewStart,
			Lines:    lines,
		})
	}
	return out
}

func kindName(k diff.Kind) string {
	switch k {
	case diff.Add:
		return "add"
	case diff.Delete:
		return "del"
	default:
		return "ctx"
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeUpstreamError(w http.ResponseWriter, err error) {
	var apiErr *ghapi.Error
	if errors.As(err, &apiErr) {
		writeError(w, apiErr.StatusCode, apiErr.Error())
		return
	}
	writeError(w, http.StatusBadGateway, err.Error())
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func spaHandler(dir string) http.Handler {
	files := http.FileServer(http.Dir(dir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(dir + r.URL.Path); err == nil || r.URL.Path == "/" {
			files.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, dir+"/index.html")
	})
}
