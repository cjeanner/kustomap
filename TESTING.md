# Testing Guide

This document describes how tests are structured, which tooling is used, and how to run them.

## Tooling

- **Test runner**: Go’s built-in `go test`. No extra test framework is required.
- **Assertions**: Standard library only (`testing.T`, `if got != want`). No testify or similar.
- **Coverage**: `go test -cover` or `go test -coverprofile=coverage.out` then `go tool cover -html=coverage.out`.

## Structure

Tests follow the usual Go layout:

- **Co-located tests**: Each package has `*_test.go` files in the same directory as the code (e.g. `internal/parser/reference_test.go` next to `reference.go`).
- **Same package for unit tests**: Tests use `package parser` (or `package repository`) so they can exercise both exported and unexported functions.
- **Table-driven tests**: One test function, many cases in a slice; easy to add cases and keep tests readable.

## What is tested

| Package      | File              | Focus |
|-------------|-------------------|--------|
| `parser`    | `reference_test.go` | `ParseReference`: HTTP/HTTPS URLs (Kustomize `//path?ref=branch` and standard), Git SSH, relative paths (./, ../, bare). |
| `parser`    | `kustomize_test.go` | Helpers: `isYAMLFile`, `resolvePath`, `getShortLabel`. |
| `repository`| `detector_test.go`  | `DetectRepository` and URL parsing: GitHub/GitLab URLs, owner/repo/path, `/tree/branch/path`, ambiguous path. |
| `repository`| `resolver_test.go`  | `findLongestMatch`: branch vs path splitting. **`ResolveBranchAndPath`** via a **mock** `RefLister` so branch resolution is tested without calling real GitHub/GitLab APIs. |

**Mocking GitHub/GitLab:** The repository package defines a `RefLister` interface and a test hook `SetTestRefLister(l RefLister)`. Tests set a mock that returns fixed branch/tag names; `ResolveBranchAndPath` then uses that list and `findLongestMatch` to resolve branch and path. No real API calls in tests.

## Running tests

```bash
# All tests
go test ./...

# With verbose output
go test -v ./...

# Single package
go test ./internal/parser/...
go test ./internal/repository/...

# Coverage (per package)
go test -cover ./...

# Coverage report (HTML)
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Adding new tests

1. Create or edit `*_test.go` in the same package.
2. Use table-driven style when you have multiple inputs/expectations:

   ```go
   func TestSomething(t *testing.T) {
       cases := []struct {
           name string
           in   string
           want string
       }{
           {"case1", "input1", "expected1"},
           {"case2", "input2", "expected2"},
       }
       for _, c := range cases {
           t.Run(c.Name, func(t *testing.T) {
               got := Something(c.in)
               if got != c.want {
                   t.Errorf("Something(%q) = %q, want %q", c.in, got, c.want)
               }
           })
       }
   }
   ```

3. For URL/branch extraction, add cases to `reference_test.go` and `detector_test.go`; for branch/path splitting, add cases to `resolver_test.go`.
