package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	formsapi "google.golang.org/api/forms/v1"

	"github.com/steipete/gogcli/internal/googleapi"
	"github.com/steipete/gogcli/internal/outfmt"
	"github.com/steipete/gogcli/internal/ui"
)

var newFormsService = googleapi.NewForms

// newFormsHTTPClient creates an authenticated HTTP client for raw Forms API
// calls (e.g. setPublishSettings). Injectable for testing.
var newFormsHTTPClient = googleapi.NewFormsHTTPClient

// formsPublishSettingsResult is the decoded response from setPublishSettings.
type formsPublishSettingsResult struct {
	PublishSettings struct {
		IsPublished           bool `json:"isPublished"`
		IsAcceptingResponses  bool `json:"isAcceptingResponses"`
		IsPublishedAsTemplate bool `json:"isPublishedAsTemplate"`
		RequiresLogin         bool `json:"requiresLogin"`
	} `json:"publishSettings"`
}

// formsSetPublishSettings calls the Forms setPublishSettings endpoint.
// The Go client's PublishSettings struct has no typed data fields in this API
// version, so we issue a raw HTTP call through an authenticated HTTP client.
// This variable is injectable for testing.
var formsSetPublishSettings = defaultFormsSetPublishSettings

func defaultFormsSetPublishSettings(
	ctx context.Context,
	svc *formsapi.Service,
	httpClient *http.Client,
	formID string,
	publishAsTemplate, requireAuth *bool,
) (*formsPublishSettingsResult, error) {
	publishSettings := map[string]any{}
	if publishAsTemplate != nil {
		publishSettings["isPublishedAsTemplate"] = *publishAsTemplate
	}
	if requireAuth != nil {
		publishSettings["requiresLogin"] = *requireAuth
	}

	body := map[string]any{
		"publishSettings": publishSettings,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal publish settings: %w", err)
	}

	endpoint := strings.TrimRight(svc.BasePath, "/") + "/v1/forms/" + formID + ":setPublishSettings"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("set publish settings: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("setPublishSettings returned %d: %s", resp.StatusCode, respBody)
	}

	var result formsPublishSettingsResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode publish settings response: %w", err)
	}

	return &result, nil
}

type FormsCmd struct {
	Get       FormsGetCmd       `cmd:"" name:"get" aliases:"info,show" help:"Get a form"`
	Create    FormsCreateCmd    `cmd:"" name:"create" aliases:"new" help:"Create a form"`
	Publish   FormsPublishCmd   `cmd:"" name:"publish" help:"Control form publish settings"`
	Responses FormsResponsesCmd `cmd:"" name:"responses" help:"Form responses"`
}

type FormsResponsesCmd struct {
	List FormsResponsesListCmd `cmd:"" name:"list" aliases:"ls" help:"List form responses"`
	Get  FormsResponseGetCmd   `cmd:"" name:"get" aliases:"info,show" help:"Get a form response"`
}

type FormsGetCmd struct {
	FormID string `arg:"" name:"formId" help:"Form ID"`
}

func (c *FormsGetCmd) Run(ctx context.Context, flags *RootFlags) error {
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	formID := strings.TrimSpace(normalizeGoogleID(c.FormID))
	if formID == "" {
		return usage("empty formId")
	}

	svc, err := newFormsService(ctx, account)
	if err != nil {
		return err
	}

	form, err := svc.Forms.Get(formID).Context(ctx).Do()
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"form":     form,
			"edit_url": formEditURL(formID),
		})
	}

	u := ui.FromContext(ctx)
	printFormSummary(u, form, formID)
	return nil
}

type FormsCreateCmd struct {
	Title       string `name:"title" help:"Form title" required:""`
	Description string `name:"description" help:"Form description"`
}

func (c *FormsCreateCmd) Run(ctx context.Context, flags *RootFlags) error {
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	title := strings.TrimSpace(c.Title)
	if title == "" {
		return usage("empty --title")
	}
	description := strings.TrimSpace(c.Description)

	if dryRunErr := dryRunExit(ctx, flags, "forms.create", map[string]any{
		"title":       title,
		"description": description,
	}); dryRunErr != nil {
		return dryRunErr
	}

	svc, err := newFormsService(ctx, account)
	if err != nil {
		return err
	}

	req := &formsapi.Form{Info: &formsapi.Info{
		Title:       title,
		Description: description,
	}}
	form, err := svc.Forms.Create(req).Context(ctx).Do()
	if err != nil {
		return err
	}

	formID := strings.TrimSpace(form.FormId)
	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"created":  true,
			"form":     form,
			"edit_url": formEditURL(formID),
		})
	}

	u := ui.FromContext(ctx)
	u.Out().Printf("created\ttrue")
	printFormSummary(u, form, formID)
	return nil
}

type FormsPublishCmd struct {
	FormID                string `arg:"" name:"formId" help:"Form ID or URL"`
	PublishAsTemplate     bool   `name:"publish-as-template" help:"Publish the form as a template"`
	RequireAuthentication bool   `name:"require-authentication" help:"Require authentication to view/submit"`
}

func (c *FormsPublishCmd) Run(ctx context.Context, kctx *kong.Context, flags *RootFlags) error {
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	formID := strings.TrimSpace(normalizeGoogleID(c.FormID))
	if formID == "" {
		return usage("empty formId")
	}

	request := map[string]any{
		"form_id": formID,
	}

	var publishAsTemplate *bool
	if flagProvided(kctx, "publish-as-template") {
		value := c.PublishAsTemplate
		publishAsTemplate = &value
		request["publish_as_template"] = value
	}

	var requireAuth *bool
	if flagProvided(kctx, "require-authentication") {
		value := c.RequireAuthentication
		requireAuth = &value
		request["require_authentication"] = value
	}

	if publishAsTemplate == nil && requireAuth == nil {
		return usage("no publish settings provided (set --publish-as-template and/or --require-authentication)")
	}

	if dryRunErr := dryRunExit(ctx, flags, "forms.publish", request); dryRunErr != nil {
		return dryRunErr
	}

	svc, err := newFormsService(ctx, account)
	if err != nil {
		return err
	}

	// Create an authenticated HTTP client for the raw setPublishSettings call.
	httpClient, err := newFormsHTTPClient(ctx, account)
	if err != nil {
		return fmt.Errorf("forms http client: %w", err)
	}

	result, err := formsSetPublishSettings(ctx, svc, httpClient, formID, publishAsTemplate, requireAuth)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"form_id":          formID,
			"publish_settings": result.PublishSettings,
		})
	}

	u := ui.FromContext(ctx)
	u.Out().Printf("form_id\t%s", formID)
	u.Out().Printf("is_published\t%v", result.PublishSettings.IsPublished)
	u.Out().Printf("is_accepting_responses\t%v", result.PublishSettings.IsAcceptingResponses)
	u.Out().Printf("is_published_as_template\t%v", result.PublishSettings.IsPublishedAsTemplate)
	u.Out().Printf("requires_login\t%v", result.PublishSettings.RequiresLogin)
	return nil
}

type FormsResponsesListCmd struct {
	FormID string `arg:"" name:"formId" help:"Form ID"`
	Max    int    `name:"max" help:"Maximum responses" default:"20"`
	Page   string `name:"page" help:"Page token"`
	Filter string `name:"filter" help:"Filter expression"`
}

func (c *FormsResponsesListCmd) Run(ctx context.Context, flags *RootFlags) error {
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	formID := strings.TrimSpace(normalizeGoogleID(c.FormID))
	if formID == "" {
		return usage("empty formId")
	}
	if c.Max <= 0 {
		return usage("--max must be > 0")
	}

	svc, err := newFormsService(ctx, account)
	if err != nil {
		return err
	}

	effectiveMax, effectivePage := applyPagination(flags, int64(c.Max), c.Page)

	call := svc.Forms.Responses.List(formID).PageSize(effectiveMax).Context(ctx)
	if page := strings.TrimSpace(effectivePage); page != "" {
		call = call.PageToken(page)
	}
	if filter := strings.TrimSpace(c.Filter); filter != "" {
		call = call.Filter(filter)
	}
	resp, err := call.Do()
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"form_id":       formID,
			"responses":     resp.Responses,
			"nextPageToken": resp.NextPageToken,
		})
	}

	u := ui.FromContext(ctx)
	u.Out().Println("RESPONSE_ID\tSUBMITTED\tEMAIL")
	for _, item := range resp.Responses {
		if item == nil {
			continue
		}
		submitted := firstFormTime(item.LastSubmittedTime, item.CreateTime)
		u.Out().Printf("%s\t%s\t%s", item.ResponseId, submitted, item.RespondentEmail)
	}
	if next := strings.TrimSpace(resp.NextPageToken); next != "" {
		u.Err().Println("# Next page: --page " + next)
	}
	return nil
}

type FormsResponseGetCmd struct {
	FormID     string `arg:"" name:"formId" help:"Form ID"`
	ResponseID string `arg:"" name:"responseId" help:"Response ID"`
}

func (c *FormsResponseGetCmd) Run(ctx context.Context, flags *RootFlags) error {
	account, err := requireAccount(flags)
	if err != nil {
		return err
	}
	formID := strings.TrimSpace(normalizeGoogleID(c.FormID))
	if formID == "" {
		return usage("empty formId")
	}
	responseID := strings.TrimSpace(c.ResponseID)
	if responseID == "" {
		return usage("empty responseId")
	}

	svc, err := newFormsService(ctx, account)
	if err != nil {
		return err
	}
	resp, err := svc.Forms.Responses.Get(formID, responseID).Context(ctx).Do()
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(ctx, os.Stdout, map[string]any{
			"response": resp,
		})
	}

	u := ui.FromContext(ctx)
	u.Out().Printf("response_id\t%s", resp.ResponseId)
	u.Out().Printf("submitted\t%s", firstFormTime(resp.LastSubmittedTime, resp.CreateTime))
	if resp.RespondentEmail != "" {
		u.Out().Printf("email\t%s", resp.RespondentEmail)
	}
	u.Out().Printf("answers\t%d", len(resp.Answers))
	if resp.TotalScore != 0 {
		u.Out().Printf("total_score\t%s", strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", resp.TotalScore), "0"), "."))
	}
	return nil
}

func printFormSummary(u *ui.UI, form *formsapi.Form, fallbackID string) {
	if u == nil || form == nil {
		return
	}
	formID := strings.TrimSpace(form.FormId)
	if formID == "" {
		formID = strings.TrimSpace(fallbackID)
	}
	u.Out().Printf("id\t%s", formID)
	if form.Info != nil {
		if form.Info.Title != "" {
			u.Out().Printf("title\t%s", form.Info.Title)
		}
		if form.Info.Description != "" {
			u.Out().Printf("description\t%s", form.Info.Description)
		}
	}
	if form.ResponderUri != "" {
		u.Out().Printf("responder_uri\t%s", form.ResponderUri)
	}
	u.Out().Printf("edit_url\t%s", formEditURL(formID))
}

func formEditURL(formID string) string {
	formID = strings.TrimSpace(formID)
	if formID == "" {
		return ""
	}
	return "https://docs.google.com/forms/d/" + formID + "/edit"
}

func firstFormTime(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
