package cmd

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type mailAttachment struct {
	Path     string
	Filename string
	MIMEType string
	Data     []byte
}

type mailOptions struct {
	From              string
	To                []string
	Cc                []string
	Bcc               []string
	Subject           string
	Body              string
	InReplyTo         string
	References        string
	AdditionalHeaders map[string]string
	Attachments       []mailAttachment
}

func buildRFC822(opts mailOptions) ([]byte, error) {
	if strings.TrimSpace(opts.From) == "" {
		return nil, errors.New("missing From")
	}
	if len(opts.To) == 0 {
		return nil, errors.New("missing To")
	}
	if strings.TrimSpace(opts.Subject) == "" {
		return nil, errors.New("missing Subject")
	}

	var b bytes.Buffer

	if err := validateHeaderValue(opts.From); err != nil {
		return nil, fmt.Errorf("invalid From: %w", err)
	}
	for _, a := range append(append([]string{}, opts.To...), append(opts.Cc, opts.Bcc...)...) {
		if err := validateHeaderValue(a); err != nil {
			return nil, fmt.Errorf("invalid address: %w", err)
		}
	}

	writeHeader(&b, "From", opts.From)
	writeHeader(&b, "To", strings.Join(opts.To, ", "))
	if len(opts.Cc) > 0 {
		writeHeader(&b, "Cc", strings.Join(opts.Cc, ", "))
	}
	if len(opts.Bcc) > 0 {
		writeHeader(&b, "Bcc", strings.Join(opts.Bcc, ", "))
	}
	if err := validateHeaderValue(opts.Subject); err != nil {
		return nil, fmt.Errorf("invalid Subject: %w", err)
	}
	writeHeader(&b, "Subject", encodeHeaderIfNeeded(opts.Subject))
	writeHeader(&b, "Date", time.Now().Format(time.RFC1123Z))
	writeHeader(&b, "MIME-Version", "1.0")
	if strings.TrimSpace(opts.InReplyTo) != "" {
		if err := validateHeaderValue(opts.InReplyTo); err != nil {
			return nil, fmt.Errorf("invalid In-Reply-To: %w", err)
		}
		writeHeader(&b, "In-Reply-To", strings.TrimSpace(opts.InReplyTo))
	}
	if strings.TrimSpace(opts.References) != "" {
		if err := validateHeaderValue(opts.References); err != nil {
			return nil, fmt.Errorf("invalid References: %w", err)
		}
		writeHeader(&b, "References", strings.TrimSpace(opts.References))
	}
	for k, v := range opts.AdditionalHeaders {
		if strings.TrimSpace(k) != "" && strings.TrimSpace(v) != "" {
			if err := validateHeaderValue(v); err != nil {
				return nil, fmt.Errorf("invalid header %s: %w", k, err)
			}
			writeHeader(&b, k, v)
		}
	}

	if len(opts.Attachments) == 0 {
		writeHeader(&b, "Content-Type", "text/plain; charset=\"utf-8\"")
		writeHeader(&b, "Content-Transfer-Encoding", "7bit")
		b.WriteString("\r\n")
		b.WriteString(opts.Body)
		if !strings.HasSuffix(opts.Body, "\r\n") {
			b.WriteString("\r\n")
		}
		return b.Bytes(), nil
	}

	boundary, err := randomBoundary()
	if err != nil {
		return nil, err
	}

	writeHeader(&b, "Content-Type", fmt.Sprintf("multipart/mixed; boundary=%q", boundary))
	b.WriteString("\r\n")

	// Body part
	b.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	b.WriteString("Content-Transfer-Encoding: 7bit\r\n\r\n")
	b.WriteString(opts.Body)
	if !strings.HasSuffix(opts.Body, "\r\n") {
		b.WriteString("\r\n")
	}

	// Attachments
	for _, a := range opts.Attachments {
		if a.Filename == "" {
			a.Filename = filepath.Base(a.Path)
		}
		if a.MIMEType == "" {
			a.MIMEType = mime.TypeByExtension(strings.ToLower(filepath.Ext(a.Filename)))
			if a.MIMEType == "" {
				a.MIMEType = "application/octet-stream"
			}
		}
		if len(a.Data) == 0 {
			data, err := os.ReadFile(a.Path)
			if err != nil {
				return nil, err
			}
			a.Data = data
		}

		b.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
		b.WriteString(fmt.Sprintf("Content-Type: %s\r\n", a.MIMEType))
		b.WriteString("Content-Transfer-Encoding: base64\r\n")
		b.WriteString(fmt.Sprintf("Content-Disposition: attachment; %s\r\n\r\n", contentDispositionFilename(a.Filename)))
		b.WriteString(wrapBase64(a.Data))
		b.WriteString("\r\n")
	}

	b.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	return b.Bytes(), nil
}

func writeHeader(b *bytes.Buffer, name, value string) {
	b.WriteString(name)
	b.WriteString(": ")
	b.WriteString(value)
	b.WriteString("\r\n")
}

func wrapBase64(b []byte) string {
	s := base64.StdEncoding.EncodeToString(b)
	const width = 76
	var out strings.Builder
	for len(s) > width {
		out.WriteString(s[:width])
		out.WriteString("\r\n")
		s = s[width:]
	}
	if len(s) > 0 {
		out.WriteString(s)
	}
	return out.String()
}

func randomBoundary() (string, error) {
	var b [18]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "gogcli_" + base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func validateHeaderValue(v string) error {
	if strings.Contains(v, "\r") || strings.Contains(v, "\n") {
		return errors.New("header value contains newline")
	}
	return nil
}

func encodeHeaderIfNeeded(v string) string {
	if isASCII(v) {
		return v
	}
	return mime.QEncoding.Encode("utf-8", v)
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func contentDispositionFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return `filename="attachment"`
	}
	if isASCII(filename) {
		return fmt.Sprintf("filename=%q", filename)
	}
	// RFC 5987 / RFC 2231 style.
	return "filename*=UTF-8''" + rfc5987Encode(filename)
}

func rfc5987Encode(s string) string {
	// url.QueryEscape uses '+' for spaces; RFC 5987 wants %20.
	esc := url.QueryEscape(s)
	return strings.ReplaceAll(esc, "+", "%20")
}
