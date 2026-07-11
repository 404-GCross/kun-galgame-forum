package service

import (
	"context"
	stderrors "errors"

	"kun-galgame-api/internal/trust"
	"kun-galgame-api/internal/trust/dto"
	"kun-galgame-api/pkg/errors"
	"kun-galgame-api/pkg/trustclient"
)

// TrustService is the BFF façade over the infra Trust & Safety client. Phase 1
// is a thin, generic report forwarder — no per-content-type logic lives here.
type TrustService struct {
	trust *trustclient.Client
}

func NewTrustService(trust *trustclient.Client) *TrustService {
	return &TrustService{trust: trust}
}

// Reasons returns the report-reason catalog for the browser dropdown.
func (s *TrustService) Reasons() []trust.ReportReason {
	return trust.GlobalReasons
}

// SubmitReport forwards a report to the trust service with the session user as
// the reporter. subject_kind / subject_id pass through untouched (trust owns
// validation, dedup, and rate-limiting).
func (s *TrustService) SubmitReport(
	ctx context.Context,
	reporterID int,
	req *dto.SubmitReportRequest,
) (*dto.SubmitReportResponse, *errors.AppError) {
	res, err := s.trust.SubmitReport(ctx, trustclient.ReportRequest{
		SubjectKind: req.SubjectKind,
		SubjectID:   req.SubjectID,
		ReasonKey:   req.ReasonKey,
		ReporterID:  int64(reporterID),
		Note:        req.Note,
		Snapshot:    req.Snapshot,
	})
	if err != nil {
		switch {
		case stderrors.Is(err, trustclient.ErrValidation):
			return nil, errors.ErrBadRequest("举报信息无效，或该内容暂不支持举报")
		case stderrors.Is(err, trustclient.ErrRateLimited):
			return nil, errors.New(errors.CodeBiz, "举报过于频繁，请稍后再试", 429)
		case stderrors.Is(err, trustclient.ErrNotConfigured):
			return nil, errors.ErrInternal("举报服务暂未启用")
		default:
			// ErrForbidden/ErrUnauthorized are server-side misconfig (site
			// binding / credentials), not the reporter's fault → 500.
			return nil, errors.ErrInternal("举报提交失败，请稍后再试")
		}
	}
	return &dto.SubmitReportResponse{ReportID: res.ReportID}, nil
}
