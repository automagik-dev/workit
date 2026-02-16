package officetext

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// ErrNoWorksheets is returned when no worksheet files are found in an XLSX archive.
var ErrNoWorksheets = errors.New("xlsx: no worksheet files found")

// extractXlsx extracts plain text from XLSX file data.
// It parses the ZIP, reads xl/sharedStrings.xml to build a shared strings table,
// then walks xl/worksheets/sheet*.xml files to produce tab-separated output.
func extractXlsx(data []byte) (string, error) {
	zr, err := openZip(data)
	if err != nil {
		return "", fmt.Errorf("xlsx: %w", err)
	}

	// Build shared strings table (may not exist in all XLSX files).
	var sharedStrings []string

	if ssData, ssErr := readZipEntry(zr, "xl/sharedStrings.xml"); ssErr == nil {
		sharedStrings = parseSharedStrings(ssData)
	}

	// Find all worksheet files, sorted by name for consistent ordering.
	var sheetNames []string

	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "xl/worksheets/sheet") && strings.HasSuffix(f.Name, ".xml") {
			sheetNames = append(sheetNames, f.Name)
		}
	}

	sort.Strings(sheetNames)

	if len(sheetNames) == 0 {
		return "", ErrNoWorksheets
	}

	multiSheet := len(sheetNames) > 1
	var result strings.Builder

	for i, name := range sheetNames {
		wsData, wsErr := readZipEntry(zr, name)
		if wsErr != nil {
			return "", fmt.Errorf("xlsx: %w", wsErr)
		}

		rows := parseWorksheet(wsData, sharedStrings)

		if multiSheet {
			if i > 0 {
				result.WriteString("\n")
			}

			result.WriteString(fmt.Sprintf("Sheet %d:\n", i+1))
		}

		for _, row := range rows {
			result.WriteString(strings.Join(row, "\t"))
			result.WriteString("\n")
		}
	}

	return result.String(), nil
}

// parseSharedStrings parses xl/sharedStrings.xml and returns a slice of strings.
func parseSharedStrings(data []byte) []string {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))

	var result []string
	var currentText strings.Builder

	inSI := false
	inT := false

	for {
		tok, tokenErr := decoder.Token()
		if tokenErr != nil {
			break // EOF or parse error; return what we have.
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "si" {
				inSI = true

				currentText.Reset()
			} else if t.Name.Local == "t" && inSI {
				inT = true
			}
		case xml.EndElement:
			if t.Name.Local == "t" && inT {
				inT = false
			} else if t.Name.Local == "si" && inSI {
				result = append(result, currentText.String())
				inSI = false
			}
		case xml.CharData:
			if inT {
				currentText.Write(t)
			}
		}
	}

	return result
}

// parseWorksheet parses a worksheet XML and returns rows of cell values.
func parseWorksheet(data []byte, sharedStrings []string) [][]string {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))

	var rows [][]string
	var currentRow []string

	inRow := false
	inCell := false
	inValue := false
	cellType := ""

	for {
		tok, tokenErr := decoder.Token()
		if tokenErr != nil {
			if !errors.Is(tokenErr, io.EOF) {
				break // parse error; return what we have
			}

			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "row":
				inRow = true
				currentRow = nil
			case "c":
				if inRow {
					inCell = true
					cellType = ""

					for _, attr := range t.Attr {
						if attr.Name.Local == "t" {
							cellType = attr.Value
						}
					}
				}
			case "v":
				if inCell {
					inValue = true
				}
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "v":
				inValue = false
			case "c":
				inCell = false
			case "row":
				if inRow && len(currentRow) > 0 {
					rows = append(rows, currentRow)
				}

				inRow = false
			}
		case xml.CharData:
			if inValue {
				val := string(t)

				if cellType == "s" {
					// Shared string reference
					idx, convErr := strconv.Atoi(strings.TrimSpace(val))
					if convErr == nil && idx >= 0 && idx < len(sharedStrings) {
						currentRow = append(currentRow, sharedStrings[idx])
					} else {
						currentRow = append(currentRow, val)
					}
				} else {
					currentRow = append(currentRow, val)
				}
			}
		}
	}

	return rows
}
