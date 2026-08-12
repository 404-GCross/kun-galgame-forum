package service

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/model"
	"kun-galgame-api/internal/admin/repository"
	"kun-galgame-api/pkg/userclient"
)

const (
	auditDefaultLimit = 30
	auditMaxLimit     = 100
)

type PermissionAuditService struct {
	repo       *repository.PermissionAuditRepository
	userClient *userclient.Client
}

func NewPermissionAuditService(repo *repository.PermissionAuditRepository, userClient *userclient.Client) *PermissionAuditService {
	return &PermissionAuditService{repo: repo, userClient: userClient}
}

func (s *PermissionAuditService) List(ctx context.Context, page, limit int) (dto.PermissionAuditPage, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = auditDefaultLimit
	}
	if limit > auditMaxLimit {
		limit = auditMaxLimit
	}
	rows, total, err := s.repo.List(ctx, (page-1)*limit, limit)
	if err != nil {
		return dto.PermissionAuditPage{}, err
	}
	return dto.PermissionAuditPage{
		Total: total,
		Items: auditItems(rows),
		Users: s.operatorMap(ctx, rows),
	}, nil
}

func (s *PermissionAuditService) operatorMap(ctx context.Context, rows []model.PermissionAuditLog) map[string]dto.PermissionAuditUser {
	out := make(map[string]dto.PermissionAuditUser)
	if s.userClient == nil || len(rows) == 0 {
		return out
	}
	seen := make(map[int]bool, len(rows))
	ids := make([]int, 0, len(rows))
	for _, r := range rows {
		if r.OperatorID > 0 && !seen[r.OperatorID] {
			seen[r.OperatorID] = true
			ids = append(ids, r.OperatorID)
		}
	}
	resolved, err := s.userClient.Users(ctx, ids)
	if err != nil {
		slog.Warn("permission audit: operator enrichment failed", "error", err)
		return out
	}
	for id, u := range resolved {
		out[strconv.Itoa(id)] = dto.PermissionAuditUser{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
	}
	return out
}

func auditItems(rows []model.PermissionAuditLog) []dto.PermissionAuditItem {
	out := make([]dto.PermissionAuditItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.PermissionAuditItem{
			ID:          r.ID,
			OperatorID:  r.OperatorID,
			SubjectKind: r.SubjectKind,
			Subject:     r.Subject,
			Action:      r.Action,
			Before:      auditRows(r.BeforeRows),
			After:       auditRows(r.AfterRows),
			CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		})
	}
	return out
}

func auditRows(deltas []model.AuditDelta) []dto.PermissionAuditRow {
	out := make([]dto.PermissionAuditRow, 0, len(deltas))
	for _, d := range deltas {
		out = append(out, dto.PermissionAuditRow{Permission: d.Permission, Effect: d.Effect})
	}
	return out
}
