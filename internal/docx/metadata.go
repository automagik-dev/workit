package docx

import (
	"strconv"
)

// DocumentMetadata holds core document properties extracted from docProps/core.xml
// and docProps/app.xml.
type DocumentMetadata struct {
	Title       string `json:"title,omitempty"`
	Author      string `json:"author,omitempty"`
	Description string `json:"description,omitempty"`
	Created     string `json:"created,omitempty"`
	Modified    string `json:"modified,omitempty"`
	Pages       int    `json:"pages,omitempty"`
}

// ReadMetadata extracts document properties from docProps/core.xml and
// docProps/app.xml. Missing or unparseable fields are left at their zero value.
func ReadMetadata(session *EditSession) (*DocumentMetadata, error) {
	meta := &DocumentMetadata{}

	// Extract from docProps/core.xml (Dublin Core metadata).
	if err := readCoreMeta(session, meta); err != nil {
		// core.xml is optional; ignore errors.
		_ = err
	}

	// Extract page count from docProps/app.xml.
	if err := readAppMeta(session, meta); err != nil {
		// app.xml is optional; ignore errors.
		_ = err
	}

	return meta, nil
}

func readCoreMeta(session *EditSession, meta *DocumentMetadata) error {
	doc, err := session.Part("docProps/core.xml")
	if err != nil {
		return err
	}

	root := doc.Root()
	if root == nil {
		return nil
	}

	// Dublin Core elements live under cp:coreProperties.
	// They use various namespaces: dc:title, dc:creator, dcterms:created, etc.
	// etree strips namespace prefixes and uses the local tag name, so we match
	// on the local name.
	for _, child := range root.ChildElements() {
		switch child.Tag {
		case "title":
			meta.Title = child.Text()
		case "creator":
			meta.Author = child.Text()
		case "description":
			meta.Description = child.Text()
		case "created":
			meta.Created = child.Text()
		case "modified":
			meta.Modified = child.Text()
		}
	}

	return nil
}

func readAppMeta(session *EditSession, meta *DocumentMetadata) error {
	doc, err := session.Part("docProps/app.xml")
	if err != nil {
		return err
	}

	root := doc.Root()
	if root == nil {
		return nil
	}

	for _, child := range root.ChildElements() {
		if child.Tag == "Pages" {
			if n, err := strconv.Atoi(child.Text()); err == nil {
				meta.Pages = n
			}
		}
	}

	return nil
}
