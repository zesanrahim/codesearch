package ghapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	defaultBaseURL = "https://api.github.com"
	acceptHeader   = "application/vnd.github+json"
	apiVersion     = "2022-11-28"
)

type RateLimit struct {
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	Reset     time.Time `json:"reset"`
}

type Error struct {
	StatusCode int
	Message    string
	URL        string
	Errors     []FieldError
}

type FieldError struct {
	Resource string `json:"resource"`
	Field    string `json:"field"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

func (e *Error) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("github %s: %d %s (%s)", e.URL, e.StatusCode, e.Message, e.Errors[0].Message)
	}
	return fmt.Sprintf("github %s: %d %s", e.URL, e.StatusCode, e.Message)
}

type Client struct {
	token   string
	baseURL string
	http    *http.Client

	mu    sync.Mutex
	etags map[string]cached
	rate  RateLimit
}

type cached struct {
	etag string
	body []byte
}

func New(token string) *Client {
	return &Client{
		token:   token,
		baseURL: defaultBaseURL,
		http:    &http.Client{Timeout: 30 * time.Second},
		etags:   make(map[string]cached),
	}
}

func (c *Client) Rate() RateLimit {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.rate
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", acceptHeader)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c *Client) recordRate(resp *http.Response) {
	limit, err1 := strconv.Atoi(resp.Header.Get("X-RateLimit-Limit"))
	remaining, err2 := strconv.Atoi(resp.Header.Get("X-RateLimit-Remaining"))
	reset, err3 := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return
	}

	c.mu.Lock()
	c.rate = RateLimit{Limit: limit, Remaining: remaining, Reset: time.Unix(reset, 0)}
	c.mu.Unlock()
}

func apiError(resp *http.Response, body []byte) error {
	var parsed struct {
		Message string       `json:"message"`
		Errors  []FieldError `json:"errors"`
	}
	_ = json.Unmarshal(body, &parsed)

	msg := parsed.Message
	if msg == "" {
		msg = http.StatusText(resp.StatusCode)
	}
	return &Error{
		StatusCode: resp.StatusCode,
		Message:    msg,
		URL:        resp.Request.URL.Path,
		Errors:     parsed.Errors,
	}
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}

	c.mu.Lock()
	prev, hasPrev := c.etags[path]
	c.mu.Unlock()
	if hasPrev {
		req.Header.Set("If-None-Match", prev.etag)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	c.recordRate(resp)

	if resp.StatusCode == http.StatusNotModified && hasPrev {
		return json.Unmarshal(prev.body, out)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return apiError(resp, body)
	}

	if etag := resp.Header.Get("ETag"); etag != "" {
		c.mu.Lock()
		c.etags[path] = cached{etag: etag, body: body}
		c.mu.Unlock()
	}

	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

func (c *Client) post(ctx context.Context, path string, in, out any) error {
	payload, err := json.Marshal(in)
	if err != nil {
		return err
	}

	req, err := c.newRequest(ctx, http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	c.recordRate(resp)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return apiError(resp, body)
	}

	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

func getPaged[T any](ctx context.Context, c *Client, path string) ([]T, error) {
	const perPage = 100

	sep := "?"
	if bytes.ContainsRune([]byte(path), '?') {
		sep = "&"
	}

	var all []T
	for page := 1; ; page++ {
		var batch []T
		url := fmt.Sprintf("%s%sper_page=%d&page=%d", path, sep, perPage, page)
		if err := c.get(ctx, url, &batch); err != nil {
			return nil, err
		}

		all = append(all, batch...)
		if len(batch) < perPage {
			return all, nil
		}
	}
}
