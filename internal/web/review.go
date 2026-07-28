package web

import (
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

func (s *Server) handlePutSummary(w http.ResponseWriter, r *http.Request) {
	key, err := reviewKey(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var payload struct {
		Summary string `json:"summary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid summary payload")
		return
	}

	state, err := review.SetSummary(key, payload.Summary)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, state)
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

	state, err := review.Load(key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	summary := payload.Summary
	if summary == "" {
		summary = state.Summary
	}

	if len(state.Drafts) == 0 && strings.TrimSpace(summary) == "" {
		writeError(w, http.StatusBadRequest,
			"nothing to submit: add a comment or a summary")
		return
	}

	comments := make([]ghapi.DraftComment, 0, len(state.Drafts))
	for _, d := range state.Drafts {
		comments = append(comments, ghapi.DraftComment{
			Path: d.Path,
			Body: d.Body,
			Line: d.Line,
			Side: d.Side,
		})
	}

	ctx, cancel := detached()
	defer cancel()

	created, err := s.gh.SubmitReview(ctx, key.Owner, key.Repo, key.Number, ghapi.ReviewRequest{
		Body:     summary,
		Event:    event,
		Comments: comments,
	})
	if err != nil {
		writeUpstreamError(w, err)
		return
	}

	if err := review.Clear(key); err != nil {
		log.Printf("clearing drafts for %s/%s#%d: %v", key.Owner, key.Repo, key.Number, err)
	}
	s.cache.invalidate(fmt.Sprintf("pr:%s/%s/%d", key.Owner, key.Repo, key.Number))

	writeJSON(w, http.StatusOK, map[string]any{
		"review":   created,
		"posted":   len(comments),
		"stateUrl": created.HTMLURL,
	})
}
