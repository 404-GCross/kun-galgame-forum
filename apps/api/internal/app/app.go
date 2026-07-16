package app

import (
	"context"
	"log/slog"

	activityHandler "kun-galgame-api/internal/activity/handler"
	activityRepo "kun-galgame-api/internal/activity/repository"
	activityService "kun-galgame-api/internal/activity/service"
	adminHandler "kun-galgame-api/internal/admin/handler"
	adminRepo "kun-galgame-api/internal/admin/repository"
	adminService "kun-galgame-api/internal/admin/service"
	communitytrust "kun-galgame-api/internal/community/trust"
	docHandler "kun-galgame-api/internal/doc/handler"
	docRepo "kun-galgame-api/internal/doc/repository"
	docService "kun-galgame-api/internal/doc/service"
	friendHandler "kun-galgame-api/internal/friendlink/handler"
	friendRepo "kun-galgame-api/internal/friendlink/repository"
	galgameClient "kun-galgame-api/internal/galgame/client"
	galgameHandler "kun-galgame-api/internal/galgame/handler"
	galgameRepo "kun-galgame-api/internal/galgame/repository"
	galgameService "kun-galgame-api/internal/galgame/service"
	homeHandler "kun-galgame-api/internal/home/handler"
	homeRepo "kun-galgame-api/internal/home/repository"
	homeService "kun-galgame-api/internal/home/service"
	imageHandler "kun-galgame-api/internal/image/handler"
	imageRepo "kun-galgame-api/internal/image/repository"
	imageService "kun-galgame-api/internal/image/service"
	"kun-galgame-api/internal/infrastructure/cache"
	cronPkg "kun-galgame-api/internal/infrastructure/cron"
	"kun-galgame-api/internal/infrastructure/database"
	"kun-galgame-api/internal/infrastructure/mail"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/internal/infrastructure/storage"
	msgHandler "kun-galgame-api/internal/message/handler"
	msgRepo "kun-galgame-api/internal/message/repository"
	msgService "kun-galgame-api/internal/message/service"
	"kun-galgame-api/internal/moemoepoint"
	rankingHandler "kun-galgame-api/internal/ranking/handler"
	rankingRepo "kun-galgame-api/internal/ranking/repository"
	rankingService "kun-galgame-api/internal/ranking/service"
	rssHandler "kun-galgame-api/internal/rss/handler"
	rssRepo "kun-galgame-api/internal/rss/repository"
	searchHandler "kun-galgame-api/internal/search/handler"
	searchRepo "kun-galgame-api/internal/search/repository"
	searchService "kun-galgame-api/internal/search/service"
	sectionHandler "kun-galgame-api/internal/section/handler"
	sectionRepo "kun-galgame-api/internal/section/repository"
	sectionService "kun-galgame-api/internal/section/service"
	toolsetHandler "kun-galgame-api/internal/toolset/handler"
	toolsetRepo "kun-galgame-api/internal/toolset/repository"
	toolsetService "kun-galgame-api/internal/toolset/service"
	topicHandler "kun-galgame-api/internal/topic/handler"
	topicRepo "kun-galgame-api/internal/topic/repository"
	topicService "kun-galgame-api/internal/topic/service"
	"kun-galgame-api/internal/trust/enforce"
	trustHandler "kun-galgame-api/internal/trust/handler"
	trustService "kun-galgame-api/internal/trust/service"
	updateHandler "kun-galgame-api/internal/update/handler"
	updateRepo "kun-galgame-api/internal/update/repository"
	"kun-galgame-api/internal/user/handler"
	"kun-galgame-api/internal/user/oauth"
	"kun-galgame-api/internal/user/repository"
	"kun-galgame-api/internal/user/service"
	websiteHandler "kun-galgame-api/internal/website/handler"
	websiteRepo "kun-galgame-api/internal/website/repository"
	websiteService "kun-galgame-api/internal/website/service"
	"kun-galgame-api/pkg/artifactclient"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/communityclient"
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/linkcheck"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/trustclient"
	"kun-galgame-api/pkg/userclient"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	Fiber       *fiber.App
	DB          *gorm.DB
	Redis       *redis.Client
	Mailer      *mail.Mailer
	Config      *config.Config
	OAuthClient *oauth.Client
	UserState   *repository.StateRepository
	UserClient  *userclient.Client

	// Handlers
	OAuthHandler                   *handler.OAuthHandler
	UserHandler                    *handler.UserHandler
	UserProfileHandler             *handler.ProfileHandler
	HomeHandler                    *homeHandler.HomeHandler
	TopicHandler                   *topicHandler.TopicHandler
	TopicDraftHandler              *topicHandler.TopicDraftHandler
	ReplyHandler                   *topicHandler.ReplyHandler
	TopicCommentHandler            *topicHandler.CommentHandler
	PollHandler                    *topicHandler.PollHandler
	MessageHandler                 *msgHandler.MessageHandler
	MessageChatHandler             *msgHandler.ChatHandler
	AdminOverviewHandler           *adminHandler.OverviewHandler
	AdminPurgeHandler              *adminHandler.PurgeHandler
	RankingHandler                 *rankingHandler.RankingHandler
	SectionHandler                 *sectionHandler.SectionHandler
	DocArticleHandler              *docHandler.ArticleHandler
	DocCategoryHandler             *docHandler.CategoryHandler
	DocTagHandler                  *docHandler.TagHandler
	WebsiteHandler                 *websiteHandler.WebsiteHandler
	WebsiteCategoryHandler         *websiteHandler.CategoryHandler
	WebsiteTagHandler              *websiteHandler.TagHandler
	UpdateHandler                  *updateHandler.UpdateHandler
	FriendLinkHandler              *friendHandler.FriendLinkHandler
	TrustHandler                   *trustHandler.TrustHandler
	RSSHandler                     *rssHandler.RSSHandler
	GalgameHandler                 *galgameHandler.GalgameHandler
	GalgameCollectionHandler       *galgameHandler.GalgameCollectionHandler
	GalgameCommunityCommentHandler *galgameHandler.CommunityCommentHandler
	ResourceCommentHandler         *galgameHandler.ResourceCommentHandler
	GalgameResourceHandler         *galgameHandler.ResourceHandler
	GalgameRatingHandler           *galgameHandler.RatingHandler
	GalgameQuizHandler             *galgameHandler.QuizHandler
	CreatorHandler                 *galgameHandler.CreatorHandler
	GalgameEntityHandler           *galgameHandler.EntityHandler
	GalgameCalendarHandler         *galgameHandler.CalendarHandler
	GalgameDraftsHandler           *galgameHandler.DraftsHandler
	GalgameWikiHandler             *galgameHandler.WikiHandler
	GalgameSubmissionHandler       *galgameHandler.SubmissionHandler
	GalgameMessageHandler          *galgameHandler.WikiMessageHandler
	GalgameCrossSourceHandler      *galgameHandler.CrossSourceHandler
	ActivityHandler                *activityHandler.ActivityHandler
	ImageHandler                   *imageHandler.ImageHandler
	SearchHandler                  *searchHandler.SearchHandler
	ToolsetHandler                 *toolsetHandler.ToolsetHandler
	ToolsetPracticalityHandler     *toolsetHandler.PracticalityHandler
	ToolsetResourceHandler         *toolsetHandler.ResourceHandler
	ToolsetUploadHandler           *toolsetHandler.UploadHandler
	CronStop                       func()
}

func New(cfg *config.Config) *App {
	// Infrastructure
	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)
	rdb := cache.NewRedis(cfg.Redis)
	// fileStorageClient: archive storage (B2). Toolset .7z/.zip/.rar uploads
	// via presigned URLs, configured via FILE_STORAGE_* env vars. This is the
	// only S3-API bucket left — inline / content images all go through
	// image_service now, so there is no separate image bed (R2) client.
	fileStorageClient := storage.NewS3(cfg.FileStorage)
	if fileStorageClient == nil {
		slog.Warn("FILE_STORAGE_* 未配置, 工具集上传将不可用")
	}
	mailer := mail.NewMailer(cfg.Mail)

	// Resolve /image/<hash> content refs to absolute CDN URLs at render time.
	// Same CDN base the image client / galgame banner walker use.
	markdown.SetContentImageCDNBase(cfg.GalgameWiki.ImageCDNBase)

	// Repositories
	userStateRepo := repository.NewStateRepository(db)
	userStatsRepo := repository.NewUserStatsRepository(db)
	userContentRepo := repository.NewUserContentRepository(db)
	messageRepository := msgRepo.NewMessageRepository(db)
	chatRepository := msgRepo.NewChatRepository(db)

	// Galgame wiki client (shared — user service needs it too). The
	// Basic-auth variant carries OAuth Client credentials so the
	// wiki-message sync cron can call /galgame/messages/feed (service
	// identity). Bearer-required endpoints still use a per-request token
	// forwarded from the user session.
	gc := galgameClient.NewGalgameClientWithBasicAuth(
		cfg.GalgameWiki.BaseURL,
		cfg.GalgameWiki.ImageCDNBase,
		cfg.OAuth.ClientID,
		cfg.OAuth.ClientSecret,
	)

	// OAuth client (used by auth service).
	oauthClient := oauth.NewClient(cfg.OAuth)

	// OAuth user-info client. Identity (name/avatar/bio/status/roles) is
	// owned by OAuth post-migration; mappers call this for batch enrichment.
	uc := userclient.New(userclient.Config{
		BaseURL:      cfg.OAuth.ServerURL,
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
		// Same image_service CDN base the wiki client uses — lets userclient
		// resolve users' avatar_image_hash into URLs (new avatars store only the
		// hash; the legacy `avatar` field is empty).
		ImageCDNBase: cfg.GalgameWiki.ImageCDNBase,
	})

	// Install the process-wide moemoepoint Awarder: OAuth is the single source
	// of truth; every change goes through it and the returned authoritative
	// balance is mirrored into the local kungal_user_state cache (no local +=).
	// See internal/moemoepoint + docs/oauth/06-moemoepoint.md.
	moemoepoint.SetDefault(moemoepoint.NewAwarder(uc, db))

	// image_service client — covers/screenshots multi-image upload path
	// (U2 / K-PR3a). ONLY construct when credentials are present, so the
	// downstream service-level `imgCli == nil` guard actually fires when
	// the operator forgot to set KUN_IMAGE_CLIENT_ID/SECRET — and
	// surfaces "图片上传服务未配置" instead of a misleading image_service
	// 401 when the user tries to upload. Mirrors the wiki side's same
	// guard pattern. A loud warn-on-startup so ops notices early.
	var imgCli *imageclient.Client
	if cfg.ImageClient.ClientID != "" && cfg.ImageClient.ClientSecret != "" {
		imgCli = imageclient.New(imageclient.Config{
			BaseURL:      cfg.ImageClient.BaseURL,
			CDNBase:      cfg.GalgameWiki.ImageCDNBase,
			ClientID:     cfg.ImageClient.ClientID,
			ClientSecret: cfg.ImageClient.ClientSecret,
		})
		slog.Info("image_service client configured", "base_url", cfg.ImageClient.BaseURL)

		// Enrich server-rendered content <img> tags with intrinsic dims +
		// ThumbHash (no-CLS aspect-ratio reservation + blur-up). The resolver
		// caches forever and the markdown renderer calls it synchronously, so
		// warm renders touch no network. Only wired when image_service is
		// configured; otherwise content images render as plain lazy <img>.
		markdown.SetContentImageMetaResolver(imgCli.NewMetaResolver(0).Resolve)
	} else {
		slog.Warn("image_service client NOT configured; /image/galgame upload will return 未配置 — set KUN_IMAGE_CLIENT_ID / KUN_IMAGE_CLIENT_SECRET")
	}

	// artifact service client (toolset archive upload/download). kungal's OAuth
	// client IS its artifact "site", so credentials default to the OAuth client
	// when KUN_ARTIFACT_CLIENT_ID/SECRET are unset. New() degrades to a no-op
	// (calls return ErrNotConfigured) when neither is set, so a dev box without
	// artifact creds still boots.
	artClientID := cfg.ArtifactClient.ClientID
	if artClientID == "" {
		artClientID = cfg.OAuth.ClientID
	}
	artClientSecret := cfg.ArtifactClient.ClientSecret
	if artClientSecret == "" {
		artClientSecret = cfg.OAuth.ClientSecret
	}
	artCli := artifactclient.New(artifactclient.Config{
		BaseURL:      cfg.ArtifactClient.BaseURL,
		ClientID:     artClientID,
		ClientSecret: artClientSecret,
	})
	if artCli.Configured() {
		slog.Info("artifact service client configured", "base_url", cfg.ArtifactClient.BaseURL)
	} else {
		slog.Warn("artifact service client NOT configured; toolset upload will return 未配置 — set KUN_ARTIFACT_CLIENT_BASE_URL + OAuth creds")
	}

	// Trust & Safety client — report intake (Phase 1) + moderator inbox proxy
	// (Phase 3). Basic auth reuses the OAuth client_id/secret; the trust service
	// reads oauth_clients.catalog_site to derive kungal's site. Degrades to a
	// no-op when KUN_TRUST_BASE_URL / OAuth creds are unset.
	trustCli := trustclient.New(trustclient.Config{
		BaseURL:      cfg.Trust.BaseURL,
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
	})
	if trustCli.Configured() {
		slog.Info("trust service client configured", "base_url", cfg.Trust.BaseURL)
	} else {
		slog.Warn("trust service client NOT configured; reporting returns 未启用 — set KUN_TRUST_BASE_URL + OAuth creds")
	}

	// Catalog read client — the galgame detail page's credits + name/character
	// reverse lookups (step 36). GET-only S2S; Basic auth reuses the OAuth
	// client_id/secret (the catalog read face imposes no site binding). Degrades
	// to a no-op (calls return ErrNotConfigured → 503) when KUN_CATALOG_API_BASE
	// / OAuth creds are unset, so a dev box without a catalog service still boots.
	catalogCli := catalogclient.New(catalogclient.Config{
		BaseURL:      cfg.Catalog.BaseURL,
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
	})
	if catalogCli.Configured() {
		slog.Info("catalog service client configured", "base_url", cfg.Catalog.BaseURL)
	} else {
		slog.Warn("catalog service client NOT configured; catalog credits proxy returns 未启用 — set KUN_CATALOG_API_BASE + OAuth creds")
	}

	// Community primitive client — the galgame comment area reroute (charter step
	// 03; kun-galgame-infra cmd/community :9282). Basic auth reuses the OAuth
	// client_id/secret (the community service derives kungal's tenant from
	// oauth_clients.catalog_site), so the S2S creds default to the OAuth client
	// when KUN_COMMUNITY_CLIENT_ID/SECRET are unset. Degrades to a no-op (calls
	// return ErrNotConfigured) when the base URL/creds are unset, so a dev box
	// without a community service boots even with the flag flipped on.
	commClientID := cfg.Community.ClientID
	if commClientID == "" {
		commClientID = cfg.OAuth.ClientID
	}
	commClientSecret := cfg.Community.ClientSecret
	if commClientSecret == "" {
		commClientSecret = cfg.OAuth.ClientSecret
	}
	communityCli := communityclient.New(communityclient.Config{
		BaseURL:      cfg.Community.BaseURL,
		ClientID:     commClientID,
		ClientSecret: commClientSecret,
	})
	// Community is now the unconditional comment backend (galgame + the three
	// resource areas). When the client is unconfigured, comment reads degrade to
	// empty pages and writes to 503 — a dev box without a community service still
	// boots.
	if communityCli.Configured() {
		slog.Info("community comment backend configured", "base_url", cfg.Community.BaseURL)
	} else {
		slog.Warn("community comment backend NOT configured; comments degrade (reads empty / writes 503) — set KUN_COMMUNITY_API_BASE + OAuth creds")
	}
	// Login-time trust Boost reporter (staff/veteran). Self-gates on client
	// config, so no Boost fires when the community client is unconfigured.
	communityBooster := communitytrust.New(communityCli, rdb, db)

	// kungal-link-live-checker client — the "report resource expired" gate.
	// Only construct when BOTH base URL + API key are set; otherwise the gate is
	// nil and MarkExpired falls back to the legacy single-report-expires flow
	// (so the feature degrades safely when the checker isn't deployed).
	var linkChecker *linkcheck.Client
	if cfg.LinkChecker.BaseURL != "" && cfg.LinkChecker.APIKey != "" {
		linkChecker = linkcheck.New(linkcheck.Config{
			BaseURL:              cfg.LinkChecker.BaseURL,
			APIKey:               cfg.LinkChecker.APIKey,
			CFAccessClientID:     cfg.LinkChecker.CFAccessClientID,
			CFAccessClientSecret: cfg.LinkChecker.CFAccessClientSecret,
		})
		slog.Info("link-live-checker gate configured",
			"base_url", cfg.LinkChecker.BaseURL,
			"cf_access", cfg.LinkChecker.CFAccessClientID != "")
	} else {
		slog.Warn("link-live-checker NOT configured; resource 报告失效 falls back to legacy single-report-expires — set LINK_CHECKER_BASE_URL / LINK_CHECKER_API_KEY")
	}

	// Services
	authService := service.NewAuthService(userStateRepo, rdb, oauthClient, uc)
	userService := service.NewUserService(userStateRepo, userStatsRepo, rdb, gc, uc, communityCli)
	userContentService := service.NewUserContentService(userContentRepo, gc, uc, communityCli)
	messageSvc := msgService.NewMessageService(messageRepository, userStateRepo, uc)
	chatSvc := msgService.NewChatService(chatRepository, uc)
	notifier := msgService.NewNotifier(messageRepository)

	// Topic
	topicRepository := topicRepo.NewTopicRepository(db)
	topicListRepo := topicRepo.NewTopicListRepository(db)
	topicTaxonomyRepo := topicRepo.NewTopicTaxonomyRepository(db)
	replyRepository := topicRepo.NewReplyRepository(db)
	topicCommentRepo := topicRepo.NewCommentRepository(db)
	pollRepository := topicRepo.NewPollRepository(db)
	draftRepository := topicRepo.NewTopicDraftRepository(db)
	topicSvc := topicService.NewTopicService(topicRepository, topicListRepo, topicTaxonomyRepo, rdb, uc, userStateRepo)
	topicWriteSvc := topicService.NewTopicWriteService(topicRepository, topicTaxonomyRepo, replyRepository, userStateRepo, rdb, notifier)
	replySvc := topicService.NewReplyService(replyRepository, topicCommentRepo, topicRepository, userStateRepo, uc, rdb)
	commentSvc := topicService.NewCommentService(replyRepository, topicCommentRepo, userStateRepo, uc, rdb)
	pollSvc := topicService.NewPollService(pollRepository, topicRepository, userStateRepo, uc, rdb)
	draftSvc := topicService.NewDraftService(draftRepository)

	// Galgame
	galgameCommentRepo := galgameRepo.NewCommentRepository(db)
	// galgameCommentSvc + galgameCommentRepo are retained for the T&S enforcement
	// adapter (galgame_comment subject_kind) + the shared UserObj/truncate helpers;
	// their read/write methods are dead since the legacy /comment routes were
	// retired (charter step 06a).
	galgameCommentSvc := galgameService.NewCommentService(galgameCommentRepo, userStateRepo, uc)
	// Community-backed comment BFF (charter step 03) — the unconditional galgame
	// comment backend on the `/comments` routes (router.go). The local repo owns
	// galgame_post_like + the legacy-id map (migration 057).
	galgameCommunityPostRepo := galgameRepo.NewCommunityPostRepository(db)
	galgameCommunityCommentSvc := galgameService.NewCommunityCommentService(communityCli, galgameCommunityPostRepo, uc, db)
	// Resource comment BFF (charter step 07) — the unconditional rating / website /
	// toolset comment backend on the `/comments` routes (router.go). Reuses the SAME
	// community client + local galgame_post_like repo (post-addressed likes are
	// region-agnostic).
	resourceCommentSvc := galgameService.NewResourceCommentService(communityCli, galgameCommunityPostRepo, uc, db)
	galgameResourceRepo := galgameRepo.NewResourceRepository(db)
	galgameResourceSvc := galgameService.NewResourceService(galgameResourceRepo, gc, uc, linkChecker)
	galgameRatingRepo := galgameRepo.NewRatingRepository(db)
	galgameRatingSvc := galgameService.NewRatingService(galgameRatingRepo, gc, uc)
	galgameQuizRepo := galgameRepo.NewQuizRepository(db)
	galgameQuizSvc := galgameService.NewQuizService(galgameQuizRepo, gc, uc)
	creatorSvc := galgameService.NewCreatorService(galgameRatingRepo, gc, uc)
	galgameLocalRepo := galgameRepo.NewGalgameRepository(db)
	galgameInteractionRepo := galgameRepo.NewGalgameInteractionRepository(db)
	galgameListRepo := galgameRepo.NewGalgameListRepository(db)
	galgameResourceMetaRepo := galgameRepo.NewGalgameResourceMetaRepository(db)
	galgameDetailRatingRepo := galgameRepo.NewGalgameDetailRatingRepository(db)
	galgameEnricher := galgameService.NewGalgameEnricher(galgameLocalRepo, galgameResourceMetaRepo, uc)
	// Core galgame service is built first: the entity (tag/official/engine)
	// detail services delegate their galgame list to it (shared local
	// filter/sort/hydrate), so they take it as a dependency.
	galgameCoreSvc := galgameService.NewGalgameService(
		galgameLocalRepo, galgameInteractionRepo, galgameListRepo,
		galgameResourceMetaRepo, galgameDetailRatingRepo, userStateRepo, gc, uc,
	)
	// Galgame collections (收藏夹): CRUD + membership. Delegates card hydration +
	// owner-name lookup to galgameCoreSvc so nothing is duplicated.
	galgameCollectionRepo := galgameRepo.NewGalgameCollectionRepository(db)
	galgameCollectionSvc := galgameService.NewCollectionService(galgameCollectionRepo, galgameCoreSvc, gc, uc)
	galgameSeriesSvc := galgameService.NewSeriesService(gc, galgameEnricher)
	galgameOfficialSvc := galgameService.NewOfficialService(gc, galgameCoreSvc)
	galgameEngineSvc := galgameService.NewEngineService(gc, galgameCoreSvc)
	galgameTagSvc := galgameService.NewTagService(gc, galgameEnricher, galgameCoreSvc)
	galgameCalendarSvc := galgameService.NewCalendarService(gc, galgameEnricher)
	galgameDraftsSvc := galgameService.NewDraftsService(gc, galgameEnricher)
	galgameWikiSvc := galgameService.NewWikiService(gc, galgameLocalRepo, uc)
	// Submission flow: submit / claim / patch-draft / delete-draft proxies
	// + local moemoepoint side effects. Per docs/galgame_wiki/07-submission.md.
	galgameSubmissionSvc := galgameService.NewSubmissionService(gc, galgameLocalRepo)
	// Cross-source read face (step 36): three-source scores + site stats (wiki
	// passthrough) and catalog credits / entity reverse-lookups (catalog S2S
	// passthrough) for the galgame detail page. Pure GET passthrough — no local
	// state.
	galgameCrossSourceSvc := galgameService.NewCrossSourceService(gc, catalogCli)

	// Wiki message stream: user notifications + admin queue + per-user
	// "read up to" cursor. The cron-driven ingestion lives in
	// galgameMessageSync below.
	galgameMessageRepo := galgameRepo.NewWikiMessageRepository(db)
	galgameMessageSvc := galgameService.NewWikiMessageService(gc, galgameMessageRepo)
	galgameMessageSync := galgameService.NewWikiMessageSync(gc, galgameLocalRepo, userStateRepo, rdb)
	// Mirrors wiki merged-revision (edit) events into galgame_activity so the
	// forum activity timeline can show galgame edits (see migration 021).
	galgameRevisionSync := galgameService.NewWikiRevisionSync(gc, db, rdb)

	// Website
	websiteRepository := websiteRepo.NewWebsiteRepository(db)
	websiteCategoryRepo := websiteRepo.NewCategoryRepository(db)
	websiteTagRepo := websiteRepo.NewTagRepository(db)
	websiteCoreSvc := websiteService.NewWebsiteService(
		websiteRepository, websiteCategoryRepo, websiteTagRepo, uc, communityCli, cfg.GalgameWiki.ImageCDNBase,
	)
	websiteCategorySvc := websiteService.NewCategoryService(websiteCategoryRepo, websiteRepository, websiteTagRepo, cfg.GalgameWiki.ImageCDNBase)
	websiteTagSvc := websiteService.NewTagService(websiteTagRepo, websiteRepository, websiteCategoryRepo, cfg.GalgameWiki.ImageCDNBase)

	// Admin
	adminOverviewRepo := adminRepo.NewOverviewRepository(db)
	adminOverviewSvc := adminService.NewOverviewService(adminOverviewRepo, gc)
	adminPurgeSvc := adminService.NewPurgeService(adminRepo.NewPurgeRepository(db), uc, communityCli)

	// Doc
	docArticleRepo := docRepo.NewArticleRepository(db)
	docCategoryRepo := docRepo.NewCategoryRepository(db)
	docTagRepo := docRepo.NewTagRepository(db)
	docArticleSvc := docService.NewArticleService(docArticleRepo, docCategoryRepo, cfg.GalgameWiki.ImageCDNBase)
	docCategorySvc := docService.NewCategoryService(docCategoryRepo)
	docTagSvc := docService.NewTagService(docTagRepo)

	// Toolset
	toolsetRepository := toolsetRepo.NewToolsetRepository(db)
	toolsetResourceRepo := toolsetRepo.NewResourceRepository(db)
	toolsetCommentRepo := toolsetRepo.NewCommentRepository(db)
	toolsetPracticalityRepo := toolsetRepo.NewPracticalityRepository(db)
	toolsetPracticalitySvc := toolsetService.NewPracticalityService(toolsetPracticalityRepo)
	toolsetCommentSvc := toolsetService.NewCommentService(toolsetCommentRepo, toolsetRepository, uc, communityCli)
	toolsetResourceSvc := toolsetService.NewResourceService(toolsetResourceRepo, toolsetRepository, fileStorageClient, artCli, uc)
	toolsetUploadSvc := toolsetService.NewUploadService(artCli, rdb, db)
	toolsetCoreSvc := toolsetService.NewToolsetService(
		toolsetRepository, toolsetResourceRepo, toolsetPracticalityRepo,
		fileStorageClient, uc, toolsetPracticalitySvc, toolsetCommentSvc,
	)

	// Trust & Safety enforcement adapters — the "thin adapter" half of the
	// pipeline: each subject_kind wires hide/remove/author-lookup to existing
	// services/repos. galgame + user are ABSENT (human-only: galgame moderation
	// is wiki-side, user bans are IdP-side), so their callbacks no-op locally.
	trustRegistry := enforce.Registry{
		"forum_topic": {
			// No hard delete exists for topics → hide == remove (status=1).
			Hide: func(_ context.Context, id int) error {
				return topicRepository.UpdateFields(id, map[string]any{"status": 1})
			},
			Remove: func(_ context.Context, id int) error {
				return topicRepository.UpdateFields(id, map[string]any{"status": 1})
			},
			AuthorID: func(_ context.Context, id int) (int, error) {
				t, err := topicRepository.FindByID(id)
				if err != nil {
					return 0, nil
				}
				return t.UserID, nil
			},
		},
		"forum_reply": {
			Hide:   func(_ context.Context, id int) error { return replyRepository.SetStatus(id, 1) },
			Remove: func(_ context.Context, id int) error { return replySvc.ModerationRemove(id) },
			AuthorID: func(_ context.Context, id int) (int, error) {
				r, err := replyRepository.FindByID(id)
				if err != nil {
					return 0, nil
				}
				return r.UserID, nil
			},
		},
		"forum_comment": {
			Hide:   func(_ context.Context, id int) error { return topicCommentRepo.SetStatus(id, 1) },
			Remove: func(_ context.Context, id int) error { return commentSvc.ModerationRemove(id) },
			AuthorID: func(_ context.Context, id int) (int, error) {
				c, err := topicCommentRepo.FindCommentByID(id)
				if err != nil {
					return 0, nil
				}
				return c.UserID, nil
			},
		},
		"galgame_comment": {
			Hide: func(_ context.Context, id int) error { return galgameCommentRepo.SetStatus(id, 1) },
			Remove: func(_ context.Context, id int) error {
				if appErr := galgameCommentSvc.DeleteComment(0, true, id); appErr != nil {
					return appErr
				}
				return nil
			},
			AuthorID: func(_ context.Context, id int) (int, error) {
				c, err := galgameCommentRepo.FindByID(id)
				if err != nil {
					return 0, nil
				}
				return c.UserID, nil
			},
		},
	}
	// warn_user is record-only for now (no system-sender user for a targeted
	// notice) — pass nil; the dispatcher no-ops warn gracefully.
	trustEnforce := enforce.NewService(db, trustRegistry, nil)

	// Handlers
	app := &App{
		DB: db, Redis: rdb, Mailer: mailer, Config: cfg, OAuthClient: oauthClient,
		UserState:                      userStateRepo,
		UserClient:                     uc,
		OAuthHandler:                   handler.NewOAuthHandler(authService, cfg.Server.Mode == "prod", communityBooster),
		UserHandler:                    handler.NewUserHandler(userService, userContentService),
		UserProfileHandler:             handler.NewProfileHandler(oauthClient, uc),
		HomeHandler:                    homeHandler.NewHomeHandler(homeService.NewHomeService(homeRepo.NewHomeRepository(db), gc, uc, rdb)),
		TopicHandler:                   topicHandler.NewTopicHandler(topicSvc, topicWriteSvc),
		TopicDraftHandler:              topicHandler.NewTopicDraftHandler(draftSvc),
		ReplyHandler:                   topicHandler.NewReplyHandler(replySvc),
		TopicCommentHandler:            topicHandler.NewCommentHandler(commentSvc),
		PollHandler:                    topicHandler.NewPollHandler(pollSvc),
		MessageHandler:                 msgHandler.NewMessageHandler(messageSvc),
		MessageChatHandler:             msgHandler.NewChatHandler(chatSvc),
		AdminOverviewHandler:           adminHandler.NewOverviewHandler(adminOverviewSvc),
		AdminPurgeHandler:              adminHandler.NewPurgeHandler(adminPurgeSvc),
		RankingHandler:                 rankingHandler.NewRankingHandler(rankingService.NewRankingService(rankingRepo.NewRankingRepository(db), gc, uc)),
		SectionHandler:                 sectionHandler.NewSectionHandler(sectionService.NewSectionService(sectionRepo.NewSectionRepository(db), uc)),
		DocArticleHandler:              docHandler.NewArticleHandler(docArticleSvc),
		DocCategoryHandler:             docHandler.NewCategoryHandler(docCategorySvc),
		DocTagHandler:                  docHandler.NewTagHandler(docTagSvc),
		WebsiteHandler:                 websiteHandler.NewWebsiteHandler(websiteCoreSvc),
		WebsiteCategoryHandler:         websiteHandler.NewCategoryHandler(websiteCategorySvc),
		WebsiteTagHandler:              websiteHandler.NewTagHandler(websiteTagSvc),
		UpdateHandler:                  updateHandler.NewUpdateHandler(updateRepo.NewUpdateRepository(db)),
		FriendLinkHandler:              friendHandler.NewFriendLinkHandler(friendRepo.NewFriendLinkRepository(db), cfg.GalgameWiki.ImageCDNBase),
		TrustHandler:                   trustHandler.NewTrustHandler(trustService.NewTrustService(trustCli, cfg.Trust.Site), trustEnforce, cfg.Trust.CallbackSecret),
		RSSHandler:                     rssHandler.NewRSSHandler(rssRepo.NewRSSRepository(db), gc, uc),
		GalgameHandler:                 galgameHandler.NewGalgameHandler(galgameCoreSvc),
		GalgameCollectionHandler:       galgameHandler.NewGalgameCollectionHandler(galgameCollectionSvc),
		GalgameCommunityCommentHandler: galgameHandler.NewCommunityCommentHandler(galgameCommunityCommentSvc),
		ResourceCommentHandler:         galgameHandler.NewResourceCommentHandler(resourceCommentSvc),
		GalgameResourceHandler:         galgameHandler.NewResourceHandler(galgameResourceSvc),
		GalgameRatingHandler:           galgameHandler.NewRatingHandler(galgameRatingSvc),
		GalgameQuizHandler:             galgameHandler.NewQuizHandler(galgameQuizSvc),
		CreatorHandler:                 galgameHandler.NewCreatorHandler(creatorSvc),
		GalgameEntityHandler: galgameHandler.NewEntityHandler(
			galgameSeriesSvc, galgameOfficialSvc, galgameEngineSvc, galgameTagSvc,
		),
		GalgameCalendarHandler:     galgameHandler.NewCalendarHandler(galgameCalendarSvc),
		GalgameDraftsHandler:       galgameHandler.NewDraftsHandler(galgameDraftsSvc),
		GalgameWikiHandler:         galgameHandler.NewWikiHandler(galgameWikiSvc),
		GalgameSubmissionHandler:   galgameHandler.NewSubmissionHandler(galgameSubmissionSvc),
		GalgameMessageHandler:      galgameHandler.NewWikiMessageHandler(galgameMessageSvc),
		GalgameCrossSourceHandler:  galgameHandler.NewCrossSourceHandler(galgameCrossSourceSvc),
		ActivityHandler:            activityHandler.NewActivityHandler(activityService.NewActivityService(activityRepo.NewActivityRepository(db), gc, uc, rdb)),
		ImageHandler:               imageHandler.NewImageHandler(imageService.NewImageService(imageRepo.NewImageRepository(db), imgCli, gc)),
		SearchHandler:              searchHandler.NewSearchHandler(searchService.NewSearchService(searchRepo.NewSearchRepository(db), gc, galgameEnricher, uc)),
		ToolsetHandler:             toolsetHandler.NewToolsetHandler(toolsetCoreSvc),
		ToolsetPracticalityHandler: toolsetHandler.NewPracticalityHandler(toolsetPracticalitySvc),
		ToolsetResourceHandler:     toolsetHandler.NewResourceHandler(toolsetResourceSvc),
		ToolsetUploadHandler:       toolsetHandler.NewUploadHandler(toolsetUploadSvc),
		CronStop:                   cronPkg.Start(db, rdb, imgCli, galgameMessageSync.Run, galgameRevisionSync.Run),
	}

	// Fiber
	//
	// ReadBufferSize bumped to 16KB. Fiber's default is 4KB which is too
	// tight for our SSR request flow: the Nuxt server forwards the
	// authenticated user's cookies to /api, and once
	// pinia-plugin-persistedstate has spread several stores (user,
	// settings, sidebar, etc.) plus the OAuth session cookie across a
	// long-lived session, the Cookie header alone can creep past 4KB and
	// surface as the rather uninformative
	// "Request Header Fields Too Large" (which silently empties pages
	// because the SSR fetch fails). The frontend kunFetch only forwards
	// the session cookie now to keep things tight, but the bump here is
	// defense in depth for real-world browsers that accumulate
	// third-party cookies.
	fiberApp := fiber.New(fiber.Config{
		ErrorHandler:   globalErrorHandler,
		BodyLimit:      10 * 1024 * 1024,
		ReadBufferSize: 16 * 1024,
	})
	fiberApp.Use(recover.New())
	app.Fiber = fiberApp

	app.setupRoutes()
	return app
}

func globalErrorHandler(c fiber.Ctx, err error) error {
	if appErr, ok := err.(*errors.AppError); ok {
		return response.Error(c, appErr)
	}
	slog.Error("未处理的错误", "error", err.Error(), "path", c.Path(), "method", c.Method())
	return response.Error(c, errors.ErrInternal("服务器内部错误"))
}
