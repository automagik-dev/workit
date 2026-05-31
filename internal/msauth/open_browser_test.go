package msauth

import (
	"strings"
	"testing"
)

func TestQuoteWindowsStartURLPreservesOAuthQuerySeparators(t *testing.T) {
	url := `https://login.microsoftonline.com/organizations/oauth2/v2.0/authorize?client_id=client&scope=User.Read+Mail.Read&redirect_uri=http://localhost:8085/oauth2/callback`
	got := quoteWindowsStartURL(url)

	if !strings.HasPrefix(got, `"`) || !strings.HasSuffix(got, `"`) {
		t.Fatalf("expected quoted URL, got %q", got)
	}

	if !strings.Contains(got, `&scope=`) || !strings.Contains(got, `&redirect_uri=`) {
		t.Fatalf("expected OAuth query separators preserved, got %q", got)
	}
}

func TestQuoteWindowsStartURLEscapesEmbeddedQuotes(t *testing.T) {
	got := quoteWindowsStartURL(`https://example.test/callback?x="bad"&y=1`)
	if strings.Contains(strings.Trim(got, `"`), `"`) {
		t.Fatalf("expected embedded quotes to be escaped, got %q", got)
	}

	if !strings.Contains(got, `%22bad%22`) {
		t.Fatalf("expected escaped quote payload, got %q", got)
	}
}
