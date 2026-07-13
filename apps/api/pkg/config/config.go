package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	Redis          RedisConfig
	OAuth          OAuthConfig
	S3             S3Config // image bed (R2) — stickers, small inline images
	FileStorage    S3Config // file storage (B2) — toolset archive uploads
	Mail           MailConfig
	Search         SearchConfig
	CORS           CORSConfig
	GalgameWiki    GalgameWikiConfig
	ImageClient    ImageClientConfig
	ArtifactClient ArtifactClientConfig
	LinkChecker    LinkCheckerConfig
	Trust          TrustConfig
	Catalog        CatalogClientConfig
}

// CatalogClientConfig holds what kungal needs to read the infra Catalog service
// (kun-galgame-infra cmd/catalog :9281): the base URL for the S2S read face
// (credits + name/character reverse lookups behind the galgame detail page).
// Auth is HTTP Basic reusing the OAuth client_id/secret (wired in app.go) — the
// catalog READ face imposes no site binding, so kungal's OAuth client
// authenticates as-is. Empty BaseURL (or OAuth creds) = the integration is inert
// (the catalog proxy endpoints degrade to 503) so a dev box without a catalog
// service is harmless.
type CatalogClientConfig struct {
	BaseURL string // catalog service base, e.g. http://127.0.0.1:9281
}

// TrustConfig holds what kungal needs to integrate the infra Trust & Safety
// service (kun-galgame-infra :9283): the base URL for submitting reports S2S,
// and the HMAC secret for verifying inbound enforcement callbacks. The S2S
// Basic-auth credentials reuse the OAuth client_id/secret (wired in app.go) —
// the trust service reads oauth_clients.catalog_site to derive kungal's site.
// Empty BaseURL / CallbackSecret = the integration is inert (reports degrade,
// callbacks are rejected) so a dev box without a trust service is harmless.
type TrustConfig struct {
	BaseURL        string // trust service base, e.g. http://127.0.0.1:9283
	CallbackSecret string // HMAC secret shared with the trust subject-kind registry
	// Site is kungal's catalog_site key. Used ONLY to scope the moderator inbox
	// proxy (Phase 3) so kungal moderators see kungal's review items, not other
	// sites'. Must match the oauth_clients.catalog_site binding; a wrong value
	// yields an empty (but safe) inbox.
	Site string
}

// ArtifactClientConfig holds the credentials kungal uses to call the centralized
// artifact service (kun-galgame-infra :9279) for large-file (toolset archive)
// upload/download. Auth is HTTP Basic with an OAuth client_id/secret — the
// artifact service reuses the oauth_client table as its site registry (gated by
// artifact_enabled + artifact_site_key infra-side), so kungal's OAuth client IS
// its artifact site. ClientID/ClientSecret default to the OAuth credentials when
// unset (filled in app.go), so a single OAuth client works for both.
type ArtifactClientConfig struct {
	BaseURL      string // artifact service base, e.g. http://127.0.0.1:9279
	ClientID     string // OAuth client id (Basic auth); defaults to OAuth.ClientID
	ClientSecret string // OAuth client secret; defaults to OAuth.ClientSecret
}

// LinkCheckerConfig holds the s2s credentials kungal uses to call the
// kungal-link-live-checker service — the "report expired" gate that returns a
// conservative alive/dead/unknown verdict for a netdisk share link. When
// BaseURL/APIKey are unset the gate is skipped and a report falls back to the
// legacy single-report-expires behavior (see resource_service.MarkExpired).
type LinkCheckerConfig struct {
	BaseURL string // checker base, e.g. https://link-checker-kungal.nextmoe.dev
	APIKey  string // service Bearer key (one of the checker's LLC_API_KEYS)
	// Cloudflare Access service-token headers. The checker sits behind CF Access
	// (zero public anonymous exposure), so every request must carry
	// CF-Access-Client-Id / -Secret on top of the Bearer key. Leave both empty
	// when reaching a checker NOT behind CF Access (e.g. a local dev instance).
	CFAccessClientID     string
	CFAccessClientSecret string
}

// ImageClientConfig holds the credentials kungal uses to call the image
// service directly (multi-image upload paths for galgame covers /
// screenshots — U2). Distinct from GalgameWikiConfig.ImageCDNBase which
// is just the public URL prefix for the response-rewrite walker.
//
// Set the three env vars below; the public CDN prefix is shared with
// the walker (GalgameWikiConfig.ImageCDNBase) so it isn't duplicated.
// When ClientID/ClientSecret are unset, kungal's upload endpoints
// return a clear error (no silent fallback to a misconfigured client).
type ImageClientConfig struct {
	BaseURL      string // image service base, e.g. http://127.0.0.1:9278
	ClientID     string // OAuth client id (Basic auth)
	ClientSecret string // OAuth client secret
}

type GalgameWikiConfig struct {
	BaseURL string
	// ImageCDNBase is the image_service public CDN prefix (no trailing
	// slash), identical to the wiki's KUN_IMAGE_PUBLIC_BASE_URL. Wiki
	// returns image_service-backed banners as banner="" + a
	// banner_image_hash; kungal resolves the hash → CDN URL server-side
	// (in the galgame client) so every downstream banner stays a plain
	// usable URL. See docs/galgame_wiki/07-submission.md §banner and
	// docs/image_service/06-integration-guide.md.
	ImageCDNBase string
}

type ServerConfig struct {
	Port string
	Mode string // "dev" or "prod"
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int // seconds
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type OAuthConfig struct {
	ServerURL    string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	JWTSecret    string
}

type S3Config struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
}

type MailConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

type SearchConfig struct {
	MeilisearchURL string
	MeilisearchKey string
}

type CORSConfig struct {
	AllowOrigins string
}

func Load() (*Config, error) {
	dbURL, err := requireEnv("KUN_DATABASE_URL")
	if err != nil {
		return nil, err
	}
	oauthServerURL, err := requireEnv("OAUTH_SERVER_URL")
	if err != nil {
		return nil, err
	}
	oauthClientID, err := requireEnv("OAUTH_CLIENT_ID")
	if err != nil {
		return nil, err
	}
	oauthClientSecret, err := requireEnv("OAUTH_CLIENT_SECRET")
	if err != nil {
		return nil, err
	}
	oauthRedirectURI, err := requireEnv("OAUTH_REDIRECT_URI")
	if err != nil {
		return nil, err
	}

	return &Config{
		Server: ServerConfig{
			Port: envOrDefault("SERVER_PORT", "2334"),
			Mode: envOrDefault("SERVER_MODE", "dev"),
		},
		Database: DatabaseConfig{
			URL:             dbURL,
			MaxOpenConns:    envOrDefaultInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    envOrDefaultInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: envOrDefaultInt("DB_CONN_MAX_LIFETIME", 300),
		},
		Redis: RedisConfig{
			Host:     envOrDefault("REDIS_HOST", "127.0.0.1"),
			Port:     envOrDefault("REDIS_PORT", "6379"),
			Password: envOrDefault("REDIS_PASSWORD", ""),
			DB:       envOrDefaultInt("REDIS_DB", 0),
		},
		OAuth: OAuthConfig{
			ServerURL:    oauthServerURL,
			ClientID:     oauthClientID,
			ClientSecret: oauthClientSecret,
			RedirectURI:  oauthRedirectURI,
			JWTSecret:    envOrDefault("JWT_SECRET", ""),
		},
		S3: S3Config{
			Endpoint:  envOrDefault("S3_ENDPOINT", ""),
			Region:    envOrDefault("S3_REGION", ""),
			Bucket:    envOrDefault("S3_BUCKET", ""),
			AccessKey: envOrDefault("S3_ACCESS_KEY", ""),
			SecretKey: envOrDefault("S3_SECRET_KEY", ""),
		},
		// Separate bucket for archive uploads (Backblaze B2 in production).
		// Toolset .7z/.zip/.rar files live here; the image bed (R2) above
		// is dedicated to small images. Falls back to the image-bed config
		// so a dev box without B2 still serves a single bucket — but the
		// CORS rules of the two buckets are usually different, so prod
		// must set FILE_STORAGE_* explicitly.
		FileStorage: S3Config{
			Endpoint:  envOrDefault("FILE_STORAGE_ENDPOINT", ""),
			Region:    envOrDefault("FILE_STORAGE_REGION", ""),
			Bucket:    envOrDefault("FILE_STORAGE_BUCKET", ""),
			AccessKey: envOrDefault("FILE_STORAGE_ACCESS_KEY", ""),
			SecretKey: envOrDefault("FILE_STORAGE_SECRET_KEY", ""),
		},
		Mail: MailConfig{
			Host:     envOrDefault("MAIL_HOST", ""),
			Port:     envOrDefaultInt("MAIL_PORT", 587),
			User:     envOrDefault("MAIL_USER", ""),
			Password: envOrDefault("MAIL_PASSWORD", ""),
			From:     envOrDefault("MAIL_FROM", ""),
		},
		Search: SearchConfig{
			MeilisearchURL: envOrDefault("MEILISEARCH_URL", "http://127.0.0.1:7700"),
			MeilisearchKey: envOrDefault("MEILISEARCH_KEY", ""),
		},
		CORS: CORSConfig{
			AllowOrigins: envOrDefault(
				"CORS_ALLOW_ORIGINS",
				"http://127.0.0.1:2333,https://www.kungal.com",
			),
		},
		GalgameWiki: GalgameWikiConfig{
			BaseURL: envOrDefault("GALGAME_WIKI_BASE_URL", "http://127.0.0.1:9280/api"),
			// Must match the wiki's KUN_IMAGE_PUBLIC_BASE_URL exactly —
			// both build the same {base}/{hh}/{hh}/{hash}.webp layout.
			ImageCDNBase: envOrDefault("KUN_IMAGE_PUBLIC_BASE_URL", "https://image.kungal.iloveren.link"),
		},
		ImageClient: ImageClientConfig{
			BaseURL:      envOrDefault("KUN_IMAGE_CLIENT_BASE_URL", "http://127.0.0.1:9278"),
			ClientID:     envOrDefault("KUN_IMAGE_CLIENT_ID", ""),
			ClientSecret: envOrDefault("KUN_IMAGE_CLIENT_SECRET", ""),
		},
		ArtifactClient: ArtifactClientConfig{
			BaseURL:      envOrDefault("KUN_ARTIFACT_CLIENT_BASE_URL", "http://127.0.0.1:9279"),
			ClientID:     envOrDefault("KUN_ARTIFACT_CLIENT_ID", ""),
			ClientSecret: envOrDefault("KUN_ARTIFACT_CLIENT_SECRET", ""),
		},
		LinkChecker: LinkCheckerConfig{
			BaseURL:              envOrDefault("LINK_CHECKER_BASE_URL", ""),
			APIKey:               envOrDefault("LINK_CHECKER_API_KEY", ""),
			CFAccessClientID:     envOrDefault("CF_ACCESS_CLIENT_ID", ""),
			CFAccessClientSecret: envOrDefault("CF_ACCESS_CLIENT_SECRET", ""),
		},
		Trust: TrustConfig{
			BaseURL:        envOrDefault("KUN_TRUST_BASE_URL", "http://127.0.0.1:9283"),
			CallbackSecret: envOrDefault("KUN_TRUST_CALLBACK_SECRET", ""),
			Site:           envOrDefault("KUN_TRUST_SITE", "kungal"),
		},
		Catalog: CatalogClientConfig{
			BaseURL: envOrDefault("KUN_CATALOG_API_BASE", "http://127.0.0.1:9281"),
		},
	}, nil
}

func requireEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("环境变量 %s 未设置", key)
	}
	return val, nil
}

func envOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func envOrDefaultInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return fallback
}
