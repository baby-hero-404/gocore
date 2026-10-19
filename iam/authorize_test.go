package iam

import (
	"errors"
	"testing"

	"github.com/baby-hero-404/gocore/stderr"
)

type stubEnforcer struct {
	allow map[string]bool
	err   error
}

func (s stubEnforcer) Enforce(rvals ...any) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return s.allow[rvals[0].(string)], nil
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{"valid", "Bearer abc", "abc", false},
		{"case insensitive scheme", "bearer  abc ", "abc", false},
		{"missing", "", "", true},
		{"wrong scheme", "Basic abc", "", true},
		{"no token", "Bearer", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BearerToken(tt.header)
			var se *stderr.Error
			if tt.wantErr {
				if !errors.As(err, &se) || se.Code != stderr.CodeUnauthorized {
					t.Fatalf("got %v, want unauthorized", err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("got %q, %v; want %q", got, err, tt.want)
			}
		})
	}
}

func TestSubjectAndAuthorize(t *testing.T) {
	if Subject(Identity{ID: "u1"}) != "u1" || Subject(Identity{ID: "u1", Roles: []string{"a", "b"}}) != "a" {
		t.Error("Subject must prefer first role, then ID")
	}

	tests := []struct {
		name    string
		enf     Enforcer
		id      Identity
		sub     string
		want    bool
		wantErr bool
	}{
		{"nil enforcer", nil, Identity{}, "a", false, true},
		{"enforce error", stubEnforcer{err: errors.New("db")}, Identity{}, "a", false, true},
		{"primary role allowed", stubEnforcer{allow: map[string]bool{"a": true}}, Identity{Roles: []string{"a"}}, "a", true, false},
		{"fallback to other role", stubEnforcer{allow: map[string]bool{"b": true}}, Identity{Roles: []string{"a", "b"}}, "a", true, false},
		{"denied", stubEnforcer{allow: map[string]bool{}}, Identity{Roles: []string{"a", "b"}}, "a", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Authorize(tt.enf, tt.id, tt.sub, "/x", "GET")
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Errorf("got %v, %v; want %v, err=%v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}
