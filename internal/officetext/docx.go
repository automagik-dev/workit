package officetext

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// extractDocx extracts plain text from DOCX file data.
// It parses the ZIP, finds word/document.xml, and extracts all <w:t> text nodes,
// preserving paragraph breaks as newlines.
func extractDocx(data []byte) (string, error) {
	zr, err := openZip(data)
	if err != nil {
		return "", fmt.Errorf("docx: %w", err)
	}

	content, err := readZipEntry(zr, "word/document.xml")
	if err != nil {
		return "", fmt.Errorf("docx: %w", err)
	}

	return parseDocxXML(content)
}

// parseDocxXML walks the XML from word/document.xml and extracts text.
// It collects <w:t> text within each <w:p> paragraph, joining paragraphs
// with newlines.
func parseDocxXML(data []byte) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))

	var paragraphs []string
	var currentPara strings.Builder

	inParagraph := false
	inText := false

	for {
		tok, tokenErr := decoder.Token()
		if tokenErr != nil {
			if errors.Is(tokenErr, io.EOF) {
				break
			}

			return "", fmt.Errorf("docx: parse xml: %w", tokenErr)
		}

		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "p" && t.Name.Space == nsWordprocessingML {
				inParagraph = true

				currentPara.Reset()
			} else if t.Name.Local == "t" && t.Name.Space == nsWordprocessingML {
				inText = true
			}
		case xml.EndElement:
			if t.Name.Local == "t" && t.Name.Space == nsWordprocessingML {
				inText = false
			} else if t.Name.Local == "p" && t.Name.Space == nsWordprocessingML {
				if inParagraph {
					paragraphs = append(paragraphs, currentPara.String())
					inParagraph = false
				}
			}
		case xml.CharData:
			if inText {
				currentPara.Write(t)
			}
		}
	}

	return strings.Join(paragraphs, "\n"), nil
}

const nsWordprocessingML = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
