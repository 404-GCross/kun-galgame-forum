package service

import (
	"context"
	"slices"
	"testing"

	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/model"
	"kun-galgame-api/pkg/perm"
)

// fakeStore is an in-memory overrideStore for validating the service without a
// database. ReplaceForRole mirrors the real replace-all-for-role semantics.
type fakeStore struct {
	rows       []model.RolePermissionOverride
	listErr    error
	replaceErr error
}

func (f *fakeStore) ListAll(_ context.Context) ([]model.RolePermissionOverride, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	// Return a copy so callers can't mutate our backing slice.
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

// resetPerm restores the pure baseline after any test that calls ReplaceOverrides
// (which write-through Loads into the global pkg/perm table).
func resetPerm(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		perm.SetOverrides(nil)
		perm.SetUserOverrides(nil)
	})
}

// emptyUserStore is a no-op user-override lister so the shared sync can refresh
// the (empty) user layer during role-service tests.
type emptyUserStore struct{}

func (emptyUserStore) ListAll(_ context.Context) ([]model.UserPermissionOverride, error) {
	return nil, nil
}

// newSvc builds a RolePermissionService whose write-through reloads from the SAME
// fakeStore via a real PermissionOverrideSync, so perm.Can reflects a replace
// immediately — exactly as the production wiring does.
func newSvc(store *fakeStore) *RolePermissionService {
	return NewRolePermissionService(store, NewPermissionOverrideSync(store, emptyUserStore{}))
}

func grant(p perm.Permission) dto.ReplaceOverrideItem {
	return dto.ReplaceOverrideItem{Permission: string(p), Effect: perm.EffectGrant}
}
func revoke(p perm.Permission) dto.ReplaceOverrideItem {
	return dto.ReplaceOverrideItem{Permission: string(p), Effect: perm.EffectRevoke}
}

// TestReplaceRejectsRen proves ren is not editable — pinned to the full catalog.
func TestReplaceRejectsRen(t *testing.T) {
	resetPerm(t)
	svc := newSvc(&fakeStore{})
	_, appErr := svc.ReplaceOverrides(context.Background(), 1, "ren", []dto.ReplaceOverrideItem{revoke(TopicHideKey)})
	if appErr == nil {
		t.Fatal("editing ren must be rejected")
	}
	if appErr.StatusCode != 400 {
		t.Fatalf("ren rejection status = %d, want 400", appErr.StatusCode)
	}
}

// TestReplaceRejectsUserAndUnknownRole proves user (implicit) and any unknown
// role are not manageable.
func TestReplaceRejectsUserAndUnknownRole(t *testing.T) {
	resetPerm(t)
	svc := newSvc(&fakeStore{})
	for _, role := range []string{"user", "banana", ""} {
		if _, appErr := svc.ReplaceOverrides(context.Background(), 1, role, nil); appErr == nil {
			t.Errorf("role %q must be rejected as non-manageable", role)
		}
	}
}

// TestReplaceRejectsUnknownKey proves an out-of-catalog permission is rejected.
func TestReplaceRejectsUnknownKey(t *testing.T) {
	resetPerm(t)
	svc := newSvc(&fakeStore{})
	_, appErr := svc.ReplaceOverrides(context.Background(), 1, "creator",
		[]dto.ReplaceOverrideItem{{Permission: "does.not.exist", Effect: perm.EffectGrant}})
	if appErr == nil {
		t.Fatal("unknown permission key must be rejected")
	}
}

// TestReplaceRejectsInvalidEffect proves an effect other than grant/revoke is
// rejected.
func TestReplaceRejectsInvalidEffect(t *testing.T) {
	resetPerm(t)
	svc := newSvc(&fakeStore{})
	_, appErr := svc.ReplaceOverrides(context.Background(), 1, "creator",
		[]dto.ReplaceOverrideItem{{Permission: string(TopicHideKey), Effect: "toggle"}})
	if appErr == nil {
		t.Fatal("invalid effect must be rejected")
	}
}

// TestReplaceRejectsNoop proves granting a baseline key or revoking a
// non-baseline key is a validation error (the FE computes true deltas).
func TestReplaceRejectsNoop(t *testing.T) {
	resetPerm(t)
	svc := newSvc(&fakeStore{})
	// moderator already holds topic.hide → granting it is a no-op.
	if _, appErr := svc.ReplaceOverrides(context.Background(), 1, "moderator",
		[]dto.ReplaceOverrideItem{grant(TopicHideKey)}); appErr == nil {
		t.Error("granting a baseline permission must be rejected as a no-op")
	}
	// creator holds nothing → revoking topic.hide is a no-op.
	if _, appErr := svc.ReplaceOverrides(context.Background(), 1, "creator",
		[]dto.ReplaceOverrideItem{revoke(TopicHideKey)}); appErr == nil {
		t.Error("revoking a non-baseline permission must be rejected as a no-op")
	}
}

// TestReplaceRejectsDuplicate proves a duplicated permission in one request is
// rejected.
func TestReplaceRejectsDuplicate(t *testing.T) {
	resetPerm(t)
	svc := newSvc(&fakeStore{})
	_, appErr := svc.ReplaceOverrides(context.Background(), 1, "creator",
		[]dto.ReplaceOverrideItem{grant(TopicHideKey), grant(TopicHideKey)})
	if appErr == nil {
		t.Fatal("duplicate permission must be rejected")
	}
}

// TestReplaceContainmentViolation proves the moderator ⊆ admin invariant is
// enforced: with admin having revoked user.purge_content, granting it to
// moderator would make moderator exceed admin → 400.
func TestReplaceContainmentViolation(t *testing.T) {
	resetPerm(t)
	store := &fakeStore{rows: []model.RolePermissionOverride{
		{Role: "admin", Permission: string(UserPurgeKey), Effect: perm.EffectRevoke},
	}}
	svc := newSvc(store)
	_, appErr := svc.ReplaceOverrides(context.Background(), 1, "moderator",
		[]dto.ReplaceOverrideItem{grant(UserPurgeKey)})
	if appErr == nil {
		t.Fatal("containment violation (moderator ⊄ admin) must be rejected")
	}
	if appErr.StatusCode != 400 {
		t.Fatalf("containment rejection status = %d, want 400", appErr.StatusCode)
	}
}

// TestReplaceHappyPath proves a valid replace persists, write-through Loads into
// pkg/perm (Can reflects it at once), and returns a matrix showing the effective
// recompute.
func TestReplaceHappyPath(t *testing.T) {
	resetPerm(t)
	store := &fakeStore{}
	svc := newSvc(store)

	matrix, appErr := svc.ReplaceOverrides(context.Background(), 7, "creator",
		[]dto.ReplaceOverrideItem{grant(TopicHideKey)})
	if appErr != nil {
		t.Fatalf("valid replace failed: %v", appErr)
	}

	// Persisted with the operator stamp.
	if len(store.rows) != 1 || store.rows[0].UpdatedBy != 7 {
		t.Fatalf("expected 1 row stamped by operator 7, got %+v", store.rows)
	}
	// Write-through: the global resolver reflects the grant immediately.
	if !perm.Can([]string{"creator"}, TopicHideKey) {
		t.Error("creator should hold topic.hide immediately after a valid replace")
	}
	// Matrix reflects the override + the effective recompute.
	creator := matrix.Roles["creator"]
	if len(creator.Overrides) != 1 || creator.Overrides[0].Permission != string(TopicHideKey) {
		t.Errorf("matrix creator overrides = %+v, want one topic.hide grant", creator.Overrides)
	}
	if !contains(creator.Effective, string(TopicHideKey)) {
		t.Errorf("matrix creator effective %v missing topic.hide", creator.Effective)
	}
	// creator baseline is empty — the grant is an override, not a baseline key.
	if len(creator.Baseline) != 0 {
		t.Errorf("creator baseline should be empty, got %v", creator.Baseline)
	}
	// ren row is locked and full.
	if !matrix.Roles["ren"].Locked {
		t.Error("ren must be marked locked in the matrix")
	}
	if len(matrix.Catalog) != 43 {
		t.Errorf("matrix catalog has %d keys, want 43", len(matrix.Catalog))
	}
}

// TestReplaceResetRestoresBaseline proves an empty override set resets a role to
// its baseline (delete-only) and clears its effect on pkg/perm.
func TestReplaceResetRestoresBaseline(t *testing.T) {
	resetPerm(t)
	store := &fakeStore{rows: []model.RolePermissionOverride{
		{Role: "creator", Permission: string(TopicHideKey), Effect: perm.EffectGrant},
	}}
	svc := newSvc(store)
	if _, appErr := svc.ReplaceOverrides(context.Background(), 1, "creator", nil); appErr != nil {
		t.Fatalf("reset failed: %v", appErr)
	}
	if len(store.rows) != 0 {
		t.Errorf("reset should delete all creator rows, got %+v", store.rows)
	}
	if perm.Can([]string{"creator"}, TopicHideKey) {
		t.Error("creator should hold nothing after a reset")
	}
}

// Local aliases so the test reads clearly without importing perm constants twice.
const (
	TopicHideKey = perm.TopicHide
	UserPurgeKey = perm.UserPurgeContent
)

func contains(ss []string, want string) bool {
	return slices.Contains(ss, want)
}
