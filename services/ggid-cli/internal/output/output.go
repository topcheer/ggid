// Package output provides formatting utilities for CLI output.
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// Format controls how data is displayed.
type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
)

// PrintJSON pretty-prints data as indented JSON.
func PrintJSON(data any) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error formatting output: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(b))
}

// PrintRaw prints raw bytes as-is.
func PrintRaw(data []byte) {
	fmt.Println(string(data))
}

// PrintError prints an error message to stderr.
func PrintError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// PrintSuccess prints a success message to stdout.
func PrintSuccess(format string, args ...any) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// Table represents a table for CLI output.
type Table struct {
	headers []string
	rows    [][]string
}

// NewTable creates a new table with the given headers.
func NewTable(headers ...string) *Table {
	return &Table{headers: headers}
}

// AddRow adds a row to the table.
func (t *Table) AddRow(values ...string) {
	t.rows = append(t.rows, values)
}

// Print renders the table using tabwriter.
func (t *Table) Print() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	// Print headers
	fmt.Fprintln(w, strings.Join(t.headers, "\t"))
	// Print separator
	separators := make([]string, len(t.headers))
	for i := range separators {
		separators[i] = strings.Repeat("-", len(t.headers[i]))
	}
	fmt.Fprintln(w, strings.Join(separators, "\t"))
	// Print rows
	for _, row := range t.rows {
		// Pad row to match headers
		for len(row) < len(t.headers) {
			row = append(row, "")
		}
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

// Truncate truncates a string to maxLen, adding "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
