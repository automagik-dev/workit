package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/muesli/termenv"
)

type Options struct {
	Stdout io.Writer
	Stderr io.Writer
	Color  string // auto|always|never
}

type UI struct {
	out *Printer
	err *Printer
}

func New(opts Options) (*UI, error) {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	colorMode := strings.ToLower(strings.TrimSpace(opts.Color))
	if colorMode == "" {
		colorMode = "auto"
	}
	if colorMode != "auto" && colorMode != "always" && colorMode != "never" {
		return nil, errors.New("invalid --color (expected auto|always|never)")
	}

	out := termenv.NewOutput(opts.Stdout, termenv.WithProfile(termenv.EnvColorProfile()))
	errOut := termenv.NewOutput(opts.Stderr, termenv.WithProfile(termenv.EnvColorProfile()))

	outProfile := chooseProfile(out.Profile, colorMode)
	errProfile := chooseProfile(errOut.Profile, colorMode)

	return &UI{
		out: newPrinter(out, outProfile),
		err: newPrinter(errOut, errProfile),
	}, nil
}

func chooseProfile(detected termenv.Profile, mode string) termenv.Profile {
	if termenv.EnvNoColor() {
		return termenv.Ascii
	}
	switch mode {
	case "never":
		return termenv.Ascii
	case "always":
		return termenv.TrueColor
	default:
		return detected
	}
}

func (u *UI) Out() *Printer { return u.out }
func (u *UI) Err() *Printer { return u.err }

type Printer struct {
	o       *termenv.Output
	profile termenv.Profile
}

func newPrinter(o *termenv.Output, profile termenv.Profile) *Printer {
	return &Printer{o: o, profile: profile}
}

func (p *Printer) ColorEnabled() bool { return p.profile != termenv.Ascii }

func (p *Printer) line(s string) {
	_, _ = io.WriteString(p.o, s+"\n")
}

func (p *Printer) printf(format string, args ...any) {
	p.line(fmt.Sprintf(format, args...))
}

func (p *Printer) Successf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if p.ColorEnabled() {
		msg = termenv.String(msg).Foreground(p.profile.Color("#22c55e")).String()
	}
	p.line(msg)
}

func (p *Printer) Error(msg string) {
	if p.ColorEnabled() {
		msg = termenv.String(msg).Foreground(p.profile.Color("#ef4444")).String()
	}
	p.line(msg)
}

func (p *Printer) Errorf(format string, args ...any) { p.Error(fmt.Sprintf(format, args...)) }
func (p *Printer) Printf(format string, args ...any) { p.printf(format, args...) }
func (p *Printer) Println(msg string)                { p.line(msg) }

type ctxKey struct{}

func WithUI(ctx context.Context, u *UI) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

func FromContext(ctx context.Context) *UI {
	v := ctx.Value(ctxKey{})
	if v == nil {
		return nil
	}
	u, _ := v.(*UI)
	return u
}
