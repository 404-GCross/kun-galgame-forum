package service

import (
	"context"
	stderrors "errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"kun-galgame-api/internal/constants"
	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/moemoepoint"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
)

type SubmissionService struct {
	galgameClient *client.GalgameClient
	catalog       *catalogclient.Client
	galgameRepo   *repository.GalgameRepository
}

func NewSubmissionService(
	galgameClient *client.GalgameClient,
	catalog *catalogclient.Client,
	galgameRepo *repository.GalgameRepository,
) *SubmissionService {
	return &SubmissionService{
		galgameClient: galgameClient,
		catalog:       catalog,
		galgameRepo:   galgameRepo,
	}
}

const submissionSite = client.ClaimSiteKungal

type SubmitResult struct {
	GID        int    `json:"gid"`
	WorkID     int64  `json:"work_id"`
	ClaimState string `json:"claim_state"`
}

func (s *SubmissionService) Submit(
	ctx context.Context,
	accessToken string,
	form *SubmissionForm,
) (*SubmitResult, *errors.AppError) {
	if form.DisplayName() == "" {
		return nil, errors.ErrValidation("请至少填写一个语言的标题")
	}
	released, appErr := form.Released()
	if appErr != nil {
		return nil, appErr
	}
	res, err := s.catalog.SubmitWorkUser(ctx, accessToken, catalogclient.UserWorkSubmitRequest{
		Fields: form.Fields(), Released: released,
	})
	if err != nil {
		return nil, claimActionError(err)
	}
	gid := int(res.ProductWorkID)
	if patch := form.CoverPatch(); patch != nil {
		if _, err := s.catalog.CreateEditProposalUser(ctx, accessToken, catalogclient.UserEditCreateRequest{
			EntityType: catalogclient.EntityTypeWork, EntityID: res.WorkID,
			Patch: patch, Note: "投稿时提交的横幅图",
		}); err != nil {
			slog.Warn("submit: 附加横幅图失败", "work", res.WorkID, "error", err)
		}
	}
	return &SubmitResult{GID: gid, WorkID: res.WorkID, ClaimState: res.ClaimState}, nil
}

func (s *SubmissionService) Claim(
	ctx context.Context,
	accessToken string,
	uid int64,
	gid int,
) (*catalogclient.ClaimActionResult, *errors.AppError) {
	res, appErr := s.act(ctx, accessToken, gid, catalogclient.ClaimActionPublish, "")
	if appErr != nil {
		return nil, appErr
	}
	if err := s.galgameRepo.Touch(s.galgameRepo.DB().WithContext(ctx), gid); err != nil {
		slog.Warn("claim: 刷新本地 galgame resource_update_time 失败", "gid", gid, "error", err)
	}
	moemoepoint.Award(int(uid), constants.RewardCreateGalgame,
		moemoepoint.ReasonContentApproved, moemoepoint.Ref("galgame", gid),
		moemoepoint.Key("claim", strconv.Itoa(gid), strconv.FormatInt(uid, 10)))
	return res, nil
}

func (s *SubmissionService) Resubmit(
	ctx context.Context,
	accessToken string,
	gid int,
) (*catalogclient.ClaimActionResult, *errors.AppError) {
	return s.act(ctx, accessToken, gid, catalogclient.ClaimActionSubmit, "")
}

func (s *SubmissionService) Withdraw(
	ctx context.Context,
	accessToken string,
	gid int,
) (*catalogclient.ClaimActionResult, *errors.AppError) {
	return s.act(ctx, accessToken, gid, catalogclient.ClaimActionWithdraw, "")
}

func (s *SubmissionService) act(
	ctx context.Context,
	accessToken string,
	gid int,
	action string,
	reason string,
) (*catalogclient.ClaimActionResult, *errors.AppError) {
	workID, appErr := s.workIDOf(ctx, gid)
	if appErr != nil {
		return nil, appErr
	}
	res, err := s.catalog.ActOnClaimUser(ctx, accessToken, workID, action, catalogclient.UserClaimActionRequest{
		Reason: reason,
	})
	if err != nil {
		return nil, claimActionError(err)
	}
	return res, nil
}

func (s *SubmissionService) workIDOf(ctx context.Context, gid int) (int64, *errors.AppError) {
	ids, appErr := s.galgameClient.CatalogWorkIDs(ctx, []int{gid})
	if appErr != nil {
		return 0, appErr
	}
	workID, ok := ids[gid]
	if !ok {
		return 0, errors.ErrNotFound("条目不存在")
	}
	return workID, nil
}

func claimActionError(err error) *errors.AppError {
	var apiErr *catalogclient.UserAPIError
	switch {
	case stderrors.Is(err, catalogclient.ErrInsufficientScope):
		return errors.ErrReauthRequired("投稿需要新的授权，请退出登录后重新登录以授予该权限")
	case stderrors.Is(err, catalogclient.ErrUnauthorized):
		return errors.ErrAuthExpired()
	case stderrors.Is(err, catalogclient.ErrNotFound):
		return errors.ErrNotFound("条目不存在")
	case stderrors.Is(err, catalogclient.ErrNotConfigured):
		return errors.New(errors.CodeBiz, "资料库服务暂不可用", http.StatusServiceUnavailable)
	case stderrors.As(err, &apiErr):
		switch apiErr.Status {
		case http.StatusForbidden:
			return errors.ErrForbidden("你没有权限执行此操作")
		case http.StatusUnprocessableEntity:
			return errors.ErrValidation(apiErr.Message)
		case http.StatusConflict:
			return errors.New(errors.CodeBiz, apiErr.Message, http.StatusConflict)
		}
		slog.Error("claim action: 上游错误", "status", apiErr.Status, "code", apiErr.Code, "msg", apiErr.Message)
	default:
		slog.Warn("claim action: catalog 不可达", "error", err)
	}
	return errors.New(errors.CodeBiz, "资料库服务暂不可用", http.StatusServiceUnavailable)
}

var mineStates = []string{
	catalogclient.ClaimStatePending,
	catalogclient.ClaimStateDeclined,
	catalogclient.ClaimStateDraft,
}

func (s *SubmissionService) ListMine(
	ctx context.Context,
	accessToken string,
	query url.Values,
) (*catalogclient.UserClaimPage, *errors.AppError) {
	states := mineStates
	if raw := query.Get("claim_state"); raw != "" {
		states = splitCSV(raw)
	}
	page, err := s.catalog.MyClaims(ctx, accessToken, catalogclient.UserClaimFilter{
		ClaimStates: states,
		Before:      int64(atoiOr(query.Get("before"), 0)),
		Limit:       atoiOr(query.Get("limit"), 20),
		Kind:        "submitted",
	})
	if err != nil {
		return nil, claimActionError(err)
	}
	if page.Items == nil {
		page.Items = []catalogclient.UserClaimItem{}
	}
	return page, nil
}

func (s *SubmissionService) ListAudit(
	ctx context.Context,
	accessToken string,
	query url.Values,
) (*catalogclient.UserClaimPage, *errors.AppError) {
	page, err := s.catalog.MyClaims(ctx, accessToken, catalogclient.UserClaimFilter{
		Before: int64(atoiOr(query.Get("before"), 0)),
		Limit:  atoiOr(query.Get("limit"), 20),
		Kind:   "audited",
	})
	if err != nil {
		return nil, claimActionError(err)
	}
	if page.Items == nil {
		page.Items = []catalogclient.UserClaimItem{}
	}
	return page, nil
}

const wizardSearchInclude = "names,covers,refs"

const wizardDefaultLimit = 12

type WizardSearchPage struct {
	Items   []client.GalgameBrief         `json:"items"`
	Pending []catalogclient.UserClaimItem `json:"pending"`
	Total   int64                         `json:"total"`
}

func (s *SubmissionService) SearchWithPending(
	ctx context.Context,
	accessToken string,
	query url.Values,
) (*WizardSearchPage, *errors.AppError) {
	items, total, appErr := s.wizardItems(ctx, query)
	if appErr != nil {
		return nil, appErr
	}
	pending, appErr := s.wizardPending(ctx, accessToken)
	if appErr != nil {
		return nil, appErr
	}
	return &WizardSearchPage{Items: items, Pending: pending, Total: total}, nil
}

func (s *SubmissionService) wizardItems(
	ctx context.Context,
	query url.Values,
) ([]client.GalgameBrief, int64, *errors.AppError) {
	q := url.Values{
		"q":           {query.Get("q")},
		"page":        {strconv.Itoa(atoiOr(query.Get("page"), 1))},
		"limit":       {strconv.Itoa(atoiOr(query.Get("limit"), wizardDefaultLimit))},
		"claimed":     {"true"},
		"claim_state": {client.ClaimStateWizard},
		"include":     {wizardSearchInclude},
	}
	client.OpenPopulation(q)

	res, appErr := s.galgameClient.CatalogWorksSearch(ctx, q)
	if appErr != nil {
		return nil, 0, appErr
	}
	items := make([]client.GalgameBrief, 0, len(res.Items))
	for i := range res.Items {
		row := &res.Items[i]
		if !client.CatalogItemRenderable(row) || client.CatalogItemGID(row) == 0 {
			continue
		}
		b := client.CatalogItemToBrief(row)
		b.Banner = b.EffectiveBannerURL
		items = append(items, b)
	}
	return items, res.Total, nil
}

func (s *SubmissionService) wizardPending(
	ctx context.Context,
	accessToken string,
) ([]catalogclient.UserClaimItem, *errors.AppError) {
	page, err := s.catalog.MyClaims(ctx, accessToken, catalogclient.UserClaimFilter{
		ClaimStates: []string{catalogclient.ClaimStatePending, catalogclient.ClaimStateDeclined},
		Limit:       wizardDefaultLimit,
	})
	if err != nil {
		return nil, claimActionError(err)
	}
	if page.Items == nil {
		return []catalogclient.UserClaimItem{}, nil
	}
	return page.Items, nil
}
