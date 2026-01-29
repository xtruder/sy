package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xtruder/sy/internal/format"
)

var notebookCmd = &cobra.Command{
	Use:     "notebook [name]",
	Aliases: []string{"nb"},
	Short:   "Get or set current notebook",
	Long: `Display or change the current notebook.

Without arguments, shows the current notebook and lists all available notebooks.
With an argument, sets the current notebook and saves it to config.`,
	Example: `  sy nb                    # Show current notebook and list all
  sy nb Personal           # Set current notebook to "Personal"
  sy notebook --list       # List all notebooks
  sy -n XTruder ls         # Use XTruder notebook for this command only`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeNotebook,
	RunE:              runNotebook,
}

func init() {
	notebookCmd.Flags().BoolP("list", "l", false, "List all notebooks")
}

func runNotebook(cmd *cobra.Command, args []string) error {
	client := GetClient()
	listOnly, _ := cmd.Flags().GetBool("list")

	notebooks, err := client.ListNotebooks()
	if err != nil {
		return fmt.Errorf("failed to list notebooks: %w", err)
	}

	// Filter to open notebooks
	var openNotebooks []*struct {
		ID     string
		Name   string
		Closed bool
	}
	for _, nb := range notebooks {
		if !nb.Closed {
			openNotebooks = append(openNotebooks, &struct {
				ID     string
				Name   string
				Closed bool
			}{nb.ID, nb.Name, nb.Closed})
		}
	}

	if len(openNotebooks) == 0 {
		return fmt.Errorf("no open notebooks found")
	}

	// If setting a new notebook
	if len(args) > 0 {
		notebookName := args[0]

		// Find the notebook
		var found *struct {
			ID     string
			Name   string
			Closed bool
		}
		for _, nb := range openNotebooks {
			if nb.Name == notebookName || nb.ID == notebookName {
				found = nb
				break
			}
		}

		if found == nil {
			fmt.Fprintf(os.Stderr, "Notebook not found: %s\n\nAvailable notebooks:\n", notebookName)
			for _, nb := range openNotebooks {
				fmt.Fprintf(os.Stderr, "  - %s\n", nb.Name)
			}
			return fmt.Errorf("notebook not found")
		}

		// Save to config
		if err := SaveNotebookToConfig(found.Name); err != nil {
			return err
		}

		fmt.Printf("Notebook set to: %s\n", found.Name)
		fmt.Fprintf(os.Stderr, "Saved to config: %s/config.yaml\n", GetConfigDir())
		return nil
	}

	// Show current notebook and list
	currentNotebookName := viper.GetString("notebook")

	if !listOnly {
		if currentNotebookName != "" {
			fmt.Printf("Current notebook: %s\n", currentNotebookName)
		} else {
			fmt.Printf("Current notebook: %s (default - first notebook)\n", openNotebooks[0].Name)
		}
		fmt.Println()
	}

	// List all notebooks
	table := format.NewTable(
		format.Column{Header: ""},
		format.Column{Header: "ID", Width: 22},
		format.Column{Header: "Name", MaxWidth: 40},
	)

	for _, nb := range openNotebooks {
		marker := "  "
		if nb.Name == currentNotebookName || (currentNotebookName == "" && nb == openNotebooks[0]) {
			marker = "* "
		}
		table.AddRow(marker, nb.ID, nb.Name)
	}

	table.Render(os.Stdout)
	return nil
}
