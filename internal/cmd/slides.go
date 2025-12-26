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

func newSlidesCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slides",
		Short: "Google Slides (export via Drive)",
	}
	cmd.AddCommand(newSlidesExportCmd(flags))
	return cmd
}

func newSlidesExportCmd(flags *rootFlags) *cobra.Command {
	var outPathFlag string
	var format string

	cmd := &cobra.Command{
		Use:   "export <presentationId>",
		Short: "Export a Google Slides deck (pdf|pptx)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}

			presentationID := strings.TrimSpace(args[0])
			if presentationID == "" {
				return usage("empty presentationId")
			}

			svc, err := newDriveService(cmd.Context(), account)
			if err != nil {
				return err
			}

			meta, err := svc.Files.Get(presentationID).
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
			if meta.MimeType != "application/vnd.google-apps.presentation" {
				return fmt.Errorf("file is not a Google Slides presentation (mimeType=%q)", meta.MimeType)
			}

			destPath, err := resolveDriveDownloadDestPath(meta, outPathFlag)
			if err != nil {
				return err
			}

			format = strings.TrimSpace(format)
			if format == "" {
				format = "pptx"
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
	cmd.Flags().StringVar(&format, "format", "pptx", "Export format: pdf|pptx")
	return cmd
}
