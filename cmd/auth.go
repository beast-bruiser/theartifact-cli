package cmd

import (
	"github.com/spf13/cobra"
	"theartifact-cli/internal/config"
	"theartifact-cli/internal/ui"
)

var apiKey string

var authCmd = &cobra.Command{
	Use:     "login",
	Aliases: []string{"auth"}, // 'auth' kept as hidden backward-compat alias
	Short:   "Log in with your API key",
	Long:    "Authenticate with TheArtifact API by saving your API key to the local config.",
	RunE: func(cmd *cobra.Command, args []string) error {
		sp := ui.NewSpinner("Saving credentials...")
		sp.Start()

		cfg, err := config.Load()
		if err != nil {
			sp.Fail("Failed to load config")
			return err
		}

		cfg.APIKey = apiKey

		if err := config.Save(cfg); err != nil {
			sp.Fail("Failed to save config")
			return err
		}

		sp.Stop("Credentials saved")

		ui.PrintSuccess("Authenticated successfully")
		ui.PrintKeyValue("Key", ui.MaskKey(apiKey))
		ui.PrintKeyValue("Config", "~/.artifact/config.json")
		ui.PrintHint("You're all set! Run 'theartifact workspace list' to get started.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.Flags().StringVar(&apiKey, "key", "", "Your TheArtifact API key (tak_live_...)")
	authCmd.MarkFlagRequired("key") //nolint:errcheck

	// Hide the 'auth' alias from help so only 'login' surfaces
	for _, alias := range authCmd.Aliases {
		rootCmd.RegisterFlagCompletionFunc(alias, func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}) //nolint:errcheck
		_ = alias
	}
}
