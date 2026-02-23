package cmd

import (
	"strings"

	"github.com/namastexlabs/gog-cli/internal/googleapi"
)

const peopleMeResource = "people/me"

func normalizePeopleResource(raw string) string {
	resource := strings.TrimSpace(raw)
	if resource == "" {
		return ""
	}
	if resource == "me" {
		return peopleMeResource
	}
	if strings.HasPrefix(resource, "people/") {
		return resource
	}
	return "people/" + resource
}

func wrapPeopleAPIError(err error) error {
	return googleapi.WrapAPIEnablementError(err, "people")
}
