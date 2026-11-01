# DevOps Implementation — Auctasy (Cricket Auction Simulator)

> **Deliverable:** Code Viva — DevOps
> **Stack:** Go (backend) · React 19 + Vite (frontend) · PostgreSQL/Supabase · Redis · Docker · Git

---

## 1. Abstract

Auctasy is a real-time cricket auction platform where a Go backend handles financial
mutations (bids, budgets, ownership) and a React frontend streams live auction state over
WebSockets. This viva documents how DevOps practices — containerization with Docker,
orchestration with Docker Compose, environment-based configuration, and version control
with Git — are applied to make the application portable, reproducible, and deployable in a
single, consistent workflow. The design follows the **"build once, run anywhere"**
philosophy: multi-stage Docker images keep the runtime small and secure, Compose wires the
services together, and secrets stay out of the image. The result is an environment where
`docker compose up` reproduces the entire stack identically on any machine.

---

## 2. DevOps Concepts Used

| Concept | How it is applied in Auctasy |
|---|---|
| **Containerization** | Backend and frontend each get their own Docker image. External managed services (Supabase, Upstash Redis) are consumed via environment URLs rather than containerized, keeping images lean. |
| **Multi-stage builds** | Go compiles in a builder stage; only the static binary ships in the final Alpine image. Vite builds the React app in a Node stage; only static assets ship in Nginx. |
| **Immutable infrastructure** | Images are versioned and tagged; containers are disposable and never patched in place. |
| **Configuration as Environment Variables** | Secrets (`SUPABASE_SERVICE_KEY`, `SUPABASE_JWT_SECRET`, `REDIS_URL`) are injected at runtime, never baked into images. |
| **Reproducibility** | Pinned base images + `go.sum` / `package-lock.json` guarantee identical dependencies everywhere. |
| **Service orchestration** | `docker-compose.yml` defines the backend + frontend, their network, ports, health checks, and restart policy. |
| **Health checks & Observability** | `/health` endpoint exposed as a Docker `HEALTHCHECK`, enabling auto-restart and readiness gating. |
| **CI/CD (continuous integration)** | A GitHub Actions workflow runs `go test`, `go vet`, and `npm run build` on every push to catch regressions early. |
| **Security hardening** | Non-root container user, `.dockerignore` to exclude secrets/artifacts, minimal runtime images. |
| **Version control** | Git branches + commits provide the single source of truth that triggers the build pipeline. |

---

## 3. Implementation

### 3.1 Backend Dockerfile — `backend/Dockerfile`

Multi-stage build: the first stage compiles the Go server using the pinned toolchain, the
second stage runs only the compiled binary on a minimal Alpine image as a non-root user.

```dockerfile
# ---------- Stage 1: build ----------
FROM golang:1.26-alpine AS builder
WORKDIR /src

# Cache dependencies first (only re-runs when go.mod/go.sum change)
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO disabled => fully static binary that runs on a bare Alpine image
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/auctasy-server ./cmd/server

# ---------- Stage 2: runtime ----------
FROM alpine:3.20
RUN adduser -D -u 10001 appuser && apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /out/auctasy-server ./auctasy-server

USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["./auctasy-server"]
```

**Why this order?** Copying `go.mod`/`go.sum` before the source maximizes Docker layer
caching — dependencies are only re-downloaded when the module files change. `-trimpath` and
`-ldflags="-s -w"` strip build paths and debug symbols, shrinking the binary.

### 3.2 Frontend Dockerfile — `frontend/Dockerfile`

Vite produces a static bundle. It is built in a Node stage and served by Nginx — no Node
runtime ships to production.

```dockerfile
# ---------- Stage 1: build ----------
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
# Vite inlines VITE_* vars at build time; pass them as build args
ARG VITE_SUPABASE_URL
ARG VITE_SUPABASE_ANON_KEY
ARG VITE_API_URL=http://localhost:8080
ENV VITE_SUPABASE_URL=$VITE_SUPABASE_URL
ENV VITE_SUPABASE_ANON_KEY=$VITE_SUPABASE_ANON_KEY
ENV VITE_API_URL=$VITE_API_URL
RUN npm run build

# ---------- Stage 2: serve ----------
FROM nginx:1.27-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
HEALTHCHECK CMD wget -qO- http://localhost/ || exit 1
```

Because only `VITE_SUPABASE_ANON_KEY` (a public key) is used in the browser, and the
`VITE_*` variables are **build-time** inputs, no backend secret ever enters the frontend
image. Note the `frontend/nginx.conf` uses `try_files ... /index.html` so client-side
routes (React Router) resolve correctly.

### 3.3 Excluding secrets — `.dockerignore`

Placed at the repo root and in each service folder, this keeps `.env`, Git history, and
build artifacts out of the build context.

```gitignore
.env
.git
**/node_modules
**/dist
*.md
```

### 3.4 Orchestration — `docker-compose.yml`

Compose defines both services on a shared network, injects secrets from the root `.env`,
and enforces a restart policy so the stack self-heals.

```yaml
services:
  backend:
    build: ./backend
    ports:
      - "8080:8080"
    env_file: .env
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3

  frontend:
    build:
      context: ./frontend
      args:
        VITE_SUPABASE_URL: ${VITE_SUPABASE_URL}
        VITE_SUPABASE_ANON_KEY: ${VITE_SUPABASE_ANON_KEY}
        VITE_API_URL: ${VITE_API_URL:-http://localhost:8080}
    ports:
      - "8000:80"
    depends_on:
      backend:
        condition: service_healthy
    restart: unless-stopped
```

### 3.5 Operating the stack

```bash
# Build images and start the full stack
docker compose up --build

# Run detached
docker compose up -d --build

# Tail logs from a single service
docker compose logs -f backend

# Stop and remove containers + network
docker compose down
```

The Go server keeps its graceful-shutdown logic (`SIGTERM` → drain → exit), which Docker
sends automatically on `docker compose down`, so in-flight auctions stop cleanly.

### 3.6 Continuous Integration — `.github/workflows/ci.yml`

Every push builds and tests the code in a clean container, catching breakage before deploy.

```yaml
name: ci
on: [push, pull_request]
jobs:
  backend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.26' }
      - run: go test ./...
        working-directory: backend
      - run: go vet ./...
        working-directory: backend
  frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: '20' }
      - run: npm ci
        working-directory: frontend
      - run: npm run build
        working-directory: frontend
```

---

## 4. Version Control — Git

Git is the foundation the pipeline sits on: every Docker build and CI run is triggered by a
commit, and the repository is the single source of truth for both code and configuration.

```bash
git checkout -b feature/dockerize
git add backend/Dockerfile frontend/Dockerfile docker-compose.yml .dockerignore
git commit -m "chore(devops): add multi-stage Dockerfiles and compose stack"
git push -u origin feature/dockerize
```

**Key practices:** a `.gitignore` (and `.dockerignore`) keeps `.env` and secrets out of both
Git and the images; feature branches isolate work; meaningful commit messages document the
why; and the CI workflow validates each push so `main` always builds and passes tests.

> **Golden rule:** secrets live in the root `.env` (git-ignored) and are injected at runtime
> — never committed and never baked into a Docker image.
