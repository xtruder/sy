package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xtruder/sy/internal/util"
)

var mvCmd = &cobra.Command{
	Use:   "mv <source> <dest>",
	Short: "Move or rename documents",
	Long: `Move a document to a new path or rename it.

Use --rename flag to only change the title without moving.`,
	Example: `  sy mv /OldPath /NewPath              # Move within notebook
  sy mv /Doc --rename "New Title"      # Rename only
  sy mv 20260118141851-abc1234 /Folder # Move by ID`,
	Args:              cobra.RangeArgs(1, 2),
	ValidArgsFunction: completeMultipleDocumentPaths,
	RunE:              runMv,
}

func init() {
	mvCmd.Flags().String("rename", "", "New title (rename without moving)")
}

func runMv(cmd *cobra.Command, args []string) error {
	client := GetClient()
	notebook := GetCurrentNotebook()
	if notebook == nil {
		return fmt.Errorf("no notebook selected")
	}

	renameTitle, _ := cmd.Flags().GetString("rename")
	sourceRef := args[0]

	// Resolve source document ID
	sourceID, err := util.ResolveDocumentID(client, notebook, sourceRef)
	if err != nil {
		return fmt.Errorf("failed to resolve source: %w", err)
	}

	// Get source document info
	sourceInfo, err := client.GetBlockInfo(sourceID)
	if err != nil {
		return fmt.Errorf("failed to get source info: %w", err)
	}

	sourceMeta, err := util.GetDocumentMeta(client, sourceID)
	if err != nil {
		return fmt.Errorf("failed to get source metadata: %w", err)
	}

	// Handle rename-only operation
	if renameTitle != "" {
		if err := client.RenameDoc(sourceInfo.Box, sourceInfo.Path, renameTitle); err != nil {
			return fmt.Errorf("failed to rename document: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Renamed: %s -> %s\n", sourceMeta.Title, renameTitle)
		return nil
	}

	// Move operation requires destination
	if len(args) < 2 {
		return fmt.Errorf("destination path required (or use --rename)")
	}

	destDisplayPath := args[1]
	if !strings.HasPrefix(destDisplayPath, "/") {
		destDisplayPath = "/" + destDisplayPath
	}

	// Resolve destination to a document/folder ID
	// For root path, use the notebook ID as destination
	var destID string
	if destDisplayPath == "/" {
		destID = notebook.ID
	} else {
		// Resolve the display path to a document ID
		resolved, err := util.ResolveDocumentID(client, notebook, destDisplayPath)
		if err != nil {
			return fmt.Errorf("failed to resolve destination path: %w", err)
		}
		destID = resolved
	}

	// Move the document using ID-based API
	if err := client.MoveDocsByID([]string{sourceID}, destID); err != nil {
		return fmt.Errorf("failed to move document: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Moved: %s -> %s\n", sourceMeta.HPath, destDisplayPath)
	return nil
}
