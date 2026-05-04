package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"os"

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
		var sourceTokens []string
		httpClient := &http.Client{}

		for i, filePath := range ingestFiles {
			sp2 := ui.NewSpinner(fmt.Sprintf("Uploading %s...", filePath))
			sp2.Start()

			fileData, err := os.ReadFile(filePath)
			if err != nil {
				sp2.Fail(fmt.Sprintf("Cannot read file: %s", filePath))
				return fmt.Errorf("failed to read file %s: %w", filePath, err)
			}

			tokenData := uploadTokens[i]

			req, err := http.NewRequest("PUT", tokenData.URL, bytes.NewReader(fileData))
			if err != nil {
				sp2.Fail("Failed to build upload request")
				return fmt.Errorf("failed to create request for %s: %w", filePath, err)
			}
			req.Header.Set("Content-Type", ingestMimeType)

			resp, err := httpClient.Do(req)
			if err != nil {
				sp2.Fail(fmt.Sprintf("Upload failed: %s", filePath))
				return fmt.Errorf("failed to upload %s: %w", filePath, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				sp2.Fail(fmt.Sprintf("Upload rejected (HTTP %d): %s", resp.StatusCode, filePath))
				return fmt.Errorf("upload failed for %s with status %d", filePath, resp.StatusCode)
			}

			sp2.Stop(filePath)
			sourceTokens = append(sourceTokens, tokenData.UploadToken)
		}

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
