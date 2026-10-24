package cmd

import (
	"slices"
	"strings"

	"google.golang.org/api/gmail/v1"
)

// gmailLabelColorPalette is the fixed set of hex colors the Gmail API accepts
// for both LabelColor.BackgroundColor and LabelColor.TextColor. Any other value
// is rejected by the API with "400 Invalid color".
// Source: google.golang.org/api/gmail/v1 LabelColor field docs.
var gmailLabelColorPalette = []string{
	"#000000", "#434343", "#666666", "#999999", "#cccccc", "#efefef", "#f3f3f3", "#ffffff",
	"#fb4c2f", "#ffad47", "#fad165", "#16a766", "#43d692", "#4a86e8", "#a479e2", "#f691b3",
	"#f6c5be", "#ffe6c7", "#fef1d1", "#b9e4d0", "#c6f3de", "#c9daf8", "#e4d7f5", "#fcdee8",
	"#efa093", "#ffd6a2", "#fce8b3", "#89d3b2", "#a0eac9", "#a4c2f4", "#d0bcf1", "#fbc8d9",
	"#e66550", "#ffbc6b", "#fcda83", "#44b984", "#68dfa9", "#6d9eeb", "#b694e8", "#f7a7c0",
	"#cc3a21", "#eaa041", "#f2c960", "#149e60", "#3dc789", "#3c78d8", "#8e63ce", "#e07798",
	"#ac2b16", "#cf8933", "#d5ae49", "#0b804b", "#2a9c68", "#285bac", "#653e9b", "#b65775",
	"#822111", "#a46a21", "#aa8831", "#076239", "#1a764d", "#1c4587", "#41236d", "#83334c",
	"#464646", "#e7e7e7", "#0d3472", "#b6cff5", "#0d3b44", "#98d7e4", "#3d188e", "#e3d7ff",
	"#711a36", "#fbd3e0", "#8a1c0a", "#f2b2a8", "#7a2e0b", "#ffc8af", "#7a4706", "#ffdeb5",
	"#594c05", "#fbe983", "#684e07", "#fdedc1", "#0b4f30", "#b3efd3", "#04502e", "#a2dcc1",
	"#c2c2c2", "#4986e7", "#2da2bb", "#b99aff", "#994a64", "#f691b2", "#ff7537", "#ffad46",
	"#662e37", "#ebdbde", "#cca6ac", "#094228", "#42d692", "#16a765",
}

var (
	gmailLabelListVisibilities   = []string{"labelShow", "labelShowIfUnread", "labelHide"}
	gmailMessageListVisibilities = []string{"show", "hide"}
)

// GmailLabelPropsFlags are the label properties shared by create and modify-label.
type GmailLabelPropsFlags struct {
	BackgroundColor       string `name:"background-color" help:"Background color, hex from the Gmail palette (e.g. #16a765); requires --text-color"`
	TextColor             string `name:"text-color" help:"Text color, hex from the Gmail palette (e.g. #ffffff); requires --background-color"`
	LabelListVisibility   string `name:"label-list-visibility" help:"Visibility in the label list: labelShow|labelShowIfUnread|labelHide"`
	MessageListVisibility string `name:"message-list-visibility" help:"Visibility of the label on messages: show|hide"`
}

func (p GmailLabelPropsFlags) isEmpty() bool {
	return strings.TrimSpace(p.BackgroundColor) == "" && strings.TrimSpace(p.TextColor) == "" &&
		strings.TrimSpace(p.LabelListVisibility) == "" && strings.TrimSpace(p.MessageListVisibility) == ""
}

// buildLabelRequest validates the flags and builds the Label body for
// users.labels.create / users.labels.patch. Unset fields stay empty so a
// patch leaves them untouched.
func buildLabelRequest(name string, p GmailLabelPropsFlags) (*gmail.Label, error) {
	label := &gmail.Label{Name: strings.TrimSpace(name)}

	bg := strings.TrimSpace(p.BackgroundColor)
	fg := strings.TrimSpace(p.TextColor)
	if bg != "" || fg != "" {
		if bg == "" || fg == "" {
			return nil, usage("--background-color and --text-color must be set together (Gmail requires both)")
		}
		var err error
		if bg, err = validateLabelColor("--background-color", bg); err != nil {
			return nil, err
		}
		if fg, err = validateLabelColor("--text-color", fg); err != nil {
			return nil, err
		}
		label.Color = &gmail.LabelColor{BackgroundColor: bg, TextColor: fg}
	}

	var err error
	if label.LabelListVisibility, err = validateLabelEnum("--label-list-visibility", p.LabelListVisibility, gmailLabelListVisibilities); err != nil {
		return nil, err
	}
	if label.MessageListVisibility, err = validateLabelEnum("--message-list-visibility", p.MessageListVisibility, gmailMessageListVisibilities); err != nil {
		return nil, err
	}
	return label, nil
}

// validateLabelColor normalizes a hex color (case-insensitive, optional '#')
// and checks it against the Gmail palette.
func validateLabelColor(flag, raw string) (string, error) {
	c := strings.ToLower(strings.TrimSpace(raw))
	if !strings.HasPrefix(c, "#") {
		c = "#" + c
	}
	if !slices.Contains(gmailLabelColorPalette, c) {
		return "", usagef("invalid %s %q: Gmail only accepts colors from its fixed palette: %s",
			flag, raw, strings.Join(gmailLabelColorPalette, ", "))
	}
	return c, nil
}

// validateLabelEnum matches raw case-insensitively against allowed and returns
// the canonical spelling; empty input returns empty.
func validateLabelEnum(flag, raw string, allowed []string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", nil
	}
	for _, a := range allowed {
		if strings.EqualFold(a, v) {
			return a, nil
		}
	}
	return "", usagef("invalid %s %q (must be one of: %s)", flag, raw, strings.Join(allowed, ", "))
}
