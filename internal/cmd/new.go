package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new <path>",
	Short: "Create a new document",
	Long: `Create a new document at the specified path in the current notebook.

Path format: /Folder/DocumentTitle or just DocumentTitle
The last component becomes the document title.`,
	Example: `  sy new MyDocument                    # Create at notebook root
  sy new /Folder/MyDocument            # Create in folder
  sy new /Folder/Doc --content "Text"  # Create with content
  sy -n Personal new /Notes/Todo       # Create in Personal notebook`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeDocumentPath,
	RunE:              runNew,
}

func init() {
	newCmd.Flags().String("content", "", "Initial document content (use '-' for stdin)")
}

func runNew(cmd *cobra.Command, args []string) error {
	client := GetClient()
	notebook := GetCurrentNotebook()
	if notebook == nil {
		return fmt.Errorf("no notebook selected")
	}

	contentFlag, _ := cmd.Flags().GetString("content")
	docPath := args[0]

	// Normalize path
	if !strings.HasPrefix(docPath, "/") {
		docPath = "/" + docPath
	}

	// Get content
	var content string
	if contentFlag == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		content = string(data)
	} else if contentFlag != "" {
		content = contentFlag
	}

	// Create the document
	docID, err := client.CreateDocWithMd(notebook.ID, docPath, content)
	if err != nil {
		return fmt.Errorf("failed to create document: %w", err)
	}

	fmt.Printf("%s\n", docID)
	fmt.Fprintf(os.Stderr, "Created: %s:%s\n", notebook.Name, docPath)

	return nil
}
