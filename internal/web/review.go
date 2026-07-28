package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"codesearch/internal/ghapi"
	"codesearch/internal/review"
)

func reviewKey(r *http.Request) (review.Key, error) {
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil {
		return review.Key{}, fmt.Errorf("pull request number must be an integer")
	}
	return review.Key{
		Owner:  r.PathValue("owner"),
		Repo:   r.PathValue("repo"),
		Number: number,
	}, nil
}

func (s *Server) invalidatePR(k review.Key) {
	s.cache.invalidate(fmt.Sprintf("pr:%s/%s/%d", k.Owner, k.Repo, k.Number))
}

func (s *Server) headSHA(ctx context.Context, k review.Key) (string, error) {
	pr, err := s.gh.PullRequest(ctx, k.Owner, k.Repo, k.Number)
	if err != nil {
		return "", err
	}
	return pr.Head.SHA, nil
}

func (s *Server) handleCreateComment(w http.ResponseWriter, r *http.Request) {
	key, err := reviewKey(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var payload struct {
		Path     string `json:"path"`
		Line     int    `json:"line"`
		Side     string `json:"side"`
		Body     string `json:"body"`
		CommitID string `json:"commitId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid comment payload")
		return
	}

	if payload.Path == "" || payload.Line <= 0 {
		writeError(w, http.StatusBadRequest, "comment needs a path and a positive line")
		return
	}
	if strings.TrimSpace(payload.Body) == "" {
		writeError(w, http.StatusBadRequest, "comment body is empty")
		return
	}
	if payload.Side != ghapi.SideLeft && payload.Side != ghapi.SideRight {
		payload.Side = ghapi.SideRight
	}

	ctx, cancel := detached()
	defer cancel()

	commit := payload.CommitID
	if commit == "" {
		commit, err = s.headSHA(ctx, key)
		if err != nil {
			writeUpstreamError(w, err)
			return
		}
	}

	created, err := s.gh.CreateReviewComment(ctx, key.Owner, key.Repo, key.Number, ghapi.CommentRequest{
		Body:     payload.Body,
		CommitID: commit,
		Path:     payload.Path,
		Line:     payload.Line,
		Side:     payload.Side,
	})
	if err != nil {
		writeUpstreamError(w, err)
		return
	}

	if _, err := review.Delete(key, payload.Path, payload.Line, payload.Side); err != nil {
		log.Printf("clearing autosaved draft: %v", err)
	}
	s.invalidatePR(key)

	writeJSON(w, http.StatusOK, created)
}

func (s *Server) handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	key, err := reviewKey(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "comment id must be an integer")
		return
	}

	ctx, cancel := detached()
	defer cancel()

	if err := s.gh.DeleteReviewComment(ctx, key.Owner, key.Repo, id); err != nil {
		writeUpstreamError(w, err)
		return
	}
	s.invalidatePR(key)

	writeJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

func (s *Server) handleSubmitReview(w http.ResponseWriter, r *http.Request) {
	key, err := reviewKey(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var payload struct {
		Event   string `json:"event"`
		Summary string `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid submit payload")
		return
	}

	event := strings.ToUpper(strings.TrimSpace(payload.Event))
	switch event {
	case ghapi.EventApprove, ghapi.EventRequestChanges, ghapi.EventComment:
	default:
		writeError(w, http.StatusBadRequest,
			"event must be APPROVE, REQUEST_CHANGES or COMMENT")
		return
	}

	if event == ghapi.EventComment && strings.TrimSpace(payload.Summary) == "" {
		writeError(w, http.StatusBadRequest,
			"a plain comment review needs a summary")
		return
	}

	ctx, cancel := detached()
	defer cancel()

	created, err := s.gh.SubmitReview(ctx, key.Owner, key.Repo, key.Number, ghapi.ReviewRequest{
		Body:  payload.Summary,
		Event: event,
	})
	if err != nil {
		writeUpstreamError(w, err)
		return
	}
	s.invalidatePR(key)

	writeJSON(w, http.StatusOK, created)
}

func (s *Server) handleGetDrafts(w http.ResponseWriter, r *http.Request) {
	key, err := reviewKey(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	state, err := review.Load(key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) handlePutDraft(w http.ResponseWriter, r *http.Request) {
	key, err := reviewKey(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var d review.Draft
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		writeError(w, http.StatusBadRequest, "invalid draft payload")
		return
	}

	state, err := review.Upsert(key, d)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) handleDeleteDraft(w http.ResponseWriter, r *http.Request) {
	key, err := reviewKey(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	q := r.URL.Query()
	line, err := strconv.Atoi(q.Get("line"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "line must be an integer")
		return
	}

	state, err := review.Delete(key, q.Get("path"), line, q.Get("side"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, state)
}
