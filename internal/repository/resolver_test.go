package repository

import (
	"fmt"
	"testing"
)

// mockRefLister returns fixed branches/tags for tests.
type mockRefLister struct {
	branches []string
	err      error
}

func (m *mockRefLister) ListBranchesAndTags(_ *RepositoryInfo, _ string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.branches, nil
}

func TestResolveBranchAndPath_WithMock(t *testing.T) {
	mock := &mockRefLister{
		branches: []string{"main", "develop", "release/v1"},
	}
	SetTestRefLister(mock)
	defer SetTestRefLister(nil)

	repoInfo := &RepositoryInfo{
		Type:  GitHub,
		Owner: "owner",
		Repo:  "repo",
	}

	branch, path, err := ResolveBranchAndPath(repoInfo, "develop/kustomize/base", "")
	if err != nil {
		t.Fatalf("ResolveBranchAndPath: %v", err)
	}
	if branch != "develop" {
		t.Errorf("branch = %q, want develop", branch)
	}
	if path != "kustomize/base" {
		t.Errorf("path = %q, want kustomize/base", path)
	}
}

func TestResolveBranchAndPath_WithMock_GitLab(t *testing.T) {
	mock := &mockRefLister{
		branches: []string{"main", "staging"},
	}
	SetTestRefLister(mock)
	defer SetTestRefLister(nil)

	repoInfo := &RepositoryInfo{
		Type:  GitLab,
		Owner: "group",
		Repo:  "project",
	}

	branch, path, err := ResolveBranchAndPath(repoInfo, "staging/deploy", "")
	if err != nil {
		t.Fatalf("ResolveBranchAndPath: %v", err)
	}
	if branch != "staging" {
		t.Errorf("branch = %q, want staging", branch)
	}
	if path != "deploy" {
		t.Errorf("path = %q, want deploy", path)
	}
}

func TestResolveBranchAndPath_WithMock_Error(t *testing.T) {
	mock := &mockRefLister{
		err: fmt.Errorf("API rate limit"),
	}
	SetTestRefLister(mock)
	defer SetTestRefLister(nil)

	repoInfo := &RepositoryInfo{Type: GitHub, Owner: "o", Repo: "r"}
	_, _, err := ResolveBranchAndPath(repoInfo, "main/path", "")
	if err == nil {
		t.Fatal("expected error from mock, got nil")
	}
	if err.Error() != "API rate limit" {
		t.Errorf("err = %v, want API rate limit", err)
	}
}

func TestResolveBranchAndPath_WithMock_NoMatch(t *testing.T) {
	mock := &mockRefLister{
		branches: []string{"main"},
	}
	SetTestRefLister(mock)
	defer SetTestRefLister(nil)

	repoInfo := &RepositoryInfo{Type: GitHub, Owner: "o", Repo: "r"}
	_, _, err := ResolveBranchAndPath(repoInfo, "unknown-branch/path", "")
	if err == nil {
		t.Fatal("expected error when no branch matches, got nil")
	}
}

// TestResolveBranchAndPath_RealURLs validates full branch+path resolution
// for real-world ambiguous paths using a mock (no live GitHub/GitLab calls).
func TestResolveBranchAndPath_RealURLs(t *testing.T) {
	cases := []struct {
		name         string
		ambiguousPath string
		mockBranches []string // branches/tags the mock returns (longest match wins)
		wantBranch   string
		wantPath     string
	}{
		{
			name:          "openstack-k8s architecture: main + examples/va/hci/control-plane",
			ambiguousPath: "main/examples/va/hci/control-plane",
			mockBranches:  []string{"main"},
			wantBranch:    "main",
			wantPath:      "examples/va/hci/control-plane",
		},
		{
			name:          "GitLab: components/new-base + environments/...",
			ambiguousPath: "components/new-base/environments/cifmw-demo/scale-out/deployment",
			mockBranches:  []string{"main", "components/new-base"},
			wantBranch:    "components/new-base",
			wantPath:      "environments/cifmw-demo/scale-out/deployment",
		},
		{
			name:          "rhoso-gitops: branch with slashes cjt/cleaning/... + example/controlplane",
			ambiguousPath: "cjt/cleaning/test-nodeset-component/example/controlplane",
			mockBranches:  []string{"main", "cjt/cleaning/test-nodeset-component"},
			wantBranch:    "cjt/cleaning/test-nodeset-component",
			wantPath:      "example/controlplane",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mock := &mockRefLister{branches: c.mockBranches}
			SetTestRefLister(mock)
			defer SetTestRefLister(nil)

			repoInfo := &RepositoryInfo{Type: GitHub, Owner: "o", Repo: "r"}
			branch, path, err := ResolveBranchAndPath(repoInfo, c.ambiguousPath, "")
			if err != nil {
				t.Fatalf("ResolveBranchAndPath: %v", err)
			}
			if branch != c.wantBranch {
				t.Errorf("branch = %q, want %q", branch, c.wantBranch)
			}
			if path != c.wantPath {
				t.Errorf("path = %q, want %q", path, c.wantPath)
			}
		})
	}
}

func TestFindLongestMatch(t *testing.T) {
	cases := []struct {
		name      string
		branches  []string
		urlPath   string
		wantBranch string
		wantPath  string
		wantErr   bool
	}{
		{
			name:       "exact branch",
			branches:   []string{"main", "develop"},
			urlPath:    "main",
			wantBranch: "main",
			wantPath:   "",
			wantErr:    false,
		},
		{
			name:       "branch with path",
			branches:   []string{"main", "develop"},
			urlPath:    "main/deploy/overlay",
			wantBranch: "main",
			wantPath:   "deploy/overlay",
			wantErr:    false,
		},
		{
			name:       "longest match wins",
			branches:   []string{"main", "main-feature", "main-feature-x"},
			urlPath:    "main-feature-x/foo/bar",
			wantBranch: "main-feature-x",
			wantPath:   "foo/bar",
			wantErr:    false,
		},
		{
			name:       "longest match not prefix of path",
			branches:   []string{"main", "main-feature"},
			urlPath:    "main/overlay",
			wantBranch: "main",
			wantPath:   "overlay",
			wantErr:    false,
		},
		{
			name:       "path with leading/trailing slashes",
			branches:   []string{"develop"},
			urlPath:    "/develop/kustomize/base/",
			wantBranch: "develop",
			wantPath:   "kustomize/base",
			wantErr:    false,
		},
		{
			name:      "no matching branch",
			branches:  []string{"main", "develop"},
			urlPath:   "release/overlay",
			wantErr:   true,
		},
		{
			name:      "empty branches",
			branches:  []string{},
			urlPath:   "main",
			wantErr:   true,
		},
		{
			name:       "tag and branch",
			branches:   []string{"main", "v1.0", "v1.0.0"},
			urlPath:    "v1.0.0/deploy",
			wantBranch: "v1.0.0",
			wantPath:   "deploy",
			wantErr:    false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			branch, path, err := findLongestMatch(c.branches, c.urlPath)
			if c.wantErr {
				if err == nil {
					t.Fatalf("findLongestMatch: expected error, got branch=%q path=%q", branch, path)
				}
				return
			}
			if err != nil {
				t.Fatalf("findLongestMatch error: %v", err)
			}
			if branch != c.wantBranch {
				t.Errorf("branch = %q, want %q", branch, c.wantBranch)
			}
			if path != c.wantPath {
				t.Errorf("path = %q, want %q", path, c.wantPath)
			}
		})
	}
}
