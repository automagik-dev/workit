// Package docx provides read/write access to OOXML (.docx) documents.
//
// A DOCX file is a ZIP archive containing XML parts. The EditSession type
// opens a DOCX, lazily parses individual XML parts into etree Documents,
// caches them, and can save modifications atomically (write to temp, rename).
package docx

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/beevik/etree"
)

// Sentinel errors for session operations.
var errPartNotFound = errors.New("part not found in docx")

// xmlDoc holds both the raw bytes and parsed DOM for a single ZIP entry.
type xmlDoc struct {
	raw []byte          // original bytes from the ZIP
	doc *etree.Document // lazily parsed XML
}

// EditSession provides lazy, cached access to the XML parts of a DOCX file.
// It supports atomic save: modifications are written to a temp file and then
// renamed, so the source file is never partially overwritten.
type EditSession struct {
	path    string             // original DOCX file path
	zipFile *zip.ReadCloser    // opened ZIP handle
	parts   map[string]*xmlDoc // cached parsed XML DOMs (keyed by ZIP entry name)
	dirty   map[string]bool    // parts that have been modified
	rawData map[string][]byte  // raw bytes for all entries (read eagerly)
}

// Open opens a DOCX file and returns an EditSession.
// The ZIP handle remains open until Close is called.
func Open(path string) (*EditSession, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open docx %s: %w", path, err)
	}

	// Read all raw entries eagerly so we can copy them on Save.
	rawData := make(map[string][]byte, len(zr.File))

	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			_ = zr.Close()
			return nil, fmt.Errorf("open entry %s: %w", f.Name, err)
		}

		data, err := io.ReadAll(rc)
		_ = rc.Close()

		if err != nil {
			_ = zr.Close()
			return nil, fmt.Errorf("read entry %s: %w", f.Name, err)
		}

		rawData[f.Name] = data
	}

	return &EditSession{
		path:    path,
		zipFile: zr,
		parts:   make(map[string]*xmlDoc),
		dirty:   make(map[string]bool),
		rawData: rawData,
	}, nil
}

// Part returns the parsed XML document for a given part path
// (e.g. "word/document.xml"). It lazily parses on first access and caches
// the result for subsequent calls.
func (s *EditSession) Part(name string) (*etree.Document, error) {
	// Return cached parse if available.
	if xd, ok := s.parts[name]; ok && xd.doc != nil {
		return xd.doc, nil
	}

	raw, ok := s.rawData[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", errPartNotFound, name)
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(raw); err != nil {
		return nil, fmt.Errorf("parse xml %s: %w", name, err)
	}

	s.parts[name] = &xmlDoc{raw: raw, doc: doc}

	return doc, nil
}

// RawPart returns the raw bytes for a ZIP entry without parsing XML.
func (s *EditSession) RawPart(name string) ([]byte, error) {
	raw, ok := s.rawData[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", errPartNotFound, name)
	}

	return raw, nil
}

// MarkDirty marks a part as modified. Dirty parts are re-serialized from
// their etree Document when Save is called.
func (s *EditSession) MarkDirty(name string) {
	s.dirty[name] = true
}

// errDirtyPartNoParsed is returned when a dirty part has no parsed document.
var errDirtyPartNoParsed = errors.New("dirty part has no parsed document")

// Save writes the DOCX to outputPath. Unmodified entries are copied verbatim
// from the original ZIP. Dirty parts are serialized from their cached DOM.
func (s *EditSession) Save(outputPath string) error {
	// Write to a temp file first, then rename for atomicity.
	dir := filepath.Dir(outputPath)

	tmp, err := os.CreateTemp(dir, ".docx-save-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpPath := tmp.Name()

	zw := zip.NewWriter(tmp)

	// Collect entry names in a deterministic order.
	names := s.ListParts()

	for _, name := range names {
		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			_ = tmp.Close()
			_ = os.Remove(tmpPath)

			return fmt.Errorf("create zip entry %s: %w", name, err)
		}

		if s.dirty[name] {
			// Serialize the modified DOM.
			xd, ok := s.parts[name]
			if !ok || xd.doc == nil {
				_ = zw.Close()
				_ = tmp.Close()
				_ = os.Remove(tmpPath)

				return fmt.Errorf("%w: %s", errDirtyPartNoParsed, name)
			}

			xd.doc.Indent(2)

			b, err := xd.doc.WriteToBytes()
			if err != nil {
				_ = zw.Close()
				_ = tmp.Close()
				_ = os.Remove(tmpPath)

				return fmt.Errorf("serialize %s: %w", name, err)
			}

			if _, err := w.Write(b); err != nil {
				_ = zw.Close()
				_ = tmp.Close()
				_ = os.Remove(tmpPath)

				return fmt.Errorf("write %s: %w", name, err)
			}
		} else {
			// Copy unmodified entry verbatim.
			if _, err := w.Write(s.rawData[name]); err != nil {
				_ = zw.Close()
				_ = tmp.Close()
				_ = os.Remove(tmpPath)

				return fmt.Errorf("copy %s: %w", name, err)
			}
		}
	}

	if err := zw.Close(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)

		return fmt.Errorf("close zip writer: %w", err)
	}

	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf("close temp file: %w", err)
	}

	// Atomic rename.
	if err := os.Rename(tmpPath, outputPath); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf("rename to %s: %w", outputPath, err)
	}

	return nil
}

// SaveInPlace saves the modified DOCX back to its original path.
// Uses temp file + rename for atomicity.
func (s *EditSession) SaveInPlace() error {
	return s.Save(s.path)
}

// Close releases resources held by the EditSession.
func (s *EditSession) Close() error {
	if s.zipFile != nil {
		return fmt.Errorf("close zip: %w", s.zipFile.Close())
	}

	return nil
}

// ListParts returns all file paths in the DOCX ZIP, sorted alphabetically.
func (s *EditSession) ListParts() []string {
	names := make([]string, 0, len(s.rawData))

	for name := range s.rawData {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

// AddRawPart injects a new ZIP entry into the session with the given raw bytes.
// If the part already exists, it is overwritten. The part is also marked dirty
// so that subsequent Save calls serialize it from the cached DOM if parsed.
func (s *EditSession) AddRawPart(name string, data []byte) {
	s.rawData[name] = data
}

// Path returns the original file path of the DOCX.
func (s *EditSession) Path() string {
	return s.path
}
