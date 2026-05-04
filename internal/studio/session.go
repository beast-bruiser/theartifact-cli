package studio

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"theartifact-cli/internal/api"
	"theartifact-cli/internal/config"
	"theartifact-cli/internal/ui"
)

const (
	promptGlyph  = "you ▸ "
	systemPrefix = "  ◆  "
)

// ChatEntry records a single turn in the studio session.
type ChatEntry struct {
	Role    string // "user" | "system"
	Content string
	Time    time.Time
	JobID   string
}

// Session is the live interactive REPL studio session.
type Session struct {
	client        *api.Client
	workspaceID   string
	workspaceName string
	tracker       *JobTracker
	history       []ChatEntry
}

// NewSession initialises a studio session. workspaceID may be empty — the session
// will ask the user to set one before their first generation.
func NewSession(client *api.Client, workspaceID, workspaceName string) *Session {
	s := &Session{
		client:        client,
		workspaceID:   workspaceID,
		workspaceName: workspaceName,
	}
	s.tracker = NewJobTracker(client, s.onJobComplete)
	return s
}

// Run starts the interactive REPL loop, blocks until the user exits.
func (s *Session) Run() {
	ui.PrintSplash(s.workspaceName)
	s.setupSignalHandler()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		s.printPrompt()
		if !scanner.Scan() {
			// EOF (Ctrl+D)
			s.shutdown("Session ended.")
			return
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "/") {
			if s.handleSlashCommand(line) {
				return // /quit or /exit signals shutdown
			}
			continue
		}

		// Regular prompt → need a workspace to generate
		if s.workspaceID == "" {
			s.promptForWorkspace(scanner, line)
			continue
		}

		s.enqueuePrompt(line)
	}
}

// ── Workspace setup ───────────────────────────────────────────────────────────

// promptForWorkspace asks the user to pick a workspace before the first generation.
func (s *Session) promptForWorkspace(scanner *bufio.Scanner, pendingPrompt string) {
	fmt.Println()
	s.printSystem(ui.WarnStyle.Render("No workspace selected. Choose one to continue:"))
	fmt.Println()

	sp := ui.NewSpinner("Fetching workspaces...")
	sp.Start()
	workspaces, err := s.client.ListWorkspaces()
	if err != nil {
		sp.Fail("Could not fetch workspaces")
		s.printSystem(ui.MutedStyle.Render("Set one manually with /workspace <id>"))
		return
	}
	sp.Stop(fmt.Sprintf("Found %d workspace(s)", len(workspaces)))

	if len(workspaces) == 0 {
		s.printSystem(ui.WarnStyle.Render("No workspaces found. Create one at theartifact.art"))
		return
	}

	fmt.Println()
	for i, w := range workspaces {
		fmt.Printf("  %s  %s\n",
			ui.AccentStyle.Render(fmt.Sprintf("[%d]", i+1)),
			ui.BoldStyle.Render(w.Name)+ui.MutedStyle.Render("  "+w.ID),
		)
	}
	fmt.Println()

	s.printSystem(ui.MutedStyle.Render("Enter a number or paste a workspace ID:"))
	fmt.Print(ui.AccentStyle.Render("workspace ▸ "))

	if !scanner.Scan() {
		return
	}
	choice := strings.TrimSpace(scanner.Text())

	for i, w := range workspaces {
		if choice == fmt.Sprintf("%d", i+1) || strings.EqualFold(choice, w.ID) {
			s.setWorkspace(w.ID, w.Name)
			s.enqueuePrompt(pendingPrompt)
			return
		}
	}
	s.printSystem(ui.ErrorStyle.Render("Invalid selection. Use /workspace <id> to set one."))
}

// setWorkspace updates the active workspace and persists it to project config.
func (s *Session) setWorkspace(id, name string) {
	s.workspaceID = id
	s.workspaceName = name

	pc, err := config.LoadProject()
	if err == nil {
		pc.WorkspaceID = id
		pc.WorkspaceName = name
		_ = config.SaveProject(pc)
	}

	s.printSystem(fmt.Sprintf(
		"%s  Workspace set to %s",
		ui.SuccessStyle.Render("✓"),
		ui.BoldStyle.Render(name),
	))
}

// ── Prompt handling ───────────────────────────────────────────────────────────

func (s *Session) enqueuePrompt(prompt string) {
	s.addHistory("user", prompt, "")

	jobID, err := s.client.Generate(api.GenerateRequest{
		WorkspaceID: s.workspaceID,
		Prompt:      prompt,
	})
	if err != nil {
		errMsg := err.Error()

		// If the workspace is invalid/inaccessible, clear it and re-prompt
		if strings.Contains(errMsg, "forbidden") || strings.Contains(errMsg, "not_found") || strings.Contains(errMsg, "workspace") {
			s.printSystem(ui.WarnStyle.Render("⚠  Current workspace is invalid or inaccessible. Let's pick a new one."))
			s.workspaceID = ""
			s.workspaceName = ""

			pc, loadErr := config.LoadProject()
			if loadErr == nil {
				pc.WorkspaceID = ""
				pc.WorkspaceName = ""
				_ = config.SaveProject(pc)
			}

			scanner := bufio.NewScanner(os.Stdin)
			s.promptForWorkspace(scanner, prompt)
			return
		}

		s.printSystem(ui.ErrorStyle.Render("✗ Failed to queue job: " + errMsg))
		return
	}

	s.tracker.Enqueue(jobID, prompt)
	s.addHistory("system", fmt.Sprintf("Queued job %s", shortID(jobID)), jobID)

	s.printSystem(fmt.Sprintf(
		"%s  %s  %s",
		ui.AccentStyle.Render("⦿"),
		ui.MutedStyle.Render("Queued"),
		ui.AccentStyle.Render(shortID(jobID))+ui.MutedStyle.Render(" — generating in background…"),
	))
}

// ── Slash commands ────────────────────────────────────────────────────────────

// handleSlashCommand processes /cmd lines. Returns true when the session should exit.
func (s *Session) handleSlashCommand(line string) bool {
	parts := strings.Fields(line)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/quit", "/exit", "/q":
		pending := s.tracker.PendingCount()
		if pending > 0 {
			s.printSystem(ui.WarnStyle.Render(fmt.Sprintf("⚠  %d job(s) still running — exiting without waiting.", pending)))
		}
		s.shutdown("Goodbye!")
		return true

	case "/workspace", "/w":
		if len(parts) < 2 {
			if s.workspaceID != "" {
				s.printSystem(fmt.Sprintf("Current workspace: %s  %s",
					ui.BoldStyle.Render(s.workspaceName),
					ui.MutedStyle.Render("("+s.workspaceID+")"),
				))
			} else {
				s.printSystem(ui.WarnStyle.Render("No workspace set. Usage: /workspace <id>"))
			}
			return false
		}
		id := parts[1]
		name := id
		workspaces, err := s.client.ListWorkspaces()
		if err == nil {
			for _, w := range workspaces {
				if w.ID == id {
					name = w.Name
					break
				}
			}
		}
		s.setWorkspace(id, name)

	case "/jobs":
		s.tracker.PrintJobsTable()

	case "/status":
		if len(parts) < 2 {
			s.printSystem(ui.WarnStyle.Render("Usage: /status <job_id>"))
			return false
		}
		s.printSingleJobStatus(parts[1])

	case "/clear":
		fmt.Print("\033[H\033[2J")
		ui.PrintSplash(s.workspaceName)

	case "/help":
		s.printHelp()

	default:
		s.printSystem(ui.MutedStyle.Render(fmt.Sprintf("Unknown command: %s  (try /help)", cmd)))
	}
	return false
}

func (s *Session) printSingleJobStatus(partialID string) {
	snap := s.tracker.Snapshot()
	for _, j := range snap {
		if strings.HasPrefix(j.ID, partialID) || strings.HasPrefix(j.ShortID, partialID) {
			statusStr := ui.StatusStyle(j.Status).Render(j.Status)
			s.printSystem(fmt.Sprintf("Job %s · %s · %d%%", ui.AccentStyle.Render(j.ShortID), statusStr, j.Progress))
			if j.Status == "succeeded" && j.Result != nil {
				for i, a := range j.Result.Assets {
					fmt.Printf("    %s  Asset %d: %s\n", ui.MutedStyle.Render("→"), i+1, ui.ValueStyle.Render(a.URL))
				}
			} else if j.Status == "failed" {
				fmt.Printf("    %s\n", ui.ErrorStyle.Render(j.Error))
			}
			return
		}
	}
	s.printSystem(ui.WarnStyle.Render("No job found with ID: " + partialID))
}

// ── Job completion callback ───────────────────────────────────────────────────

// onJobComplete is called by the JobTracker goroutine when a job reaches a terminal state.
func (s *Session) onJobComplete(job *TrackedJob) {
	fmt.Println()

	if job.Status == "succeeded" {
		assetCount := 0
		if job.Result != nil {
			assetCount = len(job.Result.Assets)
		}
		s.printSystem(fmt.Sprintf(
			"%s  Job %s complete — %s",
			ui.SuccessStyle.Render("✓"),
			ui.AccentStyle.Render(job.ShortID),
			ui.BoldStyle.Render(fmt.Sprintf("%d asset(s) generated", assetCount)),
		))
		if job.Result != nil {
			fmt.Printf("     %s  %s\n", ui.MutedStyle.Render("Prompt:"), ui.DimStyle.Render(job.Prompt))
			// Download assets in the background so the REPL isn't blocked
			go s.downloadAssets(job)
		}
	} else {
		s.printSystem(fmt.Sprintf(
			"%s  Job %s failed — %s",
			ui.ErrorStyle.Render("✗"),
			ui.AccentStyle.Render(job.ShortID),
			ui.ErrorStyle.Render(job.Error),
		))
	}

	s.printPrompt()
}

// ── UI helpers ────────────────────────────────────────────────────────────────

func (s *Session) printPrompt() {
	fmt.Print(ui.AccentStyle.Render(promptGlyph))
}

func (s *Session) printSystem(msg string) {
	fmt.Println(systemPrefix + msg)
}

func (s *Session) printHelp() {
	fmt.Println()
	fmt.Println(ui.AccentStyle.Render("  Studio Commands"))
	fmt.Println(ui.TableBorderStyle.Render("  ──────────────────────────────────────────"))

	cmds := [][2]string{
		{"/workspace <id>", "Switch to a different workspace"},
		{"/jobs", "List all queued and completed jobs"},
		{"/status <id>", "Show status of a specific job"},
		{"/clear", "Clear the terminal screen"},
		{"/help", "Show this help message"},
		{"/quit", "Exit studio (running jobs are abandoned)"},
	}
	for _, c := range cmds {
		fmt.Printf("  %-22s %s\n",
			ui.AccentStyle.Render(c[0]),
			ui.MutedStyle.Render(c[1]),
		)
	}
	fmt.Println()
}

func (s *Session) shutdown(msg string) {
	s.tracker.Stop()
	fmt.Println()
	fmt.Println(ui.MutedStyle.Render("  " + msg))
	fmt.Println()
}

func (s *Session) addHistory(role, content, jobID string) {
	s.history = append(s.history, ChatEntry{
		Role:    role,
		Content: content,
		Time:    time.Now(),
		JobID:   jobID,
	})
}

// downloadAssets runs in a goroutine — downloads job assets to .artifact/assets/
// and prints a summary line once done without blocking the REPL.
func (s *Session) downloadAssets(job *TrackedJob) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	destDir := filepath.Join(cwd, ".artifact", "assets")

	paths, errs := api.DownloadJobAssets(job.Result, destDir)
	for _, e := range errs {
		fmt.Println()
		s.printSystem(ui.WarnStyle.Render("Download warning: " + e.Error()))
	}

	if len(paths) > 0 {
		fmt.Println()
		s.printSystem(fmt.Sprintf(
			"%s  Saved %s to %s",
			ui.SuccessStyle.Render("↓"),
			ui.BoldStyle.Render(fmt.Sprintf("%d asset(s)", len(paths))),
			ui.AccentStyle.Render(".artifact/assets/"),
		))
		s.printPrompt()
	}
}

// setupSignalHandler intercepts Ctrl+C to print a tidy exit message.
func (s *Session) setupSignalHandler() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		pending := s.tracker.PendingCount()
		fmt.Println()
		if pending > 0 {
			fmt.Println(ui.WarnStyle.Render(fmt.Sprintf("\n  ⚠  Interrupted — %d job(s) were still running.", pending)))
		}
		s.tracker.Stop()
		fmt.Println(ui.MutedStyle.Render("  Goodbye!"))
		fmt.Println()
		os.Exit(0)
	}()
}
