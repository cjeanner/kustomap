# Kustomize Visualizer

![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)
![AI-Assisted](https://img.shields.io/badge/AI--Assisted-Perplexity-blueviolet.svg)
![AI-Assisted](https://img.shields.io/badge/AI--Assisted-Cursor-0066CC.svg)

A web application to visualize and explore Kustomize overlay structures in GitOps-managed environments. This is a **port of the original Node.js application to Go**, developed with the assistance of **Perplexity** and **Cursor** (AI-assisted conversion).

## Overview

- **Visual graph**: Interactive dependency tree of bases, overlays, components, and resources (Cytoscape.js in the frontend).
- **Sources**: GitHub, GitLab (URL + optional tokens), or local directory via browser File System API.
- **API**: The Go server exposes a REST API used by the web UI:
  - `POST /api/v1/analyze` — submit a repo URL (optional `github_token` / `gitlab_token`); returns a graph `id`.
  - `GET /api/v1/graph/{id}` — fetch the analyzed graph.
  - `GET /api/v1/node/{graphID}/{nodeID}` — fetch node details.

## Prerequisites

- **Go 1.24+** (see `go.mod`)
- For container: **Docker** or **Podman**

## Build and run

### Native (Go)

```bash
git clone https://github.com/cjeanner/kustomize-visualizer.git
cd kustomize-visualizer

# Run directly
go run .

# Optional: custom port (default 3000, or set PORT)
go run . -port 8080
```

Then open **http://localhost:3000**.

### Container

```bash
# Build
podman build -t kustomize-visualizer:latest -f Containerfile .
# or: docker build -t kustomize-visualizer:latest -f Containerfile .

# Run (server listens on 3000 inside the container)
podman run --rm -d -p 8080:3000 --name kustomize-viz kustomize-visualizer:latest
# or: docker run --rm -d -p 8080:3000 --name kustomize-viz kustomize-visualizer:latest
```

Then open **http://localhost:8080**.

To stop and remove:

```bash
podman stop kustomize-viz
# or: docker stop kustomize-viz
```

## AI-assisted development

This project was created using AI tools. The **tool/AI used to port the original Node.js application to Go is Perplexity** (conversion and implementation). Ongoing development and editing use **Cursor** and its integrated AI model. The use of AI does not replace human review: all code has been reviewed and tested.

## License

Apache-2.0 — see [LICENSE](LICENSE).
