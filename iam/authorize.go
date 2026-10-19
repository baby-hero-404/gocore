package iam

import "github.com/baby-hero-404/gocore/stderr"

// Subject returns the policy subject for id: its first role, or its ID when it has no roles.
func Subject(id Identity) string {
	if len(id.Roles) > 0 {
		return id.Roles[0]
	}
	return id.ID
}

// Authorize reports whether id may perform act on obj. It enforces sub first and, when denied,
// falls back to each of the identity's other roles.
func Authorize(enforcer Enforcer, id Identity, sub, obj, act string) (bool, error) {
	if enforcer == nil {
		return false, stderr.NewInternal("casbin enforcer is not configured")
	}
	allowed, err := enforcer.Enforce(sub, obj, act)
	if err != nil {
		return false, stderr.Wrap(stderr.CodeInternal, err, "policy enforcement error")
	}
	if allowed || len(id.Roles) <= 1 {
		return allowed, nil
	}
	for _, role := range id.Roles {
		if role == sub {
			continue
		}
		if match, _ := enforcer.Enforce(role, obj, act); match {
			return true, nil
		}
	}
	return false, nil
}
