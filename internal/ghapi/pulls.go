package ghapi

import (
	"context"
	"fmt"
	"net/url"
)

func (c *Client) CurrentUser(ctx context.Context) (*User, error) {
	var u User
	if err := c.get(ctx, "/user", &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func (c *Client) SearchPullRequests(ctx context.Context, query string) ([]SearchIssue, error) {
	const perPage = 100

	var all []SearchIssue
	for page := 1; ; page++ {
		path := fmt.Sprintf("/search/issues?q=%s&per_page=%d&page=%d&sort=updated&order=desc",
			url.QueryEscape(query), perPage, page)

		var result SearchResult
		if err := c.get(ctx, path, &result); err != nil {
			return nil, err
		}

		all = append(all, result.Items...)
		if len(result.Items) < perPage || len(all) >= result.TotalCount {
			return all, nil
		}
	}
}

func (c *Client) OrgPullRequests(ctx context.Context, org string) ([]SearchIssue, error) {
	return c.SearchPullRequests(ctx, fmt.Sprintf("org:%s is:pr is:open archived:false", org))
}

func (c *Client) RepoPullRequests(ctx context.Context, owner, repo string) ([]SearchIssue, error) {
	return c.SearchPullRequests(ctx, fmt.Sprintf("repo:%s/%s is:pr is:open", owner, repo))
}

func (c *Client) PullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	var pr PullRequest
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number)
	if err := c.get(ctx, path, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

func (c *Client) PullRequestFiles(ctx context.Context, owner, repo string, number int) ([]File, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/files", owner, repo, number)
	return getPaged[File](ctx, c, path)
}

func (c *Client) ReviewComments(ctx context.Context, owner, repo string, number int) ([]ReviewComment, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/comments", owner, repo, number)
	return getPaged[ReviewComment](ctx, c, path)
}

func (c *Client) IssueComments(ctx context.Context, owner, repo string, number int) ([]IssueComment, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, number)
	return getPaged[IssueComment](ctx, c, path)
}

func (c *Client) CreateReviewComment(ctx context.Context, owner, repo string, number int, req CommentRequest) (*ReviewComment, error) {
	var created ReviewComment
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/comments", owner, repo, number)
	if err := c.post(ctx, path, req, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) DeleteReviewComment(ctx context.Context, owner, repo string, id int64) error {
	return c.del(ctx, fmt.Sprintf("/repos/%s/%s/pulls/comments/%d", owner, repo, id))
}

func (c *Client) SubmitReview(ctx context.Context, owner, repo string, number int, req ReviewRequest) (*Review, error) {
	var review Review
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", owner, repo, number)
	if err := c.post(ctx, path, req, &review); err != nil {
		return nil, err
	}
	return &review, nil
}
