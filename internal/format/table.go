package format

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// Column defines a table column
type Column struct {
	Header   string
	Width    int // Min width, 0 for auto
	MaxWidth int // Max width, 0 for unlimited
	Align    Alignment
}

// Alignment for column text
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignRight
	AlignCenter
)

// Table renders tabular data
type Table struct {
	columns []Column
	rows    [][]string
}

// NewTable creates a new table with the given columns
func NewTable(columns ...Column) *Table {
	return &Table{
		columns: columns,
		rows:    make([][]string, 0),
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(values ...string) {
	// Pad or truncate to match column count
	row := make([]string, len(t.columns))
	for i := range row {
		if i < len(values) {
			row[i] = values[i]
		}
	}
	t.rows = append(t.rows, row)
}

// RowCount returns the number of rows in the table
func (t *Table) RowCount() int {
	return len(t.rows)
}

// Render writes the table to the writer
func (t *Table) Render(w io.Writer) {
	if len(t.columns) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(t.columns))
	for i, col := range t.columns {
		widths[i] = utf8.RuneCountInString(col.Header)
		if col.Width > widths[i] {
			widths[i] = col.Width
		}
	}

	// Check row data for max widths
	for _, row := range t.rows {
		for i, cell := range row {
			cellWidth := utf8.RuneCountInString(cell)
			if cellWidth > widths[i] {
				widths[i] = cellWidth
			}
		}
	}

	// Apply max width limits
	for i, col := range t.columns {
		if col.MaxWidth > 0 && widths[i] > col.MaxWidth {
			widths[i] = col.MaxWidth
		}
	}

	// Print header
	t.printRow(w, t.getHeaders(), widths)
	t.printSeparator(w, widths)

	// Print rows
	for _, row := range t.rows {
		t.printRow(w, row, widths)
	}
}

func (t *Table) getHeaders() []string {
	headers := make([]string, len(t.columns))
	for i, col := range t.columns {
		headers[i] = col.Header
	}
	return headers
}

func (t *Table) printRow(w io.Writer, row []string, widths []int) {
	parts := make([]string, len(t.columns))
	for i, cell := range row {
		parts[i] = t.formatCell(cell, widths[i], t.columns[i].Align)
	}
	_, _ = fmt.Fprintln(w, strings.Join(parts, "  "))
}

func (t *Table) printSeparator(w io.Writer, widths []int) {
	parts := make([]string, len(widths))
	for i, width := range widths {
		parts[i] = strings.Repeat("-", width)
	}
	_, _ = fmt.Fprintln(w, strings.Join(parts, "  "))
}

func (t *Table) formatCell(value string, width int, align Alignment) string {
	runeCount := utf8.RuneCountInString(value)

	// Truncate if needed
	if runeCount > width {
		runes := []rune(value)
		if width > 3 {
			return string(runes[:width-3]) + "..."
		}
		return string(runes[:width])
	}

	// Pad based on alignment
	padding := width - runeCount
	switch align {
	case AlignRight:
		return strings.Repeat(" ", padding) + value
	case AlignCenter:
		left := padding / 2
		right := padding - left
		return strings.Repeat(" ", left) + value + strings.Repeat(" ", right)
	default: // AlignLeft
		return value + strings.Repeat(" ", padding)
	}
}

// SimpleTable is a convenience function to print a simple table
func SimpleTable(w io.Writer, headers []string, rows [][]string) {
	columns := make([]Column, len(headers))
	for i, h := range headers {
		columns[i] = Column{Header: h}
	}

	t := NewTable(columns...)
	for _, row := range rows {
		t.AddRow(row...)
	}
	t.Render(w)
}
