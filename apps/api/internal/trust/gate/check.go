package gate

import (
	"context"
	"log/slog"
	"time"

	"kun-galgame-api/pkg/trustclient"
)

const checkTimeout = 500 * time.Millisecond

const (
	DecisionAllow = "allow"
	DecisionDeny  = "deny"
	DecisionHold  = "hold"
)

type Checker interface {
	Check(ctx context.Context, req trustclient.CheckRequest) (*trustclient.CheckResult, error)
}

type CheckService struct {
	ck Checker
}

func NewCheckService(ck Checker) *CheckService {
	return &CheckService{ck: ck}
}

func (s *CheckService) Enabled() bool { return s != nil && s.ck != nil }

func (s *CheckService) Decision(ctx context.Context, text string, authorID *int64) (decision string, matched []string) {
	if !s.Enabled() {
		return DecisionAllow, nil
	}
	cctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	res, err := s.ck.Check(cctx, trustclient.CheckRequest{Text: text, AuthorID: authorID})
	if err != nil {
		slog.Warn("trust check fail-open", "err", err)
		return DecisionAllow, nil
	}
	switch res.Decision {
	case DecisionDeny:
		return DecisionDeny, res.Matched
	case DecisionHold:
		return DecisionHold, res.Matched
	default:
		return DecisionAllow, res.Matched
	}
}
