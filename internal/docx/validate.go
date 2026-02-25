package docx

import (
	"archive/zip"
	"errors"
	"fmt"
	"strings"

	"github.com/beevik/etree"
)

// Sentinel errors for validation.
var errEntryNotFound = errors.New("entry not found")

// ValidateResult contains the validation outcome.
type ValidateResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
	Skipped  bool     `json:"skipped,omitempty"` // true if external validator unavailable
}

// requiredParts lists the parts that must exist in a valid DOCX.
var requiredParts = []string{
	"[Content_Types].xml",
	"word/document.xml",
	"_rels/.rels",
}

// Validate checks a DOCX file for structural correctness.
// Performs Go-native checks:
//  1. ZIP is valid
//  2. Required parts exist ([Content_Types].xml, word/document.xml, _rels/.rels)
//  3. Content types reference actual parts
//  4. document.xml parses as valid XML
//  5. Basic OOXML structure (w:body exists, paragraphs have valid structure)
func Validate(path string) (*ValidateResult, error) {
	result := &ValidateResult{Valid: true}

	// 1. Check ZIP is valid.
	zr, err := zip.OpenReader(path)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("not a valid ZIP archive: %v", err))

		return result, nil
	}
	defer zr.Close()

	// Collect all entry names.
	entryNames := make(map[string]bool, len(zr.File))

	for _, f := range zr.File {
		entryNames[f.Name] = true
	}

	// 2. Check required parts exist.
	for _, rp := range requiredParts {
		if !entryNames[rp] {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("missing required part: %s", rp))
		}
	}

	// If critical parts are missing, we cannot continue deeper checks.
	if !result.Valid {
		return result, nil
	}

	// 3. Check content types reference actual parts.
	ctErrors := validateContentTypes(zr, entryNames)
	if len(ctErrors) > 0 {
		result.Warnings = append(result.Warnings, ctErrors...)
	}

	// 4. Parse document.xml as valid XML.
	docData, err := readZipEntry(zr, "word/document.xml")
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("cannot read word/document.xml: %v", err))

		return result, nil
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(docData); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("word/document.xml is not valid XML: %v", err))

		return result, nil
	}

	// 5. Basic OOXML structure checks.
	root := doc.Root()
	if root == nil {
		result.Valid = false
		result.Errors = append(result.Errors, "word/document.xml has no root element")

		return result, nil
	}

	if root.Tag != "document" {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("root element is <%s>, expected <w:document>", root.Tag))
	}

	body := findBody(doc)
	if body == nil {
		result.Valid = false
		result.Errors = append(result.Errors, "no w:body element found in document.xml")
	} else {
		// Check paragraphs have valid structure.
		structErrors := validateBodyStructure(body)
		if len(structErrors) > 0 {
			result.Warnings = append(result.Warnings, structErrors...)
		}
	}

	return result, nil
}

// ValidateDiff compares validation results between original and edited DOCX.
// Returns only NEW errors (present in edited but not in original).
func ValidateDiff(originalPath, editedPath string) (*ValidateResult, error) {
	origResult, err := Validate(originalPath)
	if err != nil {
		return nil, fmt.Errorf("validate original: %w", err)
	}

	editResult, err := Validate(editedPath)
	if err != nil {
		return nil, fmt.Errorf("validate edited: %w", err)
	}

	// Build sets of original errors and warnings.
	origErrors := make(map[string]bool, len(origResult.Errors))

	for _, e := range origResult.Errors {
		origErrors[e] = true
	}

	origWarnings := make(map[string]bool, len(origResult.Warnings))

	for _, w := range origResult.Warnings {
		origWarnings[w] = true
	}

	// Filter to only new errors/warnings.
	diff := &ValidateResult{Valid: true}

	for _, e := range editResult.Errors {
		if !origErrors[e] {
			diff.Valid = false
			diff.Errors = append(diff.Errors, e)
		}
	}

	for _, w := range editResult.Warnings {
		if !origWarnings[w] {
			diff.Warnings = append(diff.Warnings, w)
		}
	}

	return diff, nil
}

// readZipEntry reads the raw bytes of a named entry from an opened zip reader.
func readZipEntry(zr *zip.ReadCloser, name string) ([]byte, error) {
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("open zip entry %s: %w", name, err)
			}
			defer rc.Close()

			buf := make([]byte, 0, f.UncompressedSize64)

			for {
				tmp := make([]byte, 4096)
				n, readErr := rc.Read(tmp)
				buf = append(buf, tmp[:n]...)

				if readErr != nil {
					break
				}
			}

			return buf, nil
		}
	}

	return nil, fmt.Errorf("%w: %q", errEntryNotFound, name)
}

// validateContentTypes checks that Override entries in [Content_Types].xml
// reference parts that actually exist in the ZIP.
func validateContentTypes(zr *zip.ReadCloser, entryNames map[string]bool) []string {
	data, err := readZipEntry(zr, "[Content_Types].xml")
	if err != nil {
		return []string{fmt.Sprintf("cannot read [Content_Types].xml: %v", err)}
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(data); err != nil {
		return []string{fmt.Sprintf("[Content_Types].xml is not valid XML: %v", err)}
	}

	root := doc.Root()
	if root == nil {
		return []string{"[Content_Types].xml has no root element"}
	}

	var warnings []string

	for _, child := range root.ChildElements() {
		if child.Tag != "Override" {
			continue
		}

		partName := ""
		if a := child.SelectAttr("PartName"); a != nil {
			partName = a.Value
		}

		if partName == "" {
			continue
		}

		// PartName starts with "/", strip it for ZIP entry comparison.
		zipName := strings.TrimPrefix(partName, "/")
		if !entryNames[zipName] {
			warnings = append(warnings, fmt.Sprintf("content type override references missing part: %s", partName))
		}
	}

	return warnings
}

// validateBodyStructure checks paragraphs and tables for basic structure.
func validateBodyStructure(body *etree.Element) []string {
	var warnings []string

	pIdx := 0

	for _, child := range body.ChildElements() {
		switch child.Tag {
		case "p":
			// Check that runs contain text elements.
			for _, r := range child.ChildElements() {
				if r.Tag == "r" {
					hasContent := false

					for _, rc := range r.ChildElements() {
						if rc.Tag == "t" || rc.Tag == "br" || rc.Tag == "tab" ||
							rc.Tag == "drawing" || rc.Tag == "rPr" ||
							rc.Tag == "commentReference" || rc.Tag == "footnoteReference" ||
							rc.Tag == "endnoteReference" {
							hasContent = true

							break
						}
					}

					if !hasContent && len(r.ChildElements()) > 0 {
						// Only warn if the run has children but none are recognized content.
						warnings = append(warnings, fmt.Sprintf("paragraph %d: run has no recognized content elements", pIdx))
					}
				}
			}

			pIdx++
		case tagTbl:
			// Check table has rows.
			hasRows := false

			for _, tr := range child.ChildElements() {
				if tr.Tag == "tr" {
					hasRows = true

					break
				}
			}

			if !hasRows {
				warnings = append(warnings, "table has no rows")
			}
		}
	}

	return warnings
}
