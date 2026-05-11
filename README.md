# TheArtifact CLI

> Generate AI assets from your terminal. Powered by [theartifact.art](https://theartifact.art).

```
you ▸ a knight in full plate armour at dusk, cinematic lighting
  ◆  ⦿  Queued a1b2c3d4 — generating in background…
  ✓  Job a1b2c3d4 complete — 1 asset(s) generated
  ↓  Saved 1 asset(s) to .artifact/assets/
```

---

## Install

**macOS / Linux (Homebrew-style one-liner)**

```bash
git clone https://github.com/theartifact/theartifact-cli && \
  cd theartifact-cli && \
  go build -o theartifact . && \
  sudo mv theartifact /usr/local/bin/
```

**From source (any platform with Go 1.21+)**

```bash
git clone https://github.com/theartifact/theartifact-cli
cd theartifact-cli
go build -o theartifact .
```

Then move the binary somewhere on your `PATH`.

**Verify**

```bash
theartifact --help
```

---

## First run

```bash
theartifact
```

That's it. The CLI walks you through three steps the first time:

1. **Paste your API key** — get one at [theartifact.art/api-keys](https://theartifact.art/api-keys). Press `Enter` to open the page in your browser.
2. **Project folder is created** — a `.artifact/` directory in your current folder.
3. **Pick or create a workspace** — name it whatever you want.

Then you're in Studio. Type a prompt, get an image.

---

## Studio mode

Just run `theartifact` and start typing prompts:

```
you ▸ neon cityscape, raining
you ▸ same city but at sunrise
```

Each prompt runs in the background — keep typing, results arrive when ready. Images auto-download to `.artifact/assets/`.

### Slash commands

| Command | What it does |
|---|---|
| `/workspace <id>` | Switch workspaces |
| `/ingest` | Upload everything in `.artifact/input/` now |
| `/jobs` | List recent jobs |
| `/status <id>` | Check a specific job |
| `/clear` | Clear screen |
| `/help` | Show all commands |
| `/quit` | Exit |

### Drop-to-ingest

While Studio is running, drop any image into `.artifact/input/` — it uploads automatically and teaches your workspace that style.

---

## Project folder

Run `theartifact` in any folder and it creates this:

```
your-project/
└── .artifact/
    ├── config.json    workspace binding
    ├── brain/         knowledge files (managed by the server)
    ├── input/         drop reference images here
    └── assets/        your generated outputs
```

Each folder can be linked to a different workspace. Switch projects = switch folders.

---

## One-shot commands

Skip Studio. Useful for scripts and CI.

**Generate**
```bash
theartifact generate -w <workspace_id> -m "a dragon made of circuits"
theartifact generate -w <workspace_id> -m "neon cityscape" --count 4 --seed 42
```

**Wait for a job and download the result**
```bash
theartifact status <job_id> --wait
```

**Upload reference images**
```bash
theartifact ingest -w <workspace_id> -f ./ref1.png -f ./ref2.jpg
```

**List your workspaces**
```bash
theartifact workspace list
```

**Set up a project non-interactively (CI-friendly)**
```bash
theartifact init --name "Dark Fantasy" --no-prompt
# or link to an existing one
theartifact init --link ws_abc123 --no-prompt
```

---

## Authentication

Your API key is saved once at `~/.artifact/config.json` and reused everywhere.

To set or replace it manually:

```bash
theartifact login --key tak_live_<your-key>
```

Get a key at **[theartifact.art/api-keys](https://theartifact.art/api-keys)**.

---

## All flags at a glance

| Flag | Where | What it does |
|---|---|---|
| `-w, --workspace` | `studio`, `generate`, `ingest` | Pick a workspace |
| `-m, --message` | `generate` | The prompt |
| `--count` | `generate` | How many images (1–4) |
| `--seed` | `generate` | Reproducible output |
| `-f, --file` | `ingest` | File to upload (repeat for many) |
| `--mime-type` | `ingest` | Defaults to `image/png` |
| `--name` | `init` | Create a workspace with this name |
| `--link` | `init` | Bind to an existing workspace ID |
| `--no-prompt` | `init` | Skip all prompts (CI mode) |
| `--wait` | `status` | Poll until done |
| `--no-download` | `status` | Print URLs instead of saving |
| `--no-color` | all | Disable colors |

**Environment variables**

| Var | Purpose |
|---|---|
| `ARTIFACT_API_URL` | Point at a different backend (e.g. `http://localhost:8081`) |
| `NO_COLOR` | Disable colors |

---

## Common errors

| Message | Fix |
|---|---|
| `Not authenticated` | Run `theartifact login --key <key>` |
| `forbidden` | Your key can't access that workspace |
| `not_found` | Wrong workspace or job ID |
| `quota_exceeded` | You hit your monthly limit |
| `rate_limited` | Slow down and retry |

---

## Tips

- **Queue freely** — multiple prompts run in parallel, you don't have to wait.
- **Reproduce results** — use `--seed <n>` with the same prompt.
- **Auto-style** — drop images into `.artifact/input/` and they shape future generations.
- **Pipe-safe** — set `NO_COLOR=1` to strip ANSI codes.
- **Per-folder workspaces** — each directory remembers its own workspace.

---

## Development

```bash
# Build & run
go build -o theartifact . && ./theartifact

# Local backend
ARTIFACT_API_URL=http://localhost:8081 ./theartifact

# Test & lint
go test ./...
go vet ./...

# Cross-compile
GOOS=darwin GOARCH=arm64 go build -o theartifact-darwin-arm64 .
GOOS=linux  GOARCH=amd64 go build -o theartifact-linux-amd64 .
```

Source layout:

```
cmd/         CLI commands (one file per command)
internal/
  api/       HTTP client & API methods
  config/    global + project config
  studio/    REPL, job tracker, input watcher
  ui/        styling, prompts, spinners
```
