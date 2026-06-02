package main

import (
	"os"
	"strings"
	"testing"
)

func TestDockerfileDocumentsEnterpriseM365EnvContract(t *testing.T) {
	data, err := os.ReadFile("Dockerfile")
	if err != nil {
		t.Fatalf("read Dockerfile: %v", err)
	}

	content := string(data)
	for _, want := range []string{"WK_PUBLIC_BASE_URL", "WK_M365_CLIENT_ID", "WK_M365_TENANT_ID", "TARGETARCH", "EXPOSE 8080", "workit-auth-server"} {
		if !strings.Contains(content, want) {
			t.Fatalf("Dockerfile missing %s", want)
		}
	}
}
