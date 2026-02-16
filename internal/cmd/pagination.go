package cmd

// PaginationParamInfo describes how a Google API service names its pagination
// parameters.  Different services use either "pageSize" or "maxResults" for
// the page-size field, while all use "pageToken" for the continuation token.
type PaginationParamInfo struct {
	// MaxResultsParam is the API parameter name for page size.
	// Empty string means the service does not support cursor-based pagination
	// (e.g. Sheets is row-based).
	MaxResultsParam string
	// PageTokenParam is the API parameter name for the continuation token.
	// Empty string when pagination is not applicable.
	PageTokenParam string
}

// ServicePaginationParams maps Google Workspace service names to their
// pagination parameter conventions.
//
//	Services using pageSize:   Drive, Classroom, People, Chat, Keep
//	Services using maxResults: Calendar, Gmail, Admin, Tasks, Groups
//	Sheets: N/A (row-based, pagination not applicable)
var ServicePaginationParams = map[string]PaginationParamInfo{
	"Calendar":  {MaxResultsParam: "maxResults", PageTokenParam: "pageToken"},
	"Classroom": {MaxResultsParam: "pageSize", PageTokenParam: "pageToken"},
	"Drive":     {MaxResultsParam: "pageSize", PageTokenParam: "pageToken"},
	"Gmail":     {MaxResultsParam: "maxResults", PageTokenParam: "pageToken"},
	"People":    {MaxResultsParam: "pageSize", PageTokenParam: "pageToken"},
	"Admin":     {MaxResultsParam: "maxResults", PageTokenParam: "pageToken"},
	"Tasks":     {MaxResultsParam: "maxResults", PageTokenParam: "pageToken"},
	"Groups":    {MaxResultsParam: "maxResults", PageTokenParam: "pageToken"},
	"Sheets":    {MaxResultsParam: "", PageTokenParam: ""},
	"Chat":      {MaxResultsParam: "pageSize", PageTokenParam: "pageToken"},
	"Keep":      {MaxResultsParam: "pageSize", PageTokenParam: "pageToken"},
}

// applyPagination resolves the effective maxResults and pageToken by applying
// the precedence rule (DEC-7): per-command flags win over global flags.
//
// Parameters:
//   - flags: the global RootFlags (carries --max-results and --page-token)
//   - perCommandMax: the per-command --max/--limit value (0 means not explicitly set)
//   - perCommandPage: the per-command --page/--cursor value ("" means not set)
//
// Returns the resolved (maxResults, pageToken) pair.
//
// Interaction with --results-only: when --results-only is used with pagination,
// the nextPageToken is stripped from output.  Callers that need to paginate
// across multiple pages should omit --results-only.
func applyPagination(flags *RootFlags, perCommandMax int64, perCommandPage string) (maxResults int64, pageToken string) {
	// --- max results ---
	// Per-command flag takes precedence when explicitly set (> 0).
	if perCommandMax > 0 {
		maxResults = perCommandMax
	} else if flags != nil && int64(flags.MaxResults) > 0 {
		maxResults = int64(flags.MaxResults)
	}
	// else: maxResults stays 0 (use whatever API default the caller has)

	// --- page token ---
	// Per-command flag takes precedence when non-empty.
	if perCommandPage != "" {
		pageToken = perCommandPage
	} else if flags != nil && flags.PageToken != "" {
		pageToken = flags.PageToken
	}
	// else: pageToken stays "" (first page)

	return maxResults, pageToken
}
