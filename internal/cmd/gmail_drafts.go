package cmd

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/steipete/gogcli/internal/config"
	"github.com/steipete/gogcli/internal/googleapi"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
	"google.golang.org/api/gmail/v1"
)

func newGmailDraftsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drafts",
		Short: "Manage drafts",
	}
	cmd.AddCommand(newGmailDraftsListCmd(flags))
	cmd.AddCommand(newGmailDraftsGetCmd(flags))
	cmd.AddCommand(newGmailDraftsDeleteCmd(flags))
	cmd.AddCommand(newGmailDraftsSendCmd(flags))
	cmd.AddCommand(newGmailDraftsCreateCmd(flags))
	return cmd
}

func newGmailDraftsListCmd(flags *rootFlags) *cobra.Command {
	var max int64
	var page string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List drafts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}

			svc, err := googleapi.NewGmail(cmd.Context(), account)
			if err != nil {
				return err
			}

			resp, err := svc.Users.Drafts.List("me").MaxResults(max).PageToken(page).Do()
			if err != nil {
				return err
			}
			if outfmt.IsJSON(cmd.Context()) {
				type item struct {
					ID        string `json:"id"`
					MessageID string `json:"messageId,omitempty"`
					ThreadID  string `json:"threadId,omitempty"`
				}
				items := make([]item, 0, len(resp.Drafts))
				for _, d := range resp.Drafts {
					if d == nil {
						continue
					}
					var msgID, threadID string
					if d.Message != nil {
						msgID = d.Message.Id
						threadID = d.Message.ThreadId
					}
					items = append(items, item{ID: d.Id, MessageID: msgID, ThreadID: threadID})
				}
				return outfmt.WriteJSON(os.Stdout, map[string]any{
					"drafts":        items,
					"nextPageToken": resp.NextPageToken,
				})
			}
			if len(resp.Drafts) == 0 {
				u.Err().Println("No drafts")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tMESSAGE_ID")
			for _, d := range resp.Drafts {
				msgID := ""
				if d.Message != nil {
					msgID = d.Message.Id
				}
				fmt.Fprintf(tw, "%s\t%s\n", d.Id, msgID)
			}
			_ = tw.Flush()

			if resp.NextPageToken != "" {
				u.Err().Printf("# Next page: --page %s", resp.NextPageToken)
			}
			return nil
		},
	}

	cmd.Flags().Int64Var(&max, "max", 20, "Max results")
	cmd.Flags().StringVar(&page, "page", "", "Page token")
	return cmd
}

func newGmailDraftsGetCmd(flags *rootFlags) *cobra.Command {
	var download bool

	cmd := &cobra.Command{
		Use:   "get <draftId>",
		Short: "Get draft details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}
			draftID := args[0]

			svc, err := googleapi.NewGmail(cmd.Context(), account)
			if err != nil {
				return err
			}

			draft, err := svc.Users.Drafts.Get("me", draftID).Format("full").Do()
			if err != nil {
				return err
			}
			if draft.Message == nil {
				if outfmt.IsJSON(cmd.Context()) {
					return outfmt.WriteJSON(os.Stdout, map[string]any{"draft": draft})
				}
				u.Err().Println("Empty draft")
				return nil
			}

			msg := draft.Message
			if outfmt.IsJSON(cmd.Context()) {
				out := map[string]any{"draft": draft}
				if download {
					attachDir, err := config.EnsureGmailAttachmentsDir()
					if err != nil {
						return err
					}
					type dl struct {
						MessageID    string `json:"messageId"`
						AttachmentID string `json:"attachmentId"`
						Filename     string `json:"filename"`
						Path         string `json:"path"`
						Cached       bool   `json:"cached"`
					}
					downloaded := make([]dl, 0)
					for _, a := range collectAttachments(msg.Payload) {
						outPath, cached, err := downloadAttachment(cmd, svc, msg.Id, a, attachDir)
						if err != nil {
							return err
						}
						downloaded = append(downloaded, dl{
							MessageID:    msg.Id,
							AttachmentID: a.AttachmentID,
							Filename:     a.Filename,
							Path:         outPath,
							Cached:       cached,
						})
					}
					out["downloaded"] = downloaded
				}
				return outfmt.WriteJSON(os.Stdout, out)
			}

			u.Out().Printf("Draft-ID: %s", draft.Id)
			u.Out().Printf("Message-ID: %s", msg.Id)
			u.Out().Printf("To: %s", headerValue(msg.Payload, "To"))
			u.Out().Printf("Cc: %s", headerValue(msg.Payload, "Cc"))
			u.Out().Printf("Subject: %s", headerValue(msg.Payload, "Subject"))
			u.Out().Println("")

			body := bestBodyText(msg.Payload)
			if body != "" {
				u.Out().Println(body)
				u.Out().Println("")
			}

			attachments := collectAttachments(msg.Payload)
			if len(attachments) > 0 {
				u.Out().Println("Attachments:")
				for _, a := range attachments {
					u.Out().Printf("  - %s (%d bytes)", a.Filename, a.Size)
				}
				u.Out().Println("")
			}

			if download && msg.Id != "" && len(attachments) > 0 {
				attachDir, err := config.EnsureGmailAttachmentsDir()
				if err != nil {
					return err
				}
				for _, a := range attachments {
					outPath, cached, err := downloadAttachment(cmd, svc, msg.Id, a, attachDir)
					if err != nil {
						return err
					}
					if cached {
						u.Out().Printf("Cached: %s", outPath)
					} else {
						u.Out().Successf("Saved: %s", outPath)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&download, "download", false, "Download draft attachments")
	return cmd
}

func newGmailDraftsDeleteCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <draftId>",
		Short: "Delete a draft",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}
			draftID := args[0]

			svc, err := googleapi.NewGmail(cmd.Context(), account)
			if err != nil {
				return err
			}

			if err := svc.Users.Drafts.Delete("me", draftID).Do(); err != nil {
				return err
			}
			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{"deleted": true, "draftId": draftID})
			}
			u.Out().Printf("deleted\ttrue")
			u.Out().Printf("draft_id\t%s", draftID)
			return nil
		},
	}
}

func newGmailDraftsSendCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "send <draftId>",
		Short: "Send a draft",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}
			draftID := args[0]

			svc, err := googleapi.NewGmail(cmd.Context(), account)
			if err != nil {
				return err
			}

			msg, err := svc.Users.Drafts.Send("me", &gmail.Draft{Id: draftID}).Do()
			if err != nil {
				return err
			}
			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{
					"messageId": msg.Id,
					"threadId":  msg.ThreadId,
				})
			}
			u.Out().Printf("message_id\t%s", msg.Id)
			if msg.ThreadId != "" {
				u.Out().Printf("thread_id\t%s", msg.ThreadId)
			}
			return nil
		},
	}
}

func newGmailDraftsCreateCmd(flags *rootFlags) *cobra.Command {
	var to string
	var cc string
	var bcc string
	var subject string
	var body string
	var replyTo string
	var attach []string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a draft",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}
			if strings.TrimSpace(to) == "" || strings.TrimSpace(subject) == "" || strings.TrimSpace(body) == "" {
				return errors.New("required: --to, --subject, --body")
			}

			svc, err := googleapi.NewGmail(cmd.Context(), account)
			if err != nil {
				return err
			}

			inReplyTo, references, threadID, err := replyHeaders(cmd, svc, replyTo)
			if err != nil {
				return err
			}

			atts := make([]mailAttachment, 0, len(attach))
			for _, p := range attach {
				atts = append(atts, mailAttachment{Path: p})
			}

			raw, err := buildRFC822(mailOptions{
				From:        account,
				To:          splitCSV(to),
				Cc:          splitCSV(cc),
				Bcc:         splitCSV(bcc),
				Subject:     subject,
				Body:        body,
				InReplyTo:   inReplyTo,
				References:  references,
				Attachments: atts,
			})
			if err != nil {
				return err
			}

			msg := &gmail.Message{
				Raw: base64.RawURLEncoding.EncodeToString(raw),
			}
			if threadID != "" {
				msg.ThreadId = threadID
			}

			draft, err := svc.Users.Drafts.Create("me", &gmail.Draft{Message: msg}).Do()
			if err != nil {
				return err
			}
			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{
					"draftId":  draft.Id,
					"message":  draft.Message,
					"threadId": threadID,
				})
			}
			u.Out().Printf("draft_id\t%s", draft.Id)
			if draft.Message != nil && draft.Message.Id != "" {
				u.Out().Printf("message_id\t%s", draft.Message.Id)
			}
			if threadID != "" {
				u.Out().Printf("thread_id\t%s", threadID)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&to, "to", "", "Recipients (comma-separated, required)")
	cmd.Flags().StringVar(&cc, "cc", "", "CC recipients (comma-separated)")
	cmd.Flags().StringVar(&bcc, "bcc", "", "BCC recipients (comma-separated)")
	cmd.Flags().StringVar(&subject, "subject", "", "Subject (required)")
	cmd.Flags().StringVar(&body, "body", "", "Body (required)")
	cmd.Flags().StringVar(&replyTo, "reply-to", "", "Reply to message ID (sets In-Reply-To/References and thread)")
	cmd.Flags().StringSliceVar(&attach, "attach", nil, "Attachment file path (repeatable)")
	return cmd
}
