package cmd

import (
	"fmt"

	"theartifact-cli/internal/api"
	"theartifact-cli/internal/ui"

	"github.com/spf13/cobra"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage your workspaces",
	Long:  "List and manage your workspaces on TheArtifact.",
}

var listWorkspacesCmd = &cobra.Command{
	Use:   "list",
	Short: "List all accessible workspaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		sp := ui.NewSpinner("Fetching workspaces...")
		sp.Start()

		client, err := api.NewClient()
		if err != nil {
			sp.Fail("Authentication required")
			return err
		}

		workspaces, err := client.ListWorkspaces()
		if err != nil {
			sp.Fail("Failed to fetch workspaces")
			return fmt.Errorf("failed to list workspaces: %w", err)
		}

		sp.Stop(fmt.Sprintf("Found %d workspace(s)", len(workspaces)))

		if len(workspaces) == 0 {
			ui.PrintWarn("No workspaces found. Create one at theartifact.art")
			return nil
		}

		headers := []string{"ID", "Name", "Style Summary"}
		rows := make([][]string, len(workspaces))
		for i, w := range workspaces {
			summary := w.StyleSummary
			if summary == "" {
				summary = "—"
			}
			rows[i] = []string{w.ID, w.Name, summary}
		}
		ui.PrintTable(headers, rows)
		ui.PrintHint(fmt.Sprintf("Use a workspace ID with 'theartifact generate -w <id> -m \"your prompt\"'"))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(workspaceCmd)
	workspaceCmd.AddCommand(listWorkspacesCmd)
}
