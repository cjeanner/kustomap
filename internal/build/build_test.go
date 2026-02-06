package build

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuild_InvalidNodeID_ReturnsParseError(t *testing.T) {
	b := NewBuilder("", "")
	_, err := b.Build("not-a-valid-node-id", "")
	if err == nil {
		t.Fatal("Build() expected error for invalid node ID")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("Build() error = %v, want containing 'parse'", err)
	}
}

func TestBuild_InvalidNodeID_WithBaseURL_ReturnsParseError(t *testing.T) {
	b := NewBuilder("", "")
	_, err := b.Build("missing-at:foo/bar/path", "https://gitlab.example.com")
	if err == nil {
		t.Fatal("Build() expected error for invalid node ID")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("Build() error = %v, want containing 'parse'", err)
	}
}

func TestExtractTarGz_ReturnsTopDirAndSkipsPaxGlobalHeader(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.tar.gz")

	// Create a tar.gz that mimics GitLab: pax_global_header first, then repo-dir with a file
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	entries := []struct {
		name string
		body string
		typ  byte
	}{
		{name: paxGlobalHeader, typ: tar.TypeReg, body: "metadata"},
		{name: "repo-dir", typ: tar.TypeDir},
		{name: "repo-dir/kustomization.yaml", typ: tar.TypeReg, body: "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\n"},
	}
	for _, e := range entries {
		h := &tar.Header{Name: e.name, Mode: 0644, Size: int64(len(e.body))}
		if e.typ == tar.TypeDir {
			h.Typeflag = tar.TypeDir
			h.Size = 0
		} else {
			h.Typeflag = tar.TypeReg
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if len(e.body) > 0 {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	extractDir := filepath.Join(dir, "extract")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		t.Fatal(err)
	}

	topDir, err := extractTarGz(archivePath, extractDir)
	if err != nil {
		t.Fatalf("extractTarGz() error = %v", err)
	}
	if topDir != "repo-dir" {
		t.Errorf("extractTarGz() topDir = %q, want %q", topDir, "repo-dir")
	}

	// pax_global_header must not be present
	if _, err := os.Stat(filepath.Join(extractDir, paxGlobalHeader)); err == nil {
		t.Error("pax_global_header should not be extracted")
	}
	// repo-dir and kustomization.yaml must exist
	kustPath := filepath.Join(extractDir, "repo-dir", "kustomization.yaml")
	if _, err := os.Stat(kustPath); err != nil {
		t.Errorf("kustomization.yaml should exist after extract: %v", err)
	}
}

func TestExtractTarGz_NoTopLevelDir_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "flat.tar.gz")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	// Only add pax_global_header (skipped) and a loose file (no top-level dir)
	h := &tar.Header{Name: "only-file.txt", Typeflag: tar.TypeReg, Mode: 0644, Size: 0}
	if err := tw.WriteHeader(h); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()

	extractDir := filepath.Join(dir, "extract")
	os.MkdirAll(extractDir, 0755)

	_, err = extractTarGz(archivePath, extractDir)
	if err == nil {
		t.Fatal("extractTarGz() expected error when archive has no top-level directory")
	}
	if !strings.Contains(err.Error(), "no top-level directory") {
		t.Errorf("extractTarGz() error = %v, want containing 'no top-level directory'", err)
	}
}
