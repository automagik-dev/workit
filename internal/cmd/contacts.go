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
	"google.golang.org/api/people/v1"
)

func newContactsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contacts",
		Short: "Google Contacts (People API)",
	}
	cmd.AddCommand(newContactsSearchCmd(flags))
	cmd.AddCommand(newContactsListCmd(flags))
	cmd.AddCommand(newContactsGetCmd(flags))
	cmd.AddCommand(newContactsCreateCmd(flags))
	cmd.AddCommand(newContactsUpdateCmd(flags))
	cmd.AddCommand(newContactsDeleteCmd(flags))
	cmd.AddCommand(newContactsDirectoryCmd(flags))
	cmd.AddCommand(newContactsOtherCmd(flags))
	return cmd
}

func newContactsSearchCmd(flags *rootFlags) *cobra.Command {
	var max int64

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search contacts by name/email/phone",
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

			resp, err := svc.People.SearchContacts().
				Query(query).
				PageSize(max).
				ReadMask("names,emailAddresses,phoneNumbers").
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
				fmt.Fprintf(
					tw,
					"%s\t%s\t%s\t%s\n",
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

func primaryName(p *people.Person) string {
	if p == nil || len(p.Names) == 0 || p.Names[0] == nil {
		return ""
	}
	if p.Names[0].DisplayName != "" {
		return p.Names[0].DisplayName
	}
	return strings.TrimSpace(strings.Join([]string{p.Names[0].GivenName, p.Names[0].FamilyName}, " "))
}

func primaryEmail(p *people.Person) string {
	if p == nil || len(p.EmailAddresses) == 0 || p.EmailAddresses[0] == nil {
		return ""
	}
	return p.EmailAddresses[0].Value
}

func primaryPhone(p *people.Person) string {
	if p == nil || len(p.PhoneNumbers) == 0 || p.PhoneNumbers[0] == nil {
		return ""
	}
	return p.PhoneNumbers[0].Value
}
