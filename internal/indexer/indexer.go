package indexer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"codesearch/internal/github"
)

type Status string

const (
	StatusAbsent   Status = "absent"
	StatusQueued   Status = "queued"
	StatusCloning  Status = "cloning"
	StatusIndexing Status = "indexing"
	StatusReady    Status = "ready"
	StatusFailed   Status = "failed"
)

type Job struct {
	Owner      string    `json:"owner"`
	Repo       string    `json:"repo"`
	Status     Status    `json:"status"`
	Processed  int       `json:"processed"`
	Total      int       `json:"total"`
	Files      int       `json:"files"`
	Error      string    `json:"error,omitempty"`
	StartedAt  time.Time `json:"startedAt,omitzero"`
	FinishedAt time.Time `json:"finishedAt,omitzero"`
}

type Manager struct {
	token   string
	timeout time.Duration
	onReady func(owner, repo string)

	mu   sync.Mutex
	jobs map[string]*Job
	sem  chan struct{}
}

func (m *Manager) OnReady(fn func(owner, repo string)) {
	m.onReady = fn
}

func New(token string, workers int) *Manager {
	if workers < 1 {
		workers = 1
	}
	return &Manager{
		token:   token,
		timeout: 20 * time.Minute,
		jobs:    make(map[string]*Job),
		sem:     make(chan struct{}, workers),
	}
}

func key(owner, repo string) string { return owner + "/" + repo }

func (m *Manager) Status(owner, repo string) Job {
	m.mu.Lock()
	job, running := m.jobs[key(owner, repo)]
	if running {
		snapshot := *job
		m.mu.Unlock()
		return snapshot
	}
	m.mu.Unlock()

	if indexed(owner, repo) {
		return Job{Owner: owner, Repo: repo, Status: StatusReady}
	}
	return Job{Owner: owner, Repo: repo, Status: StatusAbsent}
}

func (m *Manager) Ensure(owner, repo string) Job {
	if owner == "" || repo == "" {
		return Job{Status: StatusFailed, Error: "owner and repo are required"}
	}

	m.mu.Lock()
	k := key(owner, repo)

	if job, running := m.jobs[k]; running {
		snapshot := *job
		m.mu.Unlock()
		return snapshot
	}

	if indexed(owner, repo) {
		m.mu.Unlock()
		return Job{Owner: owner, Repo: repo, Status: StatusReady}
	}

	job := &Job{
		Owner:     owner,
		Repo:      repo,
		Status:    StatusQueued,
		StartedAt: time.Now().UTC(),
	}
	m.jobs[k] = job
	snapshot := *job
	m.mu.Unlock()

	go m.run(k, owner, repo)
	return snapshot
}

func (m *Manager) update(k string, mutate func(*Job)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if job, ok := m.jobs[k]; ok {
		mutate(job)
	}
}

func (m *Manager) finish(k string, err error, files int) {
	m.mu.Lock()
	job, ok := m.jobs[k]
	if !ok {
		m.mu.Unlock()
		return
	}

	job.FinishedAt = time.Now().UTC()
	if err != nil {
		job.Status = StatusFailed
		job.Error = err.Error()
		m.mu.Unlock()
		return
	}

	job.Status = StatusReady
	job.Files = files
	owner, repo := job.Owner, job.Repo
	m.mu.Unlock()

	if m.onReady != nil {
		m.onReady(owner, repo)
	}

	time.AfterFunc(30*time.Second, func() {
		m.mu.Lock()
		if j, ok := m.jobs[k]; ok && j.Status == StatusReady {
			delete(m.jobs, k)
		}
		m.mu.Unlock()
	})
}

func (m *Manager) run(k, owner, repo string) {
	m.sem <- struct{}{}
	defer func() { <-m.sem }()

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
	r, err := github.NewRepo(cloneURL)
	if err != nil {
		m.finish(k, err, 0)
		return
	}

	m.update(k, func(j *Job) { j.Status = StatusCloning })
	if err := github.CloneRepoAuth(ctx, r, m.token); err != nil {
		m.finish(k, err, 0)
		return
	}

	m.update(k, func(j *Job) { j.Status = StatusIndexing })
	idx, err := github.IndexRepoWithProgress(ctx, r, func(processed, total int) {
		m.update(k, func(j *Job) {
			j.Processed = processed
			j.Total = total
		})
	})
	if err != nil {
		m.finish(k, err, 0)
		return
	}

	files := len(idx.FileBoundaries)
	idx.Close()
	m.finish(k, nil, files)
}

func indexed(owner, repo string) bool {
	names, err := github.IndexedRepoNames()
	if err != nil {
		return false
	}
	return names[key(owner, repo)]
}
