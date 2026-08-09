package mcpserver

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

const (
	authorizeEndpointPath = "/authorize"
	tokenEndpointPath     = "/token"
)

// asProxyMeta is the minimal RFC 8414 Authorization Server Metadata document
// served at /.well-known/oauth-authorization-server when AS proxy is enabled.
type asProxyMeta struct {
	Issuer                        string   `json:"issuer"`
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	TokenEndpoint                 string   `json:"token_endpoint"`
	ResponseTypesSupported        []string `json:"response_types_supported"`
	GrantTypesSupported           []string `json:"grant_types_supported,omitempty"`
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported,omitempty"`
}

// registerASProxy registers the three OAuth Authorization Server proxy endpoints
// on mux:
//
//   - GET  /.well-known/oauth-authorization-server — RFC 8414 AS metadata
//   - GET  /authorize                              — injects audience, redirects upstream
//   - POST /token                                  — transparent proxy to upstream token endpoint
//
// This allows OAuth clients that do not send an audience parameter (e.g.
// Claude.ai) to work with Auth0, which requires audience to issue a JWT.
func registerASProxy(mux *http.ServeMux, baseURL string, cfg *OAuthConfig) {
	proxy := cfg.ASProxy

	upstreamAuthorize := proxy.UpstreamAuthorizeURL
	if upstreamAuthorize == "" {
		upstreamAuthorize = strings.TrimRight(cfg.AuthorizationServerURL, "/") + authorizeEndpointPath
	}
	upstreamToken := proxy.UpstreamTokenURL
	if upstreamToken == "" {
		upstreamToken = strings.TrimRight(cfg.AuthorizationServerURL, "/") + "/oauth/token"
	}

	// Pre-marshal the AS metadata so it can be served cheaply on every request.
	meta := asProxyMeta{
		Issuer:                        baseURL,
		AuthorizationEndpoint:         baseURL + authorizeEndpointPath,
		TokenEndpoint:                 baseURL + tokenEndpointPath,
		ResponseTypesSupported:        []string{"code"},
		GrantTypesSupported:           []string{"authorization_code", "refresh_token"},
		CodeChallengeMethodsSupported: []string{"S256"},
	}
	metaJSON, _ := json.Marshal(meta)

	// /.well-known/oauth-authorization-server — RFC 8414 AS metadata.
	// OAuth clients discover this after reading the protected resource metadata.
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(metaJSON)
	})
	mux.HandleFunc(authorizeEndpointPath, handleAuthorizeProxy(proxy.Audience, upstreamAuthorize))
	mux.HandleFunc(tokenEndpointPath, handleTokenProxy(proxy.ClientSecret, upstreamToken))
}

// handleAuthorizeProxy returns an http.HandlerFunc that injects the audience and
// offline_access scope before redirecting to the upstream authorization server.
func handleAuthorizeProxy(audience, upstreamAuthorize string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		if audience != "" {
			params.Set("audience", audience)
		}
		params.Set("scope", ensureOfflineAccess(params.Get("scope")))
		slog.Debug("oauth proxy: /authorize redirect", "audience", audience, "client_id", params.Get("client_id"), "scope", params.Get("scope"))
		target, err := url.Parse(upstreamAuthorize)
		if err != nil {
			slog.Debug("oauth proxy: /authorize parse error", "error", err)
			http.Error(w, "proxy misconfigured", http.StatusInternalServerError)
			return
		}
		target.RawQuery = params.Encode()
		http.Redirect(w, r, target.String(), http.StatusFound)
	}
}

// ensureOfflineAccess returns scope with "offline_access" appended if not already present.
func ensureOfflineAccess(scope string) string {
	if strings.Contains(scope, "offline_access") {
		return scope
	}
	if scope == "" {
		return "offline_access"
	}
	return scope + " offline_access"
}

// handleTokenProxy returns an http.HandlerFunc that proxies POST /token to the
// upstream token endpoint, optionally injecting a client secret.
func handleTokenProxy(clientSecret, upstreamToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			slog.Debug("oauth proxy: /token parse form error", "error", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		form := buildTokenForm(r.Form, clientSecret)
		slog.Debug("oauth proxy: /token request", "grant_type", form.Get("grant_type"), "client_id", form.Get("client_id"), "redirect_uri", form.Get("redirect_uri"))

		req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, upstreamToken,
			strings.NewReader(form.Encode()))
		if err != nil {
			slog.Debug("oauth proxy: /token request build error", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		// Forward Basic auth if the client sent it (confidential client auth).
		if authHeader := r.Header.Get("Authorization"); authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slog.Debug("oauth proxy: /token upstream error", "error", err, "upstream", upstreamToken)
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
			return
		}
		defer func() { _ = resp.Body.Close() }()

		body, _ := io.ReadAll(resp.Body)
		logTokenResponse(resp.StatusCode, body)

		for k, vs := range resp.Header {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
	}
}

// buildTokenForm copies the request form values and injects client_secret if
// configured and not already present.
func buildTokenForm(src url.Values, clientSecret string) url.Values {
	form := make(url.Values)
	for k, vs := range src {
		form[k] = vs
	}
	if clientSecret != "" && form.Get("client_secret") == "" {
		form.Set("client_secret", clientSecret)
	}
	return form
}

// logTokenResponse logs the upstream token response at debug level.
// On error responses the full body is logged; on success only the response keys are logged.
func logTokenResponse(statusCode int, body []byte) {
	if statusCode >= 400 {
		slog.Debug("oauth proxy: /token upstream error response", "status", statusCode, "body", string(body))
		return
	}
	var keys []string
	var respMap map[string]any
	if json.Unmarshal(body, &respMap) == nil {
		for k := range respMap {
			keys = append(keys, k)
		}
	}
	slog.Debug("oauth proxy: /token upstream success", "status", statusCode, "response_keys", keys)
}
