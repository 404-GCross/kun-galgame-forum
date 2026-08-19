package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"kun-galgame-api/pkg/errors"
)

const briefCacheTTL = 2 * time.Minute

const batchCacheMaxEntries = 4096

type batchCacheKey struct {
	id  int
	sfw bool
}

type batchCacheEntry[T any] struct {
	found  bool
	val    T
	expire time.Time
}

func cachedBatch[T any](
	mu *sync.RWMutex,
	cache map[batchCacheKey]batchCacheEntry[T],
	ids []int,
	sfw bool,
	fetch func([]int) (map[int]T, *errors.AppError),
) (map[int]T, *errors.AppError) {
	result := make(map[int]T, len(ids))
	var missing []int
	now := time.Now()
	mu.RLock()
	for _, id := range ids {
		if e, ok := cache[batchCacheKey{id, sfw}]; ok && now.Before(e.expire) {
			if e.found {
				result[id] = e.val
			}
		} else {
			missing = append(missing, id)
		}
	}
	mu.RUnlock()
	if len(missing) == 0 {
		return result, nil
	}
	fetched, appErr := fetch(missing)
	if appErr != nil {
		return nil, appErr
	}
	expire := now.Add(briefCacheTTL)
	mu.Lock()
	if len(cache) > batchCacheMaxEntries {
		clear(cache)
	}
	for _, id := range missing {
		v, ok := fetched[id]
		cache[batchCacheKey{id, sfw}] = batchCacheEntry[T]{found: ok, val: v, expire: expire}
		if ok {
			result[id] = v
		}
	}
	mu.Unlock()
	return result, nil
}

type GalgameClient struct {
	v1Base       string
	apiKey       string
	httpClient   *http.Client
	imageCDNBase string
	imageMeta    ImageMetaResolver

	briefMu     sync.RWMutex
	briefCache  map[batchCacheKey]batchCacheEntry[GalgameBrief]
	detailMu    sync.RWMutex
	detailCache map[batchCacheKey]batchCacheEntry[GalgameDetailBrief]

	labelLinkMu    sync.RWMutex
	labelLinkCache map[batchCacheKey]batchCacheEntry[string]

	tagSexualMu    sync.RWMutex
	tagSexualCache map[batchCacheKey]batchCacheEntry[bool]

	gidMu    sync.RWMutex
	gidCache map[int]gidLookupEntry
}

func New(baseURL, apiKey, imageCDNBase string) *GalgameClient {
	base := strings.TrimRight(baseURL, "/")

	// net/http defaults MaxIdleConnsPerHost to 2, which throttles a single-host
	// S2S client: concurrent callers cannot reuse keep-alives past 2 and pay a
	// fresh dial per request.
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 64

	return &GalgameClient{
		v1Base: base + "/v1",
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
			// Never follow a redirect. The catalog answers a merged entity id with
			// 301 + current_id so the caller can redirect the BROWSER in one hop;
			// auto-following swallows that and returns the survivor's record under
			// the dead id — a duplicate page on two URLs, which is what the 301
			// exists to prevent.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		imageCDNBase: imageCDNBase,
		briefCache:   map[batchCacheKey]batchCacheEntry[GalgameBrief]{},
		detailCache:  map[batchCacheKey]batchCacheEntry[GalgameDetailBrief]{},
		gidCache:     map[int]gidLookupEntry{},

		labelLinkCache: map[batchCacheKey]batchCacheEntry[string]{},
		tagSexualCache: map[batchCacheKey]batchCacheEntry[bool]{},
	}
}

func (c *GalgameClient) getFace(ctx context.Context, base, path, token string, query url.Values, apiKey string) (json.RawMessage, *errors.AppError) {
	reqURL := base + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, errors.ErrInternal("创建请求失败")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	return c.doRequest(req)
}

func (c *GalgameClient) postFace(ctx context.Context, base, path string, query url.Values, body any, apiKey string) (json.RawMessage, *errors.AppError) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, errors.ErrInternal("序列化请求失败")
	}
	reqURL := base + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(payload))
	if err != nil {
		return nil, errors.ErrInternal("创建请求失败")
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	return c.doRequest(req)
}

type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (c *GalgameClient) GetV1(ctx context.Context, path string, query url.Values) (json.RawMessage, *errors.AppError) {
	return c.getFace(ctx, c.v1Base, path, "", query, c.apiKey)
}

const catalogMovedCode = 12

func (c *GalgameClient) getV1Envelope(ctx context.Context, path string, query url.Values) (int, *apiResponse, *errors.AppError) {
	reqURL := c.v1Base + path
	if len(query) > 0 {
		reqURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return 0, nil, errors.ErrInternal("创建请求失败")
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("Galgame 服务请求失败 (传输层)",
			"method", req.Method, "url", req.URL.String(), "error", err)
		return 0, nil, errors.ErrInternal(fmt.Sprintf("Galgame 服务不可达: %v", err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, errors.ErrInternal("读取 Galgame 响应失败")
	}
	var result apiResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		slog.Error("解析 Galgame 响应失败 (非 JSON 响应)",
			"method", req.Method, "url", req.URL.String(),
			"status", resp.StatusCode, "body", bodySnippet(respBody))
		return resp.StatusCode, nil, errors.New(errors.CodeBiz,
			fmt.Sprintf("Galgame 服务返回了非预期响应 (HTTP %d)", resp.StatusCode), resp.StatusCode)
	}
	result.Data = rewriteBanners(result.Data, c.imageCDNBase)
	return resp.StatusCode, &result, nil
}

func BriefName(b *GalgameBrief) string {
	if b == nil {
		return ""
	}
	for _, n := range []string{b.NameZhCn, b.NameZhTw, b.NameJaJp, b.NameEnUs} {
		if n != "" {
			return n
		}
	}
	return ""
}

type GalgameBrief struct {
	ID                  int     `json:"id"`
	WorkID              int64   `json:"work_id"`
	VndbID              string  `json:"vndb_id"`
	NameEnUs            string  `json:"name_en_us"`
	NameJaJp            string  `json:"name_ja_jp"`
	NameZhCn            string  `json:"name_zh_cn"`
	NameZhTw            string  `json:"name_zh_tw"`
	Banner              string  `json:"banner"`
	Status              int     `json:"status"`
	ClaimState          string  `json:"claim_state,omitempty"`
	ContentLimit        string  `json:"content_limit"`
	UserID              int     `json:"user_id"`
	OriginalLanguage    string  `json:"original_language"`
	AgeLimit            string  `json:"age_limit"`
	ReleaseDate         *string `json:"release_date"`
	ReleaseDateTBA      bool    `json:"release_date_tba"`
	EffectiveBannerHash string  `json:"effective_banner_hash"`
	// rewriteBanners injects effective_banner_url into the raw JSON BEFORE this
	// struct is unmarshalled. The field must be declared or Go silently drops
	// the walker's work and every downstream DTO is left with only the hash.
	EffectiveBannerURL       string            `json:"effective_banner_url"`
	EffectiveBannerWidth     int               `json:"effective_banner_width,omitempty"`
	EffectiveBannerHeight    int               `json:"effective_banner_height,omitempty"`
	EffectiveBannerThumbhash string            `json:"effective_banner_thumbhash,omitempty"`
	Refs                     map[string]string `json:"refs,omitempty"`
}

func (b GalgameBrief) DlsiteWorkno() string { return b.Refs["dlsite"] }

type GalgameDetailBrief struct {
	GalgameBrief
	IntroEnUS string   `json:"intro_en_us"`
	IntroJaJP string   `json:"intro_ja_jp"`
	IntroZhCN string   `json:"intro_zh_cn"`
	IntroZhTW string   `json:"intro_zh_tw"`
	Officials []string `json:"officials"`
}

func (c *GalgameClient) GetBatchDetailPublic(ctx context.Context, ids []int, isSFW bool) (map[int]GalgameDetailBrief, *errors.AppError) {
	return cachedBatch(&c.detailMu, c.detailCache, ids, isSFW, func(miss []int) (map[int]GalgameDetailBrief, *errors.AppError) {
		rows, appErr := c.CatalogRowsByGIDs(ctx, miss, catalogDetailBriefInclude, contentLimitFor(isSFW))
		if appErr != nil {
			return nil, appErr
		}
		result := make(map[int]GalgameDetailBrief, len(rows))
		for gid := range rows {
			row := rows[gid]
			result[gid] = CatalogItemToDetailBrief(&row)
		}
		return result, nil
	})
}

func (c *GalgameClient) GetBatch(ctx context.Context, ids []int) (map[int]GalgameBrief, *errors.AppError) {
	return c.batchByGIDs(ctx, ids, "all")
}

func (c *GalgameClient) GetBatchPublic(ctx context.Context, ids []int, isSFW bool) (map[int]GalgameBrief, *errors.AppError) {
	return cachedBatch(&c.briefMu, c.briefCache, ids, isSFW, func(miss []int) (map[int]GalgameBrief, *errors.AppError) {
		return c.batchByGIDs(ctx, miss, contentLimitFor(isSFW))
	})
}

func (c *GalgameClient) batchByGIDs(ctx context.Context, ids []int, contentLimit string) (map[int]GalgameBrief, *errors.AppError) {
	if len(ids) == 0 {
		return map[int]GalgameBrief{}, nil
	}
	rows, appErr := c.CatalogRowsByGIDs(ctx, ids, catalogBriefInclude, contentLimit)
	if appErr != nil {
		return nil, appErr
	}
	result := make(map[int]GalgameBrief, len(rows))
	for gid := range rows {
		row := rows[gid]
		result[gid] = CatalogItemToBrief(&row)
	}
	return result, nil
}

func bodySnippet(b []byte) string {
	const max = 300
	if len(b) > max {
		return string(b[:max]) + "…"
	}
	return string(b)
}

func (c *GalgameClient) doRequest(req *http.Request) (json.RawMessage, *errors.AppError) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		slog.Error("Galgame 服务请求失败 (传输层)",
			"method", req.Method, "url", req.URL.String(), "error", err)
		return nil, errors.ErrInternal(fmt.Sprintf("Galgame 服务不可达: %v", err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("读取 Galgame 响应失败",
			"method", req.Method, "url", req.URL.String(), "error", err)
		return nil, errors.ErrInternal("读取 Galgame 响应失败")
	}

	var result apiResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		slog.Error("解析 Galgame 响应失败 (非 JSON 响应)",
			"method", req.Method, "url", req.URL.String(),
			"status", resp.StatusCode, "body", bodySnippet(respBody))
		return nil, errors.New(
			errors.CodeBiz,
			fmt.Sprintf(
				"Galgame 服务返回了非预期响应 (HTTP %d), 请确认 galgame 服务已部署对应接口",
				resp.StatusCode,
			),
			resp.StatusCode,
		)
	}

	if result.Code != 0 {
		return nil, errors.New(result.Code, result.Message, resp.StatusCode)
	}

	return rewriteBanners(result.Data, c.imageCDNBase), nil
}
