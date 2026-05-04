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
./theartifact status <job_id> --wait              # auto-downloads assets to .artifact/assets/
./theartifact status <job_id> --wait --no-download # print URLs only
./theartifact workspace list

# Lint & vet
go vet ./...

# Tests
go test ./internal/api ./internal/studio ./internal/config -v
go test ./internal/api ./internal/studio ./internal/config -cover

# Cross-compile
GOOS=darwin GOARCH=arm64 go build -o theartifact-darwin-arm64 .
GOOS=linux  GOARCH=amd64 go build -o theartifact-linux-amd64 .
```

## Architecture

Go CLI (Cobra-based) for TheArtifact API. Entry point: `main.go` → `cmd.Execute()`. Running with no subcommand enters **Studio mode** (interactive REPL). First run runs a 3-step onboarding orchestrator inline.

### Package layout

| Package | Role |
|---|---|
| `cmd/` | One file per Cobra command (`root` = onboarding orchestrator + Studio entry, `auth` = `login` alias for CI, `generate`, `ingest`, `status`, `workspace`, `studio` = explicit alias) |
| `internal/api/` | HTTP client and API methods — one file per resource (`client`, `workspaces`, `generations`, `ingest`, `jobs`, `assets`) |
| `internal/config/` | Global config (`~/.artifact/config.json`), project config (`<cwd>/.artifact/config.json`), `ScaffoldProject` |
| `internal/studio/` | Interactive REPL session (`session.go`) and background job poller (`tracker.go`) |
| `internal/ui/` | Terminal output: lipgloss styles (`theme.go`), formatted printers (`printer.go`), spinner wrapper (`spinner.go`), onboarding prompts (`onboarding.go`) |

### Key data flows

**Onboarding orchestrator** (`cmd/root.go` `RunE`): Three sequential gates run on every `theartifact` invocation:
1. **API key gate** — if `~/.artifact/config.json` has no `api_key`, call `ui.PromptAPIKey()`: prints `https://theartifact.art/api-keys`, `[Enter]` opens the browser (TTY only), accepts pasted key, saves to global config.
2. **Scaffold gate** — if `.artifact/` does not exist in cwd, call `config.ScaffoldProject(cwd)`: creates `.artifact/{brain,input,assets}/` and `brain/.README`.
3. **Workspace gate** — if `.artifact/config.json` has no `workspace_id`, fetch the API workspace list: 0 → print web link and exit; 1 → auto-select; N → numbered picker. Selection is saved to project config.

After all gates pass, `studio.NewSession(...).Run()` is called.

**Auth (non-interactive):** `theartifact login --key ...` writes the key directly to global config. Used for CI/scripts.

**Studio (default mode):** Running `theartifact` (no subcommand) opens `studio.Session.Run()` — a blocking REPL that reads stdin, calls `client.Generate()`, and enqueues jobs into `JobTracker`. The tracker polls `/v1/jobs/{id}` every 3 seconds in a background goroutine and fires `onJobComplete` when a job reaches a terminal state. On `succeeded`, assets are downloaded to `.artifact/assets/` in a goroutine so the REPL stays responsive. If the workspace becomes invalid mid-session, Studio clears it and re-prompts inline.

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

## Config files

### Global — `~/.artifact/config.json` (0600 permissions)

Shared across all projects on this machine.

| Field | Required | Description |
|---|---|---|
| `api_key` | yes | API key (`tak_live_...`) — set on first run or via `login` |
| `base_url` | no | Override API base URL (e.g. `http://localhost:8081`) |

### Project — `<cwd>/.artifact/config.json`

Created by `config.ScaffoldProject()` on first run in a directory. Binds the directory to a workspace.

| Field | Description |
|---|---|
| `workspace_id` | The workspace linked to this directory — authoritative |
| `workspace_name` | Display name cache — refreshed if empty |
| `initialized_at` | ISO timestamp of first scaffold |

`ARTIFACT_API_URL` env var overrides `base_url` in global config (takes precedence).

### Project folder layout

```
<cwd>/
  .artifact/
    config.json    ← project config (workspace binding)
    brain/         ← server-managed files (do not hand-edit; overwritten by ingest/critique jobs)
    input/         ← reference files to upload for ingest
    assets/        ← generated assets, auto-downloaded on job completion
```

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
