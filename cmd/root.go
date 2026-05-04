package cmd

import (
	"bufio"
	"fmt"
	"os"

	"theartifact-cli/internal/api"
	"theartifact-cli/internal/config"
	"theartifact-cli/internal/studio"
	"theartifact-cli/internal/ui"

	"github.com/spf13/cobra"
)

var (
	noColor         bool
	rootWorkspaceID string
)

var rootCmd = &cobra.Command{
	Use:   "theartifact",
	Short: "TheArtifact — AI-powered asset generation studio",
	Long:  "TheArtifact CLI — Type a prompt, generate assets. Powered by theartifact.art",

	RunE: func(cmd *cobra.Command, args []string) error {
		scanner := bufio.NewScanner(os.Stdin)

		// ── Step 1: ensure global API key ─────────────────────────────────────
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
			sp := ui.NewSpinner("Saving credentials…")
			sp.Start()
			if err := config.Save(globalCfg); err != nil {
				sp.Fail("Failed to save credentials")
				return err
			}
			sp.Stop("Credentials saved")
			ui.PrintSuccess("Authenticated")
			ui.PrintKeyValue("Key", ui.MaskKey(key))
			fmt.Println()
		}

		// ── Step 2: scaffold project if not yet initialized ────────────────────
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("cannot determine working directory: %w", err)
		}

		if !config.ProjectInitialized() {
			if err := config.ScaffoldProject(cwd); err != nil {
				return fmt.Errorf("failed to scaffold project: %w", err)
			}
			ui.PrintSuccess("Project initialized")
			ui.PrintKeyValue(".artifact/brain/", "server-managed knowledge files")
			ui.PrintKeyValue(".artifact/input/", "files to ingest")
			ui.PrintKeyValue(".artifact/assets/", "downloaded generation outputs")
			fmt.Println()
		}

		// ── Step 3: resolve workspace ──────────────────────────────────────────
		projCfg, err := config.LoadProject()
		if err != nil {
			return fmt.Errorf("failed to load project config: %w", err)
		}

		// Migrate legacy global workspace_id on first run
		if projCfg.WorkspaceID == "" && globalCfg.LegacyDefaultWorkspaceID != "" {
			projCfg.WorkspaceID = globalCfg.LegacyDefaultWorkspaceID
			globalCfg.LegacyDefaultWorkspaceID = ""
			_ = config.Save(globalCfg)
		}

		// Flag overrides project config
		if rootWorkspaceID != "" {
			projCfg.WorkspaceID = rootWorkspaceID
		}

		client, err := api.NewClient()
		if err != nil {
			return err
		}

		if projCfg.WorkspaceID == "" {
			projCfg, err = runWorkspaceGate(scanner, client, projCfg)
			if err != nil {
				return err
			}
			if projCfg == nil {
				return nil // no workspaces — user directed to web
			}
		}
		_ = config.SaveProject(projCfg)

		// Resolve display name if not cached
		workspaceName := projCfg.WorkspaceName
		if workspaceName == "" {
			workspaceName = projCfg.WorkspaceID
			sp := ui.NewSpinner("Connecting to workspace…")
			sp.Start()
			if workspaces, err := client.ListWorkspaces(); err == nil {
				for _, w := range workspaces {
					if w.ID == projCfg.WorkspaceID {
						workspaceName = w.Name
						projCfg.WorkspaceName = w.Name
						_ = config.SaveProject(projCfg)
						break
					}
				}
				sp.Stop(fmt.Sprintf("Connected to \"%s\"", workspaceName))
			} else {
				sp.Fail("Could not fetch workspace info")
			}
		}

		session := studio.NewSession(client, projCfg.WorkspaceID, workspaceName)
		session.Run()
		return nil
	},
}

// runWorkspaceGate fetches workspaces and lets the user pick one.
// Returns nil projCfg when there are no workspaces (user directed to web).
func runWorkspaceGate(scanner *bufio.Scanner, client *api.Client, projCfg *config.ProjectConfig) (*config.ProjectConfig, error) {
	fmt.Println()
	fmt.Println(ui.WarnStyle.Render("  No workspace linked to this project. Let's set one up."))

	sp := ui.NewSpinner("Fetching workspaces…")
	sp.Start()
	workspaces, err := client.ListWorkspaces()
	if err != nil {
		sp.Fail("Could not fetch workspaces")
		return nil, err
	}
	sp.Stop(fmt.Sprintf("Found %d workspace(s)", len(workspaces)))

	if len(workspaces) == 0 {
		ui.PrintNoWorkspacesHint()
		return nil, nil
	}

	items := make([]ui.WorkspaceItem, len(workspaces))
	for i, w := range workspaces {
		items[i] = ui.WorkspaceItem{ID: w.ID, Name: w.Name}
	}

	if len(workspaces) == 1 {
		projCfg.WorkspaceID = workspaces[0].ID
		projCfg.WorkspaceName = workspaces[0].Name
		ui.PrintSuccess(fmt.Sprintf("Auto-selected workspace: %s", workspaces[0].Name))
		return projCfg, nil
	}

	id, name := ui.PromptWorkspace(scanner, items)
	if id == "" {
		ui.PrintWarn("Invalid selection.")
		return nil, nil
	}
	projCfg.WorkspaceID = id
	projCfg.WorkspaceName = name
	ui.PrintSuccess(fmt.Sprintf("Workspace set to: %s", name))
	return projCfg, nil
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ui.PrintError(err.Error())
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color output")
	rootCmd.PersistentFlags().StringVarP(&rootWorkspaceID, "workspace", "w", "", "Workspace ID to use in studio")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if noColor || os.Getenv("NO_COLOR") != "" {
			os.Setenv("NO_COLOR", "1")
		}
		return nil
	}
}
