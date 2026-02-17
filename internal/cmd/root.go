package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/kong"
	"golang.org/x/term"

	"github.com/steipete/gogcli/internal/authclient"
	"github.com/steipete/gogcli/internal/config"
	"github.com/steipete/gogcli/internal/errfmt"
	"github.com/steipete/gogcli/internal/googleauth"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/secrets"
	"github.com/steipete/gogcli/internal/ui"
)

const (
	colorAuto  = "auto"
	colorNever = "never"
	strTrue    = "true"
)

type RootFlags struct {
	Color          string `help:"Color output: auto|always|never" default:"${color}"`
	Account        string `help:"Account email for API commands (gmail/calendar/chat/classroom/drive/docs/slides/contacts/tasks/people/sheets/forms/appscript)" aliases:"acct" short:"a"`
	Client         string `help:"OAuth client name (selects stored credentials + token bucket)" default:"${client}"`
	EnableCommands string `help:"Comma-separated list of enabled top-level commands (restricts CLI)" default:"${enabled_commands}"`
	CommandTier    string `name:"command-tier" help:"Command visibility tier: core|extended|complete (default: complete)" default:"${command_tier}" enum:"core,extended,complete"`
	JSON           bool   `help:"Output JSON to stdout (best for scripting)" default:"${json}" aliases:"machine" short:"j"`
	Plain          bool   `help:"Output stable, parseable text to stdout (TSV; no colors)" default:"${plain}" aliases:"tsv" short:"p"`
	ResultsOnly    bool   `name:"results-only" help:"In JSON mode, emit only the primary result (drops envelope fields like nextPageToken)"`
	Select         string `name:"select" aliases:"pick,project" help:"In JSON mode, select comma-separated fields (best-effort; supports dot paths). Desire path: use --fields for most commands."`
	JQ             string `name:"jq" help:"Apply jq expression to JSON output"`
	MaxResults     int    `name:"max-results" help:"Maximum number of results to return (maps to pageSize/maxResults per service)" default:"0"`
	PageToken      string `name:"page-token" help:"Page token for pagination (maps to pageToken per service)"`
	GenerateInput  bool   `name:"generate-input" help:"Print JSON input template for the command and exit" aliases:"gen-input"`
	DryRun         bool   `help:"Do not make changes; print intended actions and exit successfully" aliases:"noop,preview,dryrun" short:"n"`
	Force          bool   `help:"Skip confirmations for destructive commands" aliases:"yes,assume-yes" short:"y"`
	ReadOnly       bool   `name:"read-only" help:"Hide write commands and request read-only OAuth scopes" default:"${read_only}"`
	NoInput        bool   `help:"Never prompt; fail instead (useful for CI)" aliases:"non-interactive,noninteractive"`
	Verbose        bool   `help:"Enable verbose logging" short:"v"`
}

type CLI struct {
	RootFlags `embed:""`

	Version kong.VersionFlag `help:"Print version and exit"`

	// Action-first desire paths (agent-friendly shortcuts).
	Send     GmailSendCmd     `cmd:"" name:"send" help:"Send an email (alias for 'gmail send')"`
	Ls       DriveLsCmd       `cmd:"" name:"ls" aliases:"list" help:"List Drive files (alias for 'drive ls')"`
	Search   DriveSearchCmd   `cmd:"" name:"search" aliases:"find" help:"Search Drive files (alias for 'drive search')"`
	Open     OpenCmd          `cmd:"" name:"open" aliases:"browse" help:"Print a best-effort web URL for a Google URL/ID (offline)"`
	Download DriveDownloadCmd `cmd:"" name:"download" aliases:"dl" help:"Download a Drive file (alias for 'drive download')"`
	Upload   DriveUploadCmd   `cmd:"" name:"upload" aliases:"up,put" help:"Upload a file to Drive (alias for 'drive upload')"`
	Login    AuthAddCmd       `cmd:"" name:"login" help:"Authorize and store a refresh token (alias for 'auth add')"`
	Logout   AuthRemoveCmd    `cmd:"" name:"logout" help:"Remove a stored refresh token (alias for 'auth remove')"`
	Status   AuthStatusCmd    `cmd:"" name:"status" aliases:"st" help:"Show auth/config status (alias for 'auth status')"`
	Me       PeopleMeCmd      `cmd:"" name:"me" help:"Show your profile (alias for 'people me')"`
	Whoami   PeopleMeCmd      `cmd:"" name:"whoami" aliases:"who-am-i" help:"Show your profile (alias for 'people me')"`

	Auth       AuthCmd               `cmd:"" help:"Auth and credentials"`
	Groups     GroupsCmd             `cmd:"" aliases:"group" help:"Google Groups"`
	Drive      DriveCmd              `cmd:"" aliases:"drv" help:"Google Drive"`
	Docs       DocsCmd               `cmd:"" aliases:"doc" help:"Google Docs (export via Drive)"`
	Slides     SlidesCmd             `cmd:"" aliases:"slide" help:"Google Slides"`
	Calendar   CalendarCmd           `cmd:"" aliases:"cal" help:"Google Calendar"`
	Classroom  ClassroomCmd          `cmd:"" aliases:"class" help:"Google Classroom"`
	Time       TimeCmd               `cmd:"" help:"Local time utilities"`
	Gmail      GmailCmd              `cmd:"" aliases:"mail,email" help:"Gmail"`
	Chat       ChatCmd               `cmd:"" help:"Google Chat"`
	Contacts   ContactsCmd           `cmd:"" aliases:"contact" help:"Google Contacts"`
	Tasks      TasksCmd              `cmd:"" aliases:"task" help:"Google Tasks"`
	People     PeopleCmd             `cmd:"" aliases:"person" help:"Google People"`
	Keep       KeepCmd               `cmd:"" help:"Google Keep (Workspace only)"`
	Sheets     SheetsCmd             `cmd:"" aliases:"sheet" help:"Google Sheets"`
	Forms      FormsCmd              `cmd:"" aliases:"form" help:"Google Forms"`
	AppScript  AppScriptCmd          `cmd:"" name:"appscript" aliases:"script,apps-script" help:"Google Apps Script"`
	Sync       SyncCmd               `cmd:"" help:"Google Drive sync"`
	Config     ConfigCmd             `cmd:"" help:"Manage configuration"`
	ExitCodes  AgentExitCodesCmd     `cmd:"" name:"exit-codes" aliases:"exitcodes" help:"Print stable exit codes (alias for 'agent exit-codes')"`
	Agent      AgentCmd              `cmd:"" help:"Agent-friendly helpers"`
	Schema     SchemaCmd             `cmd:"" help:"Machine-readable command/flag schema" aliases:"help-json,helpjson"`
	VersionCmd VersionCmd            `cmd:"" name:"version" help:"Print version"`
	Completion CompletionCmd         `cmd:"" help:"Generate shell completion scripts"`
	Complete   CompletionInternalCmd `cmd:"" name:"__complete" hidden:"" help:"Internal completion helper"`
}

type exitPanic struct{ code int }

func Execute(args []string) (err error) {
	args = rewriteDesirePathArgs(args)

	parser, cli, err := newParser(helpDescription())
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			if ep, ok := r.(exitPanic); ok {
				if ep.code == 0 {
					err = nil
					return
				}
				err = &ExitError{Code: ep.code, Err: errors.New("exited")}
				return
			}
			panic(r)
		}
	}()

	// Pre-parse: check for --generate-input BEFORE full parsing so that
	// commands with required positional arguments don't fail.  We scan the
	// raw args, strip the flag, extract command tokens, and resolve the
	// command node directly from the parser model.
	if hasGenerateInput(args) {
		stripped := stripGenerateInputFlag(args)
		cmdTokens := extractCommandTokens(stripped)

		// Enforce --enable-commands in the pre-parse path so that
		// restricted commands cannot be introspected via --generate-input.
		enabledCSV := extractEnableCommands(args)
		if enabledCSV != "" {
			allow := parseEnabledCommands(enabledCSV)
			if len(allow) > 0 && !allow["*"] && !allow["all"] {
				if len(cmdTokens) > 0 && !allow[strings.ToLower(cmdTokens[0])] {
					cmdErr := usagef("command %q is not enabled (set --enable-commands to allow it)", cmdTokens[0])
					_, _ = fmt.Fprintln(os.Stderr, errfmt.Format(cmdErr))
					return cmdErr
				}
			}
		}

		node, findErr := findCommandNode(parser.Model.Node, cmdTokens)
		if findErr != nil {
			_, _ = fmt.Fprintln(os.Stderr, errfmt.Format(findErr))
			return findErr
		}
		return printGenerateInputFromNode(node)
	}

	kctx, err := parser.Parse(args)
	if err != nil {
		parsedErr := wrapParseError(err)
		_, _ = fmt.Fprintln(os.Stderr, errfmt.Format(parsedErr))
		return parsedErr
	}

	if err = enforceEnabledCommands(kctx, cli.EnableCommands); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, errfmt.Format(err))
		return err
	}

	if err = enforceCommandTier(kctx, cli.CommandTier); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, errfmt.Format(err))
		return err
	}

	if err = enforceReadOnly(kctx, cli.ReadOnly); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, errfmt.Format(err))
		return err
	}

	// --jq requires JSON output; reject early if combined with --plain.
	if cli.JQ != "" {
		if cli.Plain {
			_, _ = fmt.Fprintln(os.Stderr, "error: --jq requires --json output (incompatible with --plain)")
			return &ExitError{Code: 2, Err: errors.New("--jq requires --json output")}
		}
		// Auto-enable JSON when --jq is provided so that IsJSON(ctx) returns
		// true and commands emit JSON output for the jq pipeline to process.
		cli.JSON = true
	}

	logLevel := slog.LevelWarn
	if cli.Verbose {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))

	// Opt-in "agent mode": default to JSON when stdout is piped/non-TTY.
	// We intentionally do this after parsing so `--plain` can override it.
	if envBool("GOG_AUTO_JSON") && !cli.JSON && !cli.Plain && !term.IsTerminal(int(os.Stdout.Fd())) {
		cli.JSON = true
	}

	mode, err := outfmt.FromFlags(cli.JSON, cli.Plain)
	if err != nil {
		return newUsageError(err)
	}

	ctx := context.Background()
	ctx = outfmt.WithMode(ctx, mode)
	selectExplicit := outfmt.SelectFlagExplicitlySet(args)
	ctx = outfmt.WithJSONTransform(ctx, outfmt.JSONTransform{
		ResultsOnly:          cli.ResultsOnly,
		Select:               splitCommaList(cli.Select),
		JQ:                   cli.JQ,
		SelectExplicit:       selectExplicit,
		FieldDiscoveryWriter: os.Stderr,
	})
	ctx = authclient.WithClient(ctx, cli.Client)

	uiColor := cli.Color
	if outfmt.IsJSON(ctx) || outfmt.IsPlain(ctx) {
		uiColor = colorNever
	}

	u, err := ui.New(ui.Options{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Color:  uiColor,
	})
	if err != nil {
		return err
	}
	ctx = ui.WithUI(ctx, u)

	kctx.BindTo(ctx, (*context.Context)(nil))
	kctx.Bind(&cli.RootFlags)

	err = kctx.Run()
	if err == nil {
		return nil
	}
	// Some commands intentionally exit early with success.
	if ExitCode(err) == 0 {
		return nil
	}
	err = stableExitCode(err)

	if u := ui.FromContext(ctx); u != nil {
		msg := strings.TrimSpace(errfmt.Format(err))
		if msg != "" {
			u.Err().Error(msg)
		}
		return err
	}
	msg := strings.TrimSpace(errfmt.Format(err))
	if msg != "" {
		_, _ = fmt.Fprintln(os.Stderr, msg)
	}
	return err
}

func rewriteDesirePathArgs(args []string) []string {
	// `--fields` is already used by `calendar events` for the Calendar API `fields` parameter.
	// Agents frequently guess `--fields` to mean "select output fields", so we squat it
	// everywhere else by rewriting to the global `--select` flag.
	//
	// We avoid adding `--fields` as a real alias because Kong would treat it as a duplicate flag.
	keepFields := isCalendarEventsCommand(args)

	out := make([]string, 0, len(args))
	for i, a := range args {
		if a == "--" {
			out = append(out, args[i:]...)
			break
		}
		if keepFields {
			out = append(out, a)
			continue
		}
		if a == "--fields" {
			out = append(out, "--select")
			continue
		}
		if strings.HasPrefix(a, "--fields=") {
			out = append(out, "--select="+strings.TrimPrefix(a, "--fields="))
			continue
		}
		out = append(out, a)
	}
	return out
}

func isCalendarEventsCommand(args []string) bool {
	cmdTokens := make([]string, 0, 2)
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			break
		}
		if strings.HasPrefix(a, "-") {
			if globalFlagTakesValue(a) && i+1 < len(args) {
				i++
			}
			continue
		}
		cmdTokens = append(cmdTokens, a)
		if len(cmdTokens) >= 2 {
			break
		}
	}

	if len(cmdTokens) < 2 {
		return false
	}
	cmd0 := strings.TrimSpace(strings.ToLower(cmdTokens[0]))
	cmd1 := strings.TrimSpace(strings.ToLower(cmdTokens[1]))
	if cmd0 != "calendar" && cmd0 != "cal" {
		return false
	}
	return cmd1 == "events" || cmd1 == "ls" || cmd1 == "list"
}

func globalFlagTakesValue(flag string) bool {
	switch flag {
	case "--color", "--account", "--acct", "--client", "--enable-commands", "--command-tier", "--select", "--pick", "--project", "--jq", "-a",
		"--max-results", "--page-token":
		return true
	default:
		return false
	}
}

func wrapParseError(err error) error {
	if err == nil {
		return nil
	}
	var parseErr *kong.ParseError
	if errors.As(err, &parseErr) {
		return &ExitError{Code: 2, Err: parseErr}
	}
	return err
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch v {
	case "1", strTrue, "yes", "y", "on":
		return true
	default:
		return false
	}
}

func boolString(v bool) string {
	return strconv.FormatBool(v)
}

func newParser(description string) (*kong.Kong, *CLI, error) {
	envMode := outfmt.FromEnv()
	vars := kong.Vars{
		"auth_services":    googleauth.UserServiceCSV(),
		"color":            envOr("GOG_COLOR", "auto"),
		"calendar_weekday": envOr("GOG_CALENDAR_WEEKDAY", "false"),
		"client":           envOr("GOG_CLIENT", ""),
		"enabled_commands": envOr("GOG_ENABLE_COMMANDS", ""),
		"command_tier":     envOr("GOG_COMMAND_TIER", "complete"),
		"read_only":        boolString(envBool("GOG_READ_ONLY")),
		"json":             boolString(envMode.JSON),
		"plain":            boolString(envMode.Plain),
		"version":          VersionString(),
	}

	cli := &CLI{}
	parser, err := kong.New(
		cli,
		kong.Name("gog"),
		kong.Description(description),
		kong.ConfigureHelp(helpOptions()),
		kong.Help(helpPrinter),
		kong.Vars(vars),
		kong.Writers(os.Stdout, os.Stderr),
		kong.Exit(func(code int) { panic(exitPanic{code: code}) }),
	)
	if err != nil {
		return nil, nil, err
	}
	return parser, cli, nil
}

func baseDescription() string {
	return "Google CLI for Gmail/Calendar/Chat/Classroom/Drive/Contacts/Tasks/Sheets/Docs/Slides/People/Forms/App Script"
}

func helpDescription() string {
	desc := baseDescription()

	configPath, err := config.ConfigPath()
	configLine := "unknown"
	if err != nil {
		configLine = fmt.Sprintf("error: %v", err)
	} else if configPath != "" {
		configLine = configPath
	}

	backendInfo, err := secrets.ResolveKeyringBackendInfo()
	var backendLine string
	if err != nil {
		backendLine = fmt.Sprintf("error: %v", err)
	} else if backendInfo.Value != "" {
		backendLine = fmt.Sprintf("%s (source: %s)", backendInfo.Value, backendInfo.Source)
	}

	return fmt.Sprintf("%s\n\nConfig:\n  file: %s\n  keyring backend: %s", desc, configLine, backendLine)
}

// newUsageError wraps errors in a way main() can map to exit code 2.
func newUsageError(err error) error {
	if err == nil {
		return nil
	}
	return &ExitError{Code: 2, Err: err}
}

// generateInputFlags lists the flag names that trigger generate-input mode.
var generateInputFlags = map[string]bool{
	"--generate-input": true,
	"--gen-input":      true,
}

// hasGenerateInput returns true if the raw argument slice contains
// --generate-input or its alias --gen-input.
func hasGenerateInput(args []string) bool {
	for _, a := range args {
		if a == "--" {
			return false
		}
		if generateInputFlags[a] {
			return true
		}
	}
	return false
}

// stripGenerateInputFlag returns a copy of args with --generate-input / --gen-input removed.
func stripGenerateInputFlag(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if generateInputFlags[a] {
			continue
		}
		out = append(out, a)
	}
	return out
}

// extractEnableCommands scans the raw arg slice for --enable-commands flag/value,
// falling back to the GOG_ENABLE_COMMANDS env var.  This allows the generate-input
// pre-parse path to enforce command restrictions without full Kong parsing.
func extractEnableCommands(args []string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			break
		}
		if strings.HasPrefix(a, "--enable-commands=") {
			return strings.TrimPrefix(a, "--enable-commands=")
		}
		if a == "--enable-commands" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return os.Getenv("GOG_ENABLE_COMMANDS")
}

// extractCommandTokens pulls non-flag tokens from args (the command path).
// It skips flags and their values to isolate just the subcommand names.
//
// Only known global value-bearing flags (via globalFlagTakesValue) consume the
// next token.  Boolean flags like --json, --verbose are correctly skipped
// without swallowing the following token.  Unknown command-level flags with
// values (e.g. "--query foo") will leave "foo" as a token; findCommandNode
// will return a clear "unknown command" error rather than silently resolving
// the wrong command.
func extractCommandTokens(args []string) []string {
	tokens := make([]string, 0, 4)
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			break
		}
		if strings.HasPrefix(a, "-") {
			// Only skip the next token for flags known to take a value.
			if globalFlagTakesValue(a) && i+1 < len(args) {
				i++ // skip the known value
			}
			continue
		}
		tokens = append(tokens, a)
	}
	return tokens
}
