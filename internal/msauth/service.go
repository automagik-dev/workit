package msauth

type ServiceInfo struct {
	Service string   `json:"service"`
	User    bool     `json:"user"`
	Scopes  []string `json:"scopes"`
	APIs    []string `json:"apis,omitempty"`
	Note    string   `json:"note,omitempty"`
}

func ServicesInfo() []ServiceInfo {
	return []ServiceInfo{
		{
			Service: "m365",
			User:    true,
			Scopes:  PilotAllowedScopes(),
			APIs:    []string{"Microsoft Graph"},
			Note:    "Read-only Microsoft 365 pilot; writes remain KHAW-gated/disabled",
		},
	}
}
