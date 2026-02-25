package docx

import (
	"errors"
	"fmt"
	"strings"

	"github.com/beevik/etree"
)

// Sentinel errors for table operations.
var (
	errTableOutOfRange = errors.New("table index out of range")
	errTableNoRows     = errors.New("table has no rows")
	errRowMinOne       = errors.New("row must be >= 1")
	errColMinOne       = errors.New("col must be >= 1")
	errRowOutOfRange   = errors.New("row out of range in table")
	errColOutOfRange   = errors.New("col out of range in row of table")
)

// TableDetail describes a table's structure and content.
type TableDetail struct {
	Index int        `json:"index"`
	Rows  int        `json:"rows"`
	Cols  int        `json:"cols"`
	Data  [][]string `json:"data"` // cell text content
}

// ListTables returns info about all tables in the document.
func ListTables(session *EditSession) ([]TableDetail, error) {
	doc, err := session.Part("word/document.xml")
	if err != nil {
		return nil, err
	}

	body := findBody(doc)
	if body == nil {
		return nil, errNoBody
	}

	var tables []TableDetail

	idx := 0

	for _, child := range body.ChildElements() {
		if child.Tag != tagTbl {
			continue
		}

		td := tableDetail(child, idx)
		tables = append(tables, td)
		idx++
	}

	return tables, nil
}

// tableDetail extracts structure and content from a w:tbl element.
func tableDetail(tbl *etree.Element, index int) TableDetail {
	td := TableDetail{Index: index}

	for _, tr := range tbl.ChildElements() {
		if tr.Tag != "tr" {
			continue
		}

		var row []string

		for _, tc := range tr.ChildElements() {
			if tc.Tag != "tc" {
				continue
			}

			// A cell can contain multiple paragraphs; join with space.
			var cellParts []string

			for _, p := range tc.ChildElements() {
				if p.Tag == "p" {
					t := paragraphText(p)
					if t != "" {
						cellParts = append(cellParts, t)
					}
				}
			}

			row = append(row, strings.Join(cellParts, " "))
		}

		td.Data = append(td.Data, row)
		td.Rows++

		if len(row) > td.Cols {
			td.Cols = len(row)
		}
	}

	return td
}

// AddTableRow appends a new row to a table.
// tableIdx is 0-based. values is comma-separated cell values.
func AddTableRow(session *EditSession, tableIdx int, values string) error {
	doc, err := session.Part("word/document.xml")
	if err != nil {
		return err
	}

	body := findBody(doc)
	if body == nil {
		return errNoBody
	}

	tbl := findTableByIndex(body, tableIdx)
	if tbl == nil {
		return fmt.Errorf("%w: %d", errTableOutOfRange, tableIdx)
	}

	// Find the last row to clone its structure (cell count and formatting).
	lastRow := findLastRow(tbl)
	if lastRow == nil {
		return fmt.Errorf("%w: %d", errTableNoRows, tableIdx)
	}

	// Parse the comma-separated values.
	cellValues := strings.Split(values, ",")

	for i := range cellValues {
		cellValues[i] = strings.TrimSpace(cellValues[i])
	}

	// Build a new row by cloning the last row's structure.
	newRow := cloneRowStructure(lastRow, cellValues)
	tbl.AddChild(newRow)

	session.MarkDirty("word/document.xml")

	return nil
}

// UpdateTableCell updates a specific cell's text content.
// tableIdx is 0-based, row/col are 1-based (matching user mental model).
func UpdateTableCell(session *EditSession, tableIdx, row, col int, value string) error {
	if row < 1 {
		return fmt.Errorf("%w, got %d", errRowMinOne, row)
	}

	if col < 1 {
		return fmt.Errorf("%w, got %d", errColMinOne, col)
	}

	doc, err := session.Part("word/document.xml")
	if err != nil {
		return err
	}

	body := findBody(doc)
	if body == nil {
		return errNoBody
	}

	tbl := findTableByIndex(body, tableIdx)
	if tbl == nil {
		return fmt.Errorf("%w: %d", errTableOutOfRange, tableIdx)
	}

	// Find the target row (1-based).
	tr := findRowByIndex(tbl, row)
	if tr == nil {
		return fmt.Errorf("%w: row %d in table %d", errRowOutOfRange, row, tableIdx)
	}

	// Find the target cell (1-based).
	tc := findCellByIndex(tr, col)
	if tc == nil {
		return fmt.Errorf("%w: col %d in row %d of table %d", errColOutOfRange, col, row, tableIdx)
	}

	// Set the cell text. Find the first paragraph, or create one.
	setCellText(tc, value)

	session.MarkDirty("word/document.xml")

	return nil
}

// DeleteTableRow removes a row from a table.
// tableIdx is 0-based, row is 1-based.
func DeleteTableRow(session *EditSession, tableIdx, row int) error {
	if row < 1 {
		return fmt.Errorf("%w, got %d", errRowMinOne, row)
	}

	doc, err := session.Part("word/document.xml")
	if err != nil {
		return err
	}

	body := findBody(doc)
	if body == nil {
		return errNoBody
	}

	tbl := findTableByIndex(body, tableIdx)
	if tbl == nil {
		return fmt.Errorf("%w: %d", errTableOutOfRange, tableIdx)
	}

	tr := findRowByIndex(tbl, row)
	if tr == nil {
		return fmt.Errorf("%w: row %d in table %d", errRowOutOfRange, row, tableIdx)
	}

	tbl.RemoveChild(tr)

	session.MarkDirty("word/document.xml")

	return nil
}

// findTableByIndex returns the table at the given 0-based index.
func findTableByIndex(body *etree.Element, idx int) *etree.Element {
	count := 0

	for _, child := range body.ChildElements() {
		if child.Tag == tagTbl {
			if count == idx {
				return child
			}

			count++
		}
	}

	return nil
}

// findLastRow returns the last w:tr element in a table.
func findLastRow(tbl *etree.Element) *etree.Element {
	var last *etree.Element

	for _, child := range tbl.ChildElements() {
		if child.Tag == "tr" {
			last = child
		}
	}

	return last
}

// findRowByIndex returns the row at the given 1-based index.
func findRowByIndex(tbl *etree.Element, idx int) *etree.Element {
	count := 0

	for _, child := range tbl.ChildElements() {
		if child.Tag == "tr" {
			count++

			if count == idx {
				return child
			}
		}
	}

	return nil
}

// findCellByIndex returns the cell at the given 1-based column index.
func findCellByIndex(tr *etree.Element, idx int) *etree.Element {
	count := 0

	for _, child := range tr.ChildElements() {
		if child.Tag == "tc" {
			count++

			if count == idx {
				return child
			}
		}
	}

	return nil
}

// cloneRowStructure creates a new w:tr element modeled after the source row,
// but with new cell values. If fewer values are provided than the row has cells,
// remaining cells get empty text. Extra values create additional cells.
func cloneRowStructure(sourceRow *etree.Element, values []string) *etree.Element {
	newRow := etree.NewElement("w:tr")

	// Copy row properties if present.
	for _, child := range sourceRow.ChildElements() {
		if child.Tag == "trPr" {
			newRow.AddChild(child.Copy())

			break
		}
	}

	// Count source cells and collect their properties.
	var sourceCells []*etree.Element

	for _, child := range sourceRow.ChildElements() {
		if child.Tag == "tc" {
			sourceCells = append(sourceCells, child)
		}
	}

	// Create cells: match source cell count or value count, whichever is larger.
	cellCount := len(sourceCells)
	if len(values) > cellCount {
		cellCount = len(values)
	}

	for i := 0; i < cellCount; i++ {
		tc := etree.NewElement("w:tc")

		// Clone cell properties from the corresponding source cell if available.
		if i < len(sourceCells) {
			for _, child := range sourceCells[i].ChildElements() {
				if child.Tag == "tcPr" {
					tc.AddChild(child.Copy())

					break
				}
			}
		}

		// Create a paragraph with the value.
		p := tc.CreateElement("w:p")
		r := p.CreateElement("w:r")
		t := r.CreateElement("w:t")

		if i < len(values) {
			t.SetText(values[i])

			if len(values[i]) > 0 && (values[i][0] == ' ' || values[i][len(values[i])-1] == ' ') {
				t.CreateAttr("xml:space", "preserve")
			}
		}

		newRow.AddChild(tc)
	}

	return newRow
}

// setCellText sets the text content of a table cell. If the cell has paragraphs,
// the first paragraph's text is replaced. If it has no paragraphs, one is created.
func setCellText(tc *etree.Element, text string) {
	// Find the first paragraph in the cell.
	var firstP *etree.Element

	for _, child := range tc.ChildElements() {
		if child.Tag == "p" {
			firstP = child

			break
		}
	}

	if firstP == nil {
		// Create a paragraph.
		firstP = tc.CreateElement("w:p")
	}

	// Clear existing runs from the paragraph.
	var toRemove []*etree.Element

	for _, child := range firstP.ChildElements() {
		if child.Tag == "r" {
			toRemove = append(toRemove, child)
		}
	}

	for _, child := range toRemove {
		firstP.RemoveChild(child)
	}

	// Add a new run with the text.
	r := firstP.CreateElement("w:r")
	t := r.CreateElement("w:t")
	t.SetText(text)

	if len(text) > 0 && (text[0] == ' ' || text[len(text)-1] == ' ') {
		t.CreateAttr("xml:space", "preserve")
	}
}
