package cmd

import (
	"errors"
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

const contactsReadMask = "names,emailAddresses,phoneNumbers"

func newContactsListCmd(flags *rootFlags) *cobra.Command {
	var max int64
	var page string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List contacts",
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

			resp, err := svc.People.Connections.List("people/me").
				PersonFields(contactsReadMask).
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
				items := make([]item, 0, len(resp.Connections))
				for _, p := range resp.Connections {
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
			if len(resp.Connections) == 0 {
				u.Err().Println("No contacts")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "RESOURCE\tNAME\tEMAIL\tPHONE")
			for _, p := range resp.Connections {
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

func newContactsGetCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "get <resourceName|email>",
		Short: "Get a contact by resource name (people/...) or email",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}
			identifier := strings.TrimSpace(args[0])
			if identifier == "" {
				return errors.New("empty identifier")
			}

			svc, err := googleapi.NewPeople(cmd.Context(), account)
			if err != nil {
				return err
			}

			var p *people.Person
			if strings.HasPrefix(identifier, "people/") {
				p, err = svc.People.Get(identifier).PersonFields(contactsReadMask).Do()
				if err != nil {
					return err
				}
			} else {
				// Fallback: search and pick first match.
				resp, err := svc.People.SearchContacts().
					Query(identifier).
					PageSize(10).
					ReadMask(contactsReadMask).
					Do()
				if err != nil {
					return err
				}
				for _, r := range resp.Results {
					if r.Person == nil {
						continue
					}
					if strings.EqualFold(primaryEmail(r.Person), identifier) {
						p = r.Person
						break
					}
					if p == nil {
						p = r.Person
					}
				}
				if p == nil {
					if outfmt.IsJSON(cmd.Context()) {
						return outfmt.WriteJSON(os.Stdout, map[string]any{"found": false})
					}
					u.Err().Println("Not found")
					return nil
				}
			}

			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{"contact": p})
			}

			u.Out().Printf("resource\t%s", p.ResourceName)
			u.Out().Printf("name\t%s", primaryName(p))
			if e := primaryEmail(p); e != "" {
				u.Out().Printf("email\t%s", e)
			}
			if ph := primaryPhone(p); ph != "" {
				u.Out().Printf("phone\t%s", ph)
			}
			return nil
		},
	}
}

func newContactsCreateCmd(flags *rootFlags) *cobra.Command {
	var given string
	var family string
	var email string
	var phone string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new contact",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}
			if strings.TrimSpace(given) == "" {
				return errors.New("required: --given")
			}

			svc, err := googleapi.NewPeople(cmd.Context(), account)
			if err != nil {
				return err
			}

			p := &people.Person{
				Names: []*people.Name{{
					GivenName:  strings.TrimSpace(given),
					FamilyName: strings.TrimSpace(family),
				}},
			}
			if strings.TrimSpace(email) != "" {
				p.EmailAddresses = []*people.EmailAddress{{Value: strings.TrimSpace(email)}}
			}
			if strings.TrimSpace(phone) != "" {
				p.PhoneNumbers = []*people.PhoneNumber{{Value: strings.TrimSpace(phone)}}
			}

			created, err := svc.People.CreateContact(p).Do()
			if err != nil {
				return err
			}
			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{"contact": created})
			}
			u.Out().Printf("resource\t%s", created.ResourceName)
			return nil
		},
	}

	cmd.Flags().StringVar(&given, "given", "", "Given name (required)")
	cmd.Flags().StringVar(&family, "family", "", "Family name")
	cmd.Flags().StringVar(&email, "email", "", "Email address")
	cmd.Flags().StringVar(&phone, "phone", "", "Phone number")
	return cmd
}

func newContactsUpdateCmd(flags *rootFlags) *cobra.Command {
	var given string
	var family string
	var email string
	var phone string

	cmd := &cobra.Command{
		Use:   "update <resourceName>",
		Short: "Update an existing contact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}
			resourceName := strings.TrimSpace(args[0])
			if !strings.HasPrefix(resourceName, "people/") {
				return errors.New("resourceName must start with people/")
			}

			svc, err := googleapi.NewPeople(cmd.Context(), account)
			if err != nil {
				return err
			}

			existing, err := svc.People.Get(resourceName).PersonFields(contactsReadMask).Do()
			if err != nil {
				return err
			}

			updateFields := make([]string, 0, 3)

			if cmd.Flags().Changed("given") || cmd.Flags().Changed("family") {
				curGiven := ""
				curFamily := ""
				if len(existing.Names) > 0 && existing.Names[0] != nil {
					curGiven = existing.Names[0].GivenName
					curFamily = existing.Names[0].FamilyName
				}
				if cmd.Flags().Changed("given") {
					curGiven = strings.TrimSpace(given)
				}
				if cmd.Flags().Changed("family") {
					curFamily = strings.TrimSpace(family)
				}
				name := &people.Name{GivenName: curGiven, FamilyName: curFamily}
				existing.Names = []*people.Name{name}
				updateFields = append(updateFields, "names")
			}
			if cmd.Flags().Changed("email") {
				if strings.TrimSpace(email) == "" {
					existing.EmailAddresses = nil
				} else {
					existing.EmailAddresses = []*people.EmailAddress{{Value: strings.TrimSpace(email)}}
				}
				updateFields = append(updateFields, "emailAddresses")
			}
			if cmd.Flags().Changed("phone") {
				if strings.TrimSpace(phone) == "" {
					existing.PhoneNumbers = nil
				} else {
					existing.PhoneNumbers = []*people.PhoneNumber{{Value: strings.TrimSpace(phone)}}
				}
				updateFields = append(updateFields, "phoneNumbers")
			}

			if len(updateFields) == 0 {
				return errors.New("no updates provided")
			}

			updated, err := svc.People.UpdateContact(resourceName, existing).
				UpdatePersonFields(strings.Join(updateFields, ",")).
				Do()
			if err != nil {
				return err
			}
			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{"contact": updated})
			}
			u.Out().Printf("resource\t%s", updated.ResourceName)
			return nil
		},
	}

	cmd.Flags().StringVar(&given, "given", "", "Given name")
	cmd.Flags().StringVar(&family, "family", "", "Family name")
	cmd.Flags().StringVar(&email, "email", "", "Email address (empty clears)")
	cmd.Flags().StringVar(&phone, "phone", "", "Phone number (empty clears)")
	return cmd
}

func newContactsDeleteCmd(flags *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <resourceName>",
		Short: "Delete a contact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())
			account, err := requireAccount(flags)
			if err != nil {
				return err
			}
			resourceName := strings.TrimSpace(args[0])
			if !strings.HasPrefix(resourceName, "people/") {
				return errors.New("resourceName must start with people/")
			}

			svc, err := googleapi.NewPeople(cmd.Context(), account)
			if err != nil {
				return err
			}
			if _, err := svc.People.DeleteContact(resourceName).Do(); err != nil {
				return err
			}
			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{"deleted": true, "resource": resourceName})
			}
			u.Out().Printf("deleted\ttrue")
			u.Out().Printf("resource\t%s", resourceName)
			return nil
		},
	}
}
