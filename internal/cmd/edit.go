package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xtruder/sy/internal/api"
	"github.com/xtruder/sy/internal/util"
)

var editCmd = &cobra.Command{
	Use:   "edit <doc>",
	Short: "Edit documents interactively or inline",
	Long: `Edit documents using $EDITOR or inline modifications.

Interactive mode (default): exports to temp file, opens editor, imports on save.
Inline mode: use flags to append, prepend, or modify specific blocks.

Content supports full markdown notation:
  # H1, ## H2, ### H3    - Headings
  **bold**, *italic*     - Formatting  
  - item, 1. item        - Lists
  ` + "```lang ... ```" + `       - Code blocks
  > quote                - Blockquotes

Use --code to read content from a source file (auto-wrapped as code block).`,
	Example: `  sy edit /Doc                            # Open in $EDITOR
  sy edit /Doc --md                        # Edit as pure markdown (new block IDs)
  sy edit 20260118141851-abc1234           # Edit by ID
  sy edit /Doc --append --content "## New Section"
  sy edit /Doc --append --code main.go     # Append file as code block
  sy edit /Doc --prepend --code config.yaml
  sy edit /Doc --after <blockID> --code utils.py
  sy edit /Doc --replace <blockID> --content "New content"
  sy edit /Doc --delete <blockID>`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeDocumentPath,
	RunE:              runEdit,
}

func init() {
	editCmd.Flags().String("editor", "", "Editor to use (default: $EDITOR or vim)")
	editCmd.Flags().Bool("append", false, "Append content to end of document")
	editCmd.Flags().Bool("prepend", false, "Prepend content to start of document")
	editCmd.Flags().String("after", "", "Block ID to insert content after")
	editCmd.Flags().String("replace", "", "Block ID to replace")
	editCmd.Flags().String("delete", "", "Block ID to delete")
	editCmd.Flags().String("content", "", "Content to insert (or use --code for file)")
	editCmd.Flags().String("code", "", "Source file to use as content (wrapped as code block)")
	editCmd.Flags().String("lang", "", "Language for code block (auto-detected from extension if not specified)")
	editCmd.Flags().Bool("md", false, "Edit as pure markdown (strips IAL, creates new block IDs on save)")
}

func runEdit(cmd *cobra.Command, args []string) error {
	client := GetClient()
	notebook := GetCurrentNotebook()

	doAppend, _ := cmd.Flags().GetBool("append")
	doPrepend, _ := cmd.Flags().GetBool("prepend")
	afterBlock, _ := cmd.Flags().GetString("after")
	replaceBlock, _ := cmd.Flags().GetString("replace")
	deleteBlock, _ := cmd.Flags().GetString("delete")
	content, _ := cmd.Flags().GetString("content")
	editorFlag, _ := cmd.Flags().GetString("editor")
	codeFile, _ := cmd.Flags().GetString("code")
	lang, _ := cmd.Flags().GetString("lang")

	pureMarkdown, _ := cmd.Flags().GetBool("md")

	docRef := args[0]

	// Resolve document ID
	docID, err := util.ResolveDocumentID(client, notebook, docRef)
	if err != nil {
		return err
	}

	// Handle delete (no content needed)
	if deleteBlock != "" {
		if err := client.DeleteBlock(deleteBlock); err != nil {
			return fmt.Errorf("failed to delete block: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Deleted block: %s\n", deleteBlock)
		return nil
	}

	// Determine content source: --content or --code
	if codeFile != "" {
		codeContent, err := os.ReadFile(codeFile)
		if err != nil {
			return fmt.Errorf("failed to read code file: %w", err)
		}
		if lang == "" {
			lang = detectLanguage(codeFile)
		}
		content = formatCodeBlock(string(codeContent), lang)
	}

	// Expand block references without anchor text to include titles
	if content != "" {
		content = expandBlockReferences(client, content)
	}

	// Handle replace
	if replaceBlock != "" {
		if content == "" {
			return fmt.Errorf("--content or --code is required with --replace")
		}
		if err := client.UpdateBlock(replaceBlock, content); err != nil {
			return fmt.Errorf("failed to replace block: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Replaced block: %s\n", replaceBlock)
		return nil
	}

	// Handle after
	if afterBlock != "" {
		if content == "" {
			return fmt.Errorf("--content or --code is required with --after")
		}
		newID, err := client.InsertBlockAfter(afterBlock, content)
		if err != nil {
			return fmt.Errorf("failed to insert block: %w", err)
		}
		fmt.Printf("%s\n", newID)
		fmt.Fprintf(os.Stderr, "Inserted block after %s\n", afterBlock)
		return nil
	}

	// Handle append
	if doAppend {
		if content == "" {
			return fmt.Errorf("--content or --code is required with --append")
		}
		return appendToDocument(client, docID, content)
	}

	// Handle prepend
	if doPrepend {
		if content == "" {
			return fmt.Errorf("--content or --code is required with --prepend")
		}
		return prependToDocument(client, docID, content)
	}

	// If --content or --code provided without position flag, default to append
	if content != "" {
		return appendToDocument(client, docID, content)
	}

	// Interactive mode - open in editor
	return runInteractiveEdit(client, docID, editorFlag, pureMarkdown)
}

func appendToDocument(client interface{}, docID, content string) error {
	apiClient := GetClient()

	children, err := apiClient.GetChildBlocks(docID)
	if err != nil {
		return fmt.Errorf("failed to get document children: %w", err)
	}

	var newID string
	if len(children) == 0 {
		newID, err = apiClient.InsertBlockAsChild(docID, content)
	} else {
		lastChild := children[len(children)-1]
		newID, err = apiClient.InsertBlockAfter(lastChild.ID, content)
	}

	if err != nil {
		return fmt.Errorf("failed to append content: %w", err)
	}

	fmt.Printf("%s\n", newID)
	fmt.Fprintf(os.Stderr, "Appended content\n")
	return nil
}

func prependToDocument(client interface{}, docID, content string) error {
	apiClient := GetClient()

	newID, err := apiClient.InsertBlockAsChild(docID, content)
	if err != nil {
		return fmt.Errorf("failed to prepend content: %w", err)
	}

	fmt.Printf("%s\n", newID)
	fmt.Fprintf(os.Stderr, "Prepended content\n")
	return nil
}

func formatCodeBlock(content, lang string) string {
	content = strings.TrimRight(content, "\n")
	return fmt.Sprintf("```%s\n%s\n```", lang, content)
}

// blockRefPattern matches block references without anchor text: ((block-id))
// Does not match refs that already have anchor text: ((block-id "text")) or ((block-id 'text'))
var blockRefPattern = regexp.MustCompile(`\(\((\d{14}-[a-z0-9]{7})\)\)`)

// expandBlockReferences finds block references without anchor text and adds titles
func expandBlockReferences(client *api.Client, content string) string {
	return blockRefPattern.ReplaceAllStringFunc(content, func(match string) string {
		// Extract block ID from ((id))
		id := blockRefPattern.FindStringSubmatch(match)[1]

		// Look up the title
		title, err := client.GetBlockTitle(id)
		if err != nil || title == "" {
			// Keep original if lookup fails
			return match
		}

		// Escape quotes in title
		title = strings.ReplaceAll(title, `"`, `\"`)

		// Return with anchor text
		return fmt.Sprintf(`((%s "%s"))`, id, title)
	})
}

func detectLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	langMap := map[string]string{
		".go": "go", ".py": "python", ".js": "javascript", ".ts": "typescript",
		".tsx": "tsx", ".jsx": "jsx", ".rs": "rust", ".c": "c", ".cpp": "cpp",
		".cc": "cpp", ".h": "c", ".hpp": "cpp", ".java": "java", ".kt": "kotlin",
		".swift": "swift", ".rb": "ruby", ".php": "php", ".sh": "bash",
		".bash": "bash", ".zsh": "zsh", ".fish": "fish", ".ps1": "powershell",
		".sql": "sql", ".html": "html", ".htm": "html", ".css": "css",
		".scss": "scss", ".sass": "sass", ".less": "less", ".json": "json",
		".yaml": "yaml", ".yml": "yaml", ".toml": "toml", ".xml": "xml",
		".md": "markdown", ".lua": "lua", ".r": "r", ".R": "r", ".pl": "perl",
		".ex": "elixir", ".exs": "elixir", ".erl": "erlang", ".hs": "haskell",
		".ml": "ocaml", ".fs": "fsharp", ".cs": "csharp", ".vb": "vbnet",
		".scala": "scala", ".clj": "clojure", ".lisp": "lisp", ".scm": "scheme",
		".rkt": "racket", ".vim": "vim", ".dockerfile": "dockerfile",
		".makefile": "makefile", ".mk": "makefile", ".nginx": "nginx",
		".conf": "ini", ".ini": "ini", ".env": "bash", ".graphql": "graphql",
		".gql": "graphql", ".proto": "protobuf", ".asm": "asm", ".s": "asm",
		".sol": "solidity", ".zig": "zig", ".nim": "nim", ".v": "v", ".d": "d",
		".dart": "dart", ".groovy": "groovy", ".gradle": "groovy",
	}

	if lang, ok := langMap[ext]; ok {
		return lang
	}

	base := strings.ToLower(filepath.Base(filename))
	specialFiles := map[string]string{
		"dockerfile": "dockerfile", "makefile": "makefile",
		"cmakelists.txt": "cmake", "jenkinsfile": "groovy",
		"vagrantfile": "ruby", "gemfile": "ruby", "rakefile": "ruby",
		"go.mod": "gomod", "go.sum": "gosum", "cargo.toml": "toml",
		"cargo.lock": "toml", "package.json": "json", "tsconfig.json": "json",
		"composer.json": "json", "requirements.txt": "text", "setup.py": "python",
		"pyproject.toml": "toml", "build.gradle": "groovy", "pom.xml": "xml",
	}

	if lang, ok := specialFiles[base]; ok {
		return lang
	}

	return ""
}

func runInteractiveEdit(client interface{}, docID, editorFlag string, pureMarkdown bool) error {
	apiClient := GetClient()

	content, meta, err := exportDocument(apiClient, docID, true)
	if err != nil {
		return fmt.Errorf("failed to export document: %w", err)
	}

	// Strip IAL for pure markdown mode
	if pureMarkdown {
		content = stripIAL(content)
	}

	tmpFile, err := os.CreateTemp("", "sy-edit-*.md")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmpFile.WriteString(content); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	_ = tmpFile.Close()

	origInfo, err := os.Stat(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to stat temp file: %w", err)
	}

	editor := editorFlag
	if editor == "" {
		editor = os.Getenv("EDITOR")
		if editor == "" {
			editor = os.Getenv("VISUAL")
			if editor == "" {
				editor = "vim"
			}
		}
	}

	editorCmd := exec.Command(editor, tmpPath)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("editor failed: %w", err)
	}

	newInfo, err := os.Stat(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to stat temp file after edit: %w", err)
	}

	if newInfo.ModTime().Equal(origInfo.ModTime()) {
		fmt.Fprintln(os.Stderr, "No changes made")
		return nil
	}

	newContent, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to read modified file: %w", err)
	}

	_, bodyContent, err := util.UnmarshalFrontmatter(string(newContent))
	if err != nil {
		return fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	_, bodyContent = util.StripH1Title(bodyContent)
	bodyContent = strings.TrimSpace(bodyContent)

	if err := apiClient.UpdateBlock(docID, bodyContent); err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Updated: %s\n", meta.HPath)
	return nil
}
