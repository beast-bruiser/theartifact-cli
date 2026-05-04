# TheArtifact CLI

A fast, terminal-native client for [theartifact.art](https://theartifact.art) — generate AI assets, ingest reference files, and manage your workspaces from the command line.

---

## Quick Start

```bash
# Just run it — the CLI handles everything else
theartifact
```

On first run the CLI walks you through three steps automatically:

**1. API key** — prints the key dashboard URL, offers to open your browser, then accepts a pasted key:
```
  ● Welcome to TheArtifact
  AI-powered asset generation · theartifact.art

  You need an API key to continue.
  https://theartifact.art/api-keys

  [Enter] open in browser · or paste your key now: tak_live_...
  ✓  Authenticated
```

**2. Project scaffold** — creates a `.artifact/` folder in the current directory:
```
  ✓  Project initialized
  .artifact/brain/    server-managed knowledge files
  .artifact/input/    files to ingest
  .artifact/assets/   downloaded generation outputs
```

**3. Workspace selection** — picks your workspace and links it to this directory:
```
  [1]  Dark Fantasy    ws_abc123
  [2]  Sci-Fi Brand    ws_def456

  workspace ▸ 1
  ✓  Workspace set to: Dark Fantasy
```

After first run, `theartifact` drops straight into Studio Mode every time.

---

## Installation

### Build from source (requires Go 1.21+)

```bash
git clone https://github.com/theartifact/theartifact-cli
cd theartifact-cli
go build -o theartifact .

# Move to PATH
mv theartifact /usr/local/bin/
```

---

## Project structure

Each directory you run `theartifact` in becomes a project. On first run the CLI scaffolds:

```
your-project/
  .artifact/
    config.json    ← workspace binding (workspace_id, workspace_name)
    brain/         ← server-managed knowledge files (do not hand-edit)
    input/         ← reference files to upload for ingest
    assets/        ← generated assets, auto-downloaded on completion
```

The `.artifact/` folder is project-local — you can have different workspaces in different directories.

> **Note:** Add `.artifact/config.json` to `.gitignore` if you don't want workspace bindings committed. The `brain/`, `input/`, and `assets/` directories are safe to gitignore too.

---

## Authentication

Authentication happens automatically on first run. To set or update your key manually:

```bash
theartifact login --key tak_live_<your-key>
```

Your key is stored at `~/.artifact/config.json` with `0600` permissions and reused across all projects.

> Get your API key at **[theartifact.art/api-keys](https://theartifact.art/api-keys)**.

---

## Studio Mode (Interactive)

Studio is the primary way to use the CLI — just like `claude` launches Claude Code.

```bash
# Open studio (uses the workspace linked to this directory)
theartifact

# Open studio with a specific workspace (overrides project config)
theartifact -w <workspace_id>

# Explicit subcommand (same thing)
theartifact studio
```

### Studio slash commands

| Command | Description |
|---------|-------------|
| `/workspace` | Show current workspace |
| `/workspace <id>` | Switch to a different workspace mid-session |
| `/jobs` | List all queued and completed jobs in this session |
| `/status <id>` | Show details for a specific job |
| `/clear` | Clear the terminal screen |
| `/help` | Show all available commands |
| `/quit` | Exit studio |

### Example session

```
  ● Studio Mode · Workspace: Dark Fantasy
  Type a prompt to generate · /help for commands · /quit to exit

you ▸ a knight in full plate armour at dusk, cinematic lighting
  ◆  ⦿  Queued a1b2c3d4 — generating in background…

you ▸ the same knight from above, but victorious on a battlefield
  ◆  ⦿  Queued b2c3d4e5 — generating in background…

  ✓  Job a1b2c3d4 complete — 1 asset(s) generated
     Prompt: a knight in full plate armour at dusk, cinematic lighting
  ↓  Saved 1 asset(s) to .artifact/assets/

you ▸
```

> **Tip:** Queue multiple prompts without waiting — each runs as an independent background job and announces when it's done. Assets are downloaded automatically to `.artifact/assets/`.

---

## One-Shot Commands

These work without entering Studio Mode — useful for scripts and CI.

### Generate

```bash
# Start a generation job and print the job ID
theartifact generate -w <workspace_id> -m "a dragon made of circuits"

# Generate multiple images
theartifact generate -w <workspace_id> -m "neon cityscape" --count 4

# Reproducible generation with a fixed seed
theartifact generate -w <workspace_id> -m "forest at midnight" --seed 42
```

### Check job status

```bash
# One-time status check
theartifact status <job_id>

# Poll until the job finishes and auto-download assets to .artifact/assets/
theartifact status <job_id> --wait

# Poll but print URLs instead of downloading
theartifact status <job_id> --wait --no-download
```

### Ingest reference files

Upload images to a workspace's brain so future generations are styled by them.

```bash
# Single file
theartifact ingest -w <workspace_id> -f ./reference.png

# Multiple files
theartifact ingest -w <workspace_id> -f ./ref1.png -f ./ref2.jpg

# Track the resulting job
theartifact status <job_id> --wait
```

The CLI handles the full 3-step upload flow automatically:
1. Requests presigned upload URLs from the API
2. Uploads each file as a binary `PUT` to the signed URL
3. Triggers the ingestion job with the returned upload tokens

### List workspaces

```bash
theartifact workspace list
```

---

## Configuration

### Global config — `~/.artifact/config.json`

Stores your API key. Shared across all projects on this machine.

| Field | Description |
|-------|-------------|
| `api_key` | Your API key — set on first run or via `theartifact login` |
| `base_url` | Override API base URL (e.g. `http://localhost:8081` for local dev) |

### Project config — `<cwd>/.artifact/config.json`

Stores the workspace binding for the current directory. Created automatically on first run.

| Field | Description |
|-------|-------------|
| `workspace_id` | The workspace linked to this directory |
| `workspace_name` | Display name cache — refreshed automatically |
| `initialized_at` | Timestamp of first init |

### Environment variables

| Variable | Description |
|----------|-------------|
| `ARTIFACT_API_URL` | Override API base URL (takes precedence over config `base_url`) |
| `NO_COLOR` | Set to any value to disable color output |

### Flags

| Flag | Commands | Description |
|------|----------|-------------|
| `--no-color` | all | Disable color output |
| `-w, --workspace` | `studio`, `generate`, `ingest` | Workspace ID override |
| `--wait` | `status` | Poll until job completes |
| `--no-download` | `status` | Print asset URLs instead of downloading |

---

## Error Reference

| Error | Meaning |
|-------|---------|
| `Not authenticated` | Run `theartifact login --key <key>` or re-run `theartifact` |
| `forbidden` | Your API key doesn't have access to this workspace |
| `quota_exceeded` | Monthly generation quota exhausted |
| `rate_limited` | Too many requests — slow down or wait |
| `not_found` | Workspace or job ID doesn't exist |

---

## Tips

- **Batch generation**: Queue multiple prompts in Studio before any complete — they run in parallel.
- **Auto-download**: Assets land in `.artifact/assets/` automatically. Use `--no-download` to get URLs instead.
- **Reproducible results**: Use `--seed <number>` for the same output from the same prompt.
- **CI/scripts**: Use one-shot commands (`generate`, `status --wait --no-download`) for automation.
- **Pipe-safe output**: Add `--no-color` or set `NO_COLOR=1` to strip ANSI codes.
- **Multiple projects**: Each directory gets its own `.artifact/config.json` workspace binding.

---

## Project Source Structure

```
theartifact-cli/
├── main.go
├── cmd/
│   ├── root.go        # Entry point → onboarding orchestrator → Studio
│   ├── auth.go        # login --key (non-interactive/CI)
│   ├── generate.go    # generate command
│   ├── ingest.go      # ingest command
│   ├── status.go      # status command (--wait, --no-download)
│   ├── studio.go      # studio subcommand alias
│   └── workspace.go   # workspace list
└── internal/
    ├── api/           # HTTP client, resource methods, asset downloader
    ├── config/        # Global + project config, ScaffoldProject
    ├── studio/        # REPL session + background job tracker
    └── ui/            # Theme, printer, spinner, onboarding prompts
```
