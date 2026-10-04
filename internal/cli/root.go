package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/popey17/Ubinote_cli/internal/api"
	"github.com/popey17/Ubinote_cli/internal/config"
)

func Execute() error {
	root := &cobra.Command{
		Use:           "ubinote",
		Short:         "CLI for ubinote personal notes",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI()
		},
	}

	root.AddCommand(
		newLoginCmd(),
		newLogoutCmd(),
		newRegisterCmd(),
		newConfigCmd(),
		newListCmd(),
		newViewCmd(),
		newCreateCmd(),
		newEditCmd(),
		newDeleteCmd(),
		newTUICmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

func loadClient(requireAuth bool) (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	creds, err := config.LoadCredentials()
	if err != nil {
		return nil, err
	}
	if requireAuth && creds.Token == "" {
		return nil, fmt.Errorf("not logged in; run: ubinote login")
	}
	return api.New(cfg.APIURL, creds.Token), nil
}

func handleAuthError(err error) error {
	if err == nil {
		return nil
	}
	if apiErrIs(err, api.ErrUnauthorized) {
		_ = config.ClearCredentials()
		return fmt.Errorf("session expired; run: ubinote login")
	}
	return err
}

func apiErrIs(err, target error) bool {
	return errors.Is(err, target)
}
