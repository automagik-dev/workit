package config

import (
	"encoding/json"
	"errors"
	"os"
)

type ClientCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type googleCredentialsFile struct {
	Installed *struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	} `json:"installed"`
	Web *struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	} `json:"web"`
}

func ParseGoogleOAuthClientJSON(b []byte) (ClientCredentials, error) {
	var f googleCredentialsFile
	if err := json.Unmarshal(b, &f); err != nil {
		return ClientCredentials{}, err
	}

	var clientID, clientSecret string
	if f.Installed != nil {
		clientID, clientSecret = f.Installed.ClientID, f.Installed.ClientSecret
	} else if f.Web != nil {
		clientID, clientSecret = f.Web.ClientID, f.Web.ClientSecret
	}
	if clientID == "" || clientSecret == "" {
		return ClientCredentials{}, errors.New("invalid credentials.json (expected installed/web client_id and client_secret)")
	}
	return ClientCredentials{ClientID: clientID, ClientSecret: clientSecret}, nil
}

func WriteClientCredentials(c ClientCredentials) error {
	_, err := EnsureDir()
	if err != nil {
		return err
	}
	path, err := ClientCredentialsPath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	return nil
}

func ReadClientCredentials() (ClientCredentials, error) {
	path, err := ClientCredentialsPath()
	if err != nil {
		return ClientCredentials{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ClientCredentials{}, &CredentialsMissingError{Path: path, Cause: err}
		}
		return ClientCredentials{}, err
	}
	var c ClientCredentials
	if err := json.Unmarshal(b, &c); err != nil {
		return ClientCredentials{}, err
	}
	if c.ClientID == "" || c.ClientSecret == "" {
		return ClientCredentials{}, errors.New("stored credentials.json is missing client_id/client_secret")
	}
	return c, nil
}

type CredentialsMissingError struct {
	Path  string
	Cause error
}

func (e *CredentialsMissingError) Error() string {
	return "oauth credentials missing"
}

func (e *CredentialsMissingError) Unwrap() error {
	return e.Cause
}
