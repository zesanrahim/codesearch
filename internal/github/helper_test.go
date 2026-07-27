package github

import "testing"

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantOrg string
		want    string
	}{
		{"https", "https://github.com/facebook/react", "facebook", "react"},
		{"https with .git", "https://github.com/facebook/react.git", "facebook", "react"},
		{"https trailing slash", "https://github.com/facebook/react/", "facebook", "react"},
		{"https .git and slash", "https://github.com/facebook/react.git/", "facebook", "react"},
		{"ssh", "git@github.com:facebook/react.git", "facebook", "react"},
		{"ssh without .git", "git@github.com:facebook/react", "facebook", "react"},
		{"bare path", "facebook/react", "facebook", "react"},
		{"surrounding space", "  https://github.com/facebook/react  ", "facebook", "react"},
		{"non-github host", "https://gitlab.com/group/project", "group", "project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org, name, err := ParseRepoURL(tt.url)
			if err != nil {
				t.Fatalf("ParseRepoURL(%q) returned error: %v", tt.url, err)
			}
			if org != tt.wantOrg || name != tt.want {
				t.Errorf("ParseRepoURL(%q) = (%q, %q), want (%q, %q)",
					tt.url, org, name, tt.wantOrg, tt.want)
			}
		})
	}
}

func TestParseRepoURLDistinguishesForks(t *testing.T) {
	upstreamOrg, upstreamName, err := ParseRepoURL("https://github.com/facebook/react")
	if err != nil {
		t.Fatalf("upstream: %v", err)
	}
	forkOrg, forkName, err := ParseRepoURL("https://github.com/myfork/react")
	if err != nil {
		t.Fatalf("fork: %v", err)
	}

	if upstreamName != forkName {
		t.Fatalf("expected both to be named react, got %q and %q", upstreamName, forkName)
	}
	if upstreamOrg == forkOrg {
		t.Errorf("forks share org %q, so they would share a cache key", upstreamOrg)
	}
}

func TestParseRepoURLRejectsUnparseable(t *testing.T) {
	for _, url := range []string{"", "   ", "react", "https://github.com/"} {
		if org, name, err := ParseRepoURL(url); err == nil {
			t.Errorf("ParseRepoURL(%q) = (%q, %q), want an error", url, org, name)
		}
	}
}
