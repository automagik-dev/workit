package cmd

import (
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steipete/gogcli/internal/config"
	"github.com/steipete/gogcli/internal/googleauth"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/secrets"
	"github.com/steipete/gogcli/internal/ui"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication and accounts",
	}

	cmd.AddCommand(newAuthCredentialsCmd())
	cmd.AddCommand(newAuthAddCmd())
	cmd.AddCommand(newAuthListCmd())
	cmd.AddCommand(newAuthRemoveCmd())
	cmd.AddCommand(newAuthTokensCmd())
	return cmd
}

func newAuthCredentialsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "credentials <credentials.json>",
		Short: "Store OAuth client credentials",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())
			inPath := args[0]
			b, err := os.ReadFile(inPath)
			if err != nil {
				return err
			}

			creds, err := config.ParseGoogleOAuthClientJSON(b)
			if err != nil {
				return err
			}

			if err := config.WriteClientCredentials(creds); err != nil {
				return err
			}

			outPath, _ := config.ClientCredentialsPath()
			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{
					"saved": true,
					"path":  outPath,
				})
			}
			u.Out().Printf("path\t%s", outPath)
			return nil
		},
	}
}

func newAuthTokensCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tokens",
		Short: "Manage stored refresh tokens",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List stored tokens (by key only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			u := ui.FromContext(cmd.Context())
			store, err := secrets.OpenDefault()
			if err != nil {
				return err
			}
			keys, err := store.Keys()
			if err != nil {
				return err
			}
			if len(keys) == 0 {
				if outfmt.IsJSON(cmd.Context()) {
					return outfmt.WriteJSON(os.Stdout, map[string]any{"keys": []string{}})
				}
				u.Err().Println("No tokens stored")
				return nil
			}
			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{"keys": keys})
			}
			for _, k := range keys {
				u.Out().Println(k)
			}
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <email>",
		Short: "Delete a stored refresh token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())
			store, err := secrets.OpenDefault()
			if err != nil {
				return err
			}
			email := args[0]
			if err := store.DeleteToken(email); err != nil {
				return err
			}
			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{
					"deleted": true,
					"email":   email,
				})
			}
			u.Out().Printf("deleted\ttrue")
			u.Out().Printf("email\t%s", email)
			return nil
		},
	})

	return cmd
}

func newAuthAddCmd() *cobra.Command {
	var manual bool
	var forceConsent bool
	var servicesCSV string

	cmd := &cobra.Command{
		Use:   "add <email>",
		Short: "Authorize and store a refresh token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())

			email := args[0]

			var services []googleauth.Service
			if strings.EqualFold(strings.TrimSpace(servicesCSV), "") || strings.EqualFold(strings.TrimSpace(servicesCSV), "all") {
				services = googleauth.AllServices()
			} else {
				parts := strings.Split(servicesCSV, ",")
				seen := make(map[googleauth.Service]struct{})
				for _, p := range parts {
					svc, err := googleauth.ParseService(p)
					if err != nil {
						return err
					}
					if _, ok := seen[svc]; ok {
						continue
					}
					seen[svc] = struct{}{}
					services = append(services, svc)
				}
			}
			if len(services) == 0 {
				return cmd.Help()
			}

			scopes, err := googleauth.ScopesForServices(services)
			if err != nil {
				return err
			}

			refreshToken, err := googleauth.Authorize(cmd.Context(), googleauth.AuthorizeOptions{
				Services:     services,
				Scopes:       scopes,
				Manual:       manual,
				ForceConsent: forceConsent,
			})
			if err != nil {
				return err
			}

			store, err := secrets.OpenDefault()
			if err != nil {
				return err
			}
			serviceNames := make([]string, 0, len(services))
			for _, svc := range services {
				serviceNames = append(serviceNames, string(svc))
			}
			sort.Strings(serviceNames)

			if err := store.SetToken(email, secrets.Token{
				Email:        email,
				Services:     serviceNames,
				Scopes:       scopes,
				RefreshToken: refreshToken,
			}); err != nil {
				return err
			}
			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{
					"stored":   true,
					"email":    email,
					"services": serviceNames,
				})
			}
			u.Out().Printf("email\t%s", email)
			u.Out().Printf("services\t%s", strings.Join(serviceNames, ","))
			return nil
		},
	}

	cmd.Flags().BoolVar(&manual, "manual", false, "Browserless auth flow (paste redirect URL)")
	cmd.Flags().BoolVar(&forceConsent, "force-consent", false, "Force consent screen to obtain a refresh token")
	cmd.Flags().StringVar(&servicesCSV, "services", "all", "Services to authorize: all or comma-separated gmail,calendar,drive,contacts")
	return cmd
}

func newAuthListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List stored accounts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			u := ui.FromContext(cmd.Context())
			store, err := secrets.OpenDefault()
			if err != nil {
				return err
			}
			tokens, err := store.ListTokens()
			if err != nil {
				return err
			}
			sort.Slice(tokens, func(i, j int) bool { return tokens[i].Email < tokens[j].Email })
			if outfmt.IsJSON(cmd.Context()) {
				type item struct {
					Email     string   `json:"email"`
					Services  []string `json:"services,omitempty"`
					Scopes    []string `json:"scopes,omitempty"`
					CreatedAt string   `json:"created_at,omitempty"`
				}
				out := make([]item, 0, len(tokens))
				for _, t := range tokens {
					created := ""
					if !t.CreatedAt.IsZero() {
						created = t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
					}
					out = append(out, item{
						Email:     t.Email,
						Services:  t.Services,
						Scopes:    t.Scopes,
						CreatedAt: created,
					})
				}
				return outfmt.WriteJSON(os.Stdout, map[string]any{"accounts": out})
			}
			if len(tokens) == 0 {
				u.Err().Println("No tokens stored")
				return nil
			}
			for _, t := range tokens {
				created := ""
				if !t.CreatedAt.IsZero() {
					created = t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
				}
				u.Out().Printf("%s\t%s\t%s", t.Email, strings.Join(t.Services, ","), created)
			}
			return nil
		},
	}
}

func newAuthRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <email>",
		Short: "Remove a stored refresh token",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			u := ui.FromContext(cmd.Context())
			email := args[0]
			store, err := secrets.OpenDefault()
			if err != nil {
				return err
			}
			if err := store.DeleteToken(email); err != nil {
				return err
			}
			if outfmt.IsJSON(cmd.Context()) {
				return outfmt.WriteJSON(os.Stdout, map[string]any{
					"deleted": true,
					"email":   email,
				})
			}
			u.Out().Printf("deleted\ttrue")
			u.Out().Printf("email\t%s", email)
			return nil
		},
	}
}
