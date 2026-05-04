package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"theartifact-cli/internal/api"
	"theartifact-cli/internal/ui"
)

var (
	statusWait       bool
	statusNoDownload bool
)

var statusCmd = &cobra.Command{
	Use:   "status <job_id>",
	Short: "Check job status",
	Long:  "Check the current status of a job. Use --wait to poll until completion.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		jobID := args[0]

		client, err := api.NewClient()
		if err != nil {
			return err
		}

		sp := ui.NewSpinner("Checking job status...")
		sp.Start()

		for {
			job, err := client.GetJob(jobID)
			if err != nil {
				sp.Fail("Failed to fetch job status")
				return fmt.Errorf("failed to get job status: %w", err)
			}

			statusLabel := ui.StatusStyle(job.Status).Render(job.Status)
			sp.UpdateMessage(fmt.Sprintf("Status: %s  (%d%%)", statusLabel, job.Progress))

			if job.Status == "succeeded" {
				sp.Stop("Job complete")

				ui.PrintSuccess("Generation complete!")
				ui.PrintKeyValue("Job ID", job.ID)

				if job.Result != nil {
					ui.PrintKeyValue("Model", job.Result.Manifest.Model)
					ui.PrintKeyValue("Prompt", job.Result.Manifest.Prompt)

					if len(job.Result.Assets) > 0 && !statusNoDownload {
						downloadAndPrint(job.Result, ".artifact/assets")
					} else if len(job.Result.Assets) > 0 {
						printAssetURLs(job.Result)
					}
				}
				break

			} else if job.Status == "failed" {
				sp.Fail(fmt.Sprintf("Job failed: %s", job.Error))
				return fmt.Errorf("job failed: %s", job.Error)
			}

			if !statusWait {
				sp.Stop(fmt.Sprintf("Status: %s (%d%%)", job.Status, job.Progress))
				ui.PrintKeyValue("Job ID", job.ID)
				ui.PrintKeyValue("Status", ui.StatusStyle(job.Status).Render(job.Status))
				ui.PrintKeyValue("Progress", fmt.Sprintf("%d%%", job.Progress))
				ui.PrintHint(fmt.Sprintf("Run with --wait to poll until complete: theartifact status %s --wait", job.ID))
				break
			}

			time.Sleep(2 * time.Second)
		}

		return nil
	},
}

func downloadAndPrint(result *api.JobResult, destDir string) {
	sp := ui.NewSpinner(fmt.Sprintf("Downloading %d asset(s)…", len(result.Assets)))
	sp.Start()

	paths, errs := api.DownloadJobAssets(result, destDir)
	for _, e := range errs {
		ui.PrintWarn(e.Error())
	}

	if len(paths) > 0 {
		sp.Stop(fmt.Sprintf("Saved %d asset(s) to %s/", len(paths), destDir))
		for _, p := range paths {
			ui.PrintKeyValue("→", p)
		}
	} else {
		sp.Fail("No assets downloaded")
	}
}

func printAssetURLs(result *api.JobResult) {
	ui.PrintSectionHeader(fmt.Sprintf("%d Asset(s) Generated", len(result.Assets)))
	headers := []string{"#", "Asset ID", "URL"}
	rows := make([][]string, len(result.Assets))
	for i, asset := range result.Assets {
		rows[i] = []string{fmt.Sprintf("%d", i+1), asset.ID, asset.URL}
	}
	ui.PrintTable(headers, rows)
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolVar(&statusWait, "wait", false, "Poll until the job completes")
	statusCmd.Flags().BoolVar(&statusNoDownload, "no-download", false, "Print asset URLs instead of downloading")
}
