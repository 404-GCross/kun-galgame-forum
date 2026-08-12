package service

import (
	"context"
	"slices"
	"strings"
	"testing"

	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/model"
	"kun-galgame-api/pkg/perm"
)

type fakeStore struct {
	rows       []model.RolePermissionOverride
	listErr    error
	replaceErr error
}

func (f *fakeStore) ListAll(_ context.Context) ([]model.RolePermissionOverride, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return append([]model.RolePermissionOverride{}, f.rows...), nil
}

func (f *fakeStore) ReplaceForRole(_ context.Context, role string, rows []model.RolePermissionOverride, operatorUID int) error {
	if f.replaceErr != nil {
		return f.replaceErr
	}
	kept := make([]model.RolePermissionOverride, 0, len(f.rows))
	for _, r := range f.rows {
		if r.Role != role {
			kept = append(kept, r)
		}
	}
	for i := range rows {
		rows[i].Role = role
		rows[i].UpdatedBy = operatorUID
	}
	f.rows = append(kept, rows...)
	return nil
}

func resetPerm(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		perm.SetOverrides(nil)
		perm.SetUserOverrides(nil)
	})
}

type emptyUserStore struct{}

func (emptyUserStore) ListAll(_ context.Context) ([]model.UserPermissionOverride, error) {
	return nil, nil
}

func newSvc(store *fakeStore) *RolePermissionService {
	return NewRolePermissionService(store, NewPermissionOverrideSync(store, emptyUserStore{}))
}

func grant(p perm.Permission) dto.ReplaceOverrideItem {
	return dto.ReplaceOverrideItem{Permission: string(p), Effect: perm.EffectGrant}
}
func revoke(p perm.Permission) dto.ReplaceOverrideItem {
	return dto.ReplaceOverrideItem{Permission: string(p), Effect: perm.EffectRevoke}
}

var renOperatorRoles = []string{"ren"}

func TestReplaceRejectsRen(t *testing.T) {
	resetPerm(t)
	svc := newSvc(&fakeStore{})
	_, appErr := svc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, "ren", []dto.ReplaceOverrideItem{revoke(TopicHideKey)})
	if appErr == nil {
		t.Fatal("editing ren must be rejected")
	}
	if appErr.StatusCode != 400 {
		t.Fatalf("ren rejection status = %d, want 400", appErr.StatusCode)
	}
}

func TestReplaceRejectsUserAndUnknownRole(t *testing.T) {
	resetPerm(t)
	svc := newSvc(&fakeStore{})
	for _, role := range []string{"user", "banana", ""} {
		if _, appErr := svc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, role, nil); appErr == nil {
			t.Errorf("role %q must be rejected as non-manageable", role)
		}
	}
}

func TestReplaceRejectsUnknownKey(t *testing.T) {
	resetPerm(t)
	svc := newSvc(&fakeStore{})
	_, appErr := svc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, "creator",
		[]dto.ReplaceOverrideItem{{Permission: "does.not.exist", Effect: perm.EffectGrant}})
	if appErr == nil {
		t.Fatal("unknown permission key must be rejected")
	}
}

func TestReplaceRejectsInvalidEffect(t *testing.T) {
	resetPerm(t)
	svc := newSvc(&fakeStore{})
	_, appErr := svc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, "creator",
		[]dto.ReplaceOverrideItem{{Permission: string(TopicHideKey), Effect: "toggle"}})
	if appErr == nil {
		t.Fatal("invalid effect must be rejected")
	}
}

func TestReplaceRejectsNoop(t *testing.T) {
	resetPerm(t)
	svc := newSvc(&fakeStore{})
	if _, appErr := svc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, "moderator",
		[]dto.ReplaceOverrideItem{grant(TopicHideKey)}); appErr == nil {
		t.Error("granting a baseline permission must be rejected as a no-op")
	}
	if _, appErr := svc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, "creator",
		[]dto.ReplaceOverrideItem{revoke(TopicHideKey)}); appErr == nil {
		t.Error("revoking a non-baseline permission must be rejected as a no-op")
	}
}

func TestReplaceRejectsDuplicate(t *testing.T) {
	resetPerm(t)
	svc := newSvc(&fakeStore{})
	_, appErr := svc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, "creator",
		[]dto.ReplaceOverrideItem{grant(TopicHideKey), grant(TopicHideKey)})
	if appErr == nil {
		t.Fatal("duplicate permission must be rejected")
	}
}

func TestReplaceContainmentViolation(t *testing.T) {
	resetPerm(t)
	store := &fakeStore{rows: []model.RolePermissionOverride{
		{Role: "admin", Permission: string(UserPurgeKey), Effect: perm.EffectRevoke},
	}}
	svc := newSvc(store)
	_, appErr := svc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, "moderator",
		[]dto.ReplaceOverrideItem{grant(UserPurgeKey)})
	if appErr == nil {
		t.Fatal("containment violation (moderator ⊄ admin) must be rejected")
	}
	if appErr.StatusCode != 400 {
		t.Fatalf("containment rejection status = %d, want 400", appErr.StatusCode)
	}
}

func TestReplaceHappyPath(t *testing.T) {
	resetPerm(t)
	store := &fakeStore{}
	svc := newSvc(store)

	matrix, appErr := svc.ReplaceOverrides(context.Background(), 7, renOperatorRoles, "creator",
		[]dto.ReplaceOverrideItem{grant(TopicHideKey)})
	if appErr != nil {
		t.Fatalf("valid replace failed: %v", appErr)
	}

	if len(store.rows) != 1 || store.rows[0].UpdatedBy != 7 {
		t.Fatalf("expected 1 row stamped by operator 7, got %+v", store.rows)
	}
	if !perm.Can([]string{"creator"}, TopicHideKey) {
		t.Error("creator should hold topic.hide immediately after a valid replace")
	}
	creator := matrix.Roles["creator"]
	if len(creator.Overrides) != 1 || creator.Overrides[0].Permission != string(TopicHideKey) {
		t.Errorf("matrix creator overrides = %+v, want one topic.hide grant", creator.Overrides)
	}
	if !contains(creator.Effective, string(TopicHideKey)) {
		t.Errorf("matrix creator effective %v missing topic.hide", creator.Effective)
	}
	if len(creator.Baseline) != 0 {
		t.Errorf("creator baseline should be empty, got %v", creator.Baseline)
	}
	if !matrix.Roles["ren"].Locked {
		t.Error("ren must be marked locked in the matrix")
	}
	if want := len(perm.Catalog()); len(matrix.Catalog) != want {
		t.Errorf("matrix catalog has %d keys, want %d", len(matrix.Catalog), want)
	}
}

func TestReplaceResetRestoresBaseline(t *testing.T) {
	resetPerm(t)
	store := &fakeStore{rows: []model.RolePermissionOverride{
		{Role: "creator", Permission: string(TopicHideKey), Effect: perm.EffectGrant},
	}}
	svc := newSvc(store)
	if _, appErr := svc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, "creator", nil); appErr != nil {
		t.Fatalf("reset failed: %v", appErr)
	}
	if len(store.rows) != 0 {
		t.Errorf("reset should delete all creator rows, got %+v", store.rows)
	}
	if perm.Can([]string{"creator"}, TopicHideKey) {
		t.Error("creator should hold nothing after a reset")
	}
}

var adminOperatorRoles = []string{"admin"}

func TestReplaceRankAdminCannotEditAdminRole(t *testing.T) {
	resetPerm(t)
	svc := newSvc(&fakeStore{})
	_, appErr := svc.ReplaceOverrides(context.Background(), 100, adminOperatorRoles, "admin",
		[]dto.ReplaceOverrideItem{revoke(UserPurgeKey)})
	if appErr == nil || appErr.StatusCode != 400 {
		t.Fatalf("admin editing the admin role must be rejected with 400, got %v", appErr)
	}
}

func TestReplaceRankRenCanEditAdminRole(t *testing.T) {
	resetPerm(t)
	svc := newSvc(&fakeStore{})
	if _, appErr := svc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, "admin",
		[]dto.ReplaceOverrideItem{revoke(perm.AdminDashboard)}); appErr != nil {
		t.Fatalf("ren editing the admin role must succeed, got %v", appErr)
	}
}

func TestReplacePossessionAddedRow(t *testing.T) {
	resetPerm(t)
	perm.SetUserOverrides(map[int][]perm.Override{
		100: {{Permission: perm.TopicHide, Effect: perm.EffectRevoke}},
	})
	svc := newSvc(&fakeStore{})
	_, appErr := svc.ReplaceOverrides(context.Background(), 100, adminOperatorRoles, "creator",
		[]dto.ReplaceOverrideItem{grant(TopicHideKey)})
	if appErr == nil || appErr.StatusCode != 400 {
		t.Fatalf("adding an unheld permission must be rejected with 400, got %v", appErr)
	}
	if !strings.Contains(appErr.Message, "不可增删自己未持有的权限") {
		t.Errorf("expected possession error, got %q", appErr.Message)
	}
}

func TestReplacePossessionCarriedOverPasses(t *testing.T) {
	resetPerm(t)
	perm.SetUserOverrides(map[int][]perm.Override{
		100: {{Permission: perm.TopicHide, Effect: perm.EffectRevoke}},
	})
	store := &fakeStore{rows: []model.RolePermissionOverride{
		{Role: "creator", Permission: string(TopicHideKey), Effect: perm.EffectGrant},
	}}
	svc := newSvc(store)
	matrix, appErr := svc.ReplaceOverrides(context.Background(), 100, adminOperatorRoles, "creator",
		[]dto.ReplaceOverrideItem{grant(TopicHideKey), grant(perm.DocEdit)})
	if appErr != nil {
		t.Fatalf("carrying over an unheld row while adding a held one must pass, got %v", appErr)
	}
	creator := matrix.Roles["creator"]
	if !contains(creator.Effective, string(TopicHideKey)) || !contains(creator.Effective, string(perm.DocEdit)) {
		t.Errorf("creator effective %v should hold both topic.hide and doc.edit", creator.Effective)
	}
}

func TestReplacePossessionRemovalChecked(t *testing.T) {
	resetPerm(t)
	perm.SetUserOverrides(map[int][]perm.Override{
		100: {{Permission: perm.TopicHide, Effect: perm.EffectRevoke}},
	})
	store := &fakeStore{rows: []model.RolePermissionOverride{
		{Role: "creator", Permission: string(TopicHideKey), Effect: perm.EffectGrant},
	}}
	svc := newSvc(store)
	_, appErr := svc.ReplaceOverrides(context.Background(), 100, adminOperatorRoles, "creator",
		[]dto.ReplaceOverrideItem{})
	if appErr == nil || appErr.StatusCode != 400 {
		t.Fatalf("removing an unheld override row must be rejected with 400, got %v", appErr)
	}
	if !strings.Contains(appErr.Message, "不可增删自己未持有的权限") {
		t.Errorf("expected possession error, got %q", appErr.Message)
	}
}

const (
	TopicHideKey = perm.TopicHide
	UserPurgeKey = perm.UserPurgeContent
)

func contains(ss []string, want string) bool {
	return slices.Contains(ss, want)
}
