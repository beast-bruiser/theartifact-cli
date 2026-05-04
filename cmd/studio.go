package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"theartifact-cli/internal/api"
	"theartifact-cli/internal/config"
	"theartifact-cli/internal/studio"
	"theartifact-cli/internal/ui"
)

var studioWorkspaceID string

var studioCmd = &cobra.Command{
	Use:   "studio",
	Short: "Open the interactive generation studio",
	Long:  "Launch the interactive studio session. Same as running 'theartifact' with no arguments.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		projCfg, _ := config.LoadProject()

		workspaceID := studioWorkspaceID
		if workspaceID == "" && projCfg != nil {
			workspaceID = projCfg.WorkspaceID
		}

		// Persist flag value to project config
		if studioWorkspaceID != "" && (projCfg == nil || studioWorkspaceID != projCfg.WorkspaceID) {
			if projCfg == nil {
				projCfg = &config.ProjectConfig{}
			}
			projCfg.WorkspaceID = studioWorkspaceID
			_ = config.SaveProject(projCfg)
		}

		workspaceName := workspaceID
		if workspaceID != "" {
			sp := ui.NewSpinner("Connecting to workspace…")
			sp.Start()
			if workspaces, err := client.ListWorkspaces(); err == nil {
				for _, w := range workspaces {
					if w.ID == workspaceID {
						workspaceName = w.Name
						break
					}
				}
				sp.Stop(fmt.Sprintf("Connected to \"%s\"", workspaceName))
			} else {
				sp.Fail("Could not fetch workspace info")
			}
		}

		session := studio.NewSession(client, workspaceID, workspaceName)
		session.Run()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(studioCmd)
	studioCmd.Flags().StringVarP(&studioWorkspaceID, "workspace", "w", "", "Workspace ID (optional)")
}
