package cacik

import (
	"testing"

	messages "github.com/cucumber/messages/go/v21"
	"github.com/stretchr/testify/require"
)

func TestNewTable(t *testing.T) {
	t.Run("creates table from raw data", func(t *testing.T) {
		data := [][]string{
			{"name", "email"},
			{"Alice", "alice@test.com"},
			{"Bob", "bob@test.com"},
		}

		table := NewTable(data)

		require.Equal(t, 3, table.Len())
		require.Equal(t, []string{"name", "email"}, table.Headers())
	})

	t.Run("empty data creates empty table", func(t *testing.T) {
		table := NewTable([][]string{})

		require.Equal(t, 0, table.Len())
		require.Empty(t, table.Headers())
	})

	t.Run("single row table (header only)", func(t *testing.T) {
		data := [][]string{
			{"name", "email"},
		}

		table := NewTable(data)

		require.Equal(t, 1, table.Len())
		require.Equal(t, []string{"name", "email"}, table.Headers())
	})
}

func TestNewTableFromDataTable(t *testing.T) {
	t.Run("converts Gherkin DataTable", func(t *testing.T) {
		dt := &messages.DataTable{
			Rows: []*messages.TableRow{
				{Cells: []*messages.TableCell{{Value: "name"}, {Value: "age"}}},
				{Cells: []*messages.TableCell{{Value: "Alice"}, {Value: "30"}}},
				{Cells: []*messages.TableCell{{Value: "Bob"}, {Value: "25"}}},
			},
		}

		table := NewTableFromDataTable(dt)

		require.Equal(t, 3, table.Len())
		require.Equal(t, []string{"name", "age"}, table.Headers())
	})

	t.Run("nil DataTable creates empty table", func(t *testing.T) {
		table := NewTableFromDataTable(nil)

		require.Equal(t, 0, table.Len())
		require.Empty(t, table.Headers())
	})

	t.Run("DataTable with no rows creates empty table", func(t *testing.T) {
		dt := &messages.DataTable{
			Rows: []*messages.TableRow{},
		}

		table := NewTableFromDataTable(dt)

		require.Equal(t, 0, table.Len())
	})
}

func TestRow_Get(t *testing.T) {
	data := [][]string{
		{"name", "email", "age"},
		{"Alice", "alice@test.com", "30"},
		{"Bob", "bob@test.com", "25"},
	}
	table := NewTable(data)

	t.Run("returns value by column header", func(t *testing.T) {
		var names []string
		for _, row := range table.SkipHeader() {
			names = append(names, row.Get("name"))
		}
		require.Equal(t, []string{"Alice", "Bob"}, names)
	})

	t.Run("case-insensitive header lookup", func(t *testing.T) {
		for _, row := range table.SkipHeader() {
			require.Equal(t, row.Get("name"), row.Get("NAME"))
			require.Equal(t, row.Get("name"), row.Get("Name"))
			break // only need the first row
		}
	})

	t.Run("returns empty string for unknown column", func(t *testing.T) {
		for _, row := range table.SkipHeader() {
			require.Equal(t, "", row.Get("nonexistent"))
			break
		}
	})
}

func TestRow_Cell(t *testing.T) {
	data := [][]string{
		{"10", "20"},
		{"30", "40"},
	}
	table := NewTable(data)

	t.Run("returns value by column index", func(t *testing.T) {
		var results []string
		for _, row := range table.All() {
			results = append(results, row.Cell(0))
		}
		require.Equal(t, []string{"10", "30"}, results)
	})

	t.Run("returns empty string for out-of-range index", func(t *testing.T) {
		for _, row := range table.All() {
			require.Equal(t, "", row.Cell(-1))
			require.Equal(t, "", row.Cell(99))
			break
		}
	})
}

func TestRow_Values(t *testing.T) {
	data := [][]string{
		{"a", "b", "c"},
		{"1", "2", "3"},
	}
	table := NewTable(data)

	for _, row := range table.SkipHeader() {
		require.Equal(t, []string{"1", "2", "3"}, row.Values())
	}
}

func TestRow_Len(t *testing.T) {
	data := [][]string{
		{"a", "b"},
		{"1", "2"},
	}
	table := NewTable(data)

	for _, row := range table.All() {
		require.Equal(t, 2, row.Len())
		break
	}
}

func TestTable_All(t *testing.T) {
	data := [][]string{
		{"name", "email"},
		{"Alice", "alice@test.com"},
		{"Bob", "bob@test.com"},
	}
	table := NewTable(data)

	t.Run("iterates over all rows including header", func(t *testing.T) {
		var indices []int
		var firstCells []string
		for i, row := range table.All() {
			indices = append(indices, i)
			firstCells = append(firstCells, row.Cell(0))
		}
		require.Equal(t, []int{0, 1, 2}, indices)
		require.Equal(t, []string{"name", "Alice", "Bob"}, firstCells)
	})

	t.Run("supports early break", func(t *testing.T) {
		count := 0
		for range table.All() {
			count++
			break
		}
		require.Equal(t, 1, count)
	})
}

func TestTable_SkipHeader(t *testing.T) {
	data := [][]string{
		{"name", "email"},
		{"Alice", "alice@test.com"},
		{"Bob", "bob@test.com"},
	}
	table := NewTable(data)

	t.Run("iterates over data rows only", func(t *testing.T) {
		var indices []int
		var names []string
		for i, row := range table.SkipHeader() {
			indices = append(indices, i)
			names = append(names, row.Get("name"))
		}
		require.Equal(t, []int{0, 1}, indices)
		require.Equal(t, []string{"Alice", "Bob"}, names)
	})

	t.Run("empty data rows when table has only header", func(t *testing.T) {
		headerOnly := NewTable([][]string{{"name", "email"}})
		count := 0
		for range headerOnly.SkipHeader() {
			count++
		}
		require.Equal(t, 0, count)
	})

	t.Run("Get still works using first row as headers", func(t *testing.T) {
		for _, row := range table.SkipHeader() {
			// Get should use the first row ("name", "email") as headers
			email := row.Get("email")
			require.NotEmpty(t, email)
			break
		}
	})
}

func TestTable_Headers_IsCopy(t *testing.T) {
	data := [][]string{
		{"name", "email"},
		{"Alice", "alice@test.com"},
	}
	table := NewTable(data)

	headers := table.Headers()
	headers[0] = "MODIFIED"

	// Original should not be affected
	require.Equal(t, []string{"name", "email"}, table.Headers())
}

func TestRow_Values_IsCopy(t *testing.T) {
	data := [][]string{
		{"a"},
		{"1"},
	}
	table := NewTable(data)

	for _, row := range table.SkipHeader() {
		values := row.Values()
		values[0] = "MODIFIED"
		require.Equal(t, "1", row.Cell(0))
	}
}
