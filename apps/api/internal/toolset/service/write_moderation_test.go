package service

import (
	"context"
	"testing"

	"kun-galgame-api/internal/toolset/dto"
	"kun-galgame-api/internal/trust/gate"
	"kun-galgame-api/pkg/trustclient"
)

type scriptedChecker struct{ decision string }

func (c scriptedChecker) Check(_ context.Context, _ trustclient.CheckRequest) (*trustclient.CheckResult, error) {
	return &trustclient.CheckResult{Decision: c.decision, Matched: []string{"坏词"}}, nil
}

func TestCreateToolsetDeniedNothingPersisted(t *testing.T) {
	svc := NewToolsetService(
		nil, nil, nil, nil, nil, nil, nil,
		gate.NewCheckService(scriptedChecker{decision: gate.DecisionDeny}),
		gate.NewScanService(nil),
	)
	resp, appErr := svc.Create(context.Background(), 7, &dto.CreateToolsetRequest{
		Name: "某工具", Description: "简介", Type: "translator",
		Version: "1.0.0", Aliases: []string{"别名"},
	})
	if appErr == nil {
		t.Fatal("deny must return an error")
	}
	if appErr.StatusCode != 422 {
		t.Fatalf("deny status = %d, want 422", appErr.StatusCode)
	}
	if resp != nil {
		t.Fatalf("deny must persist nothing, got %+v", resp)
	}
}
