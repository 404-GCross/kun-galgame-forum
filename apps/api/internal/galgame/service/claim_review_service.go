package service

import (
	"context"
	"net/url"
	"strconv"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/pkg/catalogclient"
	"kun-galgame-api/pkg/errors"
)

type ClaimReviewService struct {
	galgameClient *client.GalgameClient
	catalog       *catalogclient.Client
}

func NewClaimReviewService(
	galgameClient *client.GalgameClient,
	catalog *catalogclient.Client,
) *ClaimReviewService {
	return &ClaimReviewService{galgameClient: galgameClient, catalog: catalog}
}

const pendingQueueLimit = 30

type PendingQueuePage struct {
	Items      []client.CatalogWorkListItem `json:"items"`
	NextCursor string                       `json:"next_cursor"`
}

func (s *ClaimReviewService) PendingQueue(
	ctx context.Context,
	query url.Values,
) (*PendingQueuePage, *errors.AppError) {
	q := url.Values{
		"claimed":     {"true"},
		"claim_state": {catalogclient.ClaimStatePending},
		"site":        {submissionSite},
		"sort":        {"updated"},
		"limit":       {strconv.Itoa(atoiOr(query.Get("limit"), pendingQueueLimit))},
		"include":     {wizardSearchInclude},
	}
	if cursor := query.Get("cursor"); cursor != "" {
		q.Set("cursor", cursor)
	}
	client.OpenPopulation(q)
	page, appErr := s.galgameClient.CatalogWorksList(ctx, q)
	if appErr != nil {
		return nil, appErr
	}
	items := page.Items
	if items == nil {
		items = []client.CatalogWorkListItem{}
	}
	return &PendingQueuePage{Items: items, NextCursor: page.NextCursor}, nil
}

var reviewActions = map[string]bool{
	catalogclient.ClaimActionApprove: true,
	catalogclient.ClaimActionDecline: true,
	catalogclient.ClaimActionBan:     true,
	catalogclient.ClaimActionUnban:   true,
}

func (s *ClaimReviewService) Review(
	ctx context.Context,
	accessToken string,
	gid int,
	action string,
	reason string,
) (*catalogclient.ClaimActionResult, *errors.AppError) {
	if !reviewActions[action] {
		return nil, errors.ErrBadRequest("未知的审核动作")
	}
	if action == catalogclient.ClaimActionDecline && reason == "" {
		return nil, errors.ErrValidation("拒绝时必须填写理由")
	}
	ids, appErr := s.galgameClient.CatalogWorkIDs(ctx, []int{gid})
	if appErr != nil {
		return nil, appErr
	}
	workID, ok := ids[gid]
	if !ok {
		return nil, errors.ErrNotFound("条目不存在")
	}
	res, err := s.catalog.ActOnClaimUser(ctx, accessToken, workID, action, catalogclient.UserClaimActionRequest{
		Reason: reason,
	})
	if err != nil {
		return nil, claimActionError(err)
	}
	return res, nil
}
