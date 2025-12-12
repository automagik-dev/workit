package config

import (
	"os"
	"path/filepath"
)

const AppName = "gogcli"

func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, AppName), nil
}

func EnsureDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func ClientCredentialsPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

func DriveDownloadsDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "drive-downloads"), nil
}

func EnsureDriveDownloadsDir() (string, error) {
	dir, err := DriveDownloadsDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func GmailAttachmentsDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gmail-attachments"), nil
}

func EnsureGmailAttachmentsDir() (string, error) {
	dir, err := GmailAttachmentsDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
