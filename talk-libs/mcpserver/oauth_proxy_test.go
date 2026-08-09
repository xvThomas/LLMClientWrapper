package mcpserver

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestRegisterASProxy_ASMetadata_ContentType verifies that the AS metadata
// endpoint sets the correct Content-Type header.
func TestRegisterASProxy_ASMetadata_ContentType(t *testing.T) {
	cfg := &OAuthConfig{
		AuthorizationServerURL: "https://auth.example.com",
		ASProxy:                &ASProxyConfig{Audience: "my-api"},
	}
	mux := http.NewServeMux()
	registerASProxy(mux, "http://localhost:8080", cfg)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// TestRegisterASProxy_Authorize_NoAudience verifies that no audience parameter
// is injected when ASProxyConfig.Audience is empty.
func TestRegisterASProxy_Authorize_NoAudience(t *testing.T) {
	cfg := &OAuthConfig{
		AuthorizationServerURL: "https://auth.example.com",
		ASProxy:                &ASProxyConfig{Audience: ""},
	}
	mux := http.NewServeMux()
	registerASProxy(mux, "http://localhost:8080", cfg)

	req := httptest.NewRequest(http.MethodGet, "/authorize?client_id=abc&scope=openid", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rr.Code)
	}
	loc := rr.Header().Get("Location")
	parsed, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("invalid Location URL: %v", err)
	}
	if parsed.Query().Has("audience") {
		t.Errorf("audience should not be present in redirect, got %q", loc)
	}
}

// TestRegisterASProxy_Authorize_EmptyScope verifies that when no scope is
// provided, the injected scope is exactly "offline_access" (no leading space).
func TestRegisterASProxy_Authorize_EmptyScope(t *testing.T) {
	cfg := &OAuthConfig{
		AuthorizationServerURL: "https://auth.example.com",
		ASProxy:                &ASProxyConfig{Audience: "my-api"},
	}
	mux := http.NewServeMux()
	registerASProxy(mux, "http://localhost:8080", cfg)

	req := httptest.NewRequest(http.MethodGet, "/authorize?client_id=abc", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rr.Code)
	}
	loc := rr.Header().Get("Location")
	parsed, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("invalid Location URL: %v", err)
	}
	if scope := parsed.Query().Get("scope"); scope != "offline_access" {
		t.Errorf("scope = %q, want %q", scope, "offline_access")
	}
}

// TestRegisterASProxy_Token_ClientSecretNotOverwritten verifies that a
// client_secret already present in the form body is not replaced by the
// proxy-configured secret.
func TestRegisterASProxy_Token_ClientSecretNotOverwritten(t *testing.T) {
	var capturedSecret string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		capturedSecret = r.FormValue("client_secret")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok"}`))
	}))
	defer upstream.Close()

	cfg := &OAuthConfig{
		AuthorizationServerURL: "https://auth.example.com",
		ASProxy: &ASProxyConfig{
			Audience:         "my-api",
			UpstreamTokenURL: upstream.URL,
			ClientSecret:     "proxy-secret",
		},
	}
	mux := http.NewServeMux()
	registerASProxy(mux, "http://localhost:8080", cfg)

	form := "grant_type=authorization_code&code=abc&client_secret=client-provided-secret"
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", rr.Code, rr.Body.String())
	}
	if capturedSecret != "client-provided-secret" {
		t.Errorf("client_secret was overwritten: got %q, want %q", capturedSecret, "client-provided-secret")
	}
}

// TestRegisterASProxy_Token_AuthorizationHeaderForwarded verifies that a
// Basic Authorization header sent by the client is forwarded to the upstream
// token endpoint.
func TestRegisterASProxy_Token_AuthorizationHeaderForwarded(t *testing.T) {
	var capturedAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tok"}`))
	}))
	defer upstream.Close()

	cfg := &OAuthConfig{
		AuthorizationServerURL: "https://auth.example.com",
		ASProxy: &ASProxyConfig{
			Audience:         "my-api",
			UpstreamTokenURL: upstream.URL,
		},
	}
	mux := http.NewServeMux()
	registerASProxy(mux, "http://localhost:8080", cfg)

	form := "grant_type=authorization_code&code=abc"
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if capturedAuth != "Basic dXNlcjpwYXNz" {
		t.Errorf("Authorization header not forwarded: got %q", capturedAuth)
	}
}

// TestRegisterASProxy_Token_UpstreamUnreachable verifies that a network-level
// failure contacting the upstream token endpoint returns 502 Bad Gateway.
func TestRegisterASProxy_Token_UpstreamUnreachable(t *testing.T) {
	cfg := &OAuthConfig{
		AuthorizationServerURL: "https://auth.example.com",
		ASProxy: &ASProxyConfig{
			Audience:         "my-api",
			UpstreamTokenURL: "http://127.0.0.1:1", // port 1 is always refused
		},
	}
	mux := http.NewServeMux()
	registerASProxy(mux, "http://localhost:8080", cfg)

	form := "grant_type=authorization_code&code=abc"
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("expected 502 for unreachable upstream, got %d", rr.Code)
	}
}
