package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/steipete/gogcli/internal/googleapi"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

const directoryReadMask = "names,emailAddresses"

func newContactsDirectoryCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "directory",
		Short: "Google Workspace directory",
	}
	cmd.AddCommand(newContactsDirectoryListCmd(flags))
	cmd.AddCommand(newContactsDirectorySearchCmd(flags))
	return cmd
}

func newContactsDirectoryListCmd(flags *rootFlags) *cobra.Command {
	var max int64
	var page string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List people from the Workspace directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}

			svc, err := googleapi.NewPeople(cmd.Context(), account)
			if err != nil {
				return err
			}

			resp, err := svc.People.ListDirectoryPeople().
				ReadMask(directoryReadMask).
				PageSize(max).
				PageToken(page).
				Do()
			if err != nil {
				return err
			}
			if outfmt.IsJSON(cmd.Context()) {
				type item struct {
					Resource string `json:"resource"`
					Name     string `json:"name,omitempty"`
					Email    string `json:"email,omitempty"`
				}
				items := make([]item, 0, len(resp.People))
				for _, p := range resp.People {
					if p == nil {
						continue
					}
					items = append(items, item{
						Resource: p.ResourceName,
						Name:     primaryName(p),
						Email:    primaryEmail(p),
					})
				}
				return outfmt.WriteJSON(os.Stdout, map[string]any{
					"people":        items,
					"nextPageToken": resp.NextPageToken,
				})
			}

			if len(resp.People) == 0 {
				u.Err().Println("No results")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "RESOURCE\tNAME\tEMAIL")
			for _, p := range resp.People {
				if p == nil {
					continue
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\n",
					p.ResourceName,
					sanitizeTab(primaryName(p)),
					sanitizeTab(primaryEmail(p)),
				)
			}
			_ = tw.Flush()

			if resp.NextPageToken != "" {
				u.Err().Printf("# Next page: --page %s", resp.NextPageToken)
			}
			return nil
		},
	}

	cmd.Flags().Int64Var(&max, "max", 50, "Max results")
	cmd.Flags().StringVar(&page, "page", "", "Page token")
	return cmd
}

func newContactsDirectorySearchCmd(flags *rootFlags) *cobra.Command {
	var max int64
	var page string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search people in the Workspace directory",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}
			query := strings.Join(args, " ")

			svc, err := googleapi.NewPeople(cmd.Context(), account)
			if err != nil {
				return err
			}

			resp, err := svc.People.SearchDirectoryPeople().
				Query(query).
				ReadMask(directoryReadMask).
				PageSize(max).
				PageToken(page).
				Do()
			if err != nil {
				return err
			}
			if outfmt.IsJSON(cmd.Context()) {
				type item struct {
					Resource string `json:"resource"`
					Name     string `json:"name,omitempty"`
					Email    string `json:"email,omitempty"`
				}
				items := make([]item, 0, len(resp.People))
				for _, p := range resp.People {
					if p == nil {
						continue
					}
					items = append(items, item{
						Resource: p.ResourceName,
						Name:     primaryName(p),
						Email:    primaryEmail(p),
					})
				}
				return outfmt.WriteJSON(os.Stdout, map[string]any{
					"people":        items,
					"nextPageToken": resp.NextPageToken,
				})
			}

			if len(resp.People) == 0 {
				u.Err().Println("No results")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "RESOURCE\tNAME\tEMAIL")
			for _, p := range resp.People {
				if p == nil {
					continue
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\n",
					p.ResourceName,
					sanitizeTab(primaryName(p)),
					sanitizeTab(primaryEmail(p)),
				)
			}
			_ = tw.Flush()

			if resp.NextPageToken != "" {
				u.Err().Printf("# Next page: --page %s", resp.NextPageToken)
			}
			return nil
		},
	}

	cmd.Flags().Int64Var(&max, "max", 50, "Max results")
	cmd.Flags().StringVar(&page, "page", "", "Page token")
	return cmd
}

func newContactsOtherCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "other",
		Short: "Other contacts (people you've interacted with)",
	}
	cmd.AddCommand(newContactsOtherListCmd(flags))
	cmd.AddCommand(newContactsOtherSearchCmd(flags))
	return cmd
}

func newContactsOtherListCmd(flags *rootFlags) *cobra.Command {
	var max int64
	var page string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List other contacts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}

			svc, err := googleapi.NewPeople(cmd.Context(), account)
			if err != nil {
				return err
			}

			resp, err := svc.OtherContacts.List().
				ReadMask(contactsReadMask).
				PageSize(max).
				PageToken(page).
				Do()
			if err != nil {
				return err
			}
			if outfmt.IsJSON(cmd.Context()) {
				type item struct {
					Resource string `json:"resource"`
					Name     string `json:"name,omitempty"`
					Email    string `json:"email,omitempty"`
					Phone    string `json:"phone,omitempty"`
				}
				items := make([]item, 0, len(resp.OtherContacts))
				for _, p := range resp.OtherContacts {
					if p == nil {
						continue
					}
					items = append(items, item{
						Resource: p.ResourceName,
						Name:     primaryName(p),
						Email:    primaryEmail(p),
						Phone:    primaryPhone(p),
					})
				}
				return outfmt.WriteJSON(os.Stdout, map[string]any{
					"contacts":      items,
					"nextPageToken": resp.NextPageToken,
				})
			}

			if len(resp.OtherContacts) == 0 {
				u.Err().Println("No results")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "RESOURCE\tNAME\tEMAIL\tPHONE")
			for _, p := range resp.OtherContacts {
				if p == nil {
					continue
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
					p.ResourceName,
					sanitizeTab(primaryName(p)),
					sanitizeTab(primaryEmail(p)),
					sanitizeTab(primaryPhone(p)),
				)
			}
			_ = tw.Flush()

			if resp.NextPageToken != "" {
				u.Err().Printf("# Next page: --page %s", resp.NextPageToken)
			}
			return nil
		},
	}

	cmd.Flags().Int64Var(&max, "max", 100, "Max results")
	cmd.Flags().StringVar(&page, "page", "", "Page token")
	return cmd
}

func newContactsOtherSearchCmd(flags *rootFlags) *cobra.Command {
	var max int64

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search other contacts",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}
			query := strings.Join(args, " ")

			svc, err := googleapi.NewPeople(cmd.Context(), account)
			if err != nil {
				return err
			}

			resp, err := svc.OtherContacts.Search().
				Query(query).
				ReadMask(contactsReadMask).
				PageSize(max).
				Do()
			if err != nil {
				return err
			}
			if outfmt.IsJSON(cmd.Context()) {
				type item struct {
					Resource string `json:"resource"`
					Name     string `json:"name,omitempty"`
					Email    string `json:"email,omitempty"`
					Phone    string `json:"phone,omitempty"`
				}
				items := make([]item, 0, len(resp.Results))
				for _, r := range resp.Results {
					p := r.Person
					if p == nil {
						continue
					}
					items = append(items, item{
						Resource: p.ResourceName,
						Name:     primaryName(p),
						Email:    primaryEmail(p),
						Phone:    primaryPhone(p),
					})
				}
				return outfmt.WriteJSON(os.Stdout, map[string]any{"contacts": items})
			}

			if len(resp.Results) == 0 {
				u.Err().Println("No results")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "RESOURCE\tNAME\tEMAIL\tPHONE")
			for _, r := range resp.Results {
				p := r.Person
				if p == nil {
					continue
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
					p.ResourceName,
					sanitizeTab(primaryName(p)),
					sanitizeTab(primaryEmail(p)),
					sanitizeTab(primaryPhone(p)),
				)
			}
			_ = tw.Flush()
			return nil
		},
	}

	cmd.Flags().Int64Var(&max, "max", 50, "Max results")
	return cmd
}
