package build

import (
	"testing"

	"github.com/cjeanner/kustomap/internal/repository"
)

func TestParseNodeID(t *testing.T) {
	tests := []struct {
		name    string
		nodeID  string
		want    *NodeIDParts
		wantErr bool
	}{
		{
			name:   "github with path",
			nodeID: "github:foo/bar/deploy/overlay@main",
			want: &NodeIDParts{
				Type:  repository.GitHub,
				Owner: "foo",
				Repo:  "bar",
				Path:  "deploy/overlay",
				Ref:   "main",
			},
		},
		{
			name:   "gitlab root path",
			nodeID: "gitlab:org/repo@v1.0",
			want: &NodeIDParts{
				Type:  repository.GitLab,
				Owner: "org",
				Repo:  "repo",
				Path:  "",
				Ref:   "v1.0",
			},
		},
		{
			name:    "missing colon",
			nodeID:  "githubfoo/bar@main",
			wantErr: true,
		},
		{
			name:    "missing at",
			nodeID:  "github:foo/bar/deploy",
			wantErr: true,
		},
		{
			name:    "unsupported type",
			nodeID:  "bitbucket:foo/bar@main",
			wantErr: true,
		},
		{
			name:   "ref with slash (branch or path)",
			nodeID: "gitlab:owner/repo/env/overlay@components/new-base",
			want: &NodeIDParts{
				Type:  repository.GitLab,
				Owner: "owner",
				Repo:  "repo",
				Path:  "env/overlay",
				Ref:   "components/new-base",
			},
		},
		{
			name:   "single segment path",
			nodeID: "github:user/proj/base@main",
			want: &NodeIDParts{
				Type:  repository.GitHub,
				Owner: "user",
				Repo:  "proj",
				Path:  "base",
				Ref:   "main",
			},
		},
		{
			name:    "empty ref",
			nodeID:  "github:foo/bar/path@",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseNodeID(tt.nodeID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseNodeID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Type != tt.want.Type || got.Owner != tt.want.Owner || got.Repo != tt.want.Repo || got.Path != tt.want.Path || got.Ref != tt.want.Ref {
				t.Errorf("ParseNodeID() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
