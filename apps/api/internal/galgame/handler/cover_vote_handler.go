package handler

// The best-cover vote BFF (wave 176) — and kungal's FIRST user-token write to
// the catalog.
//
// Every other catalog write this service makes is S2S: kungal authenticates
// with its own client credential and ASSERTS the acting user in the payload
// (edit_handler.go). This one forwards the user's OWN OAuth access token, the
// one the session already holds in Redis, and the catalog derives the actor and
// the site from it. The forum therefore asserts nothing here — it cannot vote
// for somebody, only relay somebody voting.
//
// The consequence a user can see is the SCOPE: a token minted before
// `catalog:edit` was added to the authorize request does not carry it, and no
// refresh can widen a grant. That 403 is the one denial with an action behind
// it ("log out and back in"), so it gets its own house code rather than the
// generic 你没有权限 — see errors.ErrReauthRequired.

import (
	stderrors "errors"
	"log/slog"
	"net/http"
	"strconv"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/middleware"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/response"

	"github.com/gofiber/fiber/v3"
)

// CoverVoteHandler votes on a galgame's covers. It holds the catalog client for
// the write and the galgame client for the gid → registry work id bridge: a
// kungal URL carries a gid, the catalog addresses works by registry id, and the
// two id spaces overlap — an untranslated id votes on a different game.
type CoverVoteHandler struct {
	catalog       *catalogclient.Client
	galgameClient *client.GalgameClient
}

func NewCoverVoteHandler(catalog *catalogclient.Client, galgameClient *client.GalgameClient) *CoverVoteHandler {
	return &CoverVoteHandler{catalog: catalog, galgameClient: galgameClient}
}

var errVoteDown = errors.New(errors.CodeBiz, "封面投票服务暂不可用", http.StatusServiceUnavailable)

// Vote — PUT /api/galgame/:gid/cover/:coverId/vote
//
// Casts or MOVES the user's single ballot for this game onto this cover.
func (h *CoverVoteHandler) Vote(c fiber.Ctx) error {
	return h.cast(c, true)
}

// Unvote — DELETE /api/galgame/:gid/cover/:coverId/vote
//
// Withdraws the ballot. Idempotent upstream: nothing to withdraw is still a
// success.
func (h *CoverVoteHandler) Unvote(c fiber.Ctx) error {
	return h.cast(c, false)
}

func (h *CoverVoteHandler) cast(c fiber.Ctx, vote bool) error {
	if _, appErr := middleware.MustGetUser(c); appErr != nil {
		return response.Error(c, appErr)
	}
	gid, appErr := parseGid(c)
	if appErr != nil {
		return response.Error(c, appErr)
	}
	coverID, err := strconv.ParseInt(c.Params("coverId"), 10, 64)
	if err != nil || coverID <= 0 {
		return response.Error(c, errors.ErrBadRequest("无效的封面 ID"))
	}
	// The browser never holds the access token (kun_session keeps it opaque in
	// Redis), which is exactly why this write has to traverse kungal at all.
	token := middleware.GetAccessToken(c)
	if token == "" {
		return response.Error(c, errors.ErrAuthExpired())
	}
	if h.catalog == nil || h.galgameClient == nil {
		return response.Error(c, errVoteDown)
	}
	workID, appErr := h.workIDOf(c, gid)
	if appErr != nil {
		return response.Error(c, appErr)
	}

	var result *catalogclient.CoverVoteResult
	var voteErr error
	if vote {
		result, voteErr = h.catalog.VoteCover(c.Context(), token, workID, coverID)
	} else {
		result, voteErr = h.catalog.UnvoteCover(c.Context(), token, workID, coverID)
	}
	if voteErr != nil {
		return coverVoteError(c, voteErr)
	}
	return response.OK(c, fiber.Map{
		"cover_id":   result.CoverID,
		"vote_count": result.VoteCount,
		"voted":      result.Voted,
	})
}

// workIDOf translates a kungal gid into the registry work id the catalog votes
// on. A gid the registry does not know has nothing to vote on.
func (h *CoverVoteHandler) workIDOf(c fiber.Ctx, gid int64) (int64, *errors.AppError) {
	ids, appErr := h.galgameClient.CatalogWorkIDs(c.Context(), []int{int(gid)})
	if appErr != nil {
		return 0, appErr
	}
	workID, ok := ids[int(gid)]
	if !ok {
		return 0, errors.ErrNotFound("条目不存在")
	}
	return workID, nil
}

// coverVoteError maps a user-plane error onto the house envelope.
//
// The three that matter are told apart deliberately: an insufficient scope is
// the user's OWN session being too old and is fixed by re-login; a 401 is a
// dead token and follows the site's ordinary session-expiry path; a 404 is a
// cover that is not on this work and passes through as such.
func coverVoteError(c fiber.Ctx, err error) error {
	var apiErr *catalogclient.UserAPIError
	switch {
	case stderrors.Is(err, catalogclient.ErrInsufficientScope):
		return response.Error(c, errors.ErrReauthRequired(
			"投票需要新的授权，请退出登录后重新登录以授予该权限"))
	case stderrors.Is(err, catalogclient.ErrUnauthorized):
		return response.Error(c, errors.ErrAuthExpired())
	case stderrors.Is(err, catalogclient.ErrNotFound):
		return response.Error(c, errors.ErrNotFound("条目或封面不存在"))
	case stderrors.Is(err, catalogclient.ErrNotConfigured):
		return response.Error(c, errVoteDown)
	case stderrors.As(err, &apiErr):
		if apiErr.Status == http.StatusForbidden {
			return response.Error(c, errors.ErrForbidden("你没有权限执行此操作"))
		}
		slog.Error("galgame cover vote: upstream error",
			"status", apiErr.Status, "code", apiErr.Code, "msg", apiErr.Message)
		return response.Error(c, errVoteDown)
	default:
		slog.Warn("galgame cover vote: catalog unreachable", "error", err)
		return response.Error(c, errVoteDown)
	}
}
