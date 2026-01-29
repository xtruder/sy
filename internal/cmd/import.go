package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xtruder/sy/internal/util"
)

var importCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import markdown/kramdown documents",
	Long: `Import markdown or kramdown files into SiYuan.

When importing a file exported with 'sy export':
  - If siyuan_doc_id is present, updates the existing document
  - Block IDs in IAL are preserved

For new documents:
  - Use --to flag to specify target path: /Folder/DocTitle
  - Or include target info in frontmatter`,
	Example: `  sy import file.md                # Update doc (uses frontmatter doc_id)
  sy import file.md --to /Folder   # Import to folder
  sy import ./folder/              # Import all .md files
  sy -n Personal import doc.md     # Import to Personal notebook`,
	RunE: runImport,
}

func init() {
	importCmd.Flags().String("to", "", "Target path for new documents: /Folder/DocTitle")
	importCmd.Flags().String("title", "", "Document title (overrides frontmatter/H1)")
	importCmd.Flags().String("content", "", "Document content (for inline creation, use '-' for stdin)")
}

func runImport(cmd *cobra.Command, args []string) error {
	targetPath, _ := cmd.Flags().GetString("to")
	titleOverride, _ := cmd.Flags().GetString("title")
	inlineContent, _ := cmd.Flags().GetString("content")

	// Handle inline content creation
	if inlineContent != "" {
		if targetPath == "" {
			return fmt.Errorf("--to flag is required when using --content")
		}

		content := inlineContent
		if content == "-" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
			content = string(data)
		}

		return importContent(targetPath, titleOverride, content)
	}

	// Handle file import
	if len(args) == 0 {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			if targetPath == "" {
				return fmt.Errorf("--to flag is required when reading from stdin without frontmatter")
			}
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
			return importContent(targetPath, titleOverride, string(data))
		}
		return fmt.Errorf("no file specified and no stdin data")
	}

	filePath := args[0]

	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to access %s: %w", filePath, err)
	}

	if info.IsDir() {
		return importDirectory(filePath, targetPath)
	}

	return importFile(filePath, targetPath, titleOverride)
}

func importContent(targetPath, titleOverride, content string) error {
	notebook := GetCurrentNotebook()

	// Parse frontmatter if present
	meta, bodyContent, err := util.UnmarshalFrontmatter(content)
	if err != nil {
		return fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// If we have a doc ID in frontmatter, this is an update to existing doc
	if meta != nil && meta.DocID != "" && util.IsBlockID(meta.DocID) {
		return updateExistingDocument(meta.DocID, bodyContent)
	}

	// Otherwise, create a new document
	return createNewDocument(notebook, targetPath, titleOverride, meta, bodyContent)
}

func updateExistingDocument(docID, content string) error {
	client := GetClient()

	_, content = util.StripH1Title(content)
	content = strings.TrimSpace(content)

	if err := client.UpdateBlock(docID, content); err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Updated document: %s\n", docID)
	return nil
}

func createNewDocument(notebook interface{}, targetPath, titleOverride string, meta *util.DocumentMeta, content string) error {
	client := GetClient()
	nb := GetCurrentNotebook()
	if nb == nil {
		return fmt.Errorf("no notebook selected")
	}

	// Determine path and title
	docPath, title := resolveNewDocPath(targetPath, meta, titleOverride)

	// Strip H1 title if present and use it as title if none specified
	if h1Title, remaining := util.StripH1Title(content); h1Title != "" {
		if title == "" {
			title = h1Title
		}
		content = remaining
	}

	if title == "" {
		return fmt.Errorf("no title specified (use --title, --to path, frontmatter, or include H1 heading)")
	}

	// Build full path
	fullPath := docPath
	if fullPath == "/" || fullPath == "" {
		fullPath = "/" + title
	} else {
		fullPath = docPath + "/" + title
	}

	// Create the document
	newDocID, err := client.CreateDocWithMd(nb.ID, fullPath, content)
	if err != nil {
		return fmt.Errorf("failed to create document: %w", err)
	}

	fmt.Printf("%s\n", newDocID)
	fmt.Fprintf(os.Stderr, "Created: %s:%s\n", nb.Name, fullPath)
	return nil
}

func importFile(filePath, targetPath, titleOverride string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	baseName := filepath.Base(filePath)
	fileTitle := strings.TrimSuffix(baseName, filepath.Ext(baseName))

	if titleOverride == "" {
		titleOverride = fileTitle
	}

	return importContent(targetPath, titleOverride, string(content))
}

func importDirectory(dirPath, targetPath string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".markdown" {
			return nil
		}

		fmt.Fprintf(os.Stderr, "Importing: %s\n", path)
		return importFile(path, targetPath, "")
	})
}

func resolveNewDocPath(targetPath string, meta *util.DocumentMeta, titleOverride string) (docPath, title string) {
	if targetPath != "" {
		// Normalize path
		if !strings.HasPrefix(targetPath, "/") {
			targetPath = "/" + targetPath
		}

		// Check if target is just a folder or includes title
		parts := strings.Split(strings.Trim(targetPath, "/"), "/")
		if len(parts) == 1 {
			// Could be folder or title - if titleOverride is set, treat as folder
			if titleOverride != "" {
				docPath = "/" + parts[0]
				title = titleOverride
			} else {
				// Treat as title at root
				docPath = "/"
				title = parts[0]
			}
		} else {
			// Multiple parts - last one is title
			docPath = "/" + strings.Join(parts[:len(parts)-1], "/")
			title = parts[len(parts)-1]
		}
	} else if meta != nil && meta.HPath != "" {
		// Extract path from hpath (remove notebook name prefix)
		hpath := meta.HPath
		// hpath format: /NotebookName/Path/Title
		parts := strings.Split(strings.Trim(hpath, "/"), "/")
		if len(parts) >= 2 {
			// Skip notebook name, use rest
			if len(parts) == 2 {
				docPath = "/"
				title = parts[1]
			} else {
				docPath = "/" + strings.Join(parts[1:len(parts)-1], "/")
				title = parts[len(parts)-1]
			}
		}
	}

	if titleOverride != "" {
		title = titleOverride
	}

	return docPath, title
}
