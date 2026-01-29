package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xtruder/sy/internal/api"
	"github.com/xtruder/sy/internal/format"
	"github.com/xtruder/sy/internal/util"
)

var lsCmd = &cobra.Command{
	Use:   "ls [path]",
	Short: "List documents",
	Long: `List documents in the current notebook.

Without arguments, lists documents at the root of the current notebook.
With a path argument, lists documents in that folder.
Use -r flag for recursive listing.`,
	Example: `  sy ls                    # List docs at notebook root
  sy ls /Folder            # List docs in folder
  sy ls -r                 # List all docs recursively  
  sy ls -r /Folder         # List folder recursively
  sy -n Personal ls        # List in Personal notebook`,
	ValidArgsFunction: completeDocumentPath,
	RunE:              runLs,
}

func init() {
	lsCmd.Flags().BoolP("recursive", "r", false, "List recursively")
	lsCmd.Flags().String("sort", "name", "Sort by: name, updated, created")
}

func runLs(cmd *cobra.Command, args []string) error {
	notebook := GetCurrentNotebook()
	if notebook == nil {
		return fmt.Errorf("no notebook selected")
	}

	recursive, _ := cmd.Flags().GetBool("recursive")

	// Determine path
	docPath := "/"
	if len(args) > 0 {
		docPath = args[0]
		if docPath[0] != '/' {
			docPath = "/" + docPath
		}
	}

	return listDocs(notebook, docPath, recursive)
}

func listDocs(notebook *api.Notebook, displayPath string, recursive bool) error {
	client := GetClient()

	// Resolve display path to storage path
	storagePath, err := util.ResolvePathToStoragePath(client, notebook, displayPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	table := format.NewTable(
		format.Column{Header: "ID", Width: 22},
		format.Column{Header: "Title", MaxWidth: 40},
		format.Column{Header: "Path", MaxWidth: 50},
	)

	if recursive {
		if err := listDocsRecursive(client, notebook, storagePath, displayPath, table); err != nil {
			return err
		}
	} else {
		docs, err := client.ListDocsByPath(notebook.ID, storagePath)
		if err != nil {
			// "no such file or directory" means the document has no children (leaf node)
			if strings.Contains(err.Error(), "no such file or directory") {
				fmt.Fprintf(os.Stderr, "No sub-documents in %s:%s\n", notebook.Name, displayPath)
				return nil
			}
			return fmt.Errorf("failed to list docs: %w", err)
		}
		for _, doc := range docs {
			docDisplayPath := path.Join(displayPath, doc.Title())
			table.AddRow(doc.ID, doc.Title(), docDisplayPath)
		}
	}

	if table.RowCount() == 0 {
		fmt.Fprintf(os.Stderr, "No documents found in %s:%s\n", notebook.Name, displayPath)
		return nil
	}

	fmt.Fprintf(os.Stderr, "Notebook: %s\n\n", notebook.Name)
	table.Render(os.Stdout)
	return nil
}

// listDocsRecursive recursively lists all documents under a path
func listDocsRecursive(client *api.Client, notebook *api.Notebook, storagePath, displayPath string, table *format.Table) error {
	docs, err := client.ListDocsByPath(notebook.ID, storagePath)
	if err != nil {
		// Silently ignore "no such file or directory" errors - these are leaf documents
		// with incorrect count values in SiYuan's database
		if strings.Contains(err.Error(), "no such file or directory") {
			return nil
		}
		// Log other errors as warnings but continue
		fmt.Fprintf(os.Stderr, "Warning: failed to list %s: %v\n", storagePath, err)
		return nil
	}

	for _, doc := range docs {
		docDisplayPath := path.Join(displayPath, doc.Title())
		table.AddRow(doc.ID, doc.Title(), docDisplayPath)

		// If document has sub-documents, recurse
		if doc.SubFileCount > 0 {
			childStoragePath := storagePath
			if childStoragePath == "/" {
				childStoragePath = "/" + doc.ID
			} else {
				childStoragePath = storagePath + "/" + doc.ID
			}
			// Errors are handled inside - won't propagate
			_ = listDocsRecursive(client, notebook, childStoragePath, docDisplayPath, table)
		}
	}

	return nil
}
