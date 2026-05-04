# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build ./...
go build -o theartifact .

# Run — launches Studio mode interactively (prompts for auth if needed)
./theartifact

# Run with local backend
ARTIFACT_API_URL=http://localhost:8081 ./theartifact

# One-shot commands
./theartifact login --key tak_live_...
./theartifact generate -w <id> -m "prompt"
./theartifact status <job_id> --wait
./theartifact workspace list

# Lint & vet
go vet ./...

# Tests
go test ./internal/api ./internal/studio -v
go test ./internal/api ./internal/studio -cover

# Cross-compile
GOOS=darwin GOARCH=arm64 go build -o theartifact-darwin-arm64 .
GOOS=linux  GOARCH=amd64 go build -o theartifact-linux-amd64 .
```

## Architecture

Go CLI (Cobra-based) for TheArtifact API. Entry point: `main.go` → `cmd.Execute()`. Running with no subcommand enters **Studio mode** (interactive REPL). Auth is prompted inline on first run.

### Package layout

| Package | Role |
|---|---|
| `cmd/` | One file per Cobra command (`root` = studio entry + inline auth gate, `auth` = `login`/`auth` alias, `generate`, `ingest`, `status`, `workspace`, `studio` = explicit alias) |
| `internal/api/` | HTTP client and API methods — one file per resource (`client`, `workspaces`, `generations`, `ingest`, `jobs`) |
| `internal/config/` | Read/write `~/.artifact/config.json` (`api_key`, `default_workspace_id`, `base_url`) |
| `internal/studio/` | Interactive REPL session (`session.go`) and background job poller (`tracker.go`) |
| `internal/ui/` | Terminal output: lipgloss styles (`theme.go`), formatted printers (`printer.go`), spinner wrapper (`spinner.go`) |

### Key data flows

**Auth (inline gate):** Running `theartifact` with no API key triggers `runLoginOnboarding()` in `cmd/root.go` — prompts for key, validates prefix format (`tak_live_`/`tak_test_`), saves to config, then continues into Studio. `theartifact login --key ...` also works for non-interactive auth.

**Studio (default mode):** Running `theartifact` (no subcommand) opens `studio.Session.Run()` — a blocking REPL that reads stdin, calls `client.Generate()`, and enqueues jobs into `JobTracker`. The tracker polls `/v1/jobs/{id}` every 3 seconds in a background goroutine and fires `onJobComplete` when a job reaches a terminal state. If the workspace is missing or invalid, Studio prompts inline (workspace picker or auto-recovery on `forbidden`).

**Ingest (3-step orchestration):**
1. `POST /v1/uploads` with file count → signed PUT URLs + upload tokens
2. HTTP PUT each binary to the signed URL (no auth header — presigned URL)
3. `POST /v1/ingest` with upload tokens → returns job ID

### API client pattern

All API calls follow the same resty pattern — `SetResult` for success body, `SetError(&APIError{})` globally for 4xx/5xx. Every method checks `resp.IsError()` and unwraps the error body. Base URL is resolved in `api.resolveBaseURL()`: env `ARTIFACT_API_URL` > config `base_url` > default `https://api.theartifact.art`. Bearer token set once in `api.NewClient()`.

**Error envelope:** `APIError` maps `{ "code": "...", "message": "..." }` matching the spec.

### UI conventions

- Progress/errors → **stderr** via `ui.PrintInfo`, `ui.PrintError`, `ui.PrintWarn`, spinners.
- Data output (tables, key-value results) → **stdout** via `ui.PrintSuccess`, `ui.PrintKeyValue`, `ui.PrintTable`.
- All colors come from `internal/ui/theme.go` — never hardcode lipgloss colors in command files.
- `ui.StatusStyle(status)` maps job status strings → lipgloss styles.

## Config file

`~/.artifact/config.json` — stored with 0600 permissions.

| Field | Required | Description |
|---|---|---|
| `api_key` | yes | API key (`tak_live_...`) — set on first run or via `login` |
| `default_workspace_id` | no | Auto-persisted when a workspace is selected in Studio or via `-w` |
| `base_url` | no | Override API base URL for local dev (e.g. `http://localhost:8081`) |

Base URL can also be set via `ARTIFACT_API_URL` env var (takes precedence over config).

## API reference

Full spec: `PUBLIC_API_CLI_SPEC.md`. Key facts for implementation:

### Auth & keys

- API keys: `tak_live_<32 base62>` — issued via web dashboard only. First 12 chars are the `key_prefix` (safe to display). Full key shown **once** at creation, never again.
- API key auth covers all resource endpoints. JWT auth (Supabase session) is only for `/v1/keys` (key management).

### Job lifecycle

```
queued → running → succeeded
                 → failed
```

**The spec uses `succeeded`/`running`, not `complete`/`processing`.** The current code and `StatusStyle()` use the older names — reconcile before adding new status handling.

`job_id` format is `job_<base62>`, not a UUID. Treat as opaque string.

### Generation — field support

| Field | CLI | Spec |
|---|---|---|
| `workspace_id` | ✓ | ✓ |
| `prompt` | ✓ | ✓ |
| `count` | ✓ (`--count`) | 1–4 images, default 1 |
| `seed` | ✓ (`--seed`) | int64, reproducibility |
| `refs` | — | `[]upload_token` for reference images |
| `webhook_url` | — | HTTPS callback on terminal state |

### Uploads — `mime_type` field

`POST /v1/uploads` accepts `{ "count": N, "mime_type": "image/png" }`. The CLI sends `mime_type` via `--mime-type` flag on `ingest` (default `image/png`).

### Job result — additional fields by job kind

Beyond `manifest` and `assets`, results include:

- `graph_fragment` — returned by generation, ingest, and critique jobs. Merge into local `./graph.json` by node ID; do not overwrite the whole file.
- `brain_files` — returned by ingest and critique jobs. Write only the listed files; do not delete others locally.

Asset `url` values are presigned and **expire in 24 hours** — download immediately, never cache the URL.

### Endpoints not yet implemented

- `GET /v1/workspaces/{id}` — single workspace metadata
- `GET /v1/workspaces/{id}/brain` — download brain files to `./brain/`
- `GET /v1/workspaces/{id}/graph` — download KG snapshot to `./graph.json`
- `POST /v1/critiques` — critique a generated or uploaded asset
- `DELETE /v1/jobs/{id}` — cancel a queued/running job
- `/v1/keys` (POST, GET, DELETE) — key management (requires JWT, not API key)

### Idempotency

All job POST endpoints accept `Idempotency-Key: <string>` header. Same key + same body within 24 h replays the original 202. Same key + different body → 409. Useful for CI pipelines.

### `--wait` polling — recommended backoff (not yet implemented)

Spec recommends: 2 s initial → 4 s after 10 s elapsed → cap at 30 s. Current `status --wait` uses a flat 2 s sleep. Studio `JobTracker` polls every 3 s flat.

### Error codes

| HTTP | `code` |
|---|---|
| 400/422 | `invalid_request` |
| 401 | `unauthorized` |
| 403 | `forbidden` |
| 404 | `not_found` |
| 409 | `idempotency_conflict` |
| 429 | `rate_limited` / `quota_exceeded` |
| 5xx | `internal_error` |

Job-level error codes (in `error` field on failed jobs): `quota_exceeded`, `workspace_not_found`, `invalid_reference`, `job_failed`, `webhook_unreachable`.
