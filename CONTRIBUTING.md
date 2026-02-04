# Contributing

We welcome contributions! Here's how you can help.

## Reporting issues

1. Check if the issue already exists in GitHub Issues.
2. If not, create a new issue with:
   * Clear title and description
   * Steps to reproduce
   * Expected vs actual behavior
   * Screenshots (if applicable)

## Submitting pull requests

1. Fork the repository.
2. Create a feature branch:
   ```shell
   git checkout -b feature/amazing-feature
   ```
3. Make your changes following the coding style (see below).
4. Run tests and ensure they pass:
   ```shell
   go test ./...
   go build .
   ```
5. Commit with clear messages:
   ```shell
   git commit -m "feat: add amazing feature"
   ```
6. Push to your fork:
   ```shell
   git push origin feature/amazing-feature
   ```
7. Open a Pull Request with:
   * Clear description of changes
   * Link to related issue (if applicable)
   * Screenshots/demos (if UI changes)

## Development guidelines

### Code style

* **Go**: Use Go 1.24+ (see `go.mod`). Format with `gofmt` (or `go fmt ./...`). Run `go vet ./...` before committing.
* Follow existing code style in the repo.
* Prefer table-driven tests; keep tests in `*_test.go` next to the code (see [TESTING.md](TESTING.md)).

### Project layout

```
.
├── main.go                 # Entrypoint, embed web, start server
├── main_test.go
├── go.mod, go.sum
├── internal/               # Go packages (not importable outside this module)
│   ├── export/             # Graph export
│   ├── fetcher/            # Git content fetch
│   ├── parser/             # Kustomize YAML parsing, reference parsing
│   ├── repository/         # Repo detection, branch/path resolution
│   ├── server/             # HTTP API and static file serving
│   ├── storage/            # In-memory graph storage
│   └── types/              # Shared types
└── web/                    # Frontend (embedded in binary)
    ├── index.html
    ├── css/
    └── js/
```

### Adding features

* Keep packages focused; add tests in the same package (`*_test.go`).
* For API changes, update handlers in `internal/server` and document in README if user-facing.
* For frontend changes, edit files under `web/`; the app serves them embedded.

### Ideas for contribution

* Add support for more Git providers (e.g. Bitbucket, Gitea).
* API export graph as PNG/SVG (Mermaid export exists in `internal/export`).
* Search and filter nodes in the UI.
* Performance improvements for large repositories.
* More unit and integration tests (see [TESTING.md](TESTING.md)).
* Documentation improvements.
