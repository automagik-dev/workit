package docx_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/automagik-dev/workit/internal/docx"
)

func TestListTables(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	tables, err := docx.ListTables(session)
	if err != nil {
		t.Fatalf("ListTables: %v", err)
	}

	if len(tables) != 1 {
		t.Fatalf("table count = %d, want 1", len(tables))
	}

	tbl := tables[0]
	if tbl.Index != 0 {
		t.Errorf("table index = %d, want 0", tbl.Index)
	}

	if tbl.Rows != 3 {
		t.Errorf("rows = %d, want 3", tbl.Rows)
	}

	if tbl.Cols != 3 {
		t.Errorf("cols = %d, want 3", tbl.Cols)
	}

	// Verify header row.
	if len(tbl.Data) < 1 {
		t.Fatal("no data rows")
	}

	header := tbl.Data[0]
	if len(header) < 3 {
		t.Fatalf("header cols = %d, want 3", len(header))
	}

	if header[0] != "Name" || header[1] != "Role" || header[2] != "Status" {
		t.Errorf("header = %v, want [Name Role Status]", header)
	}

	// Verify data rows.
	if tbl.Data[1][0] != "Alice" {
		t.Errorf("cell [1][0] = %q, want Alice", tbl.Data[1][0])
	}

	if tbl.Data[2][1] != "Designer" {
		t.Errorf("cell [2][1] = %q, want Designer", tbl.Data[2][1])
	}
}

func TestAddTableRow(t *testing.T) {
	path := ensureSampleDOCX(t)
	tmp := filepath.Join(t.TempDir(), "add-row.docx")
	copyFile(t, path, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	err = docx.AddTableRow(session, 0, "Carol, Manager, Active")
	if err != nil {
		t.Fatalf("AddTableRow: %v", err)
	}

	if saveErr := session.SaveInPlace(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	session.Close()

	// Re-open and verify.
	session2, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer session2.Close()

	tables, err := docx.ListTables(session2)
	if err != nil {
		t.Fatalf("ListTables: %v", err)
	}

	if len(tables) != 1 {
		t.Fatalf("table count = %d, want 1", len(tables))
	}

	tbl := tables[0]
	if tbl.Rows != 4 {
		t.Fatalf("rows = %d, want 4", tbl.Rows)
	}

	// Verify new row content.
	newRow := tbl.Data[3]
	if len(newRow) < 3 {
		t.Fatalf("new row cols = %d, want >= 3", len(newRow))
	}

	if newRow[0] != "Carol" {
		t.Errorf("new row[0] = %q, want Carol", newRow[0])
	}

	if newRow[1] != "Manager" {
		t.Errorf("new row[1] = %q, want Manager", newRow[1])
	}

	if newRow[2] != "Active" {
		t.Errorf("new row[2] = %q, want Active", newRow[2])
	}
}

func TestUpdateTableCell(t *testing.T) {
	path := ensureSampleDOCX(t)
	tmp := filepath.Join(t.TempDir(), "update-cell.docx")
	copyFile(t, path, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	// Update Alice's role (row 2, col 2) to "Lead Engineer".
	err = docx.UpdateTableCell(session, 0, 2, 2, "Lead Engineer")
	if err != nil {
		t.Fatalf("UpdateTableCell: %v", err)
	}

	if saveErr := session.SaveInPlace(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	session.Close()

	// Re-open and verify.
	session2, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer session2.Close()

	tables, err := docx.ListTables(session2)
	if err != nil {
		t.Fatalf("ListTables: %v", err)
	}

	cell := tables[0].Data[1][1] // row 2 (0-indexed=1), col 2 (0-indexed=1)
	if cell != "Lead Engineer" {
		t.Errorf("cell = %q, want 'Lead Engineer'", cell)
	}

	// Verify markdown output too.
	md, err := docx.ReadAsMarkdown(session2)
	if err != nil {
		t.Fatalf("ReadAsMarkdown: %v", err)
	}

	if !strings.Contains(md, "Lead Engineer") {
		t.Errorf("markdown does not contain updated cell:\n%s", md)
	}
}

func TestDeleteTableRow(t *testing.T) {
	path := ensureSampleDOCX(t)
	tmp := filepath.Join(t.TempDir(), "delete-row.docx")
	copyFile(t, path, tmp)

	session, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	// Delete the second data row (row 3, which is Bob's row).
	err = docx.DeleteTableRow(session, 0, 3)
	if err != nil {
		t.Fatalf("DeleteTableRow: %v", err)
	}

	if saveErr := session.SaveInPlace(); saveErr != nil {
		t.Fatalf("Save: %v", saveErr)
	}

	session.Close()

	// Re-open and verify.
	session2, err := docx.Open(tmp)
	if err != nil {
		t.Fatalf("re-Open: %v", err)
	}
	defer session2.Close()

	tables, err := docx.ListTables(session2)
	if err != nil {
		t.Fatalf("ListTables: %v", err)
	}

	tbl := tables[0]
	if tbl.Rows != 2 {
		t.Fatalf("rows = %d, want 2", tbl.Rows)
	}

	// Verify Bob is gone.
	md, err := docx.ReadAsMarkdown(session2)
	if err != nil {
		t.Fatalf("ReadAsMarkdown: %v", err)
	}

	if strings.Contains(md, "Bob") {
		t.Errorf("deleted row 'Bob' still present:\n%s", md)
	}
	// Alice should remain.
	if !strings.Contains(md, "Alice") {
		t.Errorf("Alice should remain:\n%s", md)
	}
}

func TestAddTableRowOutOfRange(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	err = docx.AddTableRow(session, 99, "a,b,c")
	if err == nil {
		t.Fatal("expected error for out-of-range table index")
	}
}

func TestUpdateTableCellOutOfRange(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	err = docx.UpdateTableCell(session, 0, 99, 1, "x")
	if err == nil {
		t.Fatal("expected error for out-of-range row")
	}

	err = docx.UpdateTableCell(session, 0, 1, 99, "x")
	if err == nil {
		t.Fatal("expected error for out-of-range col")
	}
}

func TestDeleteTableRowOutOfRange(t *testing.T) {
	path := ensureSampleDOCX(t)

	session, err := docx.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer session.Close()

	err = docx.DeleteTableRow(session, 0, 99)
	if err == nil {
		t.Fatal("expected error for out-of-range row")
	}
}
