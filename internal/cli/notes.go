package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"

)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List notes",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := loadClient(true)
			if err != nil {
				return err
			}
			notes, err := client.ListNotes(context.Background())
			if err != nil {
				return handleAuthError(err)
			}
			if len(notes) == 0 {
				fmt.Println("no notes")
				return nil
			}
			for _, n := range notes {
				fmt.Printf("%s\t%s\t%s\n", n.ID, n.Title, n.UpdatedAt.Format("2006-01-02 15:04"))
			}
			return nil
		},
	}
}

func newViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <id>",
		Short: "View a note with rendered Markdown",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := loadClient(true)
			if err != nil {
				return err
			}
			note, err := client.GetNote(context.Background(), args[0])
			if err != nil {
				return handleAuthError(err)
			}
			fmt.Println(note.Title)
			fmt.Println(strings.Repeat("─", min(40, max(8, len(note.Title)))))
			rendered, err := renderMarkdown(note.Body)
			if err != nil {
				return err
			}
			fmt.Println(rendered)
			return nil
		},
	}
}

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a note",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := loadClient(true)
			if err != nil {
				return err
			}
			note, err := client.GetNote(context.Background(), args[0])
			if err != nil {
				return handleAuthError(err)
			}
			fmt.Fprintf(os.Stderr, "Delete note %q? [y/N] ", note.Title)
			reader := bufio.NewReader(os.Stdin)
			line, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(line)) != "y" {
				fmt.Println("cancelled")
				return nil
			}
			if err := client.DeleteNote(context.Background(), args[0]); err != nil {
				return handleAuthError(err)
			}
			fmt.Println("deleted")
			return nil
		},
	}
}

func newCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create a note in the TUI editor",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := loadClient(true)
			if err != nil {
				return err
			}
			return runEditor(client, "", "Untitled", "")
		},
	}
}

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit a note in the TUI editor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := loadClient(true)
			if err != nil {
				return err
			}
			note, err := client.GetNote(context.Background(), args[0])
			if err != nil {
				return handleAuthError(err)
			}
			return runEditor(client, note.ID, note.Title, note.Body)
		},
	}
}

func renderMarkdown(body string) (string, error) {
	if strings.TrimSpace(body) == "" {
		body = "_Nothing here yet._"
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		return "", err
	}
	return r.Render(body)
}
