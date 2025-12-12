package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/steipete/gogcli/internal/errfmt"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

type rootFlags struct {
	Color   string
	Account string
	Output  string
}

func Execute(args []string) error {
	flags := rootFlags{Color: envOr("GOG_COLOR", "auto")}
	flags.Output = envOr("GOG_OUTPUT", "text")

	root := &cobra.Command{
		Use:           "gog",
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			mode, err := outfmt.Parse(flags.Output)
			if err != nil {
				return err
			}
			cmd.SetContext(outfmt.WithMode(cmd.Context(), mode))

			u, err := ui.New(ui.Options{
				Stdout: os.Stdout,
				Stderr: os.Stderr,
				Color: func() string {
					if outfmt.IsJSON(cmd.Context()) {
						return "never"
					}
					return flags.Color
				}(),
			})
			if err != nil {
				return err
			}
			cmd.SetContext(ui.WithUI(cmd.Context(), u))
			return nil
		},
	}

	root.SetArgs(args)
	root.PersistentFlags().StringVar(&flags.Color, "color", flags.Color, "Color output: auto|always|never")
	root.PersistentFlags().StringVar(&flags.Account, "account", "", "Account email for API commands")
	root.PersistentFlags().StringVar(&flags.Output, "output", flags.Output, "Output format: text|json")

	root.AddCommand(newAuthCmd())
	root.AddCommand(newDriveCmd(&flags))
	root.AddCommand(newCalendarCmd(&flags))
	root.AddCommand(newGmailCmd(&flags))
	root.AddCommand(newContactsCmd(&flags))

	err := root.Execute()
	if err == nil {
		return nil
	}

	if u := ui.FromContext(root.Context()); u != nil {
		u.Err().Error(errfmt.Format(err))
		return err
	}
	_, _ = fmt.Fprintln(os.Stderr, errfmt.Format(err))
	return err
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
