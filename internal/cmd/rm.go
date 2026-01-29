package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xtruder/sy/internal/util"
)

var rmCmd = &cobra.Command{
	Use:   "rm <doc>...",
	Short: "Delete documents",
	Long: `Delete one or more documents.

By default, prompts for confirmation before deletion.
Refuses to delete documents with children unless --recursive is used.`,
	Example: `  sy rm /Doc                    # Delete with confirmation
  sy rm /Doc --force            # Delete without confirmation  
  sy rm /Folder --recursive     # Delete folder and children
  sy rm 20260118141851-abc1234  # Delete by ID
  sy rm doc1 doc2 doc3          # Delete multiple`,
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: completeMultipleDocumentPaths,
	RunE:              runRm,
}

func init() {
	rmCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
	rmCmd.Flags().BoolP("recursive", "r", false, "Delete children recursively")
}

func runRm(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	recursive, _ := cmd.Flags().GetBool("recursive")

	for _, docRef := range args {
		if err := deleteDocument(docRef, force, recursive); err != nil {
			return err
		}
	}

	return nil
}

func deleteDocument(docRef string, force, recursive bool) error {
	client := GetClient()
	notebook := GetCurrentNotebook()

	// Resolve document ID
	docID, err := util.ResolveDocumentID(client, notebook, docRef)
	if err != nil {
		return err
	}

	// Get document info
	info, err := client.GetBlockInfo(docID)
	if err != nil {
		return fmt.Errorf("failed to get document info: %w", err)
	}

	// Get metadata for display
	meta, err := util.GetDocumentMeta(client, docID)
	if err != nil {
		return fmt.Errorf("failed to get document metadata: %w", err)
	}

	// Check for children
	children, err := client.GetChildBlocks(docID)
	if err != nil {
		return fmt.Errorf("failed to check for children: %w", err)
	}

	// Check if document has sub-documents
	hasSubDocs := false
	for _, child := range children {
		if child.Type == "d" {
			hasSubDocs = true
			break
		}
	}

	if hasSubDocs && !recursive {
		return fmt.Errorf("document has children; use --recursive to delete: %s", meta.HPath)
	}

	// Confirm deletion unless --force
	if !force {
		fmt.Fprintf(os.Stderr, "Delete %s? [y/N] ", meta.HPath)
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Fprintln(os.Stderr, "Cancelled")
			return nil
		}
	}

	// Delete the document
	if err := client.RemoveDoc(info.Box, info.Path); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Deleted: %s\n", meta.HPath)
	return nil
}
