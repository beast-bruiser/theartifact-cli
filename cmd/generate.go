package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"theartifact-cli/internal/api"
	"theartifact-cli/internal/ui"
)

var (
	generateWorkspaceID string
	generatePrompt      string
	generateCount       int
	generateSeed        int64
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Start a generation job",
	Long:  "Trigger generation of new assets in a workspace based on a text prompt.",
	RunE: func(cmd *cobra.Command, args []string) error {
		sp := ui.NewSpinner("Starting generation...")
		sp.Start()

		client, err := api.NewClient()
		if err != nil {
			sp.Fail("Authentication required")
			return err
		}

		req := api.GenerateRequest{
			WorkspaceID: generateWorkspaceID,
			Prompt:      generatePrompt,
			Count:       generateCount,
			Seed:        generateSeed,
		}

		jobID, err := client.Generate(req)
		if err != nil {
			sp.Fail("Generation request failed")
			return fmt.Errorf("failed to start generation: %w", err)
		}

		sp.Stop("Generation job queued")

		ui.PrintSuccess("Job submitted successfully")
		ui.PrintKeyValue("Job ID", jobID)
		ui.PrintKeyValue("Workspace", generateWorkspaceID)
		ui.PrintKeyValue("Prompt", generatePrompt)
		ui.PrintHint(fmt.Sprintf("Track progress: theartifact status %s --wait", jobID))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVarP(&generateWorkspaceID, "workspace", "w", "", "Workspace ID")
	generateCmd.Flags().StringVarP(&generatePrompt, "message", "m", "", "Text prompt for generation")
	generateCmd.Flags().IntVar(&generateCount, "count", 0, "Number of images to generate (1–4, default 1)")
	generateCmd.Flags().Int64Var(&generateSeed, "seed", 0, "Seed for reproducible generation")
	generateCmd.MarkFlagRequired("workspace") //nolint:errcheck
	generateCmd.MarkFlagRequired("message")   //nolint:errcheck
}
