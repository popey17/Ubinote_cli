package cli

import (
	"github.com/spf13/cobra"

	"ubinote-cli/internal/api"
	"ubinote-cli/internal/tui"
)

func newTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive note browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI()
		},
	}
}

func runTUI() error {
	client, err := loadClient(false)
	if err != nil {
		return err
	}
	return tui.Run(client)
}

func runEditor(client *api.Client, id, title, body string) error {
	return tui.RunEditor(client, id, title, body)
}
