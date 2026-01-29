package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xtruder/sy/internal/api"
	"github.com/xtruder/sy/internal/format"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Full-text search documents",
	Long: `Search for documents containing the specified query.

Optionally limit search to specific block types.`,
	Example: `  sy search "keyword"                  # Search all docs
  sy search --type heading "API"       # Search only headings
  sy search --type paragraph "TODO"    # Search only paragraphs
  sy search --type code "function"     # Search only code blocks`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSearch,
}

func init() {
	searchCmd.Flags().String("type", "", "Block type filter: heading, paragraph, code, math, table, html, embed")
	searchCmd.Flags().Int("limit", 20, "Maximum number of results")
}

func runSearch(cmd *cobra.Command, args []string) error {
	client := GetClient()
	typeFilter, _ := cmd.Flags().GetString("type")
	limit, _ := cmd.Flags().GetInt("limit")

	query := args[0]

	// Build search types based on filter
	var types *api.SearchTypes
	if typeFilter != "" {
		types = &api.SearchTypes{}
		switch strings.ToLower(typeFilter) {
		case "heading", "h":
			types.Heading = true
		case "paragraph", "p":
			types.Paragraph = true
		case "code", "c":
			types.CodeBlock = true
		case "math", "m":
			types.MathBlock = true
		case "table", "t":
			types.Table = true
		case "html":
			types.HTMLBlock = true
		case "embed":
			types.EmbedBlock = true
		default:
			return fmt.Errorf("unknown block type: %s", typeFilter)
		}
	}

	// Execute search
	results, err := client.FullTextSearch(query, types)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(results.Blocks) == 0 {
		fmt.Fprintln(os.Stderr, "No results found")
		return nil
	}

	// Limit results
	blocks := results.Blocks
	if len(blocks) > limit {
		blocks = blocks[:limit]
	}

	// Output results
	table := format.NewTable(
		format.Column{Header: "ID", Width: 22},
		format.Column{Header: "Type", Width: 4},
		format.Column{Header: "Content", MaxWidth: 60},
		format.Column{Header: "Path", MaxWidth: 40},
	)

	for _, block := range blocks {
		content := truncateContent(block.Content, 60)
		path := block.HPath
		if path == "" {
			path = block.Path
		}
		table.AddRow(block.ID, block.Type, content, path)
	}

	table.Render(os.Stdout)

	if len(results.Blocks) > limit {
		fmt.Fprintf(os.Stderr, "\nShowing %d of %d results (use --limit to see more)\n", limit, len(results.Blocks))
	}

	return nil
}

// truncateContent truncates content for display, removing newlines
func truncateContent(content string, maxLen int) string {
	// Replace newlines with spaces
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\r", "")

	// Collapse multiple spaces
	for strings.Contains(content, "  ") {
		content = strings.ReplaceAll(content, "  ", " ")
	}

	content = strings.TrimSpace(content)

	if len(content) > maxLen {
		return content[:maxLen-3] + "..."
	}
	return content
}
