package build

import (
	"fmt"
	"strings"

	"github.com/cjeanner/kustomap/internal/repository"
)

// NodeIDParts holds parsed components of a graph node ID.
// Format: Type:Owner/Repo/Path@Ref (e.g. github:foo/bar/deploy/overlay@main)
type NodeIDParts struct {
	Type   repository.RepositoryType
	Owner  string
	Repo   string
	Path   string
	Ref    string
}

// ParseNodeID parses a node ID into repo type, owner, repo, path and ref.
// Returns an error if the format is invalid.
func ParseNodeID(nodeID string) (*NodeIDParts, error) {
	colon := strings.Index(nodeID, ":")
	if colon <= 0 || colon == len(nodeID)-1 {
		return nil, fmt.Errorf("invalid node ID: missing or invalid type prefix (expected type:owner/repo/path@ref)")
	}
	typStr := nodeID[:colon]
	rest := nodeID[colon+1:]

	var repoType repository.RepositoryType
	switch typStr {
	case "github":
		repoType = repository.GitHub
	case "gitlab":
		repoType = repository.GitLab
	default:
		return nil, fmt.Errorf("unsupported repository type in node ID: %s", typStr)
	}

	at := strings.LastIndex(rest, "@")
	if at < 0 {
		return nil, fmt.Errorf("invalid node ID: missing @ref")
	}
	beforeRef := rest[:at]
	ref := rest[at+1:]
	if ref == "" {
		return nil, fmt.Errorf("invalid node ID: empty ref")
	}

	parts := strings.SplitN(beforeRef, "/", 3) // owner, repo, path (path may contain /)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid node ID: expected owner/repo[/path]")
	}
	owner := parts[0]
	repo := parts[1]
	path := ""
	if len(parts) == 3 {
		path = parts[2]
	}

	return &NodeIDParts{
		Type:  repoType,
		Owner: owner,
		Repo:  repo,
		Path:  strings.Trim(path, "/"),
		Ref:   ref,
	}, nil
}
