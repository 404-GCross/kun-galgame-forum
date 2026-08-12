package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/model"
	"kun-galgame-api/pkg/perm"
	"kun-galgame-api/pkg/userclient"
)

type fakeUserStore struct {
	rows []model.UserPermissionOverride
}

func (f *fakeUserStore) ListAll(_ context.Context) ([]model.UserPermissionOverride, error) {
	return append([]model.UserPermissionOverride{}, f.rows...), nil
}

func (f *fakeUserStore) ListForUser(_ context.Context, uid int) ([]model.UserPermissionOverride, error) {
	out := make([]model.UserPermissionOverride, 0)
	for _, r := range f.rows {
		if r.UserID == uid {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeUserStore) ReplaceForUser(_ context.Context, uid int, rows []model.UserPermissionOverride, operatorUID int) error {
	kept := make([]model.UserPermissionOverride, 0, len(f.rows))
	for _, r := range f.rows {
		if r.UserID != uid {
			kept = append(kept, r)
		}
	}
	for i := range rows {
		rows[i].UserID = uid
		rows[i].UpdatedBy = operatorUID
	}
	f.rows = append(kept, rows...)
	return nil
}

type emptyRoleStore struct{}

func (emptyRoleStore) ListAll(_ context.Context) ([]model.RolePermissionOverride, error) {
	return nil, nil
}

type fakeUserClient struct {
	roles []string
	found bool
	err   error
}

func (f fakeUserClient) User(_ context.Context, id int) (userclient.User, bool, error) {
	if f.err != nil {
		return userclient.User{}, false, f.err
	}
	if !f.found {
		return userclient.User{}, false, nil
	}
	return userclient.User{ID: id, Roles: f.roles}, true, nil
}

func newUserSvc(store *fakeUserStore, client userLookup) *UserPermissionService {
	return NewUserPermissionService(store, client, NewPermissionOverrideSync(emptyRoleStore{}, store))
}

const targetUID = 4242

func TestUserReplaceRejectsRenHolder(t *testing.T) {
	resetPerm(t)
	svc := newUserSvc(&fakeUserStore{}, fakeUserClient{roles: []string{"ren"}, found: true})
	_, appErr := svc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, targetUID, []dto.ReplaceOverrideItem{revoke(TopicHideKey)})
	if appErr == nil || appErr.StatusCode != 400 {
		t.Fatalf("ren holder must be rejected with 400, got %v", appErr)
	}
}

func TestUserReplaceFailClosed(t *testing.T) {
	resetPerm(t)
	svc := newUserSvc(&fakeUserStore{}, fakeUserClient{err: errors.New("oauth down")})
	if _, appErr := svc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, targetUID, nil); appErr == nil {
		t.Error("a lookup error must fail closed")
	}
	svc2 := newUserSvc(&fakeUserStore{}, fakeUserClient{found: false})
	if _, appErr := svc2.ReplaceOverrides(context.Background(), 1, renOperatorRoles, targetUID, nil); appErr == nil {
		t.Error("a nonexistent user must fail closed")
	}
}

func TestUserReplaceRejectsUnknownKeyAndEffect(t *testing.T) {
	resetPerm(t)
	svc := newUserSvc(&fakeUserStore{}, fakeUserClient{found: true})
	if _, appErr := svc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, targetUID,
		[]dto.ReplaceOverrideItem{{Permission: "does.not.exist", Effect: perm.EffectGrant}}); appErr == nil {
		t.Error("unknown permission must be rejected")
	}
	if _, appErr := svc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, targetUID,
		[]dto.ReplaceOverrideItem{{Permission: string(TopicHideKey), Effect: "toggle"}}); appErr == nil {
		t.Error("invalid effect must be rejected")
	}
}

func TestUserReplaceRejectsDuplicate(t *testing.T) {
	resetPerm(t)
	svc := newUserSvc(&fakeUserStore{}, fakeUserClient{found: true})
	if _, appErr := svc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, targetUID,
		[]dto.ReplaceOverrideItem{grant(TopicHideKey), grant(TopicHideKey)}); appErr == nil {
		t.Error("duplicate permission must be rejected")
	}
}

func TestUserReplaceRejectsNoop(t *testing.T) {
	resetPerm(t)
	modSvc := newUserSvc(&fakeUserStore{}, fakeUserClient{roles: []string{"moderator"}, found: true})
	if _, appErr := modSvc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, targetUID,
		[]dto.ReplaceOverrideItem{grant(TopicHideKey)}); appErr == nil {
		t.Error("granting a role-held permission must be rejected as a no-op")
	}
	plainSvc := newUserSvc(&fakeUserStore{}, fakeUserClient{roles: nil, found: true})
	if _, appErr := plainSvc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, targetUID,
		[]dto.ReplaceOverrideItem{revoke(TopicHideKey)}); appErr == nil {
		t.Error("revoking a permission the role lacks must be rejected as a no-op")
	}
}

func TestUserReplaceGrantToRoleless(t *testing.T) {
	resetPerm(t)
	store := &fakeUserStore{}
	svc := newUserSvc(store, fakeUserClient{roles: nil, found: true})

	view, appErr := svc.ReplaceOverrides(context.Background(), 9, renOperatorRoles, targetUID,
		[]dto.ReplaceOverrideItem{grant(TopicHideKey)})
	if appErr != nil {
		t.Fatalf("valid grant failed: %v", appErr)
	}
	if len(store.rows) != 1 || store.rows[0].UpdatedBy != 9 {
		t.Fatalf("expected 1 row stamped by operator 9, got %+v", store.rows)
	}
	if !perm.CanUser(targetUID, nil, TopicHideKey) {
		t.Error("roleless user should hold topic.hide immediately after the grant")
	}
	if len(view.RoleEffective) != 0 {
		t.Errorf("role_effective should be empty for a roleless user, got %v", view.RoleEffective)
	}
	if len(view.Overrides) != 1 || view.Overrides[0].Permission != string(TopicHideKey) {
		t.Errorf("view overrides = %+v, want one topic.hide grant", view.Overrides)
	}
	if !contains(view.Effective, string(TopicHideKey)) {
		t.Errorf("view effective %v missing topic.hide", view.Effective)
	}
}

func TestUserReplaceResetRestores(t *testing.T) {
	resetPerm(t)
	store := &fakeUserStore{rows: []model.UserPermissionOverride{
		{UserID: targetUID, Permission: string(TopicHideKey), Effect: perm.EffectGrant},
	}}
	perm.SetUserOverrides(map[int][]perm.Override{
		targetUID: {{Permission: TopicHideKey, Effect: perm.EffectGrant}},
	})
	svc := newUserSvc(store, fakeUserClient{roles: nil, found: true})

	if _, appErr := svc.ReplaceOverrides(context.Background(), 1, renOperatorRoles, targetUID, nil); appErr != nil {
		t.Fatalf("reset failed: %v", appErr)
	}
	if len(store.rows) != 0 {
		t.Errorf("reset should delete all rows, got %+v", store.rows)
	}
	if perm.CanUser(targetUID, nil, TopicHideKey) {
		t.Error("roleless user should hold nothing after a reset")
	}
}

func TestUserReplaceRankAdminCannotEditPeerAdmin(t *testing.T) {
	resetPerm(t)
	svc := newUserSvc(&fakeUserStore{}, fakeUserClient{roles: []string{"admin"}, found: true})
	_, appErr := svc.ReplaceOverrides(context.Background(), 100, adminOperatorRoles, targetUID,
		[]dto.ReplaceOverrideItem{grant(TopicHideKey)})
	if appErr == nil || appErr.StatusCode != 400 {
		t.Fatalf("admin editing a peer admin must be rejected with 400, got %v", appErr)
	}
	if !strings.Contains(appErr.Message, "不可编辑与自身同级或更高的用户") {
		t.Errorf("expected rank error, got %q", appErr.Message)
	}
}
