package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	formsapi "google.golang.org/api/forms/v1"
	"google.golang.org/api/option"
)

func TestExecute_FormsPublish_JSON(t *testing.T) {
	origNew := newFormsService
	origPublish := formsSetPublishSettings
	origHTTPClient := newFormsHTTPClient
	t.Cleanup(func() {
		newFormsService = origNew
		formsSetPublishSettings = origPublish
		newFormsHTTPClient = origHTTPClient
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(strings.Contains(r.URL.Path, "/v1/forms/form123:setPublishSettings") && r.Method == http.MethodPost) {
			t.Logf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		ps, ok := body["publishSettings"].(map[string]any)
		if !ok {
			t.Fatalf("missing publishSettings in body: %#v", body)
		}
		if ps["isPublishedAsTemplate"] != true {
			t.Fatalf("expected isPublishedAsTemplate=true, got %v", ps["isPublishedAsTemplate"])
		}
		if _, ok := ps["requiresLogin"]; ok {
			t.Fatalf("requiresLogin should be omitted when flag is not set, got body: %#v", ps)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"publishSettings": map[string]any{
				"isPublished":           true,
				"isAcceptingResponses":  true,
				"isPublishedAsTemplate": true,
				"requiresLogin":         false,
			},
		})
	}))
	defer srv.Close()

	svc, err := formsapi.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newFormsService = func(context.Context, string) (*formsapi.Service, error) { return svc, nil }
	newFormsHTTPClient = func(context.Context, string) (*http.Client, error) { return srv.Client(), nil }

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{
				"--json",
				"--account", "a@b.com",
				"forms", "publish", "form123",
				"--publish-as-template",
			}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("unmarshal: %v (raw=%q)", err, out)
	}
	ps, ok := parsed["publish_settings"].(map[string]any)
	if !ok {
		t.Fatalf("missing publish_settings in output: %#v", parsed)
	}
	if ps["isPublishedAsTemplate"] != true {
		t.Fatalf("expected isPublishedAsTemplate=true in output, got %v", ps["isPublishedAsTemplate"])
	}
}

func TestExecute_FormsPublish_RequireAuth_JSON(t *testing.T) {
	origNew := newFormsService
	origPublish := formsSetPublishSettings
	origHTTPClient := newFormsHTTPClient
	t.Cleanup(func() {
		newFormsService = origNew
		formsSetPublishSettings = origPublish
		newFormsHTTPClient = origHTTPClient
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(strings.Contains(r.URL.Path, "/v1/forms/form123:setPublishSettings") && r.Method == http.MethodPost) {
			t.Logf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		ps, ok := body["publishSettings"].(map[string]any)
		if !ok {
			t.Fatalf("missing publishSettings in body: %#v", body)
		}
		if ps["requiresLogin"] != true {
			t.Fatalf("expected requiresLogin=true, got %v", ps["requiresLogin"])
		}
		if _, ok := ps["isPublishedAsTemplate"]; ok {
			t.Fatalf("isPublishedAsTemplate should be omitted when flag is not set, got body: %#v", ps)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"publishSettings": map[string]any{
				"isPublished":           true,
				"isAcceptingResponses":  true,
				"isPublishedAsTemplate": false,
				"requiresLogin":         true,
			},
		})
	}))
	defer srv.Close()

	svc, err := formsapi.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newFormsService = func(context.Context, string) (*formsapi.Service, error) { return svc, nil }
	newFormsHTTPClient = func(context.Context, string) (*http.Client, error) { return srv.Client(), nil }

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{
				"--json",
				"--account", "a@b.com",
				"forms", "publish", "form123",
				"--require-authentication",
			}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("unmarshal: %v (raw=%q)", err, out)
	}
	ps, ok := parsed["publish_settings"].(map[string]any)
	if !ok {
		t.Fatalf("missing publish_settings in output: %#v", parsed)
	}
	if ps["requiresLogin"] != true {
		t.Fatalf("expected requiresLogin=true in output, got %v", ps["requiresLogin"])
	}
}

func TestExecute_FormsPublish_Text(t *testing.T) {
	origNew := newFormsService
	origPublish := formsSetPublishSettings
	origHTTPClient := newFormsHTTPClient
	t.Cleanup(func() {
		newFormsService = origNew
		formsSetPublishSettings = origPublish
		newFormsHTTPClient = origHTTPClient
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(strings.Contains(r.URL.Path, "/v1/forms/form123:setPublishSettings") && r.Method == http.MethodPost) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"publishSettings": map[string]any{
				"isPublished":           true,
				"isAcceptingResponses":  true,
				"isPublishedAsTemplate": true,
				"requiresLogin":         false,
			},
		})
	}))
	defer srv.Close()

	svc, err := formsapi.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newFormsService = func(context.Context, string) (*formsapi.Service, error) { return svc, nil }
	newFormsHTTPClient = func(context.Context, string) (*http.Client, error) { return srv.Client(), nil }

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{
				"--account", "a@b.com",
				"forms", "publish", "form123",
				"--publish-as-template",
			}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	if !strings.Contains(out, "form_id\tform123") {
		t.Fatalf("expected form_id in output, got: %q", out)
	}
	if !strings.Contains(out, "is_published_as_template\ttrue") {
		t.Fatalf("expected is_published_as_template in output, got: %q", out)
	}
}

func TestExecute_FormsPublish_ExplicitFalse_JSON(t *testing.T) {
	origNew := newFormsService
	origPublish := formsSetPublishSettings
	origHTTPClient := newFormsHTTPClient
	t.Cleanup(func() {
		newFormsService = origNew
		formsSetPublishSettings = origPublish
		newFormsHTTPClient = origHTTPClient
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !(strings.Contains(r.URL.Path, "/v1/forms/form123:setPublishSettings") && r.Method == http.MethodPost) {
			t.Logf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		ps, ok := body["publishSettings"].(map[string]any)
		if !ok {
			t.Fatalf("missing publishSettings in body: %#v", body)
		}
		if ps["isPublishedAsTemplate"] != false {
			t.Fatalf("expected explicit isPublishedAsTemplate=false, got %v", ps["isPublishedAsTemplate"])
		}
		if _, ok := ps["requiresLogin"]; ok {
			t.Fatalf("requiresLogin should be omitted when flag is not set, got body: %#v", ps)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"publishSettings": map[string]any{
				"isPublished":           true,
				"isAcceptingResponses":  true,
				"isPublishedAsTemplate": false,
				"requiresLogin":         false,
			},
		})
	}))
	defer srv.Close()

	svc, err := formsapi.NewService(context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(srv.Client()),
		option.WithEndpoint(srv.URL+"/"),
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	newFormsService = func(context.Context, string) (*formsapi.Service, error) { return svc, nil }
	newFormsHTTPClient = func(context.Context, string) (*http.Client, error) { return srv.Client(), nil }

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{
				"--json",
				"--account", "a@b.com",
				"forms", "publish", "form123",
				"--publish-as-template=false",
			}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("unmarshal: %v (raw=%q)", err, out)
	}
	ps, ok := parsed["publish_settings"].(map[string]any)
	if !ok {
		t.Fatalf("missing publish_settings in output: %#v", parsed)
	}
	if ps["isPublishedAsTemplate"] != false {
		t.Fatalf("expected isPublishedAsTemplate=false in output, got %v", ps["isPublishedAsTemplate"])
	}
}

func TestExecute_FormsPublish_DryRun_JSON(t *testing.T) {
	origNew := newFormsService
	origPublish := formsSetPublishSettings
	origHTTPClient := newFormsHTTPClient
	t.Cleanup(func() {
		newFormsService = origNew
		formsSetPublishSettings = origPublish
		newFormsHTTPClient = origHTTPClient
	})
	errUnexpectedCall := errors.New("unexpected publish call")
	newFormsService = func(context.Context, string) (*formsapi.Service, error) {
		t.Fatalf("dry-run should not create forms service")
		return nil, errUnexpectedCall
	}
	formsSetPublishSettings = func(context.Context, *formsapi.Service, *http.Client, string, *bool, *bool) (*formsPublishSettingsResult, error) {
		t.Fatalf("dry-run should not call setPublishSettings")
		return nil, errUnexpectedCall
	}

	out := captureStdout(t, func() {
		_ = captureStderr(t, func() {
			if err := Execute([]string{
				"--json",
				"--dry-run",
				"--account", "a@b.com",
				"forms", "publish", "form123",
				"--publish-as-template",
				"--require-authentication",
			}); err != nil {
				t.Fatalf("Execute: %v", err)
			}
		})
	})

	var parsed struct {
		DryRun  bool   `json:"dry_run"`
		Op      string `json:"op"`
		Request struct {
			FormID                string `json:"form_id"`
			PublishAsTemplate     bool   `json:"publish_as_template"`
			RequireAuthentication bool   `json:"require_authentication"`
		} `json:"request"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("unmarshal: %v (raw=%q)", err, out)
	}
	if !parsed.DryRun || parsed.Op != "forms.publish" {
		t.Fatalf("unexpected dry-run payload: %#v", parsed)
	}
	if parsed.Request.FormID != "form123" || !parsed.Request.PublishAsTemplate || !parsed.Request.RequireAuthentication {
		t.Fatalf("unexpected request payload: %#v", parsed.Request)
	}
}
