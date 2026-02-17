package cmd

import "testing"

func TestPaginationApply_GlobalMaxWhenPerCommandZero(t *testing.T) {
	flags := &RootFlags{MaxResults: 25}
	maxResults, pageToken := applyPagination(flags, 0, "")
	if maxResults != 25 {
		t.Fatalf("expected maxResults=25, got %d", maxResults)
	}
	if pageToken != "" {
		t.Fatalf("expected empty pageToken, got %q", pageToken)
	}
}

func TestPaginationApply_GlobalMaxWinsOverPerCommandDefault(t *testing.T) {
	// When user passes --max-results 25, it should override per-command defaults
	// (e.g. drive ls default of 20). Global flag takes precedence when set.
	flags := &RootFlags{MaxResults: 25}
	maxResults, pageToken := applyPagination(flags, 10, "")
	if maxResults != 25 {
		t.Fatalf("expected maxResults=25 (global --max-results wins when set), got %d", maxResults)
	}
	if pageToken != "" {
		t.Fatalf("expected empty pageToken, got %q", pageToken)
	}
}

func TestPaginationApply_GlobalPageTokenWhenPerCommandEmpty(t *testing.T) {
	flags := &RootFlags{PageToken: "abc123"}
	maxResults, pageToken := applyPagination(flags, 0, "")
	if maxResults != 0 {
		t.Fatalf("expected maxResults=0, got %d", maxResults)
	}
	if pageToken != "abc123" {
		t.Fatalf("expected pageToken=%q, got %q", "abc123", pageToken)
	}
}

func TestPaginationApply_PerCommandPageTakesPrecedence(t *testing.T) {
	flags := &RootFlags{PageToken: "global-token"}
	maxResults, pageToken := applyPagination(flags, 0, "cmd-token")
	if maxResults != 0 {
		t.Fatalf("expected maxResults=0, got %d", maxResults)
	}
	if pageToken != "cmd-token" {
		t.Fatalf("expected pageToken=%q (per-command wins), got %q", "cmd-token", pageToken)
	}
}

func TestPaginationApply_BothZeroEmpty_ReturnsDefaults(t *testing.T) {
	flags := &RootFlags{}
	maxResults, pageToken := applyPagination(flags, 0, "")
	if maxResults != 0 {
		t.Fatalf("expected maxResults=0, got %d", maxResults)
	}
	if pageToken != "" {
		t.Fatalf("expected empty pageToken, got %q", pageToken)
	}
}

func TestPaginationApply_BothSet_GlobalMaxWins_PerCmdPageWins(t *testing.T) {
	// Global --max-results takes priority over per-command max (which is typically
	// a default, not explicitly set by the user). Per-command page token still wins.
	flags := &RootFlags{MaxResults: 50, PageToken: "global-tok"}
	maxResults, pageToken := applyPagination(flags, 5, "cmd-tok")
	if maxResults != 50 {
		t.Fatalf("expected maxResults=50 (global wins), got %d", maxResults)
	}
	if pageToken != "cmd-tok" {
		t.Fatalf("expected pageToken=%q (per-command wins), got %q", "cmd-tok", pageToken)
	}
}

func TestPaginationApply_PerCommandMax_UsedAsFallback(t *testing.T) {
	// When global --max-results is 0 (not set), per-command max is used as fallback.
	flags := &RootFlags{MaxResults: 0}
	maxResults, _ := applyPagination(flags, 20, "")
	if maxResults != 20 {
		t.Fatalf("expected maxResults=20 (per-command fallback), got %d", maxResults)
	}
}

func TestPaginationApply_NilFlags_PerCommandFallback(t *testing.T) {
	maxResults, pageToken := applyPagination(nil, 20, "tok")
	if maxResults != 20 {
		t.Fatalf("expected maxResults=20 (per-command fallback), got %d", maxResults)
	}
	if pageToken != "tok" {
		t.Fatalf("expected pageToken=%q, got %q", "tok", pageToken)
	}
}

func TestPaginationServiceParamNames(t *testing.T) {
	// Services that use pageSize
	pageSizeServices := []string{"Drive", "Classroom", "People", "Chat", "Keep"}
	for _, svc := range pageSizeServices {
		info, ok := ServicePaginationParams[svc]
		if !ok {
			t.Fatalf("missing pagination info for %s", svc)
		}
		if info.MaxResultsParam != "pageSize" {
			t.Fatalf("expected %s maxResults param = pageSize, got %q", svc, info.MaxResultsParam)
		}
		if info.PageTokenParam != "pageToken" {
			t.Fatalf("expected %s pageToken param = pageToken, got %q", svc, info.PageTokenParam)
		}
	}

	// Services that use maxResults
	maxResultsServices := []string{"Calendar", "Gmail", "Admin", "Tasks", "Groups"}
	for _, svc := range maxResultsServices {
		info, ok := ServicePaginationParams[svc]
		if !ok {
			t.Fatalf("missing pagination info for %s", svc)
		}
		if info.MaxResultsParam != "maxResults" {
			t.Fatalf("expected %s maxResults param = maxResults, got %q", svc, info.MaxResultsParam)
		}
		if info.PageTokenParam != "pageToken" {
			t.Fatalf("expected %s pageToken param = pageToken, got %q", svc, info.PageTokenParam)
		}
	}

	// Sheets: N/A
	info, ok := ServicePaginationParams["Sheets"]
	if !ok {
		t.Fatal("missing pagination info for Sheets")
	}
	if info.MaxResultsParam != "" {
		t.Fatalf("expected Sheets maxResults param empty (N/A), got %q", info.MaxResultsParam)
	}
}
