package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/stretchr/testify/require"
)

// specRouter finds the OpenAPI operation for a request. It is loaded once from the real
// spec file, so the tests check the code against the document clients rely on.
var specRouter routers.Router

func loadSpec() error {
	doc, err := openapi3.NewLoader().LoadFromFile("../../../docs/api/openapi.yaml")
	if err != nil {
		return err
	}
	if err := doc.Validate(context.Background()); err != nil {
		return err
	}
	specRouter, err = gorillamux.NewRouter(doc)
	return err
}

// validateAgainstSpec reports whether the response is one the spec allows for the request:
// a documented status code, the right content type, and a body matching the declared
// schema. matched is false for routes the spec does not describe (such as /healthz).
func validateAgainstSpec(req *http.Request, rec *httptest.ResponseRecorder) (matched bool, err error) {
	route, pathParams, err := specRouter.FindRoute(req)
	if err != nil {
		return false, nil
	}
	err = openapi3filter.ValidateResponse(req.Context(), &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request: req, PathParams: pathParams, Route: route,
		},
		Status:  rec.Code,
		Header:  rec.Header(),
		Body:    nopCloser(rec.Body.Bytes()),
		Options: &openapi3filter.Options{IncludeResponseStatus: true},
	})
	return true, err
}

// checkAgainstSpec fails the test if the response does not match docs/api/openapi.yaml.
func checkAgainstSpec(t *testing.T, req *http.Request, rec *httptest.ResponseRecorder) {
	t.Helper()
	_, err := validateAgainstSpec(req, rec)
	require.NoError(t, err, "%s %s -> %d does not match docs/api/openapi.yaml:\n%s",
		req.Method, req.URL.Path, rec.Code, rec.Body.String())
}

// The checker is only worth having if it can fail. These cases feed it responses that
// drift from the spec and require it to notice.
func TestTheContractCheckerCatchesDrift(t *testing.T) {
	respond := func(code int, contentType, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		if contentType != "" {
			rec.Header().Set("Content-Type", contentType)
		}
		rec.WriteHeader(code)
		_, _ = rec.WriteString(body)
		return rec
	}
	login := func() *http.Request {
		return httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/login", nil)
	}
	const validToken = `{"access_token":"x","token_type":"Bearer","expires_in":60}`

	tests := []struct {
		name        string
		rec         *httptest.ResponseRecorder
		wantMatched bool
		wantErr     bool
	}{
		{"a conforming response", respond(200, "application/json", validToken), true, false},
		{"a required field is missing", respond(200, "application/json", `{"access_token":"x","token_type":"Bearer"}`), true, true},
		{"a field has the wrong type", respond(200, "application/json", `{"access_token":"x","token_type":"Bearer","expires_in":"60"}`), true, true},
		{"a wrong constant", respond(200, "application/json", `{"access_token":"x","token_type":"Basic","expires_in":60}`), true, true},
		{"an undocumented status code", respond(418, "application/json", validToken), true, true},
		{"the wrong content type", respond(200, "text/plain", validToken), true, true},
		{"an error body in the wrong shape", respond(401, "application/json", `{"message":"nope"}`), true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, err := validateAgainstSpec(login(), tt.rec)

			require.Equal(t, tt.wantMatched, matched)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}

	t.Run("routes outside the spec are skipped", func(t *testing.T) {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil)
		matched, err := validateAgainstSpec(req, respond(200, "", ""))
		require.NoError(t, err)
		require.False(t, matched)
	})

	t.Run("real routes are matched, so the checker is not a silent no-op", func(t *testing.T) {
		for _, path := range []string{"/api/v1/me", "/api/v1/makes", "/api/v1/vehicles", "/api/v1/vehicles/summary",
			"/api/v1/vehicles/3f2b8c1e-9a4d-4e7b-8c5a-1d2e3f4a5b6c",
			"/api/v1/vehicles/3f2b8c1e-9a4d-4e7b-8c5a-1d2e3f4a5b6c/actions"} {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
			matched, _ := validateAgainstSpec(req, respond(500, "", ""))
			require.True(t, matched, path)
		}
	})
}

type bodyReader struct{ *bytes.Reader }

func (bodyReader) Close() error { return nil }

func nopCloser(b []byte) io.ReadCloser { return bodyReader{bytes.NewReader(b)} }
