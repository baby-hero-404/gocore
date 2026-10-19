package iammock

import (
	"github.com/baby-hero-404/gocore/iam"
	"testing"
)

func TestMockEnforcerAndValidator(t *testing.T) {
	mockEnforcer := NewEnforcer(t)
	mockEnforcer.On("Enforce", "admin", "/resource", "GET").Return(true, nil)

	ok, err := mockEnforcer.Enforce("admin", "/resource", "GET")
	if err != nil || !ok {
		t.Errorf("expected enforce true, got %v, %v", ok, err)
	}

	mockValidator := NewTokenValidator(t)
	expectedIdentity := iam.Identity{ID: "usr-42", Roles: []string{"manager"}}
	mockValidator.On("Validate", t.Context(), "valid-token").Return(expectedIdentity, nil)

	id, err := mockValidator.Validate(t.Context(), "valid-token")
	if err != nil || id.ID != "usr-42" {
		t.Errorf("expected identity usr-42, got %v, %v", id, err)
	}
}
