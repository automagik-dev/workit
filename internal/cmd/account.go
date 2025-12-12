package cmd

import (
	"errors"
	"os"
	"strings"
)

func requireAccount(flags *rootFlags) (string, error) {
	if v := strings.TrimSpace(flags.Account); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(os.Getenv("GOG_ACCOUNT")); v != "" {
		return v, nil
	}
	return "", errors.New("missing --account (or set GOG_ACCOUNT)")
}
