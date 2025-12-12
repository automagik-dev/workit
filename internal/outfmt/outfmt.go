package outfmt

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

type Mode string

const (
	ModeText Mode = "text"
	ModeJSON Mode = "json"
)

func Parse(s string) (Mode, error) {
	switch Mode(strings.ToLower(strings.TrimSpace(s))) {
	case ModeText, "":
		return ModeText, nil
	case ModeJSON:
		return ModeJSON, nil
	default:
		return "", errors.New("invalid --output (expected text|json)")
	}
}

type ctxKey struct{}

func WithMode(ctx context.Context, mode Mode) context.Context {
	return context.WithValue(ctx, ctxKey{}, mode)
}

func FromContext(ctx context.Context) Mode {
	if v := ctx.Value(ctxKey{}); v != nil {
		if m, ok := v.(Mode); ok {
			return m
		}
	}
	return ModeText
}

func IsJSON(ctx context.Context) bool {
	return FromContext(ctx) == ModeJSON
}

func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
