package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"theartifact-cli/internal/api"
	"theartifact-cli/internal/ui"
)

var (
	ingestWorkspaceID string
	ingestFiles       []string
	ingestMimeType    string
)

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Upload and ingest reference files",
	Long:  "Upload local files and ingest them into a workspace's brain.",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		ui.PrintInfo(fmt.Sprintf("Preparing to ingest %d file(s) into workspace %s", len(ingestFiles), ingestWorkspaceID))

		// ── Step 1: Request Upload URLs ────────────────────────────────────────
		sp := ui.NewSpinner("Requesting upload URLs...")
		sp.Start()

		uploadTokens, err := client.RequestUploadURLs(len(ingestFiles), ingestMimeType)
		if err != nil {
			sp.Fail("Failed to get upload URLs")
			return fmt.Errorf("failed to get upload URLs: %w", err)
		}
		if len(uploadTokens) != len(ingestFiles) {
			sp.Fail(fmt.Sprintf("Expected %d URLs, got %d", len(ingestFiles), len(uploadTokens)))
			return fmt.Errorf("API returned wrong number of upload URLs")
		}
		sp.Stop(fmt.Sprintf("Got %d upload URL(s)", len(uploadTokens)))

		// ── Step 2: Upload Binaries ────────────────────────────────────────────
		sp2 := ui.NewSpinner(fmt.Sprintf("Uploading %d file(s)…", len(ingestFiles)))
		sp2.Start()
		sourceTokens, err := api.UploadFiles(uploadTokens, ingestFiles, ingestMimeType)
		if err != nil {
			sp2.Fail("Upload failed")
			return err
		}
		sp2.Stop(fmt.Sprintf("Uploaded %d file(s)", len(ingestFiles)))

		// ── Step 3: Trigger Ingest ─────────────────────────────────────────────
		sp3 := ui.NewSpinner("Triggering ingestion...")
		sp3.Start()

		ingestReq := api.IngestRequest{
			WorkspaceID: ingestWorkspaceID,
			Sources:     sourceTokens,
		}

		jobID, err := client.TriggerIngest(ingestReq)
		if err != nil {
			sp3.Fail("Ingestion request failed")
			return fmt.Errorf("failed to trigger ingest: %w", err)
		}
		sp3.Stop("Ingestion job queued")

		ui.PrintSuccess("Files ingested successfully")
		ui.PrintKeyValue("Job ID", jobID)
		ui.PrintKeyValue("Workspace", ingestWorkspaceID)
		ui.PrintKeyValue("Files", fmt.Sprintf("%d", len(ingestFiles)))
		ui.PrintHint(fmt.Sprintf("Track progress: theartifact status %s --wait", jobID))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(ingestCmd)
	ingestCmd.Flags().StringVarP(&ingestWorkspaceID, "workspace", "w", "", "Workspace ID")
	ingestCmd.Flags().StringSliceVarP(&ingestFiles, "file", "f", []string{}, "File path(s) to upload")
	ingestCmd.Flags().StringVar(&ingestMimeType, "mime-type", "image/png", "MIME type of uploaded files")
	ingestCmd.MarkFlagRequired("workspace") //nolint:errcheck
	ingestCmd.MarkFlagRequired("file")      //nolint:errcheck
}
