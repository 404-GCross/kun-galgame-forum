package oauth

import (
	"bytes"
	stderrors "errors"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"kun-galgame-api/pkg/config"
)

// oauthHTTPTimeout caps every OAuth-server roundtrip (token exchange /
// refresh / revoke / userinfo) at 10s. Without this every login + every
// authenticated request would block indefinitely if the OAuth server hung,
// since the four hot paths all run synchronously in the request hot path.
const oauthHTTPTimeout = 10 * time.Second

// OAuth envelope error codes that callers care about. The full list is in
// docs/oauth/api-reference.md §错误码速查; here we only name the ones
// kungal branches on (banned vs. refresh-token-expired vs. invalid-grant vs.
// everything-else-treated-as-transient).
const (
	CodeAccountBanned        = 10014 // HTTP 403
	CodeRefreshTokenExpired  = 10003 // HTTP 401 — needs user to fully re-login
	CodeInvalidToken         = 10002 // HTTP 401 — bad token / client_id mismatch
	CodeInvalidGrant         = 15005 // HTTP 400 — client missing `refresh_token` grant
	CodeInvalidClientSecret  = 15008 // HTTP 400 — confidential client misconfigured
)

// Error is a structured OAuth-server error. It captures the envelope code
// (when the response body was parseable) so middleware can branch on
// "banned" vs "transient" vs "client misconfig". Non-OAuth failures
// (network, body unreadable) are also wrapped in Error with Code == 0;
// IsTransient treats those as retryable.
type Error struct {
	Code       int    // OAuth envelope code; 0 when unparseable
	HTTPStatus int    // 0 for network errors
	Message    string // OAuth-supplied message (best effort)
}

func (e *Error) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("oauth: code=%d http=%d msg=%q", e.Code, e.HTTPStatus, e.Message)
	}
	return fmt.Sprintf("oauth: http=%d msg=%q", e.HTTPStatus, e.Message)
}

// IsBanned reports whether err is a 10014 "account banned" response.
// Callers should surface this distinctly (e.g. show a banned page rather
// than redirecting to /login, since logging in again hits the same error).
func IsBanned(err error) bool {
	var oe *Error
	return stderrors.As(err, &oe) && oe.Code == CodeAccountBanned
}

// IsRefreshTokenDead reports whether err means the refresh token is
// permanently unusable — the user must log in again from scratch. Covers
// "token expired", "invalid token" (e.g. client_id mismatch), and
// "invalid grant" (e.g. refresh_token grant not allowed for this client).
func IsRefreshTokenDead(err error) bool {
	var oe *Error
	if !stderrors.As(err, &oe) {
		return false
	}
	switch oe.Code {
	case CodeRefreshTokenExpired, CodeInvalidToken, CodeInvalidGrant, CodeInvalidClientSecret:
		return true
	}
	return false
}

// IsTransient reports whether err looks recoverable on a retry (network
// glitch, OAuth restart, 5xx, unparseable body). The middleware uses this
// to decide whether to keep the local session alive across the failure.
func IsTransient(err error) bool {
	var oe *Error
	if !stderrors.As(err, &oe) {
		// Plain network errors (rare path; usually wrapped) — treat as transient.
		return true
	}
	if oe.HTTPStatus == 0 || oe.HTTPStatus >= 500 {
		return true
	}
	// Unparseable envelope on a 4xx → can't tell, lean transient.
	if oe.Code == 0 {
		return true
	}
	return false
}

// Client calls the OAuth server via HTTP.
// It is a thin transport layer: it performs raw HTTP calls and decodes the
// standard {code, message, data} wrapper used by the OAuth server. No
// business logic lives here.
type Client struct {
	cfg        config.OAuthConfig
	httpClient *http.Client
}

// NewClient constructs an OAuth HTTP client with the given configuration.
// The HTTP client carries a per-request timeout so a hung OAuth server can't
// stall login / token refresh / logout indefinitely.
func NewClient(cfg config.OAuthConfig) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: oauthHTTPTimeout},
	}
}

// decodeEnvelope reads resp.Body, parses the standard envelope, and returns
// either the data payload (success) or a structured *Error (failure).
//
// "Success" means HTTP 200 AND envelope.Code == 0 AND envelope.Data
// non-empty. Anything else becomes a typed *Error so callers can branch
// on Code via IsBanned / IsRefreshTokenDead / IsTransient.
func decodeEnvelope(resp *http.Response) (json.RawMessage, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &Error{HTTPStatus: resp.StatusCode, Message: "读取响应体失败: " + err.Error()}
	}

	// Tolerant reader (expand→contract for the OAuth server's standard-wire
	// cutover): accept BOTH the legacy {code,message,data} envelope AND the
	// spec-compliant top-level JSON. The `code` key is present iff it's the
	// envelope; a standard error object carries `error`/`error_description`.
	var p struct {
		Code             *int            `json:"code"` // pointer: nil when absent (standard)
		Message          string          `json:"message"`
		Data             json.RawMessage `json:"data"`
		Error            string          `json:"error"`
		ErrorDescription string          `json:"error_description"`
	}
	if jerr := json.Unmarshal(body, &p); jerr != nil {
		// Unparseable body — status known, code unknown. Treated as transient.
		return nil, &Error{
			HTTPStatus: resp.StatusCode,
			Message:    fmt.Sprintf("解析响应失败: %v, body=%s", jerr, truncateBody(body)),
		}
	}

	if p.Code != nil {
		// Legacy {code,message,data} envelope.
		if resp.StatusCode == http.StatusOK && *p.Code == 0 && len(p.Data) > 0 {
			return p.Data, nil
		}
		return nil, &Error{Code: *p.Code, HTTPStatus: resp.StatusCode, Message: p.Message}
	}

	// Standard-wire response: the whole body IS the payload.
	if resp.StatusCode == http.StatusOK {
		return json.RawMessage(body), nil
	}
	// Standard OAuth error object {error, error_description}. Map the error
	// string back to the envelope code the callers branch on, so the cutover
	// preserves the "refresh dead vs transient" decision.
	msg := p.ErrorDescription
	if msg == "" {
		msg = p.Error
	}
	return nil, &Error{Code: oauthErrToCode(p.Error), HTTPStatus: resp.StatusCode, Message: msg}
}

// oauthErrToCode maps a standard RFC 6749 error string to the legacy envelope
// code kungal branches on. invalid_grant/invalid_client mean the refresh token
// is unusable (→ re-login); anything else maps to 0 (unknown → transient).
// A banned user surfaces as invalid_grant on the token endpoint and is then
// re-blocked at /userinfo (which still returns the enveloped 10014).
func oauthErrToCode(errStr string) int {
	switch errStr {
	case "invalid_grant":
		return CodeInvalidGrant
	case "invalid_client":
		return CodeInvalidClientSecret
	default:
		return 0
	}
}

// truncateBody trims a response body to a sane length for error messages
// so logs don't blow up if OAuth returns a giant HTML error page.
func truncateBody(b []byte) string {
	const max = 256
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "...(truncated)"
}

// TokenResponse represents the token data inside the OAuth response wrapper.
// /oauth/token returns { code: 0, message: "成功", data: { access_token, ... } }
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// UserInfo represents the OAuth userinfo payload.
//
// IMPORTANT: kungal post-migration relies on the integer `id` (= OAuth
// users.id) and the `roles` array. The OIDC userinfo standard only
// requires `sub` (UUID). The OAuth team must extend /oauth/userinfo to
// include `id` and `roles` so kungal can derive its userID + admin role
// without a second round-trip.
type UserInfo struct {
	ID        int      `json:"id"`
	Sub       string   `json:"sub"`
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	Picture   string   `json:"picture"`
	Roles     []string `json:"roles"`
	UpdatedAt int64    `json:"updated_at"`
}

// ExchangeCode exchanges an authorization code for access/refresh tokens.
// Returns a typed *Error on OAuth-side failures (see Error / IsBanned /
// IsTransient).
func (c *Client) ExchangeCode(code, codeVerifier string) (*TokenResponse, error) {
	payload := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  c.cfg.RedirectURI,
		"client_id":     c.cfg.ClientID,
		"client_secret": c.cfg.ClientSecret,
		"code_verifier": codeVerifier,
	}
	data, err := c.postEnvelope("/oauth/token", payload)
	if err != nil {
		return nil, err
	}
	var tok TokenResponse
	if jerr := json.Unmarshal(data, &tok); jerr != nil {
		return nil, &Error{Message: "解析 token 响应失败: " + jerr.Error()}
	}
	if tok.AccessToken == "" {
		return nil, &Error{Message: "token 响应缺 access_token"}
	}
	return &tok, nil
}

// FetchUserInfo retrieves the OAuth user info using an access token.
// Returns a typed *Error on OAuth-side failures.
func (c *Client) FetchUserInfo(accessToken string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", c.cfg.ServerURL+"/oauth/userinfo", nil)
	if err != nil {
		return nil, &Error{Message: "创建 userinfo 请求失败: " + err.Error()}
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &Error{Message: "请求 userinfo 失败: " + err.Error()}
	}
	defer resp.Body.Close()

	data, derr := decodeEnvelope(resp)
	if derr != nil {
		return nil, derr
	}
	var info UserInfo
	if jerr := json.Unmarshal(data, &info); jerr != nil {
		return nil, &Error{Message: "解析 userinfo 响应失败: " + jerr.Error()}
	}
	return &info, nil
}

// RevokeToken revokes a refresh token against the OAuth server.
func (c *Client) RevokeToken(refreshToken string) error {
	payload, err := json.Marshal(map[string]string{"token": refreshToken})
	if err != nil {
		return fmt.Errorf("序列化 revoke 请求失败: %w", err)
	}
	req, err := http.NewRequest("POST", c.cfg.ServerURL+"/oauth/revoke", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("创建 revoke 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// RefreshOAuthToken refreshes the OAuth tokens using the refresh token.
// Returns a typed *Error on OAuth-side failures — middleware switches on
// IsBanned / IsRefreshTokenDead / IsTransient to decide whether to
// preserve or invalidate the local session.
func (c *Client) RefreshOAuthToken(refreshToken string) (*TokenResponse, error) {
	payload := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     c.cfg.ClientID,
		"client_secret": c.cfg.ClientSecret,
	}
	data, err := c.postEnvelope("/oauth/token", payload)
	if err != nil {
		return nil, err
	}
	var tok TokenResponse
	if jerr := json.Unmarshal(data, &tok); jerr != nil {
		return nil, &Error{Message: "解析刷新响应失败: " + jerr.Error()}
	}
	if tok.AccessToken == "" {
		return nil, &Error{Message: "刷新响应缺 access_token"}
	}
	return &tok, nil
}

// PatchAuthMe calls PATCH /auth/me to update the authenticated user's
// profile. The body is any JSON-serialisable struct/map carrying the
// fields the user wants to change — OAuth treats omitted fields as
// "leave unchanged" so kungal can forward partial updates. Returns the
// raw refreshed user payload so callers can pass it back to the
// browser verbatim.
//
// docs/oauth/02-user-profile.md §PATCH /auth/me.
func (c *Client) PatchAuthMe(accessToken string, body any) (json.RawMessage, error) {
	payload, jerr := json.Marshal(body)
	if jerr != nil {
		return nil, &Error{Message: "序列化 PATCH /auth/me 请求失败: " + jerr.Error()}
	}
	req, rerr := http.NewRequest("PATCH", c.cfg.ServerURL+"/auth/me", bytes.NewReader(payload))
	if rerr != nil {
		return nil, &Error{Message: "创建 PATCH /auth/me 请求失败: " + rerr.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, derr := c.httpClient.Do(req)
	if derr != nil {
		return nil, &Error{Message: "请求 PATCH /auth/me 失败: " + derr.Error()}
	}
	defer resp.Body.Close()
	return decodeEnvelope(resp)
}

// UploadAvatar calls POST /auth/me/avatar with a pre-built multipart
// body. OAuth pipes the bytes to image_service, writes the resulting
// hash to the user row, and returns the image_service upload result
// (hash + variant URLs). kungal forwards the body unchanged.
//
// contentType is the value of the incoming request's Content-Type
// header (must carry the multipart boundary).
//
// docs/oauth/02-user-profile.md §POST /auth/me/avatar.
func (c *Client) UploadAvatar(accessToken string, body []byte, contentType string) (json.RawMessage, error) {
	req, rerr := http.NewRequest("POST", c.cfg.ServerURL+"/auth/me/avatar", bytes.NewReader(body))
	if rerr != nil {
		return nil, &Error{Message: "创建 POST /auth/me/avatar 请求失败: " + rerr.Error()}
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, derr := c.httpClient.Do(req)
	if derr != nil {
		return nil, &Error{Message: "请求 POST /auth/me/avatar 失败: " + derr.Error()}
	}
	defer resp.Body.Close()
	return decodeEnvelope(resp)
}

// postEnvelope POSTs a JSON-serialized payload to OAuth and decodes the
// standard envelope. Used by ExchangeCode and RefreshOAuthToken — both
// hit /oauth/token with the same wire shape but different grant_type.
func (c *Client) postEnvelope(path string, payload any) (json.RawMessage, error) {
	body, jerr := json.Marshal(payload)
	if jerr != nil {
		return nil, &Error{Message: "序列化请求失败: " + jerr.Error()}
	}
	req, rerr := http.NewRequest("POST", c.cfg.ServerURL+path, bytes.NewReader(body))
	if rerr != nil {
		return nil, &Error{Message: "创建请求失败: " + rerr.Error()}
	}
	req.Header.Set("Content-Type", "application/json")
	resp, derr := c.httpClient.Do(req)
	if derr != nil {
		return nil, &Error{Message: "请求 OAuth 失败: " + derr.Error()}
	}
	defer resp.Body.Close()
	return decodeEnvelope(resp)
}
