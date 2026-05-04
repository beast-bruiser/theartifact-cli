# TheArtifact CLI

A fast, terminal-native client for [theartifact.art](https://theartifact.art) — generate AI assets, ingest reference files, and manage your workspaces from the command line.

---

## Quick Start

```bash
# Just run it — the CLI handles everything else
theartifact
```

The CLI drops you straight into **Studio Mode**. If it's your first time, it walks you through authentication before opening the studio:

```
  ● Welcome to TheArtifact
  AI-powered asset generation · theartifact.art

  To get started, enter your API key.
  Get one at: theartifact.art/settings → API Keys

  API key ▸ tak_live_...
  ✓  Credentials saved
  ✓  Authenticated
```

You only authenticate once — the key is saved and reused on every subsequent run.

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

## Authentication

Authentication happens automatically on first run — you'll be prompted for your API key inline.

To set or update your key manually:

```bash
theartifact login --key tak_live_<your-key>
```

Your key is stored securely at `~/.artifact/config.json` with `0600` permissions.

> Get your API key at **theartifact.art → Settings → API Keys**.

---

## Studio Mode (Interactive)

Studio is the primary way to use the CLI — just like `claude` launches Claude Code.

```bash
# Open studio (uses your last workspace, or prompts you to pick one)
theartifact

# Open studio in a specific workspace
theartifact -w <workspace_id>

# Explicit subcommand (same thing)
theartifact studio -w <workspace_id>
```

### First-time workspace setup

If you haven't used a workspace yet, the CLI shows a numbered picker the moment you type your first prompt:

```
you ▸ a cyberpunk samurai in neon rain

  ◆  No workspace selected. Choose one to continue:

  [1]  Dark Fantasy    abc123-...
  [2]  Sci-Fi Brand    def456-...

workspace ▸ 1
  ◆  ✓  Workspace set to Dark Fantasy
  ◆  ⦿  Queued a1b2c3d4 — generating in background…
you ▸ 
```

Your selection is saved as the default — next time you run `theartifact`, it connects automatically.

### Studio slash commands

| Command | Description |
|---------|-------------|
| `/jobs` | List all queued and completed jobs in this session |
| `/status <id>` | Show details for a specific job (use first 8 chars of ID) |
| `/workspace <id>` | Switch to a different workspace mid-session |
| `/workspace` | Show the current workspace |
| `/clear` | Clear the terminal screen |
| `/help` | Show all available commands |
| `/quit` | Exit studio |

### Example studio session

```
  ● Studio Mode · Workspace: Dark Fantasy
  Type a prompt to generate · /help for commands · /quit to exit

you ▸ a knight in full plate armour at dusk, cinematic lighting
  ◆  ⦿  Queued a1b2c3d4 — generating in background…

you ▸ the same knight from above, but victorious on a battlefield
  ◆  ⦿  Queued b2c3d4e5 — generating in background…

you ▸ /jobs
  Job ID   │  Status      │  Progress  │  Prompt
  ──────────┼──────────────┼────────────┼───────────────────────────────
  a1b2c3d4  │  succeeded   │  100%      │  a knight in full plate armo...
  b2c3d4e5  │  running     │  60%       │  the same knight from above,...

  ✓  Job a1b2c3d4 complete — 1 asset(s) generated
     Prompt: "a knight in full plate armour at dusk, cinematic lighting"
     → Asset 1: https://cdn.theartifact.art/assets/xxx.png

you ▸ /quit
```

> **Tip:** You can queue multiple prompts without waiting — each one runs as an independent background job and announces when it's done.

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

# Poll until the job finishes (prints asset URLs on completion)
theartifact status <job_id> --wait
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

Output:
```
  ID                                    │  Name              │  Style Summary
  ──────────────────────────────────────┼────────────────────┼──────────────
  abc123-...                            │  Dark Fantasy       │  Gritty mediev...
  def456-...                            │  Sci-Fi Brand       │  Hard sci-fi, c...
```

---

## Configuration

Config is stored at `~/.artifact/config.json`:

```json
{
  "api_key": "tak_live_...",
  "default_workspace_id": "abc123-..."
}
```

| Field | Description |
|-------|-------------|
| `api_key` | Your API key — set with `theartifact login` |
| `default_workspace_id` | Last-used workspace — set automatically when you pick one in Studio |

### Environment variables

| Variable | Description |
|----------|-------------|
| `NO_COLOR` | Set to any value to disable color output |

### Flags

| Flag | Description |
|------|-------------|
| `--no-color` | Disable color output (same as `NO_COLOR=1`) |
| `-w, --workspace` | Workspace ID — applies to `studio`, `generate`, `ingest` |

---

## Error Reference

| Error | Meaning |
|-------|---------|
| `Not authenticated` | Run `theartifact login --key <key>` first |
| `forbidden` | Your API key doesn't have access to this workspace |
| `quota_exceeded` | Monthly generation quota exhausted |
| `rate_limited` | Too many requests — slow down or wait |
| `not_found` | Workspace or job ID doesn't exist |

---

## Tips

- **Batch generation**: Send multiple prompts in Studio before any complete — they all queue up and run in parallel.
- **Reproducible results**: Use `--seed <number>` to get the same output for the same prompt.
- **CI/scripts**: Use one-shot commands (`generate`, `status --wait`) instead of Studio for automation.
- **Pipe-safe output**: Add `--no-color` or set `NO_COLOR=1` to strip ANSI codes when piping to files or other tools.
- **Switch workspaces**: Use `/workspace <id>` inside Studio without restarting.

---

## Project Structure

```
theartifact-cli/
├── main.go                   # Entry point
├── cmd/
│   ├── root.go               # Root command → Studio mode
│   ├── auth.go               # login command
│   ├── generate.go           # generate command
│   ├── ingest.go             # ingest command
│   ├── status.go             # status command
│   ├── studio.go             # studio subcommand
│   └── workspace.go          # workspace subcommand
└── internal/
    ├── api/                  # API client (workspaces, jobs, generations, ingest)
    ├── config/               # Config file read/write (~/.artifact/config.json)
    ├── studio/               # Studio REPL engine + background job tracker
    └── ui/                   # Design system (theme, spinner, printer)
```
