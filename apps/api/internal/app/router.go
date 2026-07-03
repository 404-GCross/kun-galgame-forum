package app

import (
	"strings"

	"kun-galgame-api/internal/middleware"

	"github.com/gofiber/fiber/v3"
	fiberCors "github.com/gofiber/fiber/v3/middleware/cors"
)

func (a *App) setupRoutes() {
	// Session cookies are Secure (HTTPS-only) in prod, plain in dev over HTTP.
	// Drives renewSlidingSession's re-issued cookie; mirrors the login
	// handler's secure flag (NewOAuthHandler(_, cfg.Server.Mode == "prod")).
	middleware.SecureCookies = a.Config.Server.Mode == "prod"

	a.Fiber.Use(fiberCors.New(middleware.CORS(a.Config.CORS.AllowOrigins)))

	// Liveness probe for the container HEALTHCHECK (`server healthcheck`) and
	// compose depends_on gates. Plain 200 — deliberately does NOT touch DB/Redis
	// so a transient backing-store blip can't flap the container as unhealthy.
	a.Fiber.Get("/healthz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	api := a.Fiber.Group("/api")

	// ════════════════════════════════════════════
	// PUBLIC routes (no auth required)
	// Must be registered BEFORE authed group to avoid
	// Fiber Group("") middleware intercepting them.
	// ════════════════════════════════════════════

	api.Get("/home", a.HomeHandler.GetHome)

	// Auth (public). Identity changes (password / email / username / bio /
	// avatar) all live in the OAuth admin UI now — kungal owns nothing
	// that needs an /auth/* email-code flow.
	auth := api.Group("/auth")
	auth.Post("/oauth/callback", a.OAuthHandler.Callback)
	auth.Post("/logout", a.OAuthHandler.Logout)

	// User (authenticated, fixed paths — registered before :id to avoid conflicts).
	// Bio / username / email / ban / delete were here pre-OAuth; all moved to OAuth.
	userAuth := middleware.Auth(a.Redis, a.OAuthClient)
	// No rate limiter: the once-per-day gate is the `daily_check_in` flag
	// (reset at calendar midnight by the daily cron) enforced atomically in
	// StateRepository.CheckIn. A 24h-rolling rate limiter here used to block
	// legitimate next-day check-ins (its window spilled past midnight) and
	// masked the real "已签到" message with a generic "操作过于频繁" 400.
	api.Post("/user/check-in", userAuth, a.UserHandler.CheckIn)
	api.Get("/user/status", userAuth, a.UserHandler.GetStatus)
	// Unified moemoepoint ledger (own only) — fixed path, before /user/:id.
	api.Get("/user/moemoepoint/log", userAuth, a.UserHandler.GetMoemoepointLog)
	// @mention autocomplete — fixed path, MUST stay before /user/:id (else
	// "search" is captured as :id). Proxies OAuth /users/search (not cached).
	api.Get("/user/search", userAuth, a.UserHandler.SearchMention)

	// Creator-role application: forum checks its eligibility (wiki PR/galgame
	// stats + own 简评), then files on the central OAuth queue. Role grant +
	// admin review live in OAuth (contract owned there, not yet mirrored here).
	api.Get("/user/creator/status", userAuth, a.CreatorHandler.Status)
	api.Post("/user/creator/apply", userAuth, a.CreatorHandler.Apply)

	// Self-edit endpoints — proxy to OAuth /auth/me family, the session-
	// stored bearer is attached inside each handler. See
	// docs/oauth/02-user-profile.md.
	api.Put("/user/bio", userAuth, a.UserProfileHandler.UpdateBio)
	api.Put("/user/username", userAuth, a.UserProfileHandler.UpdateUsername)
	api.Post("/user/avatar", userAuth, a.UserProfileHandler.UploadAvatar)

	// User (public, parameterized — AFTER fixed paths)
	api.Get("/user/:id/floating", a.UserHandler.GetFloatingCard)
	api.Get("/user/:id", a.UserHandler.GetProfile)
	api.Get("/user/:id/galgames", a.UserHandler.GetUserGalgames)
	api.Get("/user/:id/galgame-comments", a.UserHandler.GetUserGalgameComments)
	api.Get("/user/:id/topics", a.UserHandler.GetUserTopics)
	api.Get("/user/:id/replies", a.UserHandler.GetUserReplies)
	api.Get("/user/:id/comments", a.UserHandler.GetUserComments)
	api.Get("/user/:id/resources", a.UserHandler.GetUserResources)
	api.Get("/user/:id/ratings", a.UserHandler.GetUserRatings)
	api.Get("/user/:id/toolsets", a.ToolsetHandler.GetUserToolsets)

	// Ranking (public)
	api.Get("/ranking/galgame", a.RankingHandler.GetGalgameRanking)
	api.Get("/ranking/topic", a.RankingHandler.GetTopicRanking)
	api.Get("/ranking/user", a.RankingHandler.GetUserRanking)

	// Section & Category (public)
	api.Get("/section", a.SectionHandler.GetSectionTopics)
	api.Get("/category", a.SectionHandler.GetCategories)

	// Doc (public reads)
	api.Get("/doc/article", a.DocArticleHandler.GetArticles)
	api.Get("/doc/article/:slug", a.DocArticleHandler.GetArticleBySlug)
	api.Get("/doc/category", a.DocCategoryHandler.GetCategories)
	api.Get("/doc/tag", a.DocTagHandler.GetTags)

	// Website (public reads)
	api.Get("/website-category/:name", a.WebsiteCategoryHandler.GetWebsiteCategory)
	api.Get("/website-tag", a.WebsiteTagHandler.GetWebsiteTags)
	api.Get("/website-tag/:name", a.WebsiteTagHandler.GetWebsiteTagDetail)

	// Update (public reads)
	api.Get("/update/history", a.UpdateHandler.GetHistory)
	api.Get("/update/todo", a.UpdateHandler.GetTodos)

	// Friend links (public read — rendered on /friend-links)
	api.Get("/friend-link", a.FriendLinkHandler.List)

	// Activity (public)
	api.Get("/activity", a.ActivityHandler.GetActivity)
	api.Get("/activity/tab", a.ActivityHandler.GetTab)
	api.Get("/activity/timeline", a.ActivityHandler.GetTimeline)

	// Galgame rating — moved to optAuth group below so service handlers
	// receive a non-zero `currentUserID` for logged-in callers and can
	// hydrate the per-row `isLiked` / liked-by-me state. Leaving them
	// here in the `api` group made optionalUID return 0 unconditionally,
	// which silently broke the K-PR `FindLikedSet` batch fix.

	// Resource topics (public, same as topic but filtered to resource sections)
	api.Get("/resource", a.TopicHandler.GetResourceList)

	// Search (public)
	api.Get("/search", a.SearchHandler.Search)

	// RSS (public)
	api.Get("/rss/topic", a.RSSHandler.GetTopicRSS)
	api.Get("/rss/galgame", a.RSSHandler.GetGalgameRSS)

	// Toolset resource detail (public)
	api.Get("/toolset/:id/resource/detail", a.ToolsetResourceHandler.GetResourceDetail)

	// Galgame wiki proxies (public reads)
	api.Get("/galgame", a.GalgameHandler.GetList)
	// `/galgame/check` (VNDB-ID precheck) was retired when the publish
	// wizard moved to name-based search — no FE caller remains. Removed
	// from registration; re-add only if a downstream brings the check
	// flow back.
	// /galgame/mine and /galgame/search/wizard MUST be registered BEFORE
	// /galgame/:gid below — Fiber matches by registration order and a
	// catch-all `:gid` happily binds to the literal "mine" / "search",
	// which would route to GetDetail and then fail with Atoi("mine").
	// Both endpoints require auth; inline userAuth here so we don't have
	// to predeclare the `authed` group above the optAuth section.
	api.Get("/galgame/mine", userAuth, a.GalgameSubmissionHandler.ListMine)
	api.Get(
		"/galgame/search/wizard",
		userAuth,
		a.GalgameSubmissionHandler.SearchWithPending,
	)
	// Galgame 发售月历 — proxies wiki /galgame/calendar(+pending/tba), enriched
	// with forum-local card data. Public + SFW-default. Same registration-order
	// rule as /galgame/mine above: the literal "calendar" segment must come
	// before the /galgame/:gid catch-all or it'd bind as gid="calendar".
	api.Get("/galgame/calendar", a.GalgameCalendarHandler.GetMonth)
	api.Get("/galgame/calendar/pending", a.GalgameCalendarHandler.GetPending)
	api.Get("/galgame/calendar/tba", a.GalgameCalendarHandler.GetTBA)
	api.Get("/galgame/calendar/upcoming", a.GalgameCalendarHandler.GetUpcoming)
	api.Get("/galgame/:gid/revisions", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame/:gid/revisions/:rev", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame/:gid/revisions/:rev/diff", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame/:gid/prs", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame/:gid/prs/:id", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame/:gid/links", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame/:gid/aliases", a.GalgameWikiHandler.ProxyGet)
	// `/galgame/:gid/contributors` is unused — the FE contributor view
	// reads from detail's embedded `contributor[]` array now
	// (apps/web/.../components/galgame/contributor/Container.vue notes
	// the removal). Drop the proxy route to keep the public surface
	// honest; re-add if a downstream rebuilds the standalone view.
	// NOTE: galgame detail sub-routes (/pr/all, /link/all, etc.) are
	// registered in the optAuth group below to avoid Fiber route shadowing
	// by /galgame/:gid.
	api.Get("/galgame-tag", a.GalgameEntityHandler.GetTagList)
	api.Get("/galgame-tag/search", a.GalgameEntityHandler.SearchTags)
	api.Get("/galgame-tag/multi", a.GalgameEntityHandler.GetMultiTagGalgames)
	api.Get("/galgame-tag/:name", a.GalgameEntityHandler.GetTagDetail)
	api.Get("/galgame-official", a.GalgameEntityHandler.GetOfficialList)
	api.Get("/galgame-official/search", a.GalgameEntityHandler.SearchOfficials)
	api.Get("/galgame-official/:name", a.GalgameEntityHandler.GetOfficialDetail)
	api.Get("/galgame-engine", a.GalgameEntityHandler.GetEngineList)
	api.Get("/galgame-engine/:name", a.GalgameEntityHandler.GetEngineDetail)
	api.Get("/galgame-series", a.GalgameEntityHandler.GetSeriesList)
	api.Get("/galgame-series/search", a.GalgameWikiHandler.ProxyGet)
	api.Get("/galgame-series/:id", a.GalgameEntityHandler.GetSeriesDetail)
	// `/galgame-resource` list — moved to optAuth below (FE list cards
	// show the heart icon and need the viewer's per-row like state).
	// `/galgame-rating/all` stays here in the public group: the rating
	// Card.vue doesn't render a like toggle, so optAuth would just be
	// dead-weight middleware on a high-traffic list endpoint.
	api.Get("/galgame-rating/all", a.GalgameRatingHandler.GetAllRatings)

	// ════════════════════════════════════════════
	// OPTIONAL AUTH routes (public but attach user if logged in)
	// ════════════════════════════════════════════

	optAuth := api.Group("", middleware.OptionalAuth(a.Redis, a.OAuthClient))
	// Galgame resource list + detail family — all need optionalUID for
	// the per-row `isLiked` flag (batch query via FindLikedSet on lists,
	// single query on detail).
	optAuth.Get("/galgame-resource", a.GalgameResourceHandler.GetResourceList)
	optAuth.Get("/galgame-resource/:id/detail", a.GalgameResourceHandler.GetResourceDownloadDetail)
	optAuth.Get("/galgame-resource/:id", a.GalgameResourceHandler.GetResourceDetail)

	// Galgame rating detail — currentUserID feeds containsInt against
	// likerIDs to populate the per-viewer `isLiked` flag the Like.vue
	// component renders.
	optAuth.Get("/galgame-rating/:id", a.GalgameRatingHandler.GetRatingDetail)

	// Topic (optional auth for interaction status)
	// Private per-user topic drafts. Registered here (on `api`, with an explicit
	// auth middleware) BEFORE the /topic/:tid routes below because Fiber matches
	// in REGISTRATION ORDER: a later static /topic/draft would otherwise be
	// captured by the earlier /topic/:tid param route (tid="draft" → 400).
	topicDraftAuth := middleware.Auth(a.Redis, a.OAuthClient)
	api.Get("/topic/draft", topicDraftAuth, a.TopicDraftHandler.List)
	api.Post("/topic/draft", topicDraftAuth, a.TopicDraftHandler.Save)
	api.Get("/topic/draft/:id", topicDraftAuth, a.TopicDraftHandler.Get)
	api.Delete("/topic/draft/:id", topicDraftAuth, a.TopicDraftHandler.Delete)

	optAuth.Get("/topic", a.TopicHandler.GetList)
	optAuth.Get("/topic/:tid", a.TopicHandler.GetDetail)
	optAuth.Get("/topic/:tid/upvotes", a.TopicHandler.GetUpvotes)
	optAuth.Get("/topic/:tid/reaction/history", a.TopicHandler.GetTopicReactionHistory)
	optAuth.Get("/topic/:tid/reply", a.ReplyHandler.GetReplies)
	optAuth.Get("/topic/:tid/reply/detail", a.ReplyHandler.GetReplyDetail)
	optAuth.Get("/topic/:tid/reply/locate", a.ReplyHandler.GetReplyLocate)
	optAuth.Get("/topic/:tid/poll/topic", a.PollHandler.GetPollsByTopic)
	optAuth.Get("/topic/:tid/poll/log", a.PollHandler.GetVoteLog)

	// Galgame detail sub-routes (MUST be before /:gid to avoid shadowing)
	optAuth.Get("/galgame/:gid/resource/all", a.GalgameResourceHandler.GetGalgameResources)
	optAuth.Get("/galgame/:gid/comment/all", a.GalgameCommentHandler.GetComments)
	optAuth.Get("/galgame/:gid/comment/thread/:rootId", a.GalgameCommentHandler.GetCommentThread)
	optAuth.Get("/galgame/:gid/pr/all", a.GalgameWikiHandler.GetGalgamePRs)
	optAuth.Get("/galgame/:gid/link/all", a.GalgameWikiHandler.GetGalgameLinks)
	optAuth.Get("/galgame/:gid/history/all", a.GalgameWikiHandler.GetGalgameHistory)
	optAuth.Get("/galgame/:gid", a.GalgameHandler.GetDetail)

	// Website (optional auth for like/favorite status)
	optAuth.Get("/website", a.WebsiteHandler.GetWebsites)
	optAuth.Get("/website/:domain/comment", a.WebsiteCommentHandler.GetComments)
	optAuth.Get("/website/:domain", a.WebsiteHandler.GetWebsiteDetail)

	// Toolset (optional auth for practicality "mine" field)
	optAuth.Get("/toolset", a.ToolsetHandler.GetList)
	optAuth.Get("/toolset/:id", a.ToolsetHandler.GetDetail)
	optAuth.Get("/toolset/:id/practicality", a.ToolsetPracticalityHandler.GetPracticality)
	optAuth.Get("/toolset/:id/comment/all", a.ToolsetCommentHandler.GetComments)

	// ════════════════════════════════════════════
	// AUTHENTICATED routes (require valid session)
	// ════════════════════════════════════════════

	authed := api.Group("", middleware.Auth(a.Redis, a.OAuthClient))
	authed.Get("/auth/me", a.OAuthHandler.Me)

	// Topic (authenticated)
	// Static /topic/interactions/mine wins over /topic/:tid (static segments beat
	// the param route), so feed cards hydrate the viewer's own 收藏 + reactions.
	authed.Get("/topic/interactions/mine", a.TopicHandler.MyInteractions)
	authed.Post("/topic", a.TopicHandler.Create)
	authed.Put("/topic/:tid", a.TopicHandler.Update)
	authed.Put("/topic/:tid/like", a.TopicHandler.ToggleLike)
	authed.Put("/topic/:tid/dislike", a.TopicHandler.ToggleDislike)
	authed.Put("/topic/:tid/upvote", a.TopicHandler.Upvote)
	authed.Put("/topic/:tid/favorite", a.TopicHandler.ToggleFavorite)
	authed.Put("/topic/:tid/reaction", a.TopicHandler.ToggleReaction)
	authed.Put("/topic/:tid/hide", a.TopicHandler.ToggleHide)
	authed.Put("/topic/:tid/best-answer", a.TopicHandler.SetBestAnswer)

	// Reply (authenticated)
	authed.Post("/topic/:tid/reply", a.ReplyHandler.CreateReply)
	authed.Put("/topic/:tid/reply", a.ReplyHandler.UpdateReply)
	authed.Delete("/topic/:tid/reply", a.ReplyHandler.DeleteReply)
	authed.Put("/topic/:tid/reply/like", a.ReplyHandler.ToggleReplyLike)
	authed.Put("/topic/:tid/reply/dislike", a.ReplyHandler.ToggleReplyDislike)
	authed.Put("/topic/:tid/reply/reaction", a.ReplyHandler.ToggleReplyReaction)
	authed.Put("/topic/:tid/reply/pin", a.ReplyHandler.PinReply)

	// Comment (authenticated)
	authed.Post("/topic/:tid/comment", a.TopicCommentHandler.CreateComment)
	authed.Put("/topic/:tid/comment", a.TopicCommentHandler.UpdateComment)
	authed.Put("/topic/:tid/comment/like", a.TopicCommentHandler.ToggleCommentLike)
	authed.Delete("/topic/:tid/comment", a.TopicCommentHandler.DeleteComment)

	// Poll (authenticated)
	authed.Post("/topic/:tid/poll", a.PollHandler.CreatePoll)
	authed.Put("/topic/:tid/poll", a.PollHandler.UpdatePoll)
	authed.Delete("/topic/:tid/poll", a.PollHandler.DeletePoll)
	authed.Post("/topic/:tid/poll/vote", a.PollHandler.Vote)

	// Message (authenticated)
	authed.Get("/message", a.MessageHandler.GetMessages)
	authed.Delete("/message/:id", a.MessageHandler.DeleteMessage)
	authed.Put("/message/system/read", a.MessageHandler.MarkAllRead)
	authed.Get("/message/admin", a.MessageHandler.GetSystemMessages)
	authed.Put("/message/admin/read", a.MessageHandler.MarkAdminRead)
	authed.Get("/message/nav/system", a.MessageHandler.GetNavSummary)
	authed.Get("/message/nav/contact", a.MessageChatHandler.GetNavContact)
	authed.Get("/message/chat/history", a.MessageChatHandler.GetChatHistory)
	authed.Post("/message/chat/send", a.MessageChatHandler.SendChatMessage)
	authed.Post("/message/chat/recall", a.MessageChatHandler.RecallChatMessage)

	// Image upload (authenticated)
	authed.Post("/image/topic", a.ImageHandler.UploadTopicImage)
	// Cover / banner / icon upload (doc, friend-link, website) — uploads under
	// the `topic` preset and returns {hash, url, ...}; the caller stores the
	// hash on its entity's *_image_hash column.
	authed.Post("/image/cover", a.ImageHandler.UploadCoverImage)
	// Chat / private-message inline image upload (own `message` preset).
	authed.Post("/image/message", a.ImageHandler.UploadMessageImage)
	// U2 (K-PR3a): galgame cover / screenshot upload — proxies a
	// single image to image_service under one of the gated presets and
	// returns the resulting {hash, url, ...} to the FE so it can attach
	// the hash to a covers[] or screenshots[] row on the next PUT/PR.
	authed.Post("/image/galgame", a.ImageHandler.UploadGalgameImage)

	// Report (authenticated)
	authed.Post("/report/submit", a.ReportHandler.SubmitReport)

	// Website interactions (authenticated)
	authed.Put("/website/:domain/like", a.WebsiteHandler.ToggleLike)
	authed.Put("/website/:domain/favorite", a.WebsiteHandler.ToggleFavorite)
	authed.Post("/website/:domain/comment", a.WebsiteCommentHandler.CreateComment)
	authed.Delete("/website/:domain/comment", a.WebsiteCommentHandler.DeleteComment)

	// Galgame submission flow (authenticated, any role) — see
	// docs/galgame_wiki/07-submission.md. The wizard search forces
	// include_pending=true so the caller sees their own pending hits.
	//
	// /mine + /search/wizard are registered earlier (above the
	// /galgame/:gid catch-all) because Fiber matches in registration
	// order; see the comment near api.Get("/galgame/mine", ...) above.
	authed.Post("/galgame/submit", a.GalgameSubmissionHandler.Submit)
	authed.Post("/galgame/:gid/claim", a.GalgameSubmissionHandler.Claim)
	authed.Patch("/galgame/:gid", a.GalgameSubmissionHandler.PatchDraft)
	authed.Delete("/galgame/:gid", a.GalgameSubmissionHandler.DeleteDraft)

	// Wiki message stream — user notifications + per-user read marker.
	authed.Get("/galgame/messages/mine", a.GalgameMessageHandler.MessagesMine)
	authed.Get("/galgame/messages/read-state", a.GalgameMessageHandler.GetReadState)
	authed.Put("/galgame/messages/read-state", a.GalgameMessageHandler.SetReadState)

	// Galgame interactions (authenticated, local). The static
	// /galgame/interactions/mine is declared before the :gid routes (and like
	// /galgame/messages/mine, static segments win over :gid).
	authed.Get("/galgame/interactions/mine", a.GalgameHandler.MyInteractions)
	authed.Put("/galgame/:gid/like", a.GalgameHandler.ToggleLike)
	authed.Put("/galgame/:gid/favorite", a.GalgameHandler.ToggleFavorite)
	authed.Post("/galgame/:gid/comment", a.GalgameCommentHandler.CreateComment)
	authed.Put("/galgame/:gid/comment", a.GalgameCommentHandler.UpdateComment)
	authed.Delete("/galgame/:gid/comment", a.GalgameCommentHandler.DeleteComment)
	authed.Put("/galgame/:gid/comment/like", a.GalgameCommentHandler.ToggleCommentLike)

	// Galgame resource (authenticated, local)
	authed.Post("/galgame/:gid/resource", a.GalgameResourceHandler.CreateResource)
	authed.Put("/galgame/:gid/resource", a.GalgameResourceHandler.UpdateResource)
	authed.Delete("/galgame/:gid/resource", a.GalgameResourceHandler.DeleteResource)
	authed.Put("/galgame/:gid/resource/like", a.GalgameResourceHandler.ToggleLike)
	authed.Put("/galgame/:gid/resource/valid", a.GalgameResourceHandler.MarkValid)
	authed.Put("/galgame/:gid/resource/expired", a.GalgameResourceHandler.MarkExpired)

	// Galgame rating (authenticated, local)
	authed.Post("/galgame-rating", a.GalgameRatingHandler.CreateRating)
	authed.Put("/galgame-rating/:id", a.GalgameRatingHandler.UpdateRating)
	authed.Delete("/galgame-rating/:id", a.GalgameRatingHandler.DeleteRating)
	authed.Put("/galgame-rating/:id/like", a.GalgameRatingHandler.ToggleLike)
	authed.Post("/galgame-rating/:id/comment", a.GalgameRatingHandler.CreateComment)
	authed.Put("/galgame-rating/:id/comment", a.GalgameRatingHandler.UpdateComment)
	authed.Delete("/galgame-rating/:id/comment", a.GalgameRatingHandler.DeleteComment)

	// Galgame wiki writes (authenticated + token forwarding).
	//
	// Note on PR submission (POST /galgame/:gid/prs): the integration guide
	// (docs/galgame_wiki/integration-guide.md §6) suggests letting the
	// frontend call wiki directly to skip this hop, but our kun_session
	// architecture makes the OAuth access token opaque to the browser
	// (it lives in Redis, the browser only has the session cookie). So
	// every wiki write must traverse kungal so the middleware can attach
	// the session-stored bearer token; ProxyWriteWithToken is the thin
	// shim that does that. Endpoints with kungal-local side effects
	// (Create/MergePR + PR submit/decline, which emit local "requested"/
	// "declined"/"merged" notifications the wiki doesn't) go through
	// GalgameHandler instead.
	// POST /galgame is the "admin direct publish" bypass — wiki gates it
	// to admin/moderator (see docs/galgame_wiki/01-galgame.md §POST). Most
	// users go through POST /galgame/submit instead. We mirror the gate
	// here so non-admin attempts fail fast before the wiki hop.
	authed.Post("/galgame",
		middleware.RequireModerator(),
		a.GalgameHandler.Create,
	)
	authed.Put("/galgame/:gid", a.GalgameWikiHandler.ProxyWriteWithToken("PUT"))
	authed.Put("/galgame/:gid/prs/:id/merge", a.GalgameHandler.MergePR)
	authed.Post("/galgame/:gid/revert", a.GalgameWikiHandler.ProxyWriteWithToken("POST"))
	authed.Post("/galgame/:gid/prs", a.GalgameHandler.SubmitPR)
	authed.Put("/galgame/:gid/prs/:id/decline", a.GalgameHandler.DeclinePR)
	authed.Post("/galgame/:gid/links", a.GalgameWikiHandler.ProxyWriteWithToken("POST"))
	authed.Delete("/galgame/:gid/links", a.GalgameWikiHandler.ProxyWriteWithToken("DELETE"))
	authed.Post("/galgame/:gid/aliases", a.GalgameWikiHandler.ProxyWriteWithToken("POST"))
	authed.Delete("/galgame/:gid/aliases", a.GalgameWikiHandler.ProxyWriteWithToken("DELETE"))
	authed.Delete("/galgame/:gid/contributors/:id", a.GalgameWikiHandler.ProxyWriteWithToken("DELETE"))
	authed.Put("/galgame-tag", a.GalgameWikiHandler.ProxyWriteWithToken("PUT"))
	authed.Put("/galgame-official", a.GalgameWikiHandler.ProxyWriteWithToken("PUT"))
	authed.Put("/galgame-engine", a.GalgameWikiHandler.ProxyWriteWithToken("PUT"))
	// Taxonomy create/delete (04-taxonomy.md, new). POST = any logged-in
	// user (lets users add a tag/official/engine missing for an original
	// /doujin work); PUT/DELETE = admin/moderator. Auth is enforced by
	// wiki — kungal only forwards the token, never widens/narrows it
	// (00-handbook §15.2). ToWikiPath maps /galgame-tag → /tag etc.
	authed.Post("/galgame-tag", a.GalgameWikiHandler.ProxyWriteWithToken("POST"))
	authed.Delete("/galgame-tag/:id", a.GalgameWikiHandler.ProxyWriteWithToken("DELETE"))
	authed.Post("/galgame-official", a.GalgameWikiHandler.ProxyWriteWithToken("POST"))
	authed.Delete("/galgame-official/:id", a.GalgameWikiHandler.ProxyWriteWithToken("DELETE"))
	authed.Post("/galgame-engine", a.GalgameWikiHandler.ProxyWriteWithToken("POST"))
	authed.Delete("/galgame-engine/:id", a.GalgameWikiHandler.ProxyWriteWithToken("DELETE"))

	// U3 taxonomy revisions + revert (K-PR5). ToWikiPath's kebab prefix
	// rewrite (/galgame-tag → /tag, /galgame-series → /series) maps these
	// to the wiki's per-entity revision endpoints.
	//
	// Revision GETs forward the caller's bearer (via optAuth) — the wiki
	// gates taxonomy revision history behind auth (02-revisions §"后端透传
	// Bearer 代理"), so a token-less GET 401's for everyone.
	//
	// All four entities (tag / official / engine / series) are included.
	// Series only surfaces its own name/alias/description edits — membership
	// changes (a galgame joining/leaving) are recorded as galgame-side
	// revisions — but that is still useful history, so it is no longer
	// excluded (the prior exclusion was a stale earlier-version decision).
	for _, ent := range []string{"galgame-tag", "galgame-official", "galgame-engine", "galgame-series"} {
		wikiEnt := strings.TrimPrefix(ent, "galgame-")
		// LIST is hydrated (real user name/avatar, camelCase {items,total});
		// the single-revision snapshot stays a raw proxy (the FE diff builder
		// consumes the wiki's verbatim snake_case snapshot).
		optAuth.Get("/"+ent+"/:id/revisions", a.GalgameWikiHandler.GetTaxonomyRevisions(wikiEnt))
		optAuth.Get("/"+ent+"/:id/revisions/:rev", a.GalgameWikiHandler.ProxyGetWithToken)
		authed.Post("/"+ent+"/:id/revert", a.GalgameWikiHandler.ProxyWriteWithToken("POST"))
	}
	authed.Post("/galgame-series", a.GalgameWikiHandler.ProxyWriteWithToken("POST"))
	authed.Post("/galgame-series/modal", a.GalgameWikiHandler.ProxyWriteWithToken("POST"))
	authed.Put("/galgame-series/:id", a.GalgameWikiHandler.ProxyWriteWithToken("PUT"))
	authed.Delete("/galgame-series/:id", a.GalgameWikiHandler.ProxyWriteWithToken("DELETE"))

	// Toolset (authenticated)
	authed.Post("/toolset", a.ToolsetHandler.Create)
	authed.Put("/toolset/:id", a.ToolsetHandler.Update)
	authed.Delete("/toolset/:id", a.ToolsetHandler.Delete)
	authed.Put("/toolset/:id/practicality", a.ToolsetPracticalityHandler.UpsertPracticality)
	authed.Post("/toolset/:id/comment", a.ToolsetCommentHandler.CreateComment)
	authed.Put("/toolset/:id/comment", a.ToolsetCommentHandler.UpdateComment)
	authed.Delete("/toolset/:id/comment", a.ToolsetCommentHandler.DeleteComment)
	authed.Post("/toolset/:id/resource", a.ToolsetResourceHandler.CreateResource)
	authed.Put("/toolset/:id/resource", a.ToolsetResourceHandler.UpdateResource)
	authed.Delete("/toolset/:id/resource", a.ToolsetResourceHandler.DeleteResource)
	authed.Post("/toolset/:id/upload/init", a.ToolsetUploadHandler.UploadInit)
	authed.Post("/toolset/:id/upload/complete", a.ToolsetUploadHandler.UploadComplete)
	authed.Post("/toolset/:id/upload/resume", a.ToolsetUploadHandler.UploadResume)
	authed.Post("/toolset/:id/upload/abort", a.ToolsetUploadHandler.UploadAbort)

	// ════════════════════════════════════════════
	// ADMIN routes (moderator / admin capability)
	// ════════════════════════════════════════════

	admin := authed.Group("", middleware.RequireAdmin())
	admin.Get("/admin/overview/all", a.AdminOverviewHandler.GetOverview)
	admin.Get("/admin/overview/stats", a.AdminOverviewHandler.GetStats)

	// Content moderation: kungal no longer brokers identity (account
	// ban/delete/register all live in the OAuth admin UI), but it DOES own
	// every piece of content a user publishes here — so an admin can preview
	// and one-shot purge a spam user's entire kungal footprint.
	admin.Get("/admin/user/:id/content-stats", a.AdminPurgeHandler.GetUserContentStats)
	admin.Delete("/admin/user/:id/content", a.AdminPurgeHandler.PurgeUserContent)

	// Galgame admin (moderator+): wiki submission review queue +
	// approve/decline/ban actions. Wiki requires admin/moderator on these
	// (per docs/galgame_wiki/06-admin.md + 08-messages.md); we mirror the
	// gate locally and forward via ProxyWriteWithToken so the wiki sees
	// the calling admin's identity for the revision/message side effects.
	galgameAdmin := authed.Group("", middleware.RequireModerator())
	galgameAdmin.Get("/admin/galgame/messages", a.GalgameMessageHandler.AdminMessages)
	galgameAdmin.Put(
		"/admin/galgame/:gid/status",
		a.GalgameWikiHandler.ProxyWriteWithToken("PUT"),
	)

	// Doc admin (moderator+)
	docAdmin := authed.Group("", middleware.RequireModerator())
	docAdmin.Get("/admin/doc/article", a.DocArticleHandler.GetAdminArticles)
	docAdmin.Post("/doc/article", a.DocArticleHandler.CreateArticle)
	docAdmin.Put("/doc/article", a.DocArticleHandler.UpdateArticle)
	docAdmin.Put("/doc/article/reorder", a.DocArticleHandler.ReorderArticles)
	docAdmin.Put("/doc/article/pin", a.DocArticleHandler.SetArticlePin)
	docAdmin.Delete("/doc/article", a.DocArticleHandler.DeleteArticle)
	docAdmin.Post("/doc/category", a.DocCategoryHandler.CreateCategory)
	docAdmin.Put("/doc/category", a.DocCategoryHandler.UpdateCategory)
	docAdmin.Delete("/doc/category", a.DocCategoryHandler.DeleteCategory)
	docAdmin.Post("/doc/tag", a.DocTagHandler.CreateTag)
	docAdmin.Put("/doc/tag", a.DocTagHandler.UpdateTag)
	docAdmin.Delete("/doc/tag", a.DocTagHandler.DeleteTag)

	// Website admin (moderator+)
	wsAdmin := authed.Group("", middleware.RequireModerator())
	wsAdmin.Post("/website", a.WebsiteHandler.CreateWebsite)
	wsAdmin.Put("/website/:domain", a.WebsiteHandler.UpdateWebsite)
	wsAdmin.Delete("/website/:domain", a.WebsiteHandler.DeleteWebsite)
	wsAdmin.Put("/website-category", a.WebsiteCategoryHandler.UpdateWebsiteCategory)
	wsAdmin.Post("/website-tag", a.WebsiteTagHandler.CreateWebsiteTag)
	wsAdmin.Put("/website-tag", a.WebsiteTagHandler.UpdateWebsiteTag)
	wsAdmin.Delete("/website-tag", a.WebsiteTagHandler.DeleteWebsiteTag)

	// Update admin (moderator+)
	updateAdmin := authed.Group("", middleware.RequireModerator())
	updateAdmin.Post("/update/history", a.UpdateHandler.CreateHistory)
	updateAdmin.Put("/update/history", a.UpdateHandler.UpdateHistory)
	updateAdmin.Delete("/update/history", a.UpdateHandler.DeleteHistory)
	updateAdmin.Post("/update/todo", a.UpdateHandler.CreateTodo)
	updateAdmin.Put("/update/todo", a.UpdateHandler.UpdateTodo)
	updateAdmin.Delete("/update/todo", a.UpdateHandler.DeleteTodo)

	// Friend-link admin (moderator+): CRUD + drag-reorder.
	friendAdmin := authed.Group("", middleware.RequireModerator())
	friendAdmin.Post("/admin/friend-link", a.FriendLinkHandler.Create)
	friendAdmin.Put("/admin/friend-link", a.FriendLinkHandler.Update)
	friendAdmin.Delete("/admin/friend-link", a.FriendLinkHandler.Delete)
	friendAdmin.Put("/admin/friend-link/reorder", a.FriendLinkHandler.Reorder)
}
