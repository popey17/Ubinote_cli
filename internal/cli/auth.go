package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"ubinote-cli/internal/config"
)

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Log in and store a JWT",
		RunE: func(cmd *cobra.Command, args []string) error {
			email, password, err := promptCredentials()
			if err != nil {
				return err
			}
			client, err := loadClient(false)
			if err != nil {
				return err
			}
			tok, err := client.Login(context.Background(), email, password)
			if err != nil {
				return err
			}
			if err := config.SaveCredentials(config.Credentials{Token: tok}); err != nil {
				return err
			}
			fmt.Println("logged in")
			return nil
		},
	}
}

func newRegisterCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "register",
		Short: "Register a new account and log in",
		RunE: func(cmd *cobra.Command, args []string) error {
			email, password, err := promptCredentials()
			if err != nil {
				return err
			}
			client, err := loadClient(false)
			if err != nil {
				return err
			}
			if _, _, err := client.Register(context.Background(), email, password); err != nil {
				return err
			}
			tok, err := client.Login(context.Background(), email, password)
			if err != nil {
				return err
			}
			if err := config.SaveCredentials(config.Credentials{Token: tok}); err != nil {
				return err
			}
			fmt.Println("registered and logged in")
			return nil
		},
	}
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear the stored JWT",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := loadClient(false)
			if err != nil {
				return err
			}
			if client.Token() != "" {
				_ = client.Logout(context.Background())
			}
			if err := config.ClearCredentials(); err != nil {
				return err
			}
			fmt.Println("logged out")
			return nil
		},
	}
}

func promptCredentials() (email, password string, err error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprint(os.Stderr, "Email: ")
	email, err = reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}
	email = strings.TrimSpace(email)
	if email == "" {
		return "", "", fmt.Errorf("email is required")
	}

	fmt.Fprint(os.Stderr, "Password: ")
	pw, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", "", err
	}
	password = string(pw)
	if password == "" {
		return "", "", fmt.Errorf("password is required")
	}
	return email, password, nil
}
