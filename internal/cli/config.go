package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/popey17/Ubinote_cli/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show or update CLI configuration",
	}
	cmd.AddCommand(newConfigShowCmd(), newConfigSetURLCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show API URL and login status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			creds, err := config.LoadCredentials()
			if err != nil {
				return err
			}
			dir, err := config.Dir()
			if err != nil {
				return err
			}
			loggedIn := "no"
			if creds.Token != "" {
				loggedIn = "yes"
			}
			fmt.Printf("api_url:  %s\n", cfg.APIURL)
			fmt.Printf("logged_in: %s\n", loggedIn)
			fmt.Printf("config_dir: %s\n", dir)
			return nil
		},
	}
}

func newConfigSetURLCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-url <url>",
		Short: "Set the API base URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Config{APIURL: args[0]}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("api_url set to %s\n", args[0])
			return nil
		},
	}
}
