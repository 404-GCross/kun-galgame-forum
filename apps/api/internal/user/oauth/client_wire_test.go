package oauth

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// resp builds a minimal *http.Response with the given status + JSON body for
// exercising decodeEnvelope directly.
func resp(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

// TestDecodeEnvelopeWireTolerant pins the reader against both wire formats the
// OAuth server can serve, and — more importantly — pins that the classification
// callers act on (banned / refresh-dead / transient) survives the cutover.
func TestDecodeEnvelopeWireTolerant(t *testing.T) {
	t.Run("envelope success returns inner data", func(t *testing.T) {
		data, err := decodeEnvelope(resp(200, `{"code":0,"message":"成功","data":{"access_token":"tok"}}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(data) != `{"access_token":"tok"}` {
			t.Fatalf("got data %s", data)
		}
	})

	t.Run("standard-wire success returns whole body", func(t *testing.T) {
		const body = `{"access_token":"tok","token_type":"Bearer","expires_in":900}`
		data, err := decodeEnvelope(resp(200, body))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(data) != body {
			t.Fatalf("got data %s", data)
		}
	})

	t.Run("enveloped ban is still a ban", func(t *testing.T) {
		_, err := decodeEnvelope(resp(403, `{"code":10014,"message":"账号已封禁"}`))
		if !IsBanned(err) {
			t.Fatalf("expected IsBanned, got %v", err)
		}
	})

	// RFC 6750 has no error code meaning "banned", so the OAuth server signals
	// it with a bare 403. Keying on the status is what keeps the distinct
	// banned page instead of degrading it to a generic re-login.
	t.Run("RFC 6750 403 is still a ban", func(t *testing.T) {
		_, err := decodeEnvelope(resp(403, `{"error":"invalid_token","error_description":"account banned"}`))
		if !IsBanned(err) {
			t.Fatalf("expected IsBanned, got %v", err)
		}
	})

	// Without the invalid_token mapping this lands on code 0, which IsTransient
	// reads as retryable — kungal would keep a dead session alive forever.
	t.Run("standard-wire invalid_token → refresh dead, not transient", func(t *testing.T) {
		_, err := decodeEnvelope(resp(401, `{"error":"invalid_token","error_description":"token expired"}`))
		if !IsRefreshTokenDead(err) {
			t.Fatalf("expected IsRefreshTokenDead, got %v", err)
		}
		if IsTransient(err) {
			t.Fatalf("invalid_token must not be transient")
		}
	})

	t.Run("standard-wire invalid_grant → refresh dead", func(t *testing.T) {
		_, err := decodeEnvelope(resp(400, `{"error":"invalid_grant","error_description":"refresh token revoked"}`))
		if !IsRefreshTokenDead(err) {
			t.Fatalf("expected IsRefreshTokenDead, got %v", err)
		}
	})

	t.Run("standard-wire unknown error stays transient", func(t *testing.T) {
		_, err := decodeEnvelope(resp(400, `{"error":"invalid_request","error_description":"bad"}`))
		if IsRefreshTokenDead(err) {
			t.Fatalf("invalid_request must not be refresh-dead")
		}
		if !IsTransient(err) {
			t.Fatalf("expected IsTransient for an unknown error")
		}
	})

	t.Run("5xx stays transient", func(t *testing.T) {
		_, err := decodeEnvelope(resp(503, `{"error":"temporarily_unavailable"}`))
		if !IsTransient(err) {
			t.Fatalf("expected IsTransient for 503")
		}
	})
}
