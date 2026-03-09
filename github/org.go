package github

import (
	"context"
	"github.com/google/go-github/github"
)


type Org struct {
    Name     string  `json:"name"`
    CloneURL string  `json:"clone_url"`
    Private  bool    `json:"private"`
    Repo     *Repo   `json:"-"`
}


func GetOrgRepos(ctx context.Context, orgName string) ([]*Org, error) {
	client := github.NewClient(nil)
	opt := &github.RepositoryListByOrgOptions{
		Type: "public",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allRepos []*Org
	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, orgName, opt)
		if err != nil {
			return nil, err
		}

		for _, r := range repos {
            allRepos = append(allRepos, &Org{
                Name:     r.GetName(),
                CloneURL: r.GetCloneURL(),
                Private:  r.GetPrivate(),
            })
        }

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allRepos, nil
}