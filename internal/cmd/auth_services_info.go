package cmd

import (
	"strings"

	"github.com/automagik-dev/workit/internal/googleauth"
	"github.com/automagik-dev/workit/internal/msauth"
)

type authServiceInfo struct {
	Service string   `json:"service"`
	User    bool     `json:"user"`
	Scopes  []string `json:"scopes"`
	APIs    []string `json:"apis,omitempty"`
	Note    string   `json:"note,omitempty"`
}

func appendAuthServiceInfos(googleInfos []googleauth.ServiceInfo, m365Infos []msauth.ServiceInfo) []authServiceInfo {
	out := make([]authServiceInfo, 0, len(googleInfos)+len(m365Infos))
	for _, info := range googleInfos {
		out = append(out, authServiceInfo{
			Service: string(info.Service),
			User:    info.User,
			Scopes:  append([]string(nil), info.Scopes...),
			APIs:    append([]string(nil), info.APIs...),
			Note:    info.Note,
		})
	}
	for _, info := range m365Infos {
		out = append(out, authServiceInfo{
			Service: info.Service,
			User:    info.User,
			Scopes:  append([]string(nil), info.Scopes...),
			APIs:    append([]string(nil), info.APIs...),
			Note:    info.Note,
		})
	}

	return out
}

func authServicesMarkdown(infos []authServiceInfo) string {
	if len(infos) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("| Service | User | APIs | Scopes | Notes |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, info := range infos {
		userLabel := "no"
		if info.User {
			userLabel = "yes"
		}
		b.WriteString("| ")
		b.WriteString(info.Service)
		b.WriteString(" | ")
		b.WriteString(userLabel)
		b.WriteString(" | ")
		b.WriteString(strings.Join(info.APIs, ", "))
		b.WriteString(" | ")
		b.WriteString(markdownAuthScopes(info.Scopes))
		b.WriteString(" | ")
		b.WriteString(info.Note)
		b.WriteString(" |\n")
	}

	return b.String()
}

func markdownAuthScopes(scopes []string) string {
	if len(scopes) == 0 {
		return ""
	}

	parts := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		parts = append(parts, "`"+scope+"`")
	}

	return strings.Join(parts, "<br>")
}
