package cacik

import (
	"iter"
	"strings"

	messages "github.com/cucumber/messages/go/v21"
)

// Row represents a single row in a DataTable.
type Row struct {
	cells   []string
	headers []string // reference to the table's header row (first row values)
}

// Get returns the cell value by column header name (case-insensitive).
// Returns an empty string if the column is not found or the row has fewer cells.
func (r Row) Get(col string) string {
	colLower := strings.ToLower(col)
	for i, h := range r.headers {
		if strings.ToLower(h) == colLower {
			if i < len(r.cells) {
				return r.cells[i]
			}
			return ""
		}
	}
	return ""
}

// Cell returns the cell value by column index (0-based).
// Returns an empty string if the index is out of range.
func (r Row) Cell(index int) string {
	if index < 0 || index >= len(r.cells) {
		return ""
	}
	return r.cells[index]
}

// Values returns all cell values in order.
func (r Row) Values() []string {
	cp := make([]string, len(r.cells))
	copy(cp, r.cells)
	return cp
}

// Len returns the number of cells in the row.
func (r Row) Len() int {
	return len(r.cells)
}

// Table represents a Gherkin DataTable attached to a step.
type Table struct {
	headers []string
	rows    []Row
}

// NewTable creates a Table from raw string data.
// The first row is used as column headers for Get() lookups.
func NewTable(data [][]string) Table {
	if len(data) == 0 {
		return Table{}
	}

	headers := make([]string, len(data[0]))
	copy(headers, data[0])

	rows := make([]Row, len(data))
	for i, cells := range data {
		cellsCopy := make([]string, len(cells))
		copy(cellsCopy, cells)
		rows[i] = Row{
			cells:   cellsCopy,
			headers: headers,
		}
	}

	return Table{
		headers: headers,
		rows:    rows,
	}
}

// NewTableFromDataTable creates a Table from a Gherkin DataTable message.
func NewTableFromDataTable(dt *messages.DataTable) Table {
	if dt == nil || len(dt.Rows) == 0 {
		return Table{}
	}

	data := make([][]string, len(dt.Rows))
	for i, row := range dt.Rows {
		cells := make([]string, len(row.Cells))
		for j, cell := range row.Cells {
			cells[j] = cell.Value
		}
		data[i] = cells
	}

	return NewTable(data)
}

// Headers returns the column headers (values from the first row).
func (t Table) Headers() []string {
	cp := make([]string, len(t.headers))
	copy(cp, t.headers)
	return cp
}

// Len returns the total number of rows (including the header row).
func (t Table) Len() int {
	return len(t.rows)
}

// All returns an iterator over all rows (including the header row).
// The index is 0-based.
//
// Usage:
//
//	for i, row := range table.All() {
//	    fmt.Println(i, row.Cell(0))
//	}
func (t Table) All() iter.Seq2[int, Row] {
	return func(yield func(int, Row) bool) {
		for i, row := range t.rows {
			if !yield(i, row) {
				return
			}
		}
	}
}

// SkipHeader returns an iterator over data rows only (skips the first row).
// The index is 0-based starting from the first data row.
// Row.Get(col) uses the skipped header row for column name lookups.
//
// Usage:
//
//	for i, row := range table.SkipHeader() {
//	    name := row.Get("name")
//	    fmt.Println(i, name)
//	}
func (t Table) SkipHeader() iter.Seq2[int, Row] {
	return func(yield func(int, Row) bool) {
		for i := 1; i < len(t.rows); i++ {
			if !yield(i-1, t.rows[i]) {
				return
			}
		}
	}
}
