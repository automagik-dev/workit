package officetext

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

const nsDrawingML = "http://schemas.openxmlformats.org/drawingml/2006/main"

// ErrNoSlides is returned when no slide files are found in a PPTX archive.
var ErrNoSlides = errors.New("pptx: no slide files found")

// extractPptx extracts plain text from PPTX file data.
// It parses the ZIP, finds ppt/slides/slide*.xml files, and extracts
// all <a:t> text nodes. Output is separated by "--- Slide N ---" headers.
func extractPptx(data []byte) (string, error) {
	zr, err := openZip(data)
	if err != nil {
		return "", fmt.Errorf("pptx: %w", err)
	}

	// Find all slide files, sorted for consistent ordering.
	var slideNames []string

	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			slideNames = append(slideNames, f.Name)
		}
	}

	sort.Strings(slideNames)

	if len(slideNames) == 0 {
		return "", ErrNoSlides
	}

	var result strings.Builder

	for i, name := range slideNames {
		slideData, slideErr := readZipEntry(zr, name)
		if slideErr != nil {
			return "", fmt.Errorf("pptx: %w", slideErr)
		}

		text := parseSlideXML(slideData)

		if i > 0 {
			result.WriteString("\n")
		}

		result.WriteString(fmt.Sprintf("--- Slide %d ---\n", i+1))
		result.WriteString(text)
	}

	return result.String(), nil
}

// parseSlideXML extracts text from a single slide's XML.
// It collects <a:t> text nodes, grouping them by <a:p> paragraphs.
func parseSlideXML(data []byte) string {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))

	var paragraphs []string
	var currentPara strings.Builder

	inParagraph := false
	inText := false

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
			if t.Name.Local == "p" && t.Name.Space == nsDrawingML {
				inParagraph = true

				currentPara.Reset()
			} else if t.Name.Local == "t" && t.Name.Space == nsDrawingML {
				inText = true
			}
		case xml.EndElement:
			if t.Name.Local == "t" && t.Name.Space == nsDrawingML {
				inText = false
			} else if t.Name.Local == "p" && t.Name.Space == nsDrawingML {
				if inParagraph {
					text := currentPara.String()
					if text != "" {
						paragraphs = append(paragraphs, text)
					}

					inParagraph = false
				}
			}
		case xml.CharData:
			if inText {
				currentPara.Write(t)
			}
		}
	}

	return strings.Join(paragraphs, "\n") + "\n"
}
