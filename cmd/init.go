package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"theartifact-cli/internal/api"
	"theartifact-cli/internal/config"
	"theartifact-cli/internal/ui"
)

var (
	initName     string
	initLinkID   string
	initNoPrompt bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project and link a workspace",
	Long:  "Scaffold .artifact/ dirs, create or link a workspace, and seed it with files from .artifact/input/.",
	RunE: func(cmd *cobra.Command, args []string) error {
		scanner := bufio.NewScanner(os.Stdin)

		globalCfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if globalCfg.APIKey == "" {
			key, err := ui.PromptAPIKey(scanner)
			if err != nil {
				return err
			}
			globalCfg.APIKey = key
			if err := config.Save(globalCfg); err != nil {
				return fmt.Errorf("failed to save credentials: %w", err)
			}
			ui.PrintSuccess("Authenticated")
			fmt.Println()
		}

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("cannot determine working directory: %w", err)
		}

		if !config.ProjectInitialized() {
			if err := config.ScaffoldProject(cwd); err != nil {
				return fmt.Errorf("failed to scaffold project: %w", err)
			}
			ui.PrintSuccess("Project scaffolded")
			ui.PrintKeyValue(".artifact/brain/", "server-managed knowledge files")
			ui.PrintKeyValue(".artifact/input/", "drop images here to seed your workspace")
			ui.PrintKeyValue(".artifact/assets/", "downloaded generation outputs")
			fmt.Println()
		}

		projCfg, err := config.LoadProject()
		if err != nil {
			return fmt.Errorf("failed to load project config: %w", err)
		}
		if projCfg.WorkspaceID != "" {
			ui.PrintInfo(fmt.Sprintf("Already linked to workspace: %s (%s)", projCfg.WorkspaceName, projCfg.WorkspaceID))
			ui.PrintHint("Use /workspace <id> inside Studio to switch.")
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			return err
		}

		projCfg, err = runWorkspaceGate(scanner, client, projCfg, cwd, workspaceGateOptions{
			Name:     initName,
			LinkID:   initLinkID,
			NoPrompt: initNoPrompt,
		})
		if err != nil {
			return err
		}
		if projCfg == nil {
			return nil
		}
		if err := config.SaveProject(projCfg); err != nil {
			return fmt.Errorf("failed to save project config: %w", err)
		}

		fmt.Println()
		ui.PrintSuccess(fmt.Sprintf("Workspace \"%s\" linked", projCfg.WorkspaceName))

		inputDir := filepath.Join(cwd, ".artifact", "input")
		if err := runAutoIngest(client, projCfg.WorkspaceID, inputDir, scanner, initNoPrompt); err != nil {
			ui.PrintWarn(fmt.Sprintf("Auto-ingest skipped: %v", err))
		}

		fmt.Println()
		ui.PrintHint("Run 'theartifact' to open Studio.")
		return nil
	},
}

var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true,
}

func runAutoIngest(client *api.Client, workspaceID, inputDir string, scanner *bufio.Scanner, noPrompt bool) error {
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		ui.PrintInfo("No .artifact/input/ files — skipping ingest.")
		ui.PrintHint("Drop images there and run: theartifact ingest -w <id> -f <file>")
		return nil
	}

	var files []string
	var totalSize int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !imageExts[strings.ToLower(filepath.Ext(e.Name()))] {
			continue
		}
		if info, err := e.Info(); err == nil {
			totalSize += info.Size()
		}
		files = append(files, filepath.Join(inputDir, e.Name()))
	}

	if len(files) == 0 {
		ui.PrintInfo("No image files in .artifact/input/ — skipping ingest.")
		ui.PrintHint("Drop .png/.jpg/.webp files there and run: theartifact ingest -w <id> -f <file>")
		return nil
	}

	ui.PrintInfo(fmt.Sprintf("Found %d file(s) in .artifact/input/ (%s)", len(files), formatBytes(totalSize)))

	if !noPrompt {
		fmt.Print(ui.AccentStyle.Render("  ? Upload to seed workspace? [Y/n] ▸ "))
		if scanner.Scan() {
			ans := strings.ToLower(strings.TrimSpace(scanner.Text()))
			if ans == "n" || ans == "no" {
				ui.PrintHint("Skipped. Run 'theartifact ingest' later to upload.")
				return nil
			}
		}
	}

	sp := ui.NewSpinner(fmt.Sprintf("Requesting %d upload URL(s)…", len(files)))
	sp.Start()
	uploads, err := client.RequestUploadURLs(len(files), "image/png")
	if err != nil {
		sp.Fail("Failed to get upload URLs")
		return fmt.Errorf("upload URL request failed: %w", err)
	}
	sp.Stop(fmt.Sprintf("Got %d upload URL(s)", len(uploads)))

	sp2 := ui.NewSpinner(fmt.Sprintf("Uploading %d file(s)…", len(files)))
	sp2.Start()
	tokens, err := api.UploadFiles(uploads, files, "image/png")
	if err != nil {
		sp2.Fail("Upload failed")
		return err
	}
	sp2.Stop(fmt.Sprintf("Uploaded %d file(s)", len(files)))

	sp3 := ui.NewSpinner("Queueing ingest job…")
	sp3.Start()
	jobID, err := client.TriggerIngest(api.IngestRequest{WorkspaceID: workspaceID, Sources: tokens})
	if err != nil {
		sp3.Fail("Ingest job failed")
		return err
	}
	sp3.Stop("Ingest queued")

	ui.PrintSuccess("Seed files ingested")
	ui.PrintKeyValue("Job", jobID)
	ui.PrintHint(fmt.Sprintf("Brain files appear in .artifact/brain/ when complete. Track: theartifact status %s --wait", jobID))
	return nil
}

func formatBytes(b int64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVar(&initName, "name", "", "Workspace name (skips name prompt)")
	initCmd.Flags().StringVar(&initLinkID, "link", "", "Link to an existing workspace ID")
	initCmd.Flags().BoolVar(&initNoPrompt, "no-prompt", false, "Non-interactive mode; requires --name or --link")
}
