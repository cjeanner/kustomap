package parser

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseReference_Relative(t *testing.T) {
	cases := []struct {
		name   string
		ref    string
		token  string
		typ    ReferenceType
		relPath string
	}{
		{"explicit ./", "./base", "", ReferenceRelative, "./base"},
		{"explicit ../", "../overlays/dev", "", ReferenceRelative, "../overlays/dev"},
		{"implicit relative", "deployment-02", "", ReferenceRelative, "deployment-02"},
		{"implicit with slash", "components/foo", "", ReferenceRelative, "components/foo"},
		{"nodeset", "nodeset", "", ReferenceRelative, "nodeset"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseReference(c.ref, c.token)
			if err != nil {
				t.Fatalf("ParseReference(%q) error: %v", c.ref, err)
			}
			if got.Type != c.typ {
				t.Errorf("Type = %q, want %q", got.Type, c.typ)
			}
			if got.RelativePath != c.relPath {
				t.Errorf("RelativePath = %q, want %q", got.RelativePath, c.relPath)
			}
		})
	}
}

func TestParseReference_HTTP_KustomizeFormat(t *testing.T) {
	// Kustomize format: https://github.com/org/repo//path?ref=branch
	ref := "https://github.com/owner/repo//some/overlay?ref=v1.0"
	got, err := ParseReference(ref, "")
	if err != nil {
		t.Fatalf("ParseReference error: %v", err)
	}
	if got.Type != ReferenceRemote {
		t.Errorf("Type = %q, want remote", got.Type)
	}
	if got.RepoInfo == nil {
		t.Fatal("RepoInfo is nil")
	}
	if got.RepoInfo.Owner != "owner" || got.RepoInfo.Repo != "repo" {
		t.Errorf("RepoInfo = %s/%s, want owner/repo", got.RepoInfo.Owner, got.RepoInfo.Repo)
	}
	if got.RepoInfo.Ref != "v1.0" {
		t.Errorf("Ref = %q, want v1.0", got.RepoInfo.Ref)
	}
	if got.Path != "some/overlay" {
		t.Errorf("Path = %q, want some/overlay", got.Path)
	}
}

func TestParseReference_HTTP_StandardFormat(t *testing.T) {
	// Standard: https://github.com/org/repo/path?ref=branch
	ref := "https://github.com/owner/repo/deploy/base?ref=main"
	got, err := ParseReference(ref, "")
	if err != nil {
		t.Fatalf("ParseReference error: %v", err)
	}
	if got.Type != ReferenceRemote {
		t.Errorf("Type = %q, want remote", got.Type)
	}
	if got.RepoInfo == nil {
		t.Fatal("RepoInfo is nil")
	}
	if got.RepoInfo.Owner != "owner" || got.RepoInfo.Repo != "repo" {
		t.Errorf("RepoInfo = %s/%s, want owner/repo", got.RepoInfo.Owner, got.RepoInfo.Repo)
	}
	if got.RepoInfo.Ref != "main" {
		t.Errorf("Ref = %q, want main", got.RepoInfo.Ref)
	}
	if got.Path != "deploy/base" {
		t.Errorf("Path = %q, want deploy/base", got.Path)
	}
}

func TestParseReference_HTTP_NoRef(t *testing.T) {
	ref := "https://github.com/owner/repo/deploy/base"
	got, err := ParseReference(ref, "")
	if err != nil {
		t.Fatalf("ParseReference error: %v", err)
	}
	if got.RepoInfo.Ref != "main" {
		t.Errorf("Ref (default) = %q, want main", got.RepoInfo.Ref)
	}
}

func TestParseReference_GitSSH(t *testing.T) {
	// git@github.com:org/repo.git//path?ref=branch -> converted to HTTPS and parsed
	ref := "git@github.com:owner/repo.git//kustomize/base?ref=develop"
	got, err := ParseReference(ref, "")
	if err != nil {
		t.Fatalf("ParseReference error: %v", err)
	}
	if got.Type != ReferenceRemote {
		t.Errorf("Type = %q, want remote", got.Type)
	}
	if got.RepoInfo == nil {
		t.Fatal("RepoInfo is nil")
	}
	if got.RepoInfo.Owner != "owner" || got.RepoInfo.Repo != "repo" {
		t.Errorf("RepoInfo = %s/%s, want owner/repo", got.RepoInfo.Owner, got.RepoInfo.Repo)
	}
	if got.RepoInfo.Ref != "develop" {
		t.Errorf("Ref = %q, want develop", got.RepoInfo.Ref)
	}
	if got.Path != "kustomize/base" {
		t.Errorf("Path = %q, want kustomize/base", got.Path)
	}
}

func TestParseReference_HTTP_GitLab(t *testing.T) {
	ref := "https://gitlab.com/group/subgroup/project//deploy/overlay?ref=main"
	got, err := ParseReference(ref, "")
	if err != nil {
		t.Fatalf("ParseReference error: %v", err)
	}
	if got.Type != ReferenceRemote {
		t.Errorf("Type = %q, want remote", got.Type)
	}
	if got.RepoInfo == nil {
		t.Fatal("RepoInfo is nil")
	}
	if got.RepoInfo.Owner != "group/subgroup" || got.RepoInfo.Repo != "project" {
		t.Errorf("RepoInfo = %s/%s, want group/subgroup/project", got.RepoInfo.Owner, got.RepoInfo.Repo)
	}
	if got.Path != "deploy/overlay" {
		t.Errorf("Path = %q, want deploy/overlay", got.Path)
	}
}

func TestKustomizeReference_String(t *testing.T) {
	rel := &KustomizeReference{Type: ReferenceRelative, RelativePath: "./base"}
	if got := rel.String(); got != "relative:./base" {
		t.Errorf("String() = %q, want relative:./base", got)
	}
}

// Kustomization with bare path references (no "./" prefix, no URI).
const kustomizationYAMLWithBarePaths = `---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - deployment-02

components:
  - nodeset
  - components/foo
`

func TestParseReference_KustomizationWithBarePaths(t *testing.T) {
	var kust Kustomization
	if err := yaml.Unmarshal([]byte(kustomizationYAMLWithBarePaths), &kust); err != nil {
		t.Fatalf("parse kustomization YAML: %v", err)
	}
	if len(kust.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(kust.Resources))
	}
	if len(kust.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(kust.Components))
	}

	// resources: bare path "deployment-02" (no ./ nor URI)
	refs := append([]string{kust.Resources[0]}, kust.Components...)
	wantPaths := []string{"deployment-02", "nodeset", "components/foo"}

	for i, raw := range refs {
		ref, err := ParseReference(raw, "")
		if err != nil {
			t.Fatalf("ParseReference(%q): %v", raw, err)
		}
		if ref.Type != ReferenceRelative {
			t.Errorf("ref[%d] Type = %s, want relative (bare path)", i, ref.Type)
		}
		if ref.RelativePath != wantPaths[i] {
			t.Errorf("ref[%d] RelativePath = %q, want %q", i, ref.RelativePath, wantPaths[i])
		}
	}
}

// Real kustomization.yaml content: standard Kustomize URIs with ?ref=cleaning
// (branch must be consumed when parsing to access the actual content).
const kustomizationYAMLWithRemoteComponents = `---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

components:
  # Remote refs (for CI/ArgoCD):
  - https://github.com/openstack-gitops/rhoso-gitops/components/argocd/annotations?ref=cleaning
  - https://github.com/openstack-gitops/rhoso-gitops/components/rhoso/controlplane?ref=cleaning
  - https://github.com/openstack-gitops/rhoso-gitops/components/rhoso/controlplane/services/watcher?ref=cleaning
`

func TestParseReference_RealKustomizationContent(t *testing.T) {
	var kust Kustomization
	if err := yaml.Unmarshal([]byte(kustomizationYAMLWithRemoteComponents), &kust); err != nil {
		t.Fatalf("parse kustomization YAML: %v", err)
	}
	if len(kust.Components) != 3 {
		t.Fatalf("expected 3 components, got %d", len(kust.Components))
	}

	wantRef := "cleaning"
	wantOwner := "openstack-gitops"
	wantRepo := "rhoso-gitops"
	wantPaths := []string{
		"components/argocd/annotations",
		"components/rhoso/controlplane",
		"components/rhoso/controlplane/services/watcher",
	}

	for i, rawURL := range kust.Components {
		ref, err := ParseReference(rawURL, "")
		if err != nil {
			t.Fatalf("component[%d] ParseReference(%q): %v", i, rawURL, err)
		}
		if ref.Type != ReferenceRemote {
			t.Errorf("component[%d] Type = %s, want remote", i, ref.Type)
		}
		if ref.RepoInfo == nil {
			t.Fatalf("component[%d] RepoInfo is nil", i)
		}
		if ref.RepoInfo.Ref != wantRef {
			t.Errorf("component[%d] Ref = %q, want %q (?ref= must be consumed)", i, ref.RepoInfo.Ref, wantRef)
		}
		if ref.RepoInfo.Owner != wantOwner || ref.RepoInfo.Repo != wantRepo {
			t.Errorf("component[%d] Repo = %s/%s, want %s/%s", i, ref.RepoInfo.Owner, ref.RepoInfo.Repo, wantOwner, wantRepo)
		}
		if ref.Path != wantPaths[i] {
			t.Errorf("component[%d] Path = %q, want %q", i, ref.Path, wantPaths[i])
		}
	}
}

// Kustomization with one remote component using a branch name that contains slashes
// (?ref=cjt/cleaning/test-nodeset-component) and patches (target+patch, path+target).
const kustomizationYAMLWithBranchWithSlashes = `---
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
components:
  - https://github.com/cjeanner/rhoso-gitops/components/rhoso/dataplane/nodeset?ref=cjt/cleaning/test-nodeset-component

patches:
  - target:
      kind: OpenStackDataPlaneNodeSet
    patch: |-
      - op: replace
        path: /metadata/name
        value: nodeset-02
  - path: patch-nodeset-ansible.yaml
    target:
      kind: OpenStackDataPlaneNodeSet
      name: nodeset-02
  - path: patch-nodeset-nodes.yaml
    target:
      kind: OpenStackDataPlaneNodeSet
      name: nodeset-02
  - path: patch-subscription-manager.yaml
    target:
      kind: OpenStackDataPlaneNodeSet
      name: nodeset-02
`

func TestParseReference_KustomizationWithBranchWithSlashes(t *testing.T) {
	var kust Kustomization
	if err := yaml.Unmarshal([]byte(kustomizationYAMLWithBranchWithSlashes), &kust); err != nil {
		t.Fatalf("parse kustomization YAML: %v", err)
	}
	if len(kust.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(kust.Components))
	}
	if len(kust.Patches) != 4 {
		t.Fatalf("expected 4 patches, got %d", len(kust.Patches))
	}

	rawURL := kust.Components[0]
	ref, err := ParseReference(rawURL, "")
	if err != nil {
		t.Fatalf("ParseReference(%q): %v", rawURL, err)
	}
	if ref.Type != ReferenceRemote {
		t.Errorf("Type = %s, want remote", ref.Type)
	}
	if ref.RepoInfo == nil {
		t.Fatal("RepoInfo is nil")
	}
	// Branch with slashes must be consumed as a single ref
	wantRef := "cjt/cleaning/test-nodeset-component"
	if ref.RepoInfo.Ref != wantRef {
		t.Errorf("Ref = %q, want %q (branch with slashes must be treated correctly)", ref.RepoInfo.Ref, wantRef)
	}
	if ref.RepoInfo.Owner != "cjeanner" || ref.RepoInfo.Repo != "rhoso-gitops" {
		t.Errorf("Repo = %s/%s, want cjeanner/rhoso-gitops", ref.RepoInfo.Owner, ref.RepoInfo.Repo)
	}
	wantPath := "components/rhoso/dataplane/nodeset"
	if ref.Path != wantPath {
		t.Errorf("Path = %q, want %q", ref.Path, wantPath)
	}
}
