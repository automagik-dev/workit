package googleauth

import "testing"

func TestIsReadOnlyScope(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"openid":  true,
		"email":   true,
		"profile": true,
		"https://www.googleapis.com/auth/userinfo.email":            true,
		"https://www.googleapis.com/auth/userinfo.profile":          true,
		"https://www.googleapis.com/auth/drive.readonly":            true,
		"https://www.googleapis.com/auth/spreadsheets.readonly":     true,
		"https://www.googleapis.com/auth/drive.metadata.readonly":   true,
		"https://www.googleapis.com/auth/drive":                     false,
		"https://www.googleapis.com/auth/drive.file":                false,
		"https://www.googleapis.com/auth/drive.metadata":            false,
		"https://www.googleapis.com/auth/spreadsheets":              false,
		"https://www.googleapis.com/auth/gmail.modify":              false,
		"https://www.googleapis.com/auth/readonly.but.not.suffixed": false,
	}

	for scope, want := range cases {
		if got := IsReadOnlyScope(scope); got != want {
			t.Errorf("IsReadOnlyScope(%q) = %v, want %v", scope, got, want)
		}
	}
}
