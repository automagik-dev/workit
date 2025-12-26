package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

func newDocsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Google Docs (export via Drive)",
	}
	cmd.AddCommand(newDocsExportCmd(flags))
	return cmd
}

func newDocsExportCmd(flags *rootFlags) *cobra.Command {
	var outPathFlag string
	var format string

	cmd := &cobra.Command{
		Use:   "export <docId>",
		Short: "Export a Google Doc (pdf|docx|txt)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}

			docID := strings.TrimSpace(args[0])
			if docID == "" {
				return usage("empty docId")
			}

			svc, err := newDriveService(cmd.Context(), account)
			if err != nil {
				return err
			}

			meta, err := svc.Files.Get(docID).
				SupportsAllDrives(true).
				Fields("id, name, mimeType").
				Context(cmd.Context()).
				Do()
			if err != nil {
				return err
			}
			if meta == nil {
				return errors.New("file not found")
			}
			if meta.MimeType != "application/vnd.google-apps.document" {
				return fmt.Errorf("file is not a Google Doc (mimeType=%q)", meta.MimeType)
			}

			destPath, err := resolveDriveDownloadDestPath(meta, outPathFlag)
			if err != nil {
				return err
			}

			format = strings.TrimSpace(format)
			if format == "" {
				format = "pdf"
			}

			downloadedPath, size, err := downloadDriveFile(cmd.Context(), svc, meta, destPath, format)
			if err != nil {
				return err
			}

			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{"path": downloadedPath, "size": size})
			}
			u.Out().Printf("path\t%s", downloadedPath)
			u.Out().Printf("size\t%s", formatDriveSize(size))
			return nil
		},
	}

	cmd.Flags().StringVar(&outPathFlag, "out", "", "Output file path (default: gogcli config dir)")
	cmd.Flags().StringVar(&format, "format", "pdf", "Export format: pdf|docx|txt")
	return cmd
}
