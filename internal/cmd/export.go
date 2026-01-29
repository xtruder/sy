package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xtruder/sy/internal/api"
	"github.com/xtruder/sy/internal/util"
)

var exportCmd = &cobra.Command{
	Use:     "export <doc>",
	Aliases: []string{"cat"},
	Short:   "Export documents as kramdown",
	Long: `Export documents as kramdown format (preserves block IDs via IAL).

Document can be identified by:
  - Block ID: 20260118141851-qvxnjey
  - Path: /Folder/Document

Use --md for pure markdown output (strips IAL block attributes).`,
	Example: `  sy export /Folder/Doc                 # Export to stdout
  sy cat /Folder/Doc                    # Same as export
  sy cat /Folder/Doc --md               # Export as pure markdown
  sy export /Folder/Doc -o doc.md       # Export to file
  sy -n Personal export /Doc            # Export from Personal notebook`,
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: completeDocumentPath,
	RunE:              runExport,
}

func init() {
	exportCmd.Flags().StringP("output", "o", "", "Output file or directory (default: stdout)")
	exportCmd.Flags().Bool("no-frontmatter", false, "Omit frontmatter from output")
	exportCmd.Flags().Bool("md", false, "Export as pure markdown (strips IAL block attributes)")
}

func runExport(cmd *cobra.Command, args []string) error {
	client := GetClient()
	notebook := GetCurrentNotebook()
	output, _ := cmd.Flags().GetString("output")
	noFrontmatter, _ := cmd.Flags().GetBool("no-frontmatter")
	pureMarkdown, _ := cmd.Flags().GetBool("md")

	// --md implies no frontmatter (frontmatter contains SiYuan-specific metadata)
	if pureMarkdown {
		noFrontmatter = true
	}

	docRef := args[0]

	// Resolve the document reference to an ID
	docID, err := util.ResolveDocumentID(client, notebook, docRef)
	if err != nil {
		return err
	}

	// Export the document
	content, meta, err := exportDocument(client, docID, !noFrontmatter)
	if err != nil {
		return err
	}

	// Strip IAL for pure markdown output
	if pureMarkdown {
		content = stripIAL(content)
	}

	// Write output
	if output == "" {
		fmt.Print(content)
	} else {
		info, err := os.Stat(output)
		if err == nil && info.IsDir() {
			filename := sanitizeFilename(meta.Title) + ".md"
			outputPath := filepath.Join(output, filename)
			if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Exported to %s\n", outputPath)
		} else {
			if err := os.WriteFile(output, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write file: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Exported to %s\n", output)
		}
	}

	return nil
}

// exportDocument exports a document as kramdown with optional frontmatter
func exportDocument(client *api.Client, docID string, includeFrontmatter bool) (string, *util.DocumentMeta, error) {
	meta, err := util.GetDocumentMeta(client, docID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get document metadata: %w", err)
	}

	children, err := client.GetChildBlocks(docID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get child blocks: %w", err)
	}

	var kramdownParts []string
	for _, child := range children {
		kramdown, err := client.GetBlockKramdown(child.ID)
		if err != nil {
			return "", nil, fmt.Errorf("failed to export block %s: %w", child.ID, err)
		}
		kramdownParts = append(kramdownParts, kramdown)
	}

	kramdownContent := strings.Join(kramdownParts, "\n\n")

	var content string
	if includeFrontmatter {
		content, err = util.BuildDocument(meta, kramdownContent)
		if err != nil {
			return "", nil, err
		}
	} else {
		content = kramdownContent
	}

	return content, meta, nil
}

func sanitizeFilename(title string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	filename := replacer.Replace(title)
	if len(filename) > 200 {
		filename = filename[:200]
	}
	if filename == "" {
		filename = "untitled"
	}
	return filename
}

// ialPattern matches IAL (Inline Attribute List) blocks in kramdown
// Formats:
//
//	{: id="..." updated="..."}           - standalone line
//	{: id="..." updated="..."} at end    - after content on same line
//	- {: id="..."} list item prefix      - list item IAL
var ialPattern = regexp.MustCompile(`\{:\s+[^}]*(?:id|updated)="[^"]*"[^}]*\}`)

// stripIAL removes all IAL (Inline Attribute List) blocks from kramdown content
func stripIAL(content string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		// Remove IAL from the line
		cleaned := ialPattern.ReplaceAllString(line, "")

		// Trim trailing whitespace left after IAL removal
		cleaned = strings.TrimRight(cleaned, " \t")

		// Skip lines that were only IAL (now empty)
		if strings.TrimSpace(cleaned) == "" && strings.TrimSpace(line) != "" && ialPattern.MatchString(line) {
			continue
		}

		result = append(result, cleaned)
	}

	// Clean up multiple consecutive blank lines
	output := strings.Join(result, "\n")
	multipleBlankLines := regexp.MustCompile(`\n{3,}`)
	output = multipleBlankLines.ReplaceAllString(output, "\n\n")

	return output
}
