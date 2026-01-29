package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xtruder/sy/internal/format"
)

var sqlCmd = &cobra.Command{
	Use:   "sql <query>",
	Short: "Execute SQL query",
	Long: `Execute a SQL query against SiYuan's block database.

Available tables: blocks, spans, assets, refs, file_annotation_refs

Common block columns: id, parent_id, root_id, box, path, hpath, name, 
content, type, subtype, created, updated`,
	Example: `  sy sql "SELECT * FROM blocks WHERE content LIKE '%search%' LIMIT 10"
  sy sql --format table "SELECT id, content FROM blocks LIMIT 5"
  sy sql --format tsv "SELECT id, type, content FROM blocks LIMIT 10"
  sy sql --format json "SELECT * FROM blocks WHERE type='d' LIMIT 5"`,
	Args: cobra.ExactArgs(1),
	RunE: runSQL,
}

func init() {
	sqlCmd.Flags().String("format", "table", "Output format: table, tsv, json")
}

func runSQL(cmd *cobra.Command, args []string) error {
	client := GetClient()
	formatType, _ := cmd.Flags().GetString("format")

	query := args[0]

	// Execute query
	results, err := client.ExecuteSQL(query)
	if err != nil {
		return fmt.Errorf("SQL query failed: %w", err)
	}

	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "No results")
		return nil
	}

	// Output based on format
	switch formatType {
	case "json":
		return outputJSON(results)
	case "tsv":
		return outputTSV(results)
	case "table":
		fallthrough
	default:
		return outputTable(results)
	}
}

// outputJSON outputs results as JSON array
func outputJSON(results []map[string]interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// outputTSV outputs results as tab-separated values
func outputTSV(results []map[string]interface{}) error {
	if len(results) == 0 {
		return nil
	}

	// Get column names from first row
	columns := getColumnNames(results[0])

	// Print header
	fmt.Println(strings.Join(columns, "\t"))

	// Print rows
	for _, row := range results {
		values := make([]string, len(columns))
		for i, col := range columns {
			values[i] = formatValue(row[col])
		}
		fmt.Println(strings.Join(values, "\t"))
	}

	return nil
}

// outputTable outputs results as ASCII table
func outputTable(results []map[string]interface{}) error {
	if len(results) == 0 {
		return nil
	}

	// Get column names from first row
	columns := getColumnNames(results[0])

	// Create table
	tableColumns := make([]format.Column, len(columns))
	for i, col := range columns {
		maxWidth := 50
		// Larger width for certain columns
		if col == "content" || col == "markdown" {
			maxWidth = 80
		}
		tableColumns[i] = format.Column{
			Header:   col,
			MaxWidth: maxWidth,
		}
	}

	table := format.NewTable(tableColumns...)

	// Add rows
	for _, row := range results {
		values := make([]string, len(columns))
		for i, col := range columns {
			values[i] = formatValue(row[col])
		}
		table.AddRow(values...)
	}

	table.Render(os.Stdout)
	return nil
}

// getColumnNames extracts sorted column names from a row
func getColumnNames(row map[string]interface{}) []string {
	columns := make([]string, 0, len(row))
	for k := range row {
		columns = append(columns, k)
	}

	// Sort with common columns first
	priority := map[string]int{
		"id":        0,
		"type":      1,
		"subtype":   2,
		"name":      3,
		"content":   4,
		"box":       5,
		"path":      6,
		"hpath":     7,
		"parent_id": 8,
		"root_id":   9,
		"created":   10,
		"updated":   11,
	}

	sort.Slice(columns, func(i, j int) bool {
		pi, oki := priority[columns[i]]
		pj, okj := priority[columns[j]]
		if oki && okj {
			return pi < pj
		}
		if oki {
			return true
		}
		if okj {
			return false
		}
		return columns[i] < columns[j]
	})

	return columns
}

// formatValue converts a value to string representation
func formatValue(v interface{}) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		// Replace newlines and tabs for display
		s := strings.ReplaceAll(val, "\n", "\\n")
		s = strings.ReplaceAll(s, "\t", "\\t")
		return s
	case float64:
		// JSON numbers are decoded as float64
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", val)
	}
}
