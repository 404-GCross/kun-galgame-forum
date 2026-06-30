package app

import (
	"log/slog"

	activityHandler "kun-galgame-api/internal/activity/handler"
	activityRepo "kun-galgame-api/internal/activity/repository"
	activityService "kun-galgame-api/internal/activity/service"
	adminHandler "kun-galgame-api/internal/admin/handler"
	adminRepo "kun-galgame-api/internal/admin/repository"
	adminService "kun-galgame-api/internal/admin/service"
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
	reportHandler "kun-galgame-api/internal/report/handler"
	reportRepo "kun-galgame-api/internal/report/repository"
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
	"kun-galgame-api/pkg/config"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/imageclient"
	"kun-galgame-api/pkg/linkcheck"
	"kun-galgame-api/pkg/response"
	"kun-galgame-api/pkg/userclient"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	Fiber       *fiber.App
	DB          *gorm.DB
	Redis       *redis.Client
	S3          *storage.S3Client
	Mailer      *mail.Mailer
	Config      *config.Config
	OAuthClient *oauth.Client
	UserState   *repository.StateRepository
	UserClient  *userclient.Client

	// Handlers
	OAuthHandler               *handler.OAuthHandler
	UserHandler                *handler.UserHandler
	UserProfileHandler         *handler.ProfileHandler
	HomeHandler                *homeHandler.HomeHandler
	TopicHandler               *topicHandler.TopicHandler
	ReplyHandler               *topicHandler.ReplyHandler
	TopicCommentHandler        *topicHandler.CommentHandler
	PollHandler                *topicHandler.PollHandler
	MessageHandler             *msgHandler.MessageHandler
	MessageChatHandler         *msgHandler.ChatHandler
	AdminOverviewHandler       *adminHandler.OverviewHandler
	AdminPurgeHandler          *adminHandler.PurgeHandler
	RankingHandler             *rankingHandler.RankingHandler
	SectionHandler             *sectionHandler.SectionHandler
	DocArticleHandler          *docHandler.ArticleHandler
	DocCategoryHandler         *docHandler.CategoryHandler
	DocTagHandler              *docHandler.TagHandler
	WebsiteHandler             *websiteHandler.WebsiteHandler
	WebsiteCommentHandler      *websiteHandler.CommentHandler
	WebsiteCategoryHandler     *websiteHandler.CategoryHandler
	WebsiteTagHandler          *websiteHandler.TagHandler
	UpdateHandler              *updateHandler.UpdateHandler
	FriendLinkHandler          *friendHandler.FriendLinkHandler
	ReportHandler              *reportHandler.ReportHandler
	RSSHandler                 *rssHandler.RSSHandler
	GalgameHandler             *galgameHandler.GalgameHandler
	GalgameCommentHandler      *galgameHandler.CommentHandler
	GalgameResourceHandler     *galgameHandler.ResourceHandler
	GalgameRatingHandler       *galgameHandler.RatingHandler
	CreatorHandler             *galgameHandler.CreatorHandler
	GalgameEntityHandler       *galgameHandler.EntityHandler
	GalgameWikiHandler         *galgameHandler.WikiHandler
	GalgameSubmissionHandler   *galgameHandler.SubmissionHandler
	GalgameMessageHandler      *galgameHandler.WikiMessageHandler
	ActivityHandler            *activityHandler.ActivityHandler
	ImageHandler               *imageHandler.ImageHandler
	SearchHandler              *searchHandler.SearchHandler
	ToolsetHandler             *toolsetHandler.ToolsetHandler
	ToolsetPracticalityHandler *toolsetHandler.PracticalityHandler
	ToolsetCommentHandler      *toolsetHandler.CommentHandler
	ToolsetResourceHandler     *toolsetHandler.ResourceHandler
	ToolsetUploadHandler       *toolsetHandler.UploadHandler
	CronStop                   func()
}

func New(cfg *config.Config) *App {
	// Infrastructure
	db := database.NewPostgres(cfg.Database, cfg.Server.Mode)
	rdb := cache.NewRedis(cfg.Redis)
	// Two distinct buckets:
	// - s3Client: image bed (R2). Stickers / inline images / image_service
	//   fallbacks. Configured via S3_* env vars.
	// - fileStorageClient: archive storage (B2). Toolset .7z/.zip/.rar
	//   uploads via presigned URLs. Configured via FILE_STORAGE_* env
	//   vars. B2 has different CORS rules from R2 (B2 supports browser
	//   PUT preflight cleanly), so it must be a separate client even
	//   though both implement the S3 API.
	s3Client := storage.NewS3(cfg.S3)
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
	userService := service.NewUserService(userStateRepo, userStatsRepo, rdb, gc, uc)
	userContentService := service.NewUserContentService(userContentRepo, gc, uc)
	messageSvc := msgService.NewMessageService(messageRepository, uc)
	chatSvc := msgService.NewChatService(chatRepository, uc)
	notifier := msgService.NewNotifier(messageRepository)

	// Topic
	topicRepository := topicRepo.NewTopicRepository(db)
	topicListRepo := topicRepo.NewTopicListRepository(db)
	topicTaxonomyRepo := topicRepo.NewTopicTaxonomyRepository(db)
	replyRepository := topicRepo.NewReplyRepository(db)
	topicCommentRepo := topicRepo.NewCommentRepository(db)
	pollRepository := topicRepo.NewPollRepository(db)
	topicSvc := topicService.NewTopicService(topicRepository, topicListRepo, topicTaxonomyRepo, rdb, uc, userStateRepo)
	topicWriteSvc := topicService.NewTopicWriteService(topicRepository, topicTaxonomyRepo, replyRepository, userStateRepo, rdb, notifier)
	replySvc := topicService.NewReplyService(replyRepository, topicCommentRepo, topicRepository, userStateRepo, uc, rdb)
	commentSvc := topicService.NewCommentService(replyRepository, topicCommentRepo, userStateRepo, uc, rdb)
	pollSvc := topicService.NewPollService(pollRepository, topicRepository, userStateRepo, uc, rdb)

	// Galgame
	galgameCommentRepo := galgameRepo.NewCommentRepository(db)
	galgameCommentSvc := galgameService.NewCommentService(galgameCommentRepo, userStateRepo, uc)
	galgameResourceRepo := galgameRepo.NewResourceRepository(db)
	galgameResourceSvc := galgameService.NewResourceService(galgameResourceRepo, gc, uc, linkChecker)
	galgameRatingRepo := galgameRepo.NewRatingRepository(db)
	galgameRatingSvc := galgameService.NewRatingService(galgameRatingRepo, gc, uc)
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
	galgameSeriesSvc := galgameService.NewSeriesService(gc, galgameEnricher)
	galgameOfficialSvc := galgameService.NewOfficialService(gc, galgameEnricher)
	galgameEngineSvc := galgameService.NewEngineService(gc, galgameEnricher)
	galgameTagSvc := galgameService.NewTagService(gc, galgameEnricher)
	galgameWikiSvc := galgameService.NewWikiService(gc, galgameLocalRepo, uc)
	// Submission flow: submit / claim / patch-draft / delete-draft proxies
	// + local moemoepoint side effects. Per docs/galgame_wiki/07-submission.md.
	galgameSubmissionSvc := galgameService.NewSubmissionService(gc, galgameLocalRepo)

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
	websiteCommentRepo := websiteRepo.NewCommentRepository(db)
	websiteCoreSvc := websiteService.NewWebsiteService(
		websiteRepository, websiteCategoryRepo, websiteTagRepo, websiteCommentRepo, uc, cfg.GalgameWiki.ImageCDNBase,
	)
	websiteCommentSvc := websiteService.NewCommentService(websiteCommentRepo, websiteRepository, notifier, uc)
	websiteCategorySvc := websiteService.NewCategoryService(websiteCategoryRepo, websiteRepository, websiteTagRepo, cfg.GalgameWiki.ImageCDNBase)
	websiteTagSvc := websiteService.NewTagService(websiteTagRepo, websiteRepository, websiteCategoryRepo, cfg.GalgameWiki.ImageCDNBase)

	// Admin
	adminOverviewRepo := adminRepo.NewOverviewRepository(db)
	adminOverviewSvc := adminService.NewOverviewService(adminOverviewRepo, gc)
	adminPurgeSvc := adminService.NewPurgeService(adminRepo.NewPurgeRepository(db), uc)

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
	toolsetCommentSvc := toolsetService.NewCommentService(toolsetCommentRepo, toolsetRepository, uc)
	toolsetResourceSvc := toolsetService.NewResourceService(toolsetResourceRepo, toolsetRepository, fileStorageClient, artCli, uc)
	toolsetUploadSvc := toolsetService.NewUploadService(artCli, rdb, db)
	toolsetCoreSvc := toolsetService.NewToolsetService(
		toolsetRepository, toolsetResourceRepo, toolsetCommentRepo, toolsetPracticalityRepo,
		fileStorageClient, uc, toolsetPracticalitySvc, toolsetCommentSvc,
	)

	// Handlers
	app := &App{
		DB: db, Redis: rdb, S3: s3Client, Mailer: mailer, Config: cfg, OAuthClient: oauthClient,
		UserState:              userStateRepo,
		UserClient:             uc,
		OAuthHandler:           handler.NewOAuthHandler(authService, cfg.Server.Mode == "prod"),
		UserHandler:            handler.NewUserHandler(userService, userContentService),
		UserProfileHandler:     handler.NewProfileHandler(oauthClient, uc),
		HomeHandler:            homeHandler.NewHomeHandler(homeService.NewHomeService(homeRepo.NewHomeRepository(db), gc, uc, rdb)),
		TopicHandler:           topicHandler.NewTopicHandler(topicSvc, topicWriteSvc),
		ReplyHandler:           topicHandler.NewReplyHandler(replySvc),
		TopicCommentHandler:    topicHandler.NewCommentHandler(commentSvc),
		PollHandler:            topicHandler.NewPollHandler(pollSvc),
		MessageHandler:         msgHandler.NewMessageHandler(messageSvc),
		MessageChatHandler:     msgHandler.NewChatHandler(chatSvc),
		AdminOverviewHandler:   adminHandler.NewOverviewHandler(adminOverviewSvc),
		AdminPurgeHandler:      adminHandler.NewPurgeHandler(adminPurgeSvc),
		RankingHandler:         rankingHandler.NewRankingHandler(rankingService.NewRankingService(rankingRepo.NewRankingRepository(db), gc, uc)),
		SectionHandler:         sectionHandler.NewSectionHandler(sectionService.NewSectionService(sectionRepo.NewSectionRepository(db), uc)),
		DocArticleHandler:      docHandler.NewArticleHandler(docArticleSvc),
		DocCategoryHandler:     docHandler.NewCategoryHandler(docCategorySvc),
		DocTagHandler:          docHandler.NewTagHandler(docTagSvc),
		WebsiteHandler:         websiteHandler.NewWebsiteHandler(websiteCoreSvc),
		WebsiteCommentHandler:  websiteHandler.NewCommentHandler(websiteCommentSvc),
		WebsiteCategoryHandler: websiteHandler.NewCategoryHandler(websiteCategorySvc),
		WebsiteTagHandler:      websiteHandler.NewTagHandler(websiteTagSvc),
		UpdateHandler:          updateHandler.NewUpdateHandler(updateRepo.NewUpdateRepository(db)),
		FriendLinkHandler:      friendHandler.NewFriendLinkHandler(friendRepo.NewFriendLinkRepository(db), cfg.GalgameWiki.ImageCDNBase),
		ReportHandler:          reportHandler.NewReportHandler(reportRepo.NewReportRepository(db)),
		RSSHandler:             rssHandler.NewRSSHandler(rssRepo.NewRSSRepository(db), gc, uc),
		GalgameHandler:         galgameHandler.NewGalgameHandler(galgameCoreSvc),
		GalgameCommentHandler:  galgameHandler.NewCommentHandler(galgameCommentSvc),
		GalgameResourceHandler: galgameHandler.NewResourceHandler(galgameResourceSvc),
		GalgameRatingHandler:   galgameHandler.NewRatingHandler(galgameRatingSvc),
		CreatorHandler:         galgameHandler.NewCreatorHandler(creatorSvc),
		GalgameEntityHandler: galgameHandler.NewEntityHandler(
			galgameSeriesSvc, galgameOfficialSvc, galgameEngineSvc, galgameTagSvc,
		),
		GalgameWikiHandler:         galgameHandler.NewWikiHandler(galgameWikiSvc),
		GalgameSubmissionHandler:   galgameHandler.NewSubmissionHandler(galgameSubmissionSvc),
		GalgameMessageHandler:      galgameHandler.NewWikiMessageHandler(galgameMessageSvc),
		ActivityHandler:            activityHandler.NewActivityHandler(activityService.NewActivityService(activityRepo.NewActivityRepository(db), gc, uc, rdb)),
		ImageHandler:               imageHandler.NewImageHandler(imageService.NewImageService(imageRepo.NewImageRepository(db), s3Client, imgCli, gc)),
		SearchHandler:              searchHandler.NewSearchHandler(searchService.NewSearchService(searchRepo.NewSearchRepository(db), gc, galgameEnricher, uc)),
		ToolsetHandler:             toolsetHandler.NewToolsetHandler(toolsetCoreSvc),
		ToolsetPracticalityHandler: toolsetHandler.NewPracticalityHandler(toolsetPracticalitySvc),
		ToolsetCommentHandler:      toolsetHandler.NewCommentHandler(toolsetCommentSvc),
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

func globalErrorHandler(c *fiber.Ctx, err error) error {
	if appErr, ok := err.(*errors.AppError); ok {
		return response.Error(c, appErr)
	}
	slog.Error("未处理的错误", "error", err.Error(), "path", c.Path(), "method", c.Method())
	return response.Error(c, errors.ErrInternal("服务器内部错误"))
}
