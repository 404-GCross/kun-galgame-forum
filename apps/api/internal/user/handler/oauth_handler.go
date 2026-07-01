package handler

import (
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/service"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/utils"

	"github.com/gofiber/fiber/v3"
)

type OAuthHandler struct {
	authService *service.AuthService
	secure      bool // true in production (HTTPS), false in dev (HTTP)
}

func NewOAuthHandler(authService *service.AuthService, isProd bool) *OAuthHandler {
	return &OAuthHandler{authService: authService, secure: isProd}
}

// Callback handles the OAuth code exchange: code → token → userinfo → session.
// POST /api/auth/oauth/callback
//
// SECURITY: never log the request body — it carries the one-shot authorization
// code and the PKCE code_verifier, both of which are short-lived credentials.
// Logging them (even at Debug level) leaks them to any log sink and creates a
// replay window if an attacker reads the log before the token exchange.
func (h *OAuthHandler) Callback(c fiber.Ctx) error {
	var req dto.OAuthCallbackRequest
	if err := utils.ParseAndValidate(c, &req); err != nil {
		return response.Error(c, err)
	}

	session, appErr := h.authService.OAuthCallback(c.Context(), &req)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	c.Cookie(&fiber.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    session.Token,
		MaxAge:   int(middleware.SessionTTL.Seconds()), // sliding; renewed on activity
		HTTPOnly: true,
		Secure:   h.secure,
		SameSite: "Lax",
		Path:     "/",
	})

	return response.OK(c, session.User)
}

// Logout clears the session.
// POST /api/auth/logout
func (h *OAuthHandler) Logout(c fiber.Ctx) error {
	token := c.Cookies(middleware.SessionCookieName)
	if token != "" {
		_ = h.authService.Logout(c.Context(), token)
	}

	c.Cookie(&fiber.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   h.secure,
		SameSite: "Lax",
		Path:     "/",
	})

	return response.OKMessage(c, "已登出")
}

// Me returns the current authenticated user's profile.
// GET /api/auth/me
func (h *OAuthHandler) Me(c fiber.Ctx) error {
	user, err := middleware.MustGetUser(c)
	if err != nil {
		return response.Error(c, err)
	}

	profile, appErr := h.authService.GetProfile(c.Context(), user.ID)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	// Sub isn't in the userclient brief; fill it from the session identity so
	// /auth/me carries the account's OAuth UUID (needed for account switching).
	profile.Sub = user.Sub

	return response.OK(c, profile)
}
